package boltfs

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"go.etcd.io/bbolt"
	"time"
)

type ByteFile struct {
	cursor int
	path   string
	data   []byte
}

var _ fs.File = &ByteFile{data: []byte{}}

func (bf *ByteFile) Read(data []byte) (int, error) {
	len := copy(data, bf.data)
	if len <= 0 {
		return 0, io.EOF
	}
	bf.cursor += len

	return len, nil
}

func (bf *ByteFile) Stat() (fs.FileInfo, error) {
	return bf, nil
}

func (bf *ByteFile) Close() error {
	return nil
}

func (bf *ByteFile) Name() string {
	return filepath.Base(bf.path)
}

func (bf *ByteFile) Size() int64 {
	return int64(len(bf.data))
}

func (bf *ByteFile) Mode() fs.FileMode {
	return 0
}

func (bf *ByteFile) ModTime() time.Time {
	return time.Time{}
}

func (bf *ByteFile) IsDir() bool {
	return false
}

func (bf *ByteFile) Sys() any {
	return nil
}

type BoltFS struct {
  bucket string
	*bbolt.DB
}

func NewBoltFS(db *bbolt.DB, bucketKey string) *BoltFS {
  return &BoltFS{
    bucket: bucketKey,
	  DB: db,
  }
}

var _ fs.FS = &BoltFS{}

func (f *BoltFS) Open(path string) (fs.File, error) {
	dir := filepath.Dir(path)
	filename := filepath.Base(path)
	file := ByteFile{path: path}

	err := f.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(dir))
		if bucket == nil {
			return fmt.Errorf("bucket %q does not exist", dir)
		}

		content := bucket.Get([]byte(filename))
		if content == nil {
			return fmt.Errorf("key %q in bucket %q does not exist", filename, dir)
		}

		file.data = make([]byte, len(content))
		copy(file.data, content)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start read-only transaction: %w", err)
	}

	return nil, nil
}
