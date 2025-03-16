package web

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var (
	fileHashes   = map[string]string{}
	fallbackHash = fmt.Sprintf("%x", time.Now().Unix())
)

func getFileHash(fs fs.FS, path string) string {
	if hash, ok := fileHashes[path]; ok {
		return hash
	}

	file, err := fs.Open(path)
	if err != nil {
		log.Printf("failed to open file %q for hashing: %v", path, err)
		return fallbackHash
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		log.Printf("failed to read file %q for hashing: %v", path, err)
		return fallbackHash
	}

	hash := fmt.Sprintf("%x", hasher.Sum([]byte{}))
	fileHashes[path] = hash

	return hash
}

type Template struct {
	t    template.Template
	data TemplateData
}

func (t *Template) Execute(w io.Writer, data any) error {
	t.data.Data = data
	return t.t.Execute(w, t.data)
}

func (t *Template) ExecuteFragment(w io.Writer, name string, data any) error {
	t.data.Data = data
	return t.t.ExecuteTemplate(w, name, t.data)
}

type Templater struct {
	templateFS fs.FS
	assetsFS   fs.FS
	funcs      template.FuncMap
}

func NewTemplater(templateFS, assetsFS fs.FS) *Templater {
	return &Templater{
		templateFS: templateFS,
		assetsFS:   assetsFS,
		funcs: template.FuncMap{
			"asset": func(assetPath string) string {
				// TODO: This breaks on Windows
				assetPath = path.Join("assets", assetPath)
				hash := getFileHash(assetsFS, assetPath)
				return path.Join("/", assetPath+"?h="+hash)
			},
		},
	}
}

// TODO: Don't include base.html per default.
func (t *Templater) Get(patterns ...string) (*Template, error) {
	templatePaths := make([]string, len(patterns)+1)
	templatePaths[0] = "templates/base.html.tmpl"
	for i := range patterns {
		templatePaths[i+1] = path.Join("templates", patterns[i]+".tmpl")
	}

	tmpl, err := template.New("base.html.tmpl").
		Funcs(t.funcs).
		ParseFS(t.templateFS, templatePaths...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %v: %w", templatePaths, err)
	}

	return &Template{t: *tmpl}, nil
}

func (t *Templater) GetP(patterns ...string) *Template {
	template, err := t.Get(patterns...)
	if err != nil {
		panic(err)
	}

	return template
}

func (t *Templater) Write(writer io.Writer, msg string, data any, patterns ...string) error {
	tmpl, err := t.Get(patterns...)
	if err != nil {
		return err
	}
	tmpl.data.Msg = msg

	if err := tmpl.Execute(writer, data); err != nil {
		return fmt.Errorf("failed to execute template %v: %w", patterns, err)
	}

	return nil
}

const templatePathUrlParam = "templatePath"

type TemplateData struct {
	Msg  string
	Data any
}

type TemplateModule struct {
	templater *Templater
	data      any
}

func NewTemplateModule(templateFS, assetsFS fs.FS, data any) *TemplateModule {
	return &TemplateModule{
		templater: NewTemplater(templateFS, assetsFS),
		data:      data,
	}
}

func (m *TemplateModule) Controller() *Controller {
	c := Controller{
		BasePath: "/",
		Handlers: map[Endpoint]Handler{
			{Method: http.MethodGet, Path: "/"}:                                  m.getTemplate(),
			{Method: http.MethodGet, Path: "/{" + templatePathUrlParam + ":.*}"}: m.getTemplate(),
		},
		Middleware: []HandlerProvider{middleware.Compress(5)},
	}

	return &c
}

func (m *TemplateModule) getTemplate() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		templatePath := chi.URLParam(r, templatePathUrlParam)
		if templatePath == "" {
			templatePath = "index.html"
		}

		msg := r.URL.Query().Get("msg")
		w.Header().Add(ContentTypeHeader, ContentTypeHTML)

		if err := m.templater.Write(w, msg, m.data, templatePath); err != nil {
			log.Printf("failed to execute template at path %q: %v", templatePath, err)
			w.WriteHeader(http.StatusNotFound)
			return m.templater.Write(w, "", m.data, "not-found.html")
		}

		return nil
	}
}
