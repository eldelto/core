package web

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"
)

type Templater struct {
	fileSystem fs.FS
}

func NewTemplater(fileSystem fs.FS) *Templater {
	return &Templater{fileSystem: fileSystem}
}

func (t *Templater) Get(patterns ...string) (*template.Template, error) {
	templatePaths := make([]string, len(patterns))
	for i := range patterns {
		templatePaths[i] = path.Join("templates", patterns[i]+".tmpl")
	}

	tmpl, err := template.ParseFS(t.fileSystem, templatePaths...)
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
