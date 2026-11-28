package appenv

import (
	"errors"
	"strings"
	"testing"
)

func TestNewObjectStoreBackendDefaultsToDisk(t *testing.T) {
	setRequiredEnv(t)
	clearEnv(t, "OBJECT_STORE_BACKEND")

	env, err := New()
	if err != nil {
		t.Fatalf("unexpected New error: %v", err)
	}
	if env.ObjectStoreBackend != ObjectStoreBackendDisk || env.UsesS3() {
		t.Fatalf("expected disk backend, got %q", env.ObjectStoreBackend)
	}
}

func TestNewRejectsUnknownObjectStoreBackend(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("OBJECT_STORE_BACKEND", "ftp")

	if _, err := New(); err == nil {
		t.Fatal("expected error for unknown object store backend")
	}
}

func TestNewRequiresS3SettingsForS3Backend(t *testing.T) {
	setRequiredEnv(t)
	clearEnv(t, "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET")
	t.Setenv("OBJECT_STORE_BACKEND", "s3")

	_, err := New()
	if !errors.Is(err, ErrS3ConfigRequired) {
		t.Fatalf("expected ErrS3ConfigRequired, got %v", err)
	}
	for _, name := range []string{"S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("expected error to name %s, got %v", name, err)
		}
	}

	t.Setenv("S3_ENDPOINT", "localhost:9000")
	t.Setenv("S3_ACCESS_KEY", "key")
	t.Setenv("S3_SECRET_KEY", "secret")
	t.Setenv("S3_BUCKET", "miconsul-dev")

	env, err := New()
	if err != nil {
		t.Fatalf("expected complete S3 settings to load, got %v", err)
	}
	if !env.UsesS3() || env.S3Bucket != "miconsul-dev" || env.S3UseSSL {
		t.Fatalf("unexpected S3 settings: %+v", env)
	}
}

func TestNewRejectsInvalidBucketName(t *testing.T) {
	for _, name := range []string{"Miconsul", "miconsul_dev", "-dev", "ab", "miconsul.dev"} {
		setRequiredEnv(t)
		t.Setenv("S3_BUCKET", name)

		if _, err := New(); err == nil {
			t.Fatalf("expected error for bucket name %q", name)
		}
	}
}

func TestNewAcceptsBlankS3SettingsOnDisk(t *testing.T) {
	setRequiredEnv(t)
	for _, name := range []string{"S3_ENDPOINT", "S3_REGION", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET"} {
		t.Setenv(name, "")
	}

	env, err := New()
	if err != nil {
		t.Fatalf("expected blank S3 settings to load on disk, got %v", err)
	}
	if env.UsesS3() {
		t.Fatal("expected disk backend")
	}
}
