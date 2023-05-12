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

func (t *Templater) Get(templatePath string) (*template.Template, error) {
	tmpl, err := template.ParseFS(t.fileSystem, path.Join("templates", templatePath+".tmpl"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	return tmpl, nil
}

func (t *Templater) Write(templatePath string, writer io.Writer, data any) error {
	tmpl, err := t.Get(templatePath)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(writer, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templatePath, err)
	}

	return nil
}
