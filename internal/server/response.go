package server

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"miconsul/internal/lib/objstore"

	"github.com/gofiber/fiber/v3"
)

// Redirect performs a 303 See Other redirect.
func (s *Server) Redirect(c fiber.Ctx, path string) error {
	return c.Redirect().Status(fiber.StatusSeeOther).To(path)
}

// HXRedirect tells htmx to load path as a full page navigation.
func (s *Server) HXRedirect(c fiber.Ctx, path string) error {
	c.Set("HX-Redirect", path)
	return c.SendStatus(fiber.StatusNoContent)
}

// RedirectWithToast redirects to path and shows a toast on the next page load
// through a flash message, so the URL never carries the message. HTMX requests
// get HX-Redirect instead of a 303. level is stored by name, e.g.
// view.LevelSuccess.
func (s *Server) RedirectWithToast(c fiber.Ctx, path, msg string, level fmt.Stringer) error {
	return s.redirectWithFlash(c, path, "toast", msg, "level", level.String())
}

// RedirectWithAlert redirects to path and shows an inline alert on the next
// page load through a flash message; the page renders it with CmpFlashAlert.
// HTMX requests get HX-Redirect instead of a 303. level is stored by name,
// e.g. view.LevelSuccess.
func (s *Server) RedirectWithAlert(c fiber.Ctx, path, msg string, level fmt.Stringer) error {
	return s.redirectWithFlash(c, path, "alert", msg, "alert_level", level.String())
}

// SendObject sends obj as the response with its content type, size and
// modification time. It answers 304 when the client's copy is current. When
// obj.Body can seek, it also serves a single byte range of the object (206, or
// 416 for a range outside it). It closes obj.Body: the response closes it after
// streaming, and SendObject closes it on every other path.
func (s *Server) SendObject(c fiber.Ctx, obj objstore.Object) error {
	c.Set(fiber.HeaderContentType, obj.ContentType)

	seeker, canSeek := obj.Body.(io.Seeker)
	if canSeek {
		c.Set(fiber.HeaderAcceptRanges, "bytes")
	}

	modified := obj.ModTime.UTC().Truncate(time.Second)
	if !modified.IsZero() {
		c.Set(fiber.HeaderLastModified, modified.Format(http.TimeFormat))

		since, err := http.ParseTime(c.Get(fiber.HeaderIfModifiedSince))
		if err == nil && !modified.After(since) {
			_ = obj.Body.Close()
			return c.SendStatus(fiber.StatusNotModified)
		}
	}

	if canSeek && rangeApplies(c, modified) {
		start, end, result := parseByteRange(c.Get(fiber.HeaderRange), obj.Size)
		switch result {
		case rangeSatisfiable:
			if _, err := seeker.Seek(start, io.SeekStart); err != nil {
				_ = obj.Body.Close()
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			length := end - start + 1
			c.Status(fiber.StatusPartialContent)
			c.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, obj.Size))
			part := limitedBody{Reader: io.LimitReader(obj.Body, length), Closer: obj.Body}
			return c.SendStream(part, int(length))
		case rangeUnsatisfiable:
			_ = obj.Body.Close()
			c.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes */%d", obj.Size))
			c.Status(fiber.StatusRequestedRangeNotSatisfiable)
			return nil
		}
	}

	return c.SendStream(obj.Body, int(obj.Size))
}

type limitedBody struct {
	io.Reader
	io.Closer
}

type rangeResult int

const (
	rangeIgnored rangeResult = iota
	rangeSatisfiable
	rangeUnsatisfiable
)

// rangeApplies reports whether the request asks for a range that still matches
// the object: a Range header, and an If-Range date, when present, equal to the
// modification time.
func rangeApplies(c fiber.Ctx, modified time.Time) bool {
	if c.Get(fiber.HeaderRange) == "" {
		return false
	}

	ifRange := c.Get(fiber.HeaderIfRange)
	if ifRange == "" {
		return true
	}

	since, err := http.ParseTime(ifRange)
	return err == nil && since.Equal(modified)
}

// parseByteRange reads a single "bytes=start-end", "bytes=start-" or
// "bytes=-suffix" range for an object of size bytes and returns the inclusive
// start and end offsets. Multiple ranges and malformed values are ignored, which
// serves the whole object.
func parseByteRange(header string, size int64) (start, end int64, result rangeResult) {
	spec, ok := strings.CutPrefix(header, "bytes=")
	if !ok || strings.Contains(spec, ",") {
		return 0, 0, rangeIgnored
	}

	first, last, ok := strings.Cut(spec, "-")
	if !ok {
		return 0, 0, rangeIgnored
	}

	if first == "" {
		suffix, err := strconv.ParseInt(last, 10, 64)
		if err != nil || suffix < 0 {
			return 0, 0, rangeIgnored
		}
		if suffix == 0 || size == 0 {
			return 0, 0, rangeUnsatisfiable
		}

		return max(size-suffix, 0), size - 1, rangeSatisfiable
	}

	start, err := strconv.ParseInt(first, 10, 64)
	if err != nil || start < 0 {
		return 0, 0, rangeIgnored
	}

	end = size - 1
	if last != "" {
		end, err = strconv.ParseInt(last, 10, 64)
		if err != nil || end < start {
			return 0, 0, rangeIgnored
		}
	}

	if start >= size {
		return 0, 0, rangeUnsatisfiable
	}

	return start, min(end, size-1), rangeSatisfiable
}

// redirectWithFlash redirects to path with a flash message and level. For htmx
// it answers 204 with an HX-Redirect header.
func (s *Server) redirectWithFlash(c fiber.Ctx, path, msgKey, msg, levelKey, level string) error {
	status := fiber.StatusSeeOther
	if s.IsHTMX(c) {
		status = fiber.StatusNoContent
		c.Set("HX-Redirect", path)
	}

	return c.Redirect().Status(status).With(msgKey, msg).With(levelKey, level).To(path)
}
