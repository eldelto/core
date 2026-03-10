package web

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func RenderTemplate(template *template.Template, data any) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return template.Execute(w, data)
	}
}

type Templater struct {
	templateFS fs.FS
	assetsFS   fs.FS
	funcs      template.FuncMap
}

var (
	fileHashes   = map[string]string{} // TODO: Use sync.Map instead
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

func asset(assetsFS fs.FS) func(string) string {
	return func(assetPath string) string {
		assetPath = path.Join("assets", assetPath)
		hash := getFileHash(assetsFS, assetPath)
		return path.Join("/", assetPath+"?h="+hash)
	}
}

func isURL(s string) bool {
	url, err := url.Parse(s)
	return err == nil && url.Scheme != ""
}

func NewTemplater(templateFS, assetsFS fs.FS) *Templater {
	return &Templater{
		templateFS: templateFS,
		assetsFS:   assetsFS,
		funcs: template.FuncMap{
			"asset": asset(assetsFS),
			"isURL": isURL,
		},
	}
}

// TODO: Don't include base.html per default.
func (t *Templater) Get(patterns ...string) (*template.Template, error) {
	baseTemplate := "base.html.tmpl"
	baseTemplatePath := path.Join("templates", baseTemplate)

	_, err := fs.Stat(t.templateFS, baseTemplatePath)
	includeBase := err == nil

	templatePaths := make([]string, 0, len(patterns)+1)
	if includeBase {
		templatePaths = append(templatePaths, baseTemplatePath)
	}

	for _, pattern := range patterns {
		templatePaths = append(templatePaths,
			path.Join("templates", pattern+".tmpl"))
	}

	rootTemplate := patterns[0] + ".tmpl"
	if includeBase {
		rootTemplate = baseTemplate
	}

	tmpl, err := template.New(rootTemplate).
		Funcs(t.funcs).
		ParseFS(t.templateFS, templatePaths...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %v: %w", templatePaths, err)
	}

	return tmpl, nil
}

func (t *Templater) GetP(patterns ...string) *template.Template {
	template, err := t.Get(patterns...)
	if err != nil {
		panic(err)
	}

	return template
}

func (t *Templater) Write(writer io.Writer, data any, patterns ...string) error {
	tmpl, err := t.Get(patterns...)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(writer, data); err != nil {
		return fmt.Errorf("failed to execute template %v: %w", patterns, err)
	}

	return nil
}

const templatePathUrlParam = "templatePath"

func NewTemplateModule(templateFS, assetsFS fs.FS, data any) chi.Router {
	templater := NewTemplater(templateFS, assetsFS)

	r := chi.NewRouter()
	eh := NewErrorHandlers()

	r.Use(middleware.Compress(5))
	r.Get("/", eh.Handle(getTemplate(templater, data)))
	r.Get("/{"+templatePathUrlParam+":.*}", eh.Handle(getTemplate(templater, data)))

	return r
}

func getTemplate(templater *Templater, data any) Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		templatePath := chi.URLParam(r, templatePathUrlParam)
		if templatePath == "" {
			templatePath = "index.html"
		}

		// TODO: Get from context
		msg := r.URL.Query().Get("msg")
		// TODO: Use ContentTypeMiddleware
		w.Header().Add(ContentTypeHeader, ContentTypeHTML)
		data := map[string]any{
			"data": data,
			"msg":  msg,
		}

		if err := templater.Write(w, data, templatePath); err != nil {
			log.Printf("failed to execute template at path %q: %v", templatePath, err)
			w.WriteHeader(http.StatusNotFound)
			return templater.Write(w, data, "not-found.html")
		}

		return nil
	}
}
