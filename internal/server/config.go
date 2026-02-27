package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/storage/sqlite3/v2"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const csp = "default-src 'self';base-uri 'self';font-src 'self' https: data:;" +
	"form-action 'self';" +
	"frame-ancestors 'self';" +
	"img-src 'self' data: *.dicebear.com *.pravatar.cc images.unsplash.com *.googleusercontent.com *.gravatar.com;" +
	"object-src 'none';" +
	"script-src 'self' 'unsafe-eval' unpkg.com *.jsdelivr.net;" +
	"script-src-attr 'none';" +
	"style-src 'self' https:;" +
	"upgrade-insecure-requests"

func limiterConfig() limiter.Config {
	return limiter.Config{
		Max:               100,
		Expiration:        60 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}
}

func staticConfig() static.Config {
	return static.Config{
		Compress:      true,
		ByteRange:     true,
		Browse:        false,
		IndexNames:    []string{},
		CacheDuration: staticCacheDuration(),
		MaxAge:        3600,
	}
}

func staticCacheDuration() time.Duration {
	if os.Getenv("APP_ENV") == "development" {
		return -10
	}

	return 300 * time.Second
}

func sessionConfig(path string) sqlite3.Config {
	sessPath := path
	if sessPath == "" {
		sessPath = os.Getenv("SESSION_DB_PATH")
	}
	if sessPath == "" {
		sessPath = "./fiber_session.sqlite3"
	}

	if stat, err := os.Stat(sessPath); err == nil && stat.IsDir() {
		sessPath = filepath.Join(sessPath, "fiber_session.sqlite3")
	}

	if err := os.MkdirAll(filepath.Dir(sessPath), 0o755); err != nil {
		sessPath = "./fiber_session.sqlite3"
	}

	return sqlite3.Config{
		Database:        sessPath,
		Table:           "fiber_storage",
		Reset:           false,
		GCInterval:      10 * time.Second,
		MaxOpenConns:    100,
		MaxIdleConns:    100,
		ConnMaxLifetime: 1 * time.Second,
	}
}

func helmetConfig() helmet.Config {
	return helmet.Config{
		ContentSecurityPolicy:     csp,
		XSSProtection:             "0",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "SAMEORIGIN",
		ReferrerPolicy:            "no-referrer",
		CrossOriginEmbedderPolicy: "credentialless",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "cross-origin-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "none",
	}
}

// recoverConfig returns recover middleware config object with the yield func
// executed with the request ctx to trace the panic error
func fiberAppErrorHandler(ctx fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := fiber.StatusInternalServerError

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	span := trace.SpanFromContext(ctx.Context())
	defer span.End()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	// Send custom error page
	err = ctx.Status(code).SendFile(fmt.Sprintf("./public/%d.html", code))
	if err != nil {
		// In case SendFile fails
		return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	// Return from handler
	return nil
}
