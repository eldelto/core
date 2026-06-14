package fileshare

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strconv"

	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
)

var (
	ErrInvalidPath = errors.New("invalid path")

	templater         = web.NewTemplater(TemplatesFS, AssetsFS, "templates")
	directoryTemplate = templater.GetP("directory.html")

	mailTemplater     = web.NewTemplater(TemplatesFS, nil, "templates/emails")
	mailLoginTemplate = mailTemplater.GetP("login.html")
)

func invalidPath(path string) error {
	return fmt.Errorf("path %q: %w", path, ErrInvalidPath)
}

func NewDirectoryController(service *Service) chi.Router {
	r := chi.NewRouter()
	r.Get("/*", errorHandlers.Handle(getPath(service)))
	r.Post("/download", errorHandlers.Handle(download(service)))
	r.Post("/upload", errorHandlers.Handle(initFile(service)))
	r.Put("/upload/{reference}", errorHandlers.Handle(addFileChunk(service)))
	r.Post("/upload/{reference}", errorHandlers.Handle(commitFile(service)))
	// TODO:
	// - User handling and restricting them to their own service dir
	// - Move multiple
	r.Post("/directory", errorHandlers.Handle(createDirectory(service)))
	r.Post("/delete", errorHandlers.Handle(deleteFiles(service)))

	return r
}

type directoryData struct {
	CurrentPath string
	ParentPath  string
	CurrentURL  *url.URL
	Entries     []fs.FileInfo
}

func listDirectoryContent(w http.ResponseWriter, r *http.Request, service *Service) error {
	dirPath := chi.URLParam(r, "*")
	entries, err := service.ListEntries(r.Context(), dirPath)
	if err != nil {
		return err
	}

	data := directoryData{
		CurrentPath: path.Clean(dirPath),
		ParentPath:  path.Dir(r.URL.Path),
		CurrentURL:  r.URL,
		Entries:     entries,
	}

	return directoryTemplate.Execute(w, data)
}

func preview(w http.ResponseWriter, r *http.Request, service *Service) error {
	dirPath := chi.URLParam(r, "*")
	root, err := service.userRoot(r.Context())
	if err != nil {
		return err
	}

	w.Header().Set(web.ContentDispositionHeader, "inline")
	http.ServeFileFS(w, r, root.FS(), dirPath)
	return nil
}

func getPath(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		isPreview := r.URL.Query().Get("preview")
		if isPreview == "true" {
			return preview(w, r, service)
		}

		return listDirectoryContent(w, r, service)
	}
}

func createDirectory(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := service.CreateDirectory(r.Context(),
			r.PostFormValue("path")); err != nil {
			return err
		}

		referrer := r.Header.Get(web.ReferrerHeader)
		http.Redirect(w, r, referrer, http.StatusSeeOther)
		return nil
	}
}

func deleteFiles(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		root, err := service.userRoot(r.Context())
		if err != nil {
			return err
		}

		if err := r.ParseForm(); err != nil {
			return err
		}
		paths := r.Form["paths"]

		for _, p := range paths {
			p = filepath.Clean(p)
			fmt.Println(p)
			if err := root.RemoveAll(p); err != nil {
				return err
			}
		}

		referrer := r.Header.Get(web.ReferrerHeader)
		http.Redirect(w, r, referrer, http.StatusSeeOther)
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

func download(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		root, err := service.userRoot(r.Context())
		if err != nil {
			return err
		}
		fsys := root.FS()

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

const multipartMemory = 2 * 1024 * 1024

func initFile(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// TODO:
		// - Generic form parser/validator

		if err := r.ParseMultipartForm(multipartMemory); err != nil {
			return err
		}

		path := r.MultipartForm.Value["path"][0]
		name := r.MultipartForm.Value["name"][0]
		size, err := strconv.ParseUint(r.MultipartForm.Value["size"][0], 10, 64)
		if err != nil {
			return err
		}

		reference, err := service.InitFile(r.Context(), path, name, uint(size))
		if err != nil {
			return err
		}

		w.Header().Set(web.ContentTypeHeader, web.ContentTypeText)
		_, err = fmt.Fprint(w, reference)
		return err
	}
}

func addFileChunk(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		reference := chi.URLParam(r, "reference")

		if err := r.ParseMultipartForm(multipartMemory); err != nil {
			return err
		}

		chunkHeader := r.MultipartForm.File["chunk"][0]
		chunk, err := chunkHeader.Open()
		if err != nil {
			return err
		}
		defer chunk.Close()

		return service.AddFileChunk(r.Context(), reference, chunk)
	}
}

func commitFile(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		reference := chi.URLParam(r, "reference")
		return service.CommitFile(r.Context(), reference)
	}
}
