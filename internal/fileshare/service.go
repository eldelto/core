package fileshare

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/mail"
	"os"
	"path/filepath"

	"github.com/eldelto/core/auth"
	"github.com/eldelto/core/internal/legacyweb"
	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/google/uuid"
)

const (
	userDataBucket = "user-data"
	chunkBucket    = "chunked-file"
)

type UserData struct {
	ID      uuid.UUID
	HomeDir string
}

func (ud *UserData) BucketKey() []byte {
	return []byte(ud.ID.String())
}

type chunkedFile struct {
	ID       uuid.UUID
	Path     string
	TempPath string
	Name     string
	Size     uint
}

func newChunkedFile(path, name string, size uint) (chunkedFile, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return chunkedFile{}, err
	}

	f, err := os.CreateTemp("", name)
	if err != nil {
		return chunkedFile{}, err
	}
	f.Close()

	return chunkedFile{
		ID:       id,
		Path:     path,
		TempPath: f.Name(),
		Name:     name,
		Size:     size,
	}, nil
}

func (cf *chunkedFile) BucketKey() []byte {
	return []byte(cf.ID.String())
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

	err := s.db.Write(func(tx storage.WriteTx) error {
		if err := s.root.Mkdir(dir, 0744); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create new home dir %q: %w", dir, err)
		}

		return s.setHomeDir(tx, authn, authn, dir)
	})

	return dir, err
}

func (s *Service) getHomeDir(auth legacyweb.Auth) (string, error) {
	var data UserData
	err := s.db.Read(func(tx storage.ReadTx) error {
		d, err := storage.Load[*UserData](tx, userDataBucket, []byte(auth.UserID().String()))
		data = *d
		if err != nil {
			return fmt.Errorf("get home dir for %q: %w", auth.UserID(), err)
		}
		return nil
	})
	if errors.Is(err, storage.ErrNotFound) {
		return s.initUser(auth)
	}

	return data.HomeDir, err
}

func (s *Service) setHomeDir(tx storage.WriteTx, authn, toModify legacyweb.Auth, dir string) error {
	data := UserData{
		ID:      toModify.UserID().UUID,
		HomeDir: dir,
	}
	return storage.Store(tx, userDataBucket, &data, auth.UserID(authn.UserID()))
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
		return nil, fmt.Errorf("open user root dir for %q: %w", auth.UserID(), err)
	}

	return root, nil
}

func (s *Service) ListEntries(ctx context.Context, path string) ([]fs.FileInfo, error) {
	root, err := s.userRoot(ctx)
	if err != nil {
		return nil, err
	}
	fsys := root.FS()

	path = filepath.Clean(path)
	entries, err := fs.ReadDir(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("list entries in dir %q: %w", path, err)
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

	path = filepath.Clean(path)
	if err := root.Mkdir(path, 0744); err != nil {
		return fmt.Errorf("create directory %q for user %q",
			path, auth.UserID())
	}

	return nil
}

func (s *Service) InitFile(ctx context.Context,
	path,
	name string,
	size uint) (string, error) {
	authn, err := legacyweb.GetAuth(ctx)
	if err != nil {
		return "", err
	}

	chunkedFile, err := newChunkedFile(path, name, uint(size))
	if err != nil {
		return "", err
	}

	err = s.db.Write(func(tx storage.WriteTx) error {
		return storage.Store(tx, chunkBucket, &chunkedFile, auth.UserID(authn.UserID()))
	})
	if err != nil {
		return "", fmt.Errorf("init chunked file: user=%q, path=%q, err=%w",
			authn.UserID(), path, err)
	}

	return chunkedFile.ID.String(), nil
}

func (s *Service) AddFileChunk(ctx context.Context, reference string, r io.Reader) error {
	// TODO: Chunked file should be in a user bucket.

	var cFile *chunkedFile
	err := s.db.Read(func(tx storage.ReadTx) error {
		cf, err := storage.Load[*chunkedFile](tx, chunkBucket, []byte(reference))
		cFile = cf
		return err
	})
	if err != nil {
		return fmt.Errorf("add file chunk: reference=%q, err=%w", reference, err)
	}

	path := filepath.Clean(cFile.TempPath)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open chunked file: reference=%q, path=%q, err=%w",
			reference, path, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("copy file chunk: reference=%q, path=%q, err=%w",
			reference, path, err)
	}

	return nil
}
func (s *Service) CommitFile(ctx context.Context, reference string) error {
	root, err := s.userRoot(ctx)
	if err != nil {
		return err
	}

	var cFile chunkedFile
	err = s.db.Read(func(tx storage.ReadTx) error {
		cf, err := storage.Load[*chunkedFile](tx, chunkBucket, []byte(reference))
		cFile = *cf
		return err
	})
	if err != nil {
		return fmt.Errorf("commit file: reference=%q, err=%w",
			reference, err)
	}

	temp, err := os.Open(filepath.Clean(cFile.TempPath))
	if err != nil {
		return fmt.Errorf("open temp file to commit: reference=%q, err=%w",
			reference, err)
	}
	defer temp.Close()

	path := filepath.Join(cFile.Path, cFile.Name)
	f, err := root.Create(path)
	if err != nil {
		return fmt.Errorf("create destination file: reference=%q, err=%w",
			reference, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, temp); err != nil {
		return fmt.Errorf("move temp file: reference=%q, err=%w",
			reference, err)
	}

	return nil
}

func (s *Service) DeletePaths(ctx context.Context, paths []string) error {
	root, err := s.userRoot(ctx)
	if err != nil {
		return err
	}

	for _, p := range paths {
		p = filepath.Clean(p)
		if err := root.RemoveAll(p); err != nil {
			return fmt.Errorf("delete path: path=%q, err=%q", p, err)
		}
	}

	return nil
}

func zipFile(zipper *zip.Writer, f fs.File, name string) error {
	w, err := zipper.Create(name)
	if err != nil {
		return fmt.Errorf("zip file: name=%q, err=%w", name, err)
	}

	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("copy file to zip: name=%q, err=%w", name, err)
	}
	return nil
}

// This is basically a copy of zip.Writer.AddFS.
func zipFS(zipper *zip.Writer, fsys fs.FS, prefix string) error {
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
		return fmt.Errorf("open zip item: path=%q, err=%w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat zip item: path=%q, err=%w", path, err)
	}

	if info.IsDir() {
		dir, err := fs.Sub(fsys, path)
		if err != nil {
			return fmt.Errorf("create sub fs to zip: path=%q, err=%w", path, err)
		}

		if err := zipFS(zipper, dir, info.Name()); err != nil {
			return fmt.Errorf("zip dir: path=%q, err=%w", path, err)
		}
		return nil
	}

	return zipFile(zipper, f, info.Name())
}

func (s *Service) ZipPaths(ctx context.Context, w io.Writer, paths []string) error {
	root, err := s.userRoot(ctx)
	if err != nil {
		return err
	}
	fsys := root.FS()

	zipper := zip.NewWriter(w)
	defer zipper.Close()
	for _, p := range paths {
		if err := addZipItem(zipper, fsys, p); err != nil {
			return err
		}
	}

	return nil
}
