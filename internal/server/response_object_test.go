package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"miconsul/internal/lib/objstore"

	"github.com/gofiber/fiber/v3"
)

type closeTracker struct {
	io.Reader
	closed atomic.Bool
}

func (c *closeTracker) Close() error {
	c.closed.Store(true)
	return nil
}

func objectApp(t *testing.T, content []byte, modTime time.Time) (*fiber.App, *closeTracker) {
	t.Helper()

	body := &closeTracker{Reader: bytes.NewReader(content)}
	obj := objstore.Object{Body: body, Size: int64(len(content)), ModTime: modTime, ContentType: "image/png"}

	app := fiber.New()
	app.Get("/pic", func(c fiber.Ctx) error {
		return (&Server{}).SendObject(c, obj)
	})

	return app, body
}

func drain(t *testing.T, resp *http.Response) []byte {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}

	return body
}

func waitClosed(t *testing.T, body *closeTracker) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for !body.closed.Load() {
		if time.Now().After(deadline) {
			t.Fatal("object body was never closed")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestSendObjectSendsBodyAndHeaders(t *testing.T) {
	modTime := time.Date(2026, 11, 27, 22, 13, 12, 987654321, time.UTC)
	app, body := objectApp(t, []byte("png-bytes"), modTime)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/pic", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	got := drain(t, resp)

	if resp.StatusCode != http.StatusOK || string(got) != "png-bytes" {
		t.Fatalf("status %d body %q", resp.StatusCode, got)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type = %q", ct)
	}
	if cl := resp.Header.Get("Content-Length"); cl != "9" {
		t.Fatalf("Content-Length = %q", cl)
	}
	if lm := resp.Header.Get("Last-Modified"); lm != "Fri, 27 Nov 2026 22:13:12 GMT" {
		t.Fatalf("Last-Modified = %q", lm)
	}
	waitClosed(t, body)
}

func TestSendObjectAnswers304AndClosesBody(t *testing.T) {
	modTime := time.Date(2026, 11, 27, 22, 13, 12, 987654321, time.UTC)
	app, body := objectApp(t, []byte("png-bytes"), modTime)

	req := httptest.NewRequest(http.MethodGet, "/pic", nil)
	req.Header.Set("If-Modified-Since", "Fri, 27 Nov 2026 22:13:12 GMT")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	drain(t, resp)

	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("status = %d, want 304 for a copy modified in the same second", resp.StatusCode)
	}
	if !body.closed.Load() {
		t.Fatal("the 304 path must close the body")
	}
}

func TestSendObjectSendsAgainWhenClientCopyIsOlder(t *testing.T) {
	modTime := time.Date(2026, 11, 27, 22, 13, 12, 0, time.UTC)
	app, body := objectApp(t, []byte("png-bytes"), modTime)

	req := httptest.NewRequest(http.MethodGet, "/pic", nil)
	req.Header.Set("If-Modified-Since", "Fri, 27 Nov 2026 22:13:11 GMT")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	drain(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	waitClosed(t, body)
}

func TestSendObjectHeadClosesBody(t *testing.T) {
	app, body := objectApp(t, []byte("png-bytes"), time.Now())

	resp, err := app.Test(httptest.NewRequest(http.MethodHead, "/pic", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	drain(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	waitClosed(t, body)
}

func TestSendObjectIgnoresInvalidIfModifiedSince(t *testing.T) {
	app, body := objectApp(t, []byte("x"), time.Now())

	req := httptest.NewRequest(http.MethodGet, "/pic", nil)
	req.Header.Set("If-Modified-Since", "not a date")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	drain(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	waitClosed(t, body)
}

func TestSendObjectWithUnknownModTimeSendsTheBody(t *testing.T) {
	app, body := objectApp(t, []byte("png-bytes"), time.Time{})

	req := httptest.NewRequest(http.MethodGet, "/pic", nil)
	req.Header.Set("If-Modified-Since", "Fri, 27 Nov 2026 22:13:12 GMT")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	got := drain(t, resp)

	if resp.StatusCode != http.StatusOK || string(got) != "png-bytes" {
		t.Fatalf("status %d body %q, want 200 and the body: an unknown date never matches the client's", resp.StatusCode, got)
	}
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		t.Fatalf("Last-Modified = %q, want none for an unknown date", lm)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type = %q", ct)
	}
	waitClosed(t, body)
}
