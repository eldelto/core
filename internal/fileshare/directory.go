package fileshare

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
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

func NewDirectoryController(db *storage.Storage, root *os.Root) chi.Router {
	fileSystem := root.FS()
	r := chi.NewRouter()
	r.Get("/*", errorHandlers.Handle(getPath(fileSystem)))
	r.Post("/*", errorHandlers.Handle(upload(root)))
	// TODO:
	// - User handling and restricting them to their own root dir
	// - Upload multiple
	// - Delete multiple
	// - Move multiple
	// - Create directory
	r.Delete("/*", errorHandlers.Handle(remove(root)))

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

func preview(w http.ResponseWriter, r *http.Request, fileSystem fs.FS) error {
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

	// TODO:
	// - Do we want a custom preview?
	// - Should we just mingle the mime-types so more files can be
	//   previewed? (e.g. text/csv becomes text)

	return nil
}

func download(w http.ResponseWriter, r *http.Request, fileSystem fs.FS) error {
	filename := path.Base(r.URL.Path)
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

	w.Header().Add(web.ContentDisposition, "attachment;filename=\""+filename+"\"")
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

func upload(root *os.Root) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		mr, err := r.MultipartReader()
		if err != nil {
			return err
		}

		path := chi.URLParam(r, "*")
		// TODO: Can you escape by uploading a symbolic link?
		if strings.Contains(path, "..") {
			return invalidPath(path)
		}
		if path == "" {
			path = "."
		}

		fmt.Println("base path")
		fmt.Println(path)

		for {
			// TODO: Move to function so the defers work.
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			defer part.Close()

			path := filepath.Join(path, filepath.Base(part.FileName()))
			fmt.Println(path)
			dst, err := root.Create(path)
			if err != nil {
				return err
			}
			defer dst.Close()

			if _, err := io.Copy(dst, part); err != nil {
				return err
			}
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return nil
	}
}

func remove(root *os.Root) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		mr, err := r.MultipartReader()
		if err != nil {
			return err
		}

		path := chi.URLParam(r, "*")
		// TODO: Can you escape by uploading a symbolic link?
		if strings.Contains(path, "..") {
			return invalidPath(path)
		}
		if path == "" {
			path = "."
		}

		fmt.Println("base path")
		fmt.Println(path)

		for {
			// TODO: Move to function so the defers work.
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			defer part.Close()

			path := filepath.Join(path, filepath.Base(part.FileName()))
			fmt.Println(path)
			dst, err := root.Create(path)
			if err != nil {
				return err
			}
			defer dst.Close()

			if _, err := io.Copy(dst, part); err != nil {
				return err
			}
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return nil
	}
}
