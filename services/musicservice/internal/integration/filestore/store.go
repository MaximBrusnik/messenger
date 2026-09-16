package filestore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Store persists track files on disk.
type Store struct {
	dir string
}

func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create music dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Save writes the reader body to a uniquely named file and returns its name.
func (s *Store) Save(ext string, body io.Reader) (filename string, size int64, err error) {
	tmp, err := os.CreateTemp(s.dir, "upload-*"+ext)
	if err != nil {
		return "", 0, err
	}
	defer tmp.Close()
	n, err := io.Copy(tmp, body)
	if err != nil {
		_ = os.Remove(tmp.Name())
		return "", 0, err
	}
	return filepath.Base(tmp.Name()), n, nil
}

// Path resolves a stored filename to an absolute path.
func (s *Store) Path(filename string) string {
	return filepath.Join(s.dir, filename)
}

// Remove deletes a stored file.
func (s *Store) Remove(filename string) {
	_ = os.Remove(filepath.Join(s.dir, filename))
}
