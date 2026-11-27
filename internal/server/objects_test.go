package server

import (
	"errors"
	"testing"

	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/objstore"
)

func TestNewObjectStoreDisk(t *testing.T) {
	env := &appenv.Env{StorageBackend: appenv.StorageBackendDisk, AssetsDir: t.TempDir()}

	store, err := NewObjectStore(env)
	if err != nil {
		t.Fatalf("NewObjectStore: %v", err)
	}
	if _, ok := store.(*objstore.Disk); !ok {
		t.Fatalf("expected a *objstore.Disk, got %T", store)
	}
}

func TestNewObjectStoreDiskRequiresAssetsDir(t *testing.T) {
	env := &appenv.Env{StorageBackend: appenv.StorageBackendDisk}

	store, err := NewObjectStore(env)
	if err == nil || store != nil {
		t.Fatalf("expected an error and no store, got %v, %v", store, err)
	}
}

func TestNewObjectStoreS3IsNotImplemented(t *testing.T) {
	env := &appenv.Env{StorageBackend: appenv.StorageBackendS3, AssetsDir: t.TempDir()}

	store, err := NewObjectStore(env)
	if !errors.Is(err, ErrStorageBackendNotImplemented) || store != nil {
		t.Fatalf("expected ErrStorageBackendNotImplemented and no store, got %v, %v", store, err)
	}
}

func TestNewObjectStoreRejectsUnknownBackend(t *testing.T) {
	for _, backend := range []string{"", "ftp"} {
		env := &appenv.Env{StorageBackend: backend, AssetsDir: t.TempDir()}

		if store, err := NewObjectStore(env); err == nil || store != nil {
			t.Fatalf("backend %q: expected an error and no store, got %v, %v", backend, store, err)
		}
	}
}

func TestNewObjectStoreRequiresEnv(t *testing.T) {
	if store, err := NewObjectStore(nil); err == nil || store != nil {
		t.Fatalf("expected an error and no store, got %v, %v", store, err)
	}
}

func TestWithObjectsRejectsNil(t *testing.T) {
	if err := WithObjects(nil)(&Server{}); err == nil {
		t.Fatal("expected an error for a nil store")
	}
}
