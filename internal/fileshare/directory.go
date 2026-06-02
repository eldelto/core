package fileshare

import (
	"archive/zip"
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
	fsys := root.FS()
	r := chi.NewRouter()
	r.Get("/*", errorHandlers.Handle(getPath(fsys)))
	r.Post("/*", errorHandlers.Handle(upload(root)))
	r.Post("/download", errorHandlers.Handle(download(fsys)))
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
	CurrentPath string
	ParentPath  string
	CurrentURL  *url.URL
	Entries     []fs.FileInfo
}

func listDirectoryContent(w http.ResponseWriter, r *http.Request, fsys fs.FS) error {
	dirPath := chi.URLParam(r, "*")
	// TODO: Can you escape by uploading a symbolic link?
	if strings.Contains(dirPath, "..") {
		return invalidPath(dirPath)
	}
	if dirPath == "" {
		dirPath = "."
	}

	// TODO: restrict to user's home path
	// fs.Sub(fs, dir)

	entries, err := fs.ReadDir(fsys, dirPath)
	if err != nil {
		return err
	}

	infos := make([]fs.FileInfo, len(entries))
	for i := range entries {
		info, err := entries[i].Info()
		if err != nil {
			return err
		}
		infos[i] = info
	}

	data := directoryData{
		CurrentPath: dirPath,
		ParentPath:  path.Dir(r.URL.Path),
		CurrentURL:  r.URL,
		Entries:     infos,
	}

	return directoryTemplate.Execute(w, data)
}

func preview(w http.ResponseWriter, r *http.Request, fsys fs.FS) error {
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

	w.Header().Set(web.ContentDispositionHeader, "inline")
	http.ServeFileFS(w, r, fsys, path)
	return nil
}

func getPath(fsys fs.FS) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		isPreview := r.URL.Query().Get("preview")
		if isPreview == "true" {
			return preview(w, r, fsys)
		}

		return listDirectoryContent(w, r, fsys)
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

func addZipFile(zipper *zip.Writer, f fs.File, name string) error {
	w, err := zipper.Create(name)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, f)
	return err
}

// This is basically a copy of zip.Writer.AddFS.
func addFS(zipper *zip.Writer, fsys fs.FS, prefix string) error {
	return fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !d.IsDir() && !info.Mode().IsRegular() {
			return errors.New("zip: cannot add non-regular file")
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = filepath.Join(prefix, name)
		if d.IsDir() {
			h.Name += "/"
		}
		h.Method = zip.Deflate
		fw, err := zipper.CreateHeader(h)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := fsys.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(fw, f)
		return err
	})
}

func addZipItem(zipper *zip.Writer, fsys fs.FS, path string) error {
	path = filepath.Clean(path)
	f, err := fsys.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	dir, err := fs.Sub(fsys, path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return addFS(zipper, dir, info.Name())
	}

	return addZipFile(zipper, f, info.Name())
}

func zipPaths(w io.Writer, fsys fs.FS, paths []string) error {
	zipper := zip.NewWriter(w)
	defer zipper.Close()
	for _, p := range paths {
		if err := addZipItem(zipper, fsys, p); err != nil {
			return err
		}
	}

	return nil
}

func isSingleFile(fsys fs.FS, paths []string) (bool, error) {
	if len(paths) > 1 {
		return false, nil
	}

	info, err := fs.Stat(fsys, filepath.Clean(paths[0]))
	if err != nil {
		return false, err
	}
	return !info.IsDir(), nil
}

func downloadFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, path string) error {
	path = filepath.Clean(path)

	info, err := fs.Stat(fsys, path)
	if err != nil {
		return err
	}

	w.Header().Set(web.ContentDispositionHeader,
		`attachment; filename="`+info.Name()+`"`)
	http.ServeFileFS(w, r, fsys, path)
	return nil
}

func download(fsys fs.FS) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return err
		}
		paths := r.Form["paths"]
		singleFile, err := isSingleFile(fsys, paths)
		if err != nil {
			return err
		}

		if singleFile {
			return downloadFile(w, r, fsys, paths[0])
		}

		return zipPaths(w, fsys, paths)
	}
}
