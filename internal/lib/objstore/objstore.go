// Package objstore stores and serves uploaded files by key.
package objstore

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"time"
)

var (
	ErrNotFound   = errors.New("object not found")
	ErrInvalidKey = errors.New("invalid object key")
)

// Store keeps objects addressed by key, for example "clinics/<file>".
type Store interface {
	// Put stores size bytes from body under key with the given content type,
	// replacing any object already there.
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	// Open returns the object under key. The caller closes Object.Body. It
	// returns ErrNotFound when there is no such object.
	Open(ctx context.Context, key string) (Object, error)
	// Delete removes the object under key. A missing object is not an error.
	Delete(ctx context.Context, key string) error
}

// Object is an opened object and its metadata.
type Object struct {
	Body        io.ReadCloser
	Size        int64
	ModTime     time.Time
	ContentType string
}

const (
	maxKeyLen     = 1024
	maxSegmentLen = 255
)

// ValidKey reports whether key is a valid object key on every backend: one or
// more "/"-separated segments of lowercase letters, digits, dot, dash and
// underscore, none empty or starting with a dot, each at most 255 bytes and
// the whole key at most 1024 bytes.
func ValidKey(key string) bool {
	if len(key) > maxKeyLen || !fs.ValidPath(key) {
		return false
	}

	for seg := range strings.SplitSeq(key, "/") {
		if !validSegment(seg) {
			return false
		}
	}

	return true
}

func validSegment(seg string) bool {
	if seg == "" || len(seg) > maxSegmentLen || seg[0] == '.' {
		return false
	}

	for i := 0; i < len(seg); i++ {
		switch c := seg[i]; {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		case c == '.', c == '-', c == '_':
		default:
			return false
		}
	}

	return true
}
