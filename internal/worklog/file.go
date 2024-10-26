package worklog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	errSkippedEntry = errors.New("entry is invalid and will be skipped")
)

func parseFile(path string, start, end time.Time, skipUnsupported bool) ([]Entry, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer r.Close()

	if strings.HasSuffix(path, ".csv") {
		return parseFromCSV(r, start, end)
	} else if strings.HasSuffix(path, ".org") {
		return parseFromOrg(r, start, end)
	}

	if skipUnsupported {
		return []Entry{}, nil
	}

	return nil, fmt.Errorf("unsupported file type %q", path)
}

func parseDir(path string, start, end time.Time) ([]Entry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %q: %w", path, err)
	}

	result := []Entry{}
	for _, e := range entries {
		if e.IsDir() || e.Name()[0] == '.' {
			continue
		}

		filePath := filepath.Join(path, e.Name())
		entries, err := parseFile(filePath, start, end, true)
		if err != nil {
			return nil, err
		}
		result = append(result, entries...)
	}

	return result, nil
}

func isDir(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open %q: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to get stats of %q: %w", path, err)
	}

	return info.IsDir(), nil
}

type FileSource struct {
	path string
}

func NewFileSource(path string) *FileSource {
	return &FileSource{
		path: path,
	}
}

func (s *FileSource) Name() string {
	return "file source"
}

func (s *FileSource) ValidIdentifier(identifier string) bool {
	return true
}

func (s *FileSource) FetchEntries(start, end time.Time) ([]Entry, error) {
	isDir, err := isDir(s.path)
	if err != nil {
		return nil, err
	}

	if isDir {
		return parseDir(s.path, start, end)
	}

	return parseFile(s.path, start, end, false)
}
