package objstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
)

var _ Store = (*Disk)(nil)

// Disk is a Store that keeps objects as files under a root directory. Every
// file operation goes through an os.Root, which keeps keys inside the
// directory, symbolic links included.
type Disk struct {
	root *os.Root
}

// NewDisk returns a Disk rooted at dir and creates dir when it does not exist.
func NewDisk(dir string) (*Disk, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, errors.New("disk store requires a root directory")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create disk store root: %w", err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open disk store root: %w", err)
	}

	return &Disk{root: root}, nil
}

// Close releases the root directory.
func (d *Disk) Close() error {
	return d.root.Close()
}

// Put writes body to a temporary file in the directory of key and renames it
// over the file for key. Readers never see a partial object. It creates missing
// directories.
func (d *Disk) Put(_ context.Context, key string, body io.Reader, _ int64, _ string) error {
	if !ValidKey(key) {
		return ErrInvalidKey
	}

	dir := path.Dir(key)
	if dir != "." {
		if err := d.root.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create object directory: %w", err)
		}
	}

	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return fmt.Errorf("name temporary object file: %w", err)
	}
	tmpName := path.Join(dir, ".put-"+hex.EncodeToString(suffix))

	tmp, err := d.root.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create temporary object file: %w", err)
	}

	_, err = io.Copy(tmp, body)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		d.discard(tmpName)
		return fmt.Errorf("write object: %w", err)
	}

	if err := d.root.Rename(tmpName, key); err != nil {
		d.discard(tmpName)
		return fmt.Errorf("store object: %w", err)
	}

	return nil
}

// Open returns the file for key. Its content type comes from the file
// extension, or from the first bytes when there is no known extension.
func (d *Disk) Open(_ context.Context, key string) (Object, error) {
	if !ValidKey(key) {
		return Object{}, ErrInvalidKey
	}

	f, err := d.root.Open(key)
	if errors.Is(err, os.ErrNotExist) {
		return Object{}, ErrNotFound
	}
	if err != nil {
		return Object{}, fmt.Errorf("open object: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		return Object{}, closeAfter(f, fmt.Errorf("stat object: %w", err))
	}
	if info.IsDir() {
		return Object{}, closeAfter(f, ErrNotFound)
	}

	contentType := mime.TypeByExtension(path.Ext(key))
	if contentType == "" {
		head := make([]byte, 512)
		n, err := f.Read(head)
		if err != nil && !errors.Is(err, io.EOF) {
			return Object{}, closeAfter(f, fmt.Errorf("read object: %w", err))
		}
		contentType = http.DetectContentType(head[:n])
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return Object{}, closeAfter(f, fmt.Errorf("rewind object: %w", err))
		}
	}

	return Object{
		Body:        f,
		Size:        info.Size(),
		ModTime:     info.ModTime(),
		ContentType: contentType,
	}, nil
}

// Delete removes the file for key.
func (d *Disk) Delete(_ context.Context, key string) error {
	if !ValidKey(key) {
		return ErrInvalidKey
	}

	err := d.root.Remove(key)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete object: %w", err)
	}

	return nil
}

// discard removes the temporary file name and ignores a failure to do so.
func (d *Disk) discard(name string) {
	_ = d.root.Remove(name)
}

// closeAfter closes f and returns err.
func closeAfter(f *os.File, err error) error {
	_ = f.Close()
	return err
}
