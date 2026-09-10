// Package fs provides a temporary filesystem that cleans up after itself.
package fs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/spf13/afero"
)

// TempFS is an afero.Fs backed by a newly created temporary directory.
// Close removes the directory and its contents.
type TempFS struct {
	// Fs is the underlying filesystem rooted at the temporary directory.
	afero.Fs
	tempDir string
	files   []string
}

// NewTempFS creates a TempFS under dir using MkdirTemp's pattern.
func NewTempFS(dir, pattern string) (*TempFS, error) {
	tempDir, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &TempFS{tempDir: tempDir, Fs: afero.NewBasePathFs(afero.NewOsFs(), tempDir)}, nil
}

// OpenFile opens filename, creating parent directories as needed, and records
// the path so it appears in List.
func (t *TempFS) OpenFile(filename string, flag int, perm os.FileMode) (afero.File, error) {
	dir := path.Dir(filename)
	_, err := t.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		if err = t.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	f, err := t.Fs.OpenFile(filename, flag, perm)
	if err != nil {
		return nil, err
	}
	t.files = append(t.files, filename)
	return f, nil
}

// Open opens the named file for reading using the underlying filesystem.
func (t *TempFS) Open(filename string) (fs.File, error) {
	return t.Fs.Open(filename)
}

// List returns a copy of the file paths opened through OpenFile.
func (t TempFS) List() []string {
	files := make([]string, len(t.files))
	copy(files, t.files)
	return files
}

// Close removes the temporary directory and all of its contents.
func (t *TempFS) Close() error {
	if t.tempDir == "" {
		return errors.New("os: DirFS with empty root")
	}
	if t.tempDir == "/" || t.tempDir == "./" {
		return fmt.Errorf("The template dir is invalid: %s", t.tempDir)
	}
	return os.RemoveAll(t.tempDir)
}
