package fileshare

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/internal/legacyweb"
	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

func NewDirectoryController(db *storage.Storage, service *Service) chi.Router {
	r := chi.NewRouter()
	r.Get("/*", errorHandlers.Handle(getPath(service)))
	r.Post("/*", errorHandlers.Handle(upload(service)))
	r.Post("/download", errorHandlers.Handle(download(service)))
	r.Post("/upload", errorHandlers.Handle(initFile(db)))
	r.Put("/upload/{reference}", errorHandlers.Handle(addFileChunk(db)))
	r.Post("/upload/{reference}", errorHandlers.Handle(commitFile(db, service)))
	// TODO:
	// - User handling and restricting them to their own service dir
	// - Move multiple
	// - Create directory
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
	if dirPath == "" {
		dirPath = "."
	}

	entries, err := service.ListEntries(r.Context(), dirPath)
	if err != nil {
		return err
	}

	data := directoryData{
		CurrentPath: dirPath,
		ParentPath:  path.Dir(r.URL.Path),
		CurrentURL:  r.URL,
		Entries:     entries,
	}

	return directoryTemplate.Execute(w, data)
}

func preview(w http.ResponseWriter, r *http.Request, service *Service) error {
	path := chi.URLParam(r, "*")
	// TODO: Can you escape by uploading a symbolic link?
	if strings.Contains(path, "..") {
		return invalidPath(path)
	}
	if path == "" {
		path = "."
	}

	root, err := service.userRoot(r.Context())
	if err != nil {
		return err
	}

	w.Header().Set(web.ContentDispositionHeader, "inline")
	http.ServeFileFS(w, r, root.FS(), path)
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

func upload(service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		root, err := service.userRoot(r.Context())
		if err != nil {
			return err
		}

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

type ChunkedFile struct {
	ID       uuid.UUID
	Path     string
	TempPath string
	Name     string
	Size     uint
}

func NewChunkedFile(path, name string, size uint) (ChunkedFile, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return ChunkedFile{}, err
	}

	f, err := os.CreateTemp("", name)
	if err != nil {
		return ChunkedFile{}, err
	}
	f.Close()

	return ChunkedFile{
		ID:       id,
		Path:     path,
		TempPath: f.Name(),
		Name:     name,
		Size:     size,
	}, nil
}

func (cf *ChunkedFile) Bucket() string {
	return "chunked-file"
}

func (cf *ChunkedFile) BucketKey() []byte {
	return []byte(cf.ID.String())
}

const multipartMemory = 2 * 1024 * 1024

func initFile(db *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// TODO:
		// - Validate path is allowed for upload
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

		chunkedFile, err := NewChunkedFile(path, name, uint(size))
		if err != nil {
			return err
		}

		user := auth.UserID{}

		err = db.Write(func(tx *storage.Tx) error {
			return storage.Store(tx, &chunkedFile, user)
		})
		if err != nil {
			return err
		}

		w.Header().Set(web.ContentTypeHeader, web.ContentTypeText)
		_, err = fmt.Fprint(w, chunkedFile.ID)
		return err
	}
}

func addFileChunk(db *storage.Storage) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		reference := chi.URLParam(r, "reference")

		if err := r.ParseMultipartForm(multipartMemory); err != nil {
			return err
		}

		var chunkedFile *ChunkedFile
		err := db.Read(func(tx *storage.Tx) error {
			cf, err := storage.Load[*ChunkedFile](tx, []byte(reference))
			chunkedFile = cf
			return err
		})
		if err != nil {
			return err
		}

		f, err := os.OpenFile(chunkedFile.TempPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		chunkHeader := r.MultipartForm.File["chunk"][0]
		chunk, err := chunkHeader.Open()
		if err != nil {
			return err
		}
		defer chunk.Close()

		_, err = io.Copy(f, chunk)
		return err
	}
}

func commitFile(db *storage.Storage, service *Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		root, err := service.userRoot(r.Context())
		if err != nil {
			return err
		}

		reference := chi.URLParam(r, "reference")

		var chunkedFile ChunkedFile
		err = db.Read(func(tx *storage.Tx) error {
			cf, err := storage.Load[*ChunkedFile](tx, []byte(reference))
			chunkedFile = *cf
			return err
		})
		if err != nil {
			return err
		}

		temp, err := os.Open(chunkedFile.TempPath)
		if err != nil {
			return err
		}
		defer temp.Close()

		path := filepath.Join(chunkedFile.Path, chunkedFile.Name)
		f, err := root.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(f, temp)
		return err
	}
}

type UserData struct {
	ID      uuid.UUID
	HomeDir string
}

func (ud *UserData) Bucket() string {
	return "user-data"
}

func (ud *UserData) BucketKey() []byte {
	return []byte(ud.ID.String())
}

type Service struct {
	db     *storage.Storage
	root   *os.Root
	mailer web.Mailer
}

func NewService(db *storage.Storage, root *os.Root,
	mailer web.Mailer) *Service {
	return &Service{
		db:     db,
		root:   root,
		mailer: mailer,
	}
}
func (s *Service) initUser(authn legacyweb.Auth) (string, error) {
	userAuth, ok := authn.(*legacyweb.UserAuth)
	if !ok {
		return "", fmt.Errorf("not a valid user auth")
	}
	dir := userAuth.Email.String()

	err := s.db.Write(func(tx *storage.Tx) error {
		if err := s.root.Mkdir(dir, 0744); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create new home dir %v: %w", dir, err)
		}

		return s.setHomeDir(tx, authn, authn, dir)
	})

	return dir, err
}

func (s *Service) getHomeDir(auth legacyweb.Auth) (string, error) {
	var data UserData
	err := s.db.Read(func(tx *storage.Tx) error {
		d, err := storage.Load[*UserData](tx, []byte(auth.UserID().String()))
		data = *d
		if err != nil {
			return fmt.Errorf("get home dir for %v: %w", auth.UserID(), err)
		}
		return nil
	})
	if errors.Is(err, storage.ErrNotFound) {
		return s.initUser(auth)
	}

	return data.HomeDir, err
}

func (s *Service) setHomeDir(tx *storage.Tx, authn, toModify legacyweb.Auth, dir string) error {
	data := UserData{
		ID:      toModify.UserID().UUID,
		HomeDir: dir,
	}
	return storage.Store(tx, &data, auth.UserID(authn.UserID()))
}

func (s *Service) userRoot(ctx context.Context) (*os.Root, error) {
	auth, err := legacyweb.GetAuth(ctx)
	if err != nil {
		return nil, err
	}

	homeDir, err := s.getHomeDir(auth)
	if err != nil {
		return nil, err
	}

	root, err := s.root.OpenRoot(homeDir)
	if err != nil {
		return nil, fmt.Errorf("open user root dir for %v: %w", auth.UserID(), err)
	}

	return root, nil
}

func (s *Service) ListEntries(ctx context.Context, path string) ([]fs.FileInfo, error) {
	root, err := s.userRoot(ctx)
	if err != nil {
		return nil, err
	}
	fsys := root.FS()

	entries, err := fs.ReadDir(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("list entries in dir %v: %w", path, err)
	}

	infos := make([]fs.FileInfo, len(entries))
	for i := range entries {
		info, err := entries[i].Info()
		if err != nil {
			return nil, err
		}
		infos[i] = info
	}

	return infos, nil
}

type loginData struct {
	Token auth.TokenID
}

func (s Service) SendLoginEmail(recipient mail.Address, token legacyweb.TokenID) error {
	data := loginData{Token: auth.TokenID(token)}
	sender, err := mail.ParseAddress("no-reply@eldelto.net")
	if err != nil {
		return fmt.Errorf("send login E-mail: %w", err)
	}

	fmt.Println(data)
	return s.mailer.Send(*sender, recipient, mailLoginTemplate, data)
}

func (s *Service) CreateDirectory(ctx context.Context, path string) error {
	auth, err := legacyweb.GetAuth(ctx)
	if err != nil {
		return err
	}

	root, err := s.userRoot(ctx)
	if err != nil {
		return err
	}

	if err := root.Mkdir(path, 0744); err != nil {
		return fmt.Errorf("create directory %v for user %v",
			path, auth.UserID())
	}

	return nil
}
