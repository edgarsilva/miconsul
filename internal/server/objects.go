package server

import (
	"errors"
	"fmt"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/objstore"
)

// ErrStorageBackendNotImplemented is returned by NewObjectStore for a backend
// that has no implementation yet.
var ErrStorageBackendNotImplemented = errors.New("storage backend is not implemented")

// NewObjectStore returns the object store for env.StorageBackend: files under
// env.AssetsDir for disk.
func NewObjectStore(env *appenv.Env) (objstore.Store, error) {
	if env == nil {
		return nil, errors.New("environment config is required")
	}

	switch env.StorageBackend {
	case appenv.StorageBackendDisk:
		store, err := objstore.NewDisk(env.AssetsDir)
		if err != nil {
			return nil, fmt.Errorf("create disk object store: %w", err)
		}

		return store, nil
	case appenv.StorageBackendS3:
		return nil, fmt.Errorf("%w: %s", ErrStorageBackendNotImplemented, env.StorageBackend)
	default:
		return nil, fmt.Errorf("unknown storage backend %q", env.StorageBackend)
	}
}
