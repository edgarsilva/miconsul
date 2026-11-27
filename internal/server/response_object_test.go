package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

type seekTracker struct {
	*bytes.Reader
	closed atomic.Bool
}

func (s *seekTracker) Close() error {
	s.closed.Store(true)
	return nil
}

func rangeApp(t *testing.T, content string, modTime time.Time) (*fiber.App, *seekTracker) {
	t.Helper()

	body := &seekTracker{Reader: bytes.NewReader([]byte(content))}
	obj := objstore.Object{Body: body, Size: int64(len(content)), ModTime: modTime, ContentType: "video/mp4"}

	app := fiber.New()
	app.Get("/file", func(c fiber.Ctx) error {
		return (&Server{}).SendObject(c, obj)
	})

	return app, body
}

func TestSendObjectServesByteRanges(t *testing.T) {
	const content = "0123456789"
	modTime := time.Date(2026, 11, 27, 22, 13, 12, 0, time.UTC)

	tests := []struct {
		name         string
		rangeHeader  string
		ifRange      string
		wantStatus   int
		wantBody     string
		wantRange    string
		wantAccepted bool
	}{
		{"no range sends everything", "", "", 200, content, "", true},
		{"first bytes", "bytes=0-3", "", 206, "0123", "bytes 0-3/10", true},
		{"middle bytes", "bytes=2-5", "", 206, "2345", "bytes 2-5/10", true},
		{"open ended", "bytes=7-", "", 206, "789", "bytes 7-9/10", true},
		{"suffix", "bytes=-3", "", 206, "789", "bytes 7-9/10", true},
		{"suffix longer than the object", "bytes=-50", "", 206, content, "bytes 0-9/10", true},
		{"single byte", "bytes=4-4", "", 206, "4", "bytes 4-4/10", true},
		{"end past the object is clamped", "bytes=5-500", "", 206, "56789", "bytes 5-9/10", true},
		{"start at the end is unsatisfiable", "bytes=10-", "", 416, "", "bytes */10", true},
		{"start past the end is unsatisfiable", "bytes=50-60", "", 416, "", "bytes */10", true},
		{"zero length suffix is unsatisfiable", "bytes=-0", "", 416, "", "bytes */10", true},
		{"multiple ranges are ignored", "bytes=0-1,4-5", "", 200, content, "", true},
		{"wrong unit is ignored", "items=0-3", "", 200, content, "", true},
		{"end before start is ignored", "bytes=5-2", "", 200, content, "", true},
		{"garbage is ignored", "bytes=abc", "", 200, content, "", true},
		{"negative start is ignored", "bytes=--3", "", 200, content, "", true},
		{"matching If-Range keeps the range", "bytes=0-3", "Fri, 27 Nov 2026 22:13:12 GMT", 206, "0123", "bytes 0-3/10", true},
		{"stale If-Range sends everything", "bytes=0-3", "Thu, 26 Nov 2026 22:13:12 GMT", 200, content, "", true},
		{"invalid If-Range sends everything", "bytes=0-3", "W/\"etag\"", 200, content, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, body := rangeApp(t, content, modTime)

			req := httptest.NewRequest(http.MethodGet, "/file", nil)
			if tt.rangeHeader != "" {
				req.Header.Set("Range", tt.rangeHeader)
			}
			if tt.ifRange != "" {
				req.Header.Set("If-Range", tt.ifRange)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			got := drain(t, resp)

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if string(got) != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
			if cr := resp.Header.Get("Content-Range"); cr != tt.wantRange {
				t.Fatalf("Content-Range = %q, want %q", cr, tt.wantRange)
			}
			if ar := resp.Header.Get("Accept-Ranges"); (ar == "bytes") != tt.wantAccepted {
				t.Fatalf("Accept-Ranges = %q", ar)
			}
			if tt.wantStatus == 206 {
				if cl := resp.Header.Get("Content-Length"); cl != strconv.Itoa(len(tt.wantBody)) {
					t.Fatalf("Content-Length = %q, want %d", cl, len(tt.wantBody))
				}
			}
			waitClosedSeek(t, body)
		})
	}
}

func TestSendObjectIgnoresRangeWhenBodyCannotSeek(t *testing.T) {
	app, body := objectApp(t, []byte("0123456789"), time.Now())

	req := httptest.NewRequest(http.MethodGet, "/pic", nil)
	req.Header.Set("Range", "bytes=0-3")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	got := drain(t, resp)

	if resp.StatusCode != http.StatusOK || string(got) != "0123456789" {
		t.Fatalf("status %d body %q, want the whole object", resp.StatusCode, got)
	}
	if ar := resp.Header.Get("Accept-Ranges"); ar != "" {
		t.Fatalf("Accept-Ranges = %q, want none for a body that cannot seek", ar)
	}
	waitClosed(t, body)
}

func TestSendObjectNotModifiedWinsOverRange(t *testing.T) {
	modTime := time.Date(2026, 11, 27, 22, 13, 12, 0, time.UTC)
	app, body := rangeApp(t, "0123456789", modTime)

	req := httptest.NewRequest(http.MethodGet, "/file", nil)
	req.Header.Set("Range", "bytes=0-3")
	req.Header.Set("If-Modified-Since", "Fri, 27 Nov 2026 22:13:12 GMT")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	drain(t, resp)

	if resp.StatusCode != http.StatusNotModified || !body.closed.Load() {
		t.Fatalf("status %d closed %v, want 304 and a closed body", resp.StatusCode, body.closed.Load())
	}
}

func waitClosedSeek(t *testing.T, body *seekTracker) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for !body.closed.Load() {
		if time.Now().After(deadline) {
			t.Fatal("object body was never closed")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestParseByteRange(t *testing.T) {
	tests := []struct {
		header     string
		size       int64
		start, end int64
		result     rangeResult
	}{
		{"bytes=0-0", 1, 0, 0, rangeSatisfiable},
		{"bytes=0-", 1, 0, 0, rangeSatisfiable},
		{"bytes=-1", 1, 0, 0, rangeSatisfiable},
		{"bytes=0-0", 0, 0, 0, rangeUnsatisfiable},
		{"bytes=-1", 0, 0, 0, rangeUnsatisfiable},
		{"bytes=1-1", 1, 0, 0, rangeUnsatisfiable},
		{"bytes=0-9223372036854775807", 5, 0, 4, rangeSatisfiable},
		{"bytes=99999999999999999999-", 5, 0, 0, rangeIgnored},
		{"bytes=-", 5, 0, 0, rangeIgnored},
		{"bytes=", 5, 0, 0, rangeIgnored},
		{"bytes=1", 5, 0, 0, rangeIgnored},
		{"bytes= 1-2", 5, 0, 0, rangeIgnored},
		{"BYTES=0-1", 5, 0, 0, rangeIgnored},
	}

	for _, tt := range tests {
		start, end, result := parseByteRange(tt.header, tt.size)
		if result != tt.result || (result == rangeSatisfiable && (start != tt.start || end != tt.end)) {
			t.Errorf("parseByteRange(%q, %d) = %d, %d, %d, want %d, %d, %d", tt.header, tt.size, start, end, result, tt.start, tt.end, tt.result)
		}
	}
}
