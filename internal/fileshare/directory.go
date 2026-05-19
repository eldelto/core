package fileshare

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
)

var (
	ErrInvalidPath = errors.New("invalid path")

	templater         = web.NewTemplater(TemplatesFS, AssetsFS)
	directoryTemplate = templater.GetP("directory.html")
)

func invalidPath(path string) error {
	return fmt.Errorf("path %q: %w", path, ErrInvalidPath)
}

func NewDirectoryController(db *storage.Storage, fileSystem fs.FS) chi.Router {
	r := chi.NewRouter()
	r.Get("/*", errorHandlers.Handle(getPath(fileSystem)))

	return r
}

type directoryData struct {
	CurrentURL *url.URL
	Entries    []fs.DirEntry
}

func listDirectoryContent(w http.ResponseWriter, r *http.Request, fileSystem fs.FS) error {
	path := chi.URLParam(r, "*")
	// TODO: Can you escape by uploading a symbolic link?
	if strings.Contains(path, "..") {
		return invalidPath(path)
	}
	if path == "" {
		path = "."
	}

	// TODO: restrict to user's home path
	// fs.Sub(fs, dir)

	entries, err := fs.ReadDir(fileSystem, path)
	if err != nil {
		return err
	}

	data := directoryData{
		CurrentURL: r.URL,
		Entries:    entries,
	}

	return directoryTemplate.Execute(w, data)
}

func download(w http.ResponseWriter, r *http.Request, fileSystem fs.FS) error {
	path := chi.URLParam(r, "*")
	// TODO: Can you escape by uploading a symbolic link?
	if strings.Contains(path, "..") {
		return invalidPath(path)
	}
	if path == "" {
		path = "."
	}

	// TODO: restrict to user's home path
	// fs.Sub(fs, dir)

	http.ServeFileFS(w, r, fileSystem, path)
	return nil
}

func getPath(fileSystem fs.FS) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		isDownload := r.URL.Query().Get("download")
		if isDownload == "true" {
			return download(w, r, fileSystem)
		}

		return listDirectoryContent(w, r, fileSystem)
	}
}
