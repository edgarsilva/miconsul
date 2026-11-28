// Package objstoretest provides object stores for tests.
package objstoretest

import (
	"context"
	"io"

	"miconsul/internal/lib/objstore"
)

// Failing is a Store whose every operation returns Err.
type Failing struct {
	Err error
}

var _ objstore.Store = Failing{}

// Put returns f.Err.
func (f Failing) Put(context.Context, string, io.Reader, int64, string) error {
	return f.Err
}

// Open returns f.Err.
func (f Failing) Open(context.Context, string) (objstore.Object, error) {
	return objstore.Object{}, f.Err
}

// Delete returns f.Err.
func (f Failing) Delete(context.Context, string) error {
	return f.Err
}
