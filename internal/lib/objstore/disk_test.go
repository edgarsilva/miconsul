package objstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var pngHead = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func newDisk(t *testing.T) (*Disk, string) {
	t.Helper()

	dir := t.TempDir()
	d, err := NewDisk(dir)
	if err != nil {
		t.Fatalf("NewDisk: %v", err)
	}
	t.Cleanup(func() {
		if err := d.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	return d, dir
}

func closeBody(t *testing.T, obj Object) {
	t.Helper()

	if err := obj.Body.Close(); err != nil {
		t.Errorf("close object body: %v", err)
	}
}

func TestDiskPutOpenDelete(t *testing.T) {
	d, dir := newDisk(t)
	ctx := context.Background()

	err := d.Put(ctx, "clinics/c1_ppic_ab.png", bytes.NewReader(pngHead), int64(len(pngHead)), "image/png")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "clinics", "c1_ppic_ab.png")); err != nil {
		t.Fatalf("expected file at the key's path: %v", err)
	}

	obj, err := d.Open(ctx, "clinics/c1_ppic_ab.png")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeBody(t, obj)

	got, _ := io.ReadAll(obj.Body)
	if !bytes.Equal(got, pngHead) || obj.Size != int64(len(pngHead)) {
		t.Fatalf("unexpected body or size: %d bytes, size %d", len(got), obj.Size)
	}
	if obj.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", obj.ContentType)
	}
	if obj.ModTime.IsZero() {
		t.Fatal("expected a modification time")
	}

	if err := d.Delete(ctx, "clinics/c1_ppic_ab.png"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := d.Open(ctx, "clinics/c1_ppic_ab.png"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDiskOpenSniffsTypeWithoutExtension(t *testing.T) {
	d, _ := newDisk(t)
	ctx := context.Background()

	if err := d.Put(ctx, "users/u1_avatar", bytes.NewReader(pngHead), int64(len(pngHead)), ""); err != nil {
		t.Fatalf("Put: %v", err)
	}

	obj, err := d.Open(ctx, "users/u1_avatar")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeBody(t, obj)

	if obj.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", obj.ContentType)
	}
	got, _ := io.ReadAll(obj.Body)
	if !bytes.Equal(got, pngHead) {
		t.Fatal("body must be readable from the start after sniffing")
	}
}

func TestDiskPutReplacesObject(t *testing.T) {
	d, _ := newDisk(t)
	ctx := context.Background()

	for _, content := range []string{"first", "second"} {
		if err := d.Put(ctx, "users/u1_avatar", bytes.NewReader([]byte(content)), int64(len(content)), ""); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	obj, err := d.Open(ctx, "users/u1_avatar")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer closeBody(t, obj)

	if got, _ := io.ReadAll(obj.Body); string(got) != "second" {
		t.Fatalf("got %q, want second", got)
	}
}

func TestDiskDeleteMissingIsNotAnError(t *testing.T) {
	d, _ := newDisk(t)

	if err := d.Delete(context.Background(), "users/nope"); err != nil {
		t.Fatalf("Delete of a missing object: %v", err)
	}
}

func TestDiskOpenDirectoryIsNotFound(t *testing.T) {
	d, dir := newDisk(t)
	if err := os.MkdirAll(filepath.Join(dir, "clinics"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := d.Open(context.Background(), "clinics"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a directory, got %v", err)
	}
}

func TestDiskRejectsInvalidKeys(t *testing.T) {
	d, _ := newDisk(t)
	ctx := context.Background()

	for _, key := range invalidKeys {
		if _, err := d.Open(ctx, key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Open(%q) = %v, want ErrInvalidKey", key, err)
		}
		if err := d.Put(ctx, key, bytes.NewReader(nil), 0, ""); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Put(%q) = %v, want ErrInvalidKey", key, err)
		}
		if err := d.Delete(ctx, key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Delete(%q) = %v, want ErrInvalidKey", key, err)
		}
	}
}

func TestNewDiskRequiresRoot(t *testing.T) {
	if _, err := NewDisk("  "); err == nil {
		t.Fatal("expected error for an empty root")
	}
}

func TestDiskSymlinkCannotEscapeRoot(t *testing.T) {
	d, dir := newDisk(t)
	ctx := context.Background()

	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "users"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(dir, "users", "file_link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "users", "dir_link")); err != nil {
		t.Fatal(err)
	}

	if obj, err := d.Open(ctx, "users/file_link"); err == nil {
		closeBody(t, obj)
		t.Fatal("Open followed a symlink out of the root")
	}
	if obj, err := d.Open(ctx, "users/dir_link/secret.txt"); err == nil {
		closeBody(t, obj)
		t.Fatal("Open followed a directory symlink out of the root")
	}
	if err := d.Put(ctx, "users/dir_link/planted", bytes.NewReader([]byte("x")), 1, ""); err == nil {
		t.Fatal("Put wrote through a directory symlink out of the root")
	}
	if _, err := os.Stat(filepath.Join(outside, "planted")); err == nil {
		t.Fatal("a file was written outside the root")
	}
	if err := d.Delete(ctx, "users/dir_link/secret.txt"); err == nil {
		t.Fatal("Delete followed a directory symlink out of the root")
	}
	if _, err := os.Stat(secret); err != nil {
		t.Fatalf("file outside the root was removed: %v", err)
	}
}

func TestDiskSymlinkInsideRootIsFollowed(t *testing.T) {
	d, dir := newDisk(t)
	ctx := context.Background()

	if err := d.Put(ctx, "users/real", bytes.NewReader([]byte("data")), 4, ""); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := os.Symlink("real", filepath.Join(dir, "users", "alias")); err != nil {
		t.Fatal(err)
	}

	obj, err := d.Open(ctx, "users/alias")
	if err != nil {
		t.Fatalf("Open through an in-root symlink: %v", err)
	}
	closeBody(t, obj)
}

func TestDiskPutLeavesNoTemporaryFile(t *testing.T) {
	d, dir := newDisk(t)

	if err := d.Put(context.Background(), "users/u1_avatar", bytes.NewReader([]byte("x")), 1, ""); err != nil {
		t.Fatalf("Put: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(dir, "users"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "u1_avatar" {
		t.Fatalf("unexpected directory contents: %v", entries)
	}
}

func TestDiskConcurrentUse(t *testing.T) {
	d, dir := newDisk(t)
	ctx := context.Background()

	const workers, rounds = 16, 40
	payload := func(id int) []byte { return bytes.Repeat([]byte{byte('a' + id)}, 2048+id) }

	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {

			own := fmt.Sprintf("users/own_%d", w)
			for range rounds {
				if err := d.Put(ctx, "users/shared", bytes.NewReader(payload(w)), 0, ""); err != nil {
					t.Errorf("worker %d: Put shared: %v", w, err)
					return
				}
				if err := d.Put(ctx, own, bytes.NewReader(payload(w)), 0, ""); err != nil {
					t.Errorf("worker %d: Put own: %v", w, err)
					return
				}

				checkWhole(t, d, "users/shared", true)
				checkOwn(t, d, own, payload(w))

				if w%4 == 0 {
					if err := d.Delete(ctx, "users/shared"); err != nil {
						t.Errorf("worker %d: Delete shared: %v", w, err)
						return
					}
				}
				if err := d.Delete(ctx, own); err != nil {
					t.Errorf("worker %d: Delete own: %v", w, err)
					return
				}
			}
		})
	}
	wg.Wait()

	entries, err := os.ReadDir(filepath.Join(dir, "users"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".put-") {
			t.Errorf("leftover temporary file %s", entry.Name())
		}
	}
}

func checkWhole(t *testing.T, d *Disk, key string, mayBeMissing bool) {
	t.Helper()

	obj, err := d.Open(context.Background(), key)
	if errors.Is(err, ErrNotFound) && mayBeMissing {
		return
	}
	if err != nil {
		t.Errorf("Open %s: %v", key, err)
		return
	}
	defer closeBody(t, obj)

	got, err := io.ReadAll(obj.Body)
	if err != nil {
		t.Errorf("read %s: %v", key, err)
		return
	}
	if len(got) == 0 {
		t.Errorf("%s is empty", key)
		return
	}

	want := bytes.Repeat([]byte{got[0]}, 2048+int(got[0]-'a'))
	if !bytes.Equal(got, want) {
		t.Errorf("%s is a mix of writes: %d bytes starting with %q", key, len(got), got[0])
	}
}

func checkOwn(t *testing.T, d *Disk, key string, want []byte) {
	t.Helper()

	obj, err := d.Open(context.Background(), key)
	if err != nil {
		t.Errorf("Open %s: %v", key, err)
		return
	}
	defer closeBody(t, obj)

	got, err := io.ReadAll(obj.Body)
	if err != nil || !bytes.Equal(got, want) {
		t.Errorf("%s: %d bytes, err %v, want %d bytes", key, len(got), err, len(want))
	}
}
