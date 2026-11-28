package server

import (
	"errors"
	"fmt"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/objstore"
)

// ErrObjectStoreBackendNotImplemented is returned by NewObjectStore for a backend
// that has no implementation yet.
var ErrObjectStoreBackendNotImplemented = errors.New("object store backend is not implemented")

// NewObjectStore returns the object store for env.ObjectStoreBackend: files under
// env.AssetsDir for disk.
func NewObjectStore(env *appenv.Env) (objstore.Store, error) {
	if env == nil {
		return nil, errors.New("environment config is required")
	}

	switch env.ObjectStoreBackend {
	case appenv.ObjectStoreBackendDisk:
		store, err := objstore.NewDisk(env.AssetsDir)
		if err != nil {
			return nil, fmt.Errorf("create disk object store: %w", err)
		}

		return store, nil
	case appenv.ObjectStoreBackendS3:
		return nil, fmt.Errorf("%w: %s", ErrObjectStoreBackendNotImplemented, env.ObjectStoreBackend)
	default:
		return nil, fmt.Errorf("unknown object store backend %q", env.ObjectStoreBackend)
	}
}
