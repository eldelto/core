package fileshare

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
)

const pathUrlParam = "path"
var (
	InvalidPath = errors.New("invalid path")
	
	templater        = web.NewTemplater(TemplatesFS, AssetsFS)
	directoryTemplate = templater.GetP("directory.html")
)

func invalidPath(path string) error {
	return fmt.Errorf("path %q: %w", path, InvalidPath)
}

func NewDirectoryController(db *storage.Storage) chi.Router {
	r := chi.NewRouter()
	r.Get("/", errorHandlers.Handle(getPath()))
	r.Get("/{path:.*}", errorHandlers.Handle(getPath()))

	return r
}

type directoryData struct {
	CurrentPath string
	Entries []fs.DirEntry
}

func getPath() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		path := chi.URLParam(r, pathUrlParam)
		// TODO: Can you escape by uploading a symbolic link?
		if strings.Contains(path, "..") {
			return invalidPath(path)
		}
		if path == "" {
			path = "."
		}

		fmt.Println(path)

		// TODO: restrict to user's home path
		// fs.Sub(fs, dir)

		// TODO: Use real file system
		root, err := os.OpenRoot(".")
		if err != nil {
			return err
		}
		fileSystem := root.FS()

		entries, err := fs.ReadDir(fileSystem, path)
		if err != nil {
			return err
		}

		data := directoryData{
			CurrentPath: ,
		}

		return directoryTemplate.Execute(w, entries)
	}
}
