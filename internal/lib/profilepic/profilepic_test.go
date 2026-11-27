package profilepic

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"miconsul/internal/lib/objstore"
)

func TestRandomName(t *testing.T) {
	re := regexp.MustCompile(`^u1_ppic_[0-9a-f]{32}\.png$`)

	a, err := RandomName("u1_ppic_", `C:\pics\Me.PNG`)
	if err != nil || !re.MatchString(a) {
		t.Fatalf("unexpected name %q err=%v", a, err)
	}

	b, _ := RandomName("u1_ppic_", "me.png")
	if a == b {
		t.Fatalf("names must differ, got %q twice", a)
	}

	for _, upload := range []string{"", "..", "photo", "x.exe", "a.png/../b.php"} {
		if _, err := RandomName("p_", upload); !errors.Is(err, ErrUnsupportedType) {
			t.Fatalf("expected ErrUnsupportedType for %q, got %v", upload, err)
		}
	}
}

func TestIsSafeFilename(t *testing.T) {
	cases := map[string]bool{
		"patient_ppic.png":       true,
		"a..b.png":               true,
		"":                       false,
		".":                      false,
		"..":                     false,
		"a/b.png":                false,
		`a\b.png`:                false,
		"a\x00.png":              false,
		"a?.png":                 false,
		"a b.png":                false,
		strings.Repeat("a", 256): false,
	}
	for name, want := range cases {
		if got := IsSafeFilename(name); got != want {
			t.Fatalf("IsSafeFilename(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestPath(t *testing.T) {
	dir := t.TempDir()

	got, err := Path(dir, "/patients", "a.png")
	if err != nil || got != filepath.Join(dir, "patients", "a.png") {
		t.Fatalf("unexpected path %q err=%v", got, err)
	}
	if info, err := os.Stat(filepath.Join(dir, "patients")); err != nil || !info.IsDir() {
		t.Fatalf("expected directory to exist: %v", err)
	}
	if _, err := Path(dir, "/patients", "../a.png"); !errors.Is(err, ErrInvalidFilename) {
		t.Fatalf("expected ErrInvalidFilename, got %v", err)
	}
	if _, err := Path(" ", "/patients", "a.png"); !errors.Is(err, ErrNoAssetsDir) {
		t.Fatalf("expected ErrNoAssetsDir, got %v", err)
	}
}

func TestFormFileError(t *testing.T) {
	if FormFileError(nil) != nil {
		t.Fatalf("nil must stay nil")
	}
	if !errors.Is(FormFileError(http.ErrMissingFile), ErrNotProvided) {
		t.Fatalf("missing file must map to ErrNotProvided")
	}
	if !errors.Is(FormFileError(errors.New("no uploaded file")), ErrNotProvided) {
		t.Fatalf("message match must map to ErrNotProvided")
	}
	if err := FormFileError(errors.New("boom")); errors.Is(err, ErrNotProvided) {
		t.Fatalf("other errors must not map to ErrNotProvided")
	}
}

func TestRemove(t *testing.T) {
	dir := t.TempDir()
	path, err := Path(dir, "/patients", "a.png")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := Remove(dir, "/patients", "a.png"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
	if err := Remove(dir, "/patients", "a.png"); err != nil {
		t.Fatalf("removing a missing file must not fail, got %v", err)
	}
	if err := Remove(dir, "/patients", "../a.png"); !errors.Is(err, ErrInvalidFilename) {
		t.Fatalf("expected ErrInvalidFilename, got %v", err)
	}
	if err := Remove(" ", "/patients", "a.png"); !errors.Is(err, ErrNoAssetsDir) {
		t.Fatalf("expected ErrNoAssetsDir, got %v", err)
	}
}

func TestCheckTypeAndAccept(t *testing.T) {
	if err := CheckType("me.JPG"); err != nil {
		t.Fatalf("jpg must be supported, got %v", err)
	}
	if err := CheckType("scan.tiff"); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}

	accepted := map[string]bool{}
	for mimeType := range strings.SplitSeq(Accept, ",") {
		accepted[mimeType] = false
	}
	for ext, mimeType := range contentTypes {
		if _, ok := accepted[mimeType]; !ok {
			t.Fatalf("allowed extension %q is %s, which Accept does not list", ext, mimeType)
		}
		accepted[mimeType] = true
	}
	for mimeType, used := range accepted {
		if !used {
			t.Fatalf("Accept lists %s, but no allowed extension has that type", mimeType)
		}
	}
}

func TestContentType(t *testing.T) {
	cases := map[string]string{
		"me.jpg": "image/jpeg", "me.JPEG": "image/jpeg", "me.jpe": "image/jpeg", "me.jfif": "image/jpeg",
		"me.png": "image/png", `C:\pics\Me.PNG`: "image/png", "me.webp": "image/webp",
	}
	for name, want := range cases {
		got, err := ContentType(name)
		if err != nil || got != want {
			t.Fatalf("ContentType(%q) = %q, %v, want %q", name, got, err, want)
		}
	}

	for _, name := range []string{"", "me", "me.gif", "me.heic", "me.png.exe", "a.png/../b.php"} {
		if _, err := ContentType(name); !errors.Is(err, ErrUnsupportedType) {
			t.Fatalf("ContentType(%q) = %v, want ErrUnsupportedType", name, err)
		}
	}
}

func uploadHeader(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("profilePic", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err := req.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}

	if req.MultipartForm == nil || len(req.MultipartForm.File["profilePic"]) != 1 {
		t.Fatal("expected one uploaded profilePic")
	}

	return req.MultipartForm.File["profilePic"][0]
}

type recordingStore struct {
	key         string
	body        []byte
	size        int64
	contentType string
	err         error
}

func (r *recordingStore) Put(_ context.Context, key string, body io.Reader, size int64, contentType string) error {
	if r.err != nil {
		return r.err
	}
	r.key, r.size, r.contentType = key, size, contentType
	r.body, _ = io.ReadAll(body)
	return nil
}

func (r *recordingStore) Open(context.Context, string) (objstore.Object, error) {
	return objstore.Object{}, objstore.ErrNotFound
}

func (r *recordingStore) Delete(context.Context, string) error { return nil }

func TestPutStoresUploadWithItsContentType(t *testing.T) {
	store := &recordingStore{}
	upload := uploadHeader(t, "Me.JPG", []byte("jpeg-bytes"))

	if err := Put(t.Context(), store, "clinics/c1_ppic_ab.jpg", upload); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if store.key != "clinics/c1_ppic_ab.jpg" || string(store.body) != "jpeg-bytes" || store.size != 10 || store.contentType != "image/jpeg" {
		t.Fatalf("unexpected store call: %+v", store)
	}
}

func TestPutRejectsUnsupportedTypeWithoutStoring(t *testing.T) {
	store := &recordingStore{}

	err := Put(t.Context(), store, "clinics/x.exe", uploadHeader(t, "x.exe", []byte("x")))
	if !errors.Is(err, ErrUnsupportedType) || store.key != "" {
		t.Fatalf("expected ErrUnsupportedType and no store call, got %v (key %q)", err, store.key)
	}
}

func TestPutReturnsStoreError(t *testing.T) {
	boom := errors.New("boom")
	store := &recordingStore{err: boom}

	err := Put(t.Context(), store, "clinics/c1.png", uploadHeader(t, "me.png", []byte("x")))
	if !errors.Is(err, boom) {
		t.Fatalf("expected the store error, got %v", err)
	}
}

func TestPutThroughDiskStore(t *testing.T) {
	disk, err := objstore.NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = disk.Close() })

	if err := Put(t.Context(), disk, "users/u1_avatar", uploadHeader(t, "me.png", []byte("\x89PNG\r\n\x1a\n"))); err != nil {
		t.Fatalf("Put: %v", err)
	}
	obj, err := disk.Open(t.Context(), "users/u1_avatar")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = obj.Body.Close() }()

	if obj.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", obj.ContentType)
	}
}
