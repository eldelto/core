package web

import (
	"crypto/sha256"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"path"
	"time"
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
			"asset": func(path string) string {
				path = "/assets/" + path
				hash := getFileHash(assetsFS, path)
				return path + "?h=" + hash
			},
		},
	}
}

func (t *Templater) Get(patterns ...string) (*template.Template, error) {
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
