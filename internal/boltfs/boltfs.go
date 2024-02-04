package boltfs

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"

	"time"

	"go.etcd.io/bbolt"
)

type ByteFile struct {
	cursor int
	path   string
	data   []byte
}

var _ fs.File = &ByteFile{data: []byte{}}

func (bf *ByteFile) Read(data []byte) (int, error) {
	log.Println("reading")
	len := copy(data, bf.data[bf.cursor:])
	if len <= 0 {
		return 0, io.EOF
	}
	bf.cursor += len

	return len, nil
}

func (bf *ByteFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		bf.cursor = int(offset)
	case io.SeekCurrent:
		bf.cursor += int(offset)
	case io.SeekEnd:
		bf.cursor = (len(bf.data) - 1) - int(offset)
	default:
		panic(fmt.Errorf("unknown 'whence': %d", whence))
	}

	if bf.cursor < 0 || bf.cursor >= len(bf.data) {
		return 0, fmt.Errorf("cursor value %d is outside of bounds: %w",
			bf.cursor, io.EOF)
	}

	return int64(bf.cursor), nil
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
	bucket []byte
	*bbolt.DB
}

func NewBoltFS(db *bbolt.DB, bucketKey []byte) *BoltFS {
	return &BoltFS{
		bucket: bucketKey,
		DB:     db,
	}
}

var _ fs.FS = &BoltFS{}

func (f *BoltFS) Open(path string) (fs.File, error) {
	log.Println("opening: " + path)
	filename := filepath.Base(path)
	file := ByteFile{path: path}

	err := f.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(f.bucket)
		if bucket == nil {
			return fmt.Errorf("bucket %q does not exist", f.bucket)
		}

		content := bucket.Get([]byte(filename))
		if content == nil {
			return fmt.Errorf("key %q in bucket %q does not exist", filename, f.bucket)
		}

		log.Println("before copy")
		file.data = make([]byte, len(content))
		copy(file.data, content)
		log.Println("after copy")

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start read-only transaction: %w", err)
	}

	return &file, nil
}
