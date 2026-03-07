package server

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"miconsul/internal/lib/appenv"

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

var (
	errorPagesOnce sync.Once
	errorPages     = map[int]string{}
)

func limiterConfig() limiter.Config {
	return limiter.Config{
		Max:               100,
		Expiration:        60 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}
}

func staticConfig(appEnv appenv.Environment) static.Config {
	return static.Config{
		Compress:      true,
		ByteRange:     true,
		Browse:        false,
		IndexNames:    []string{},
		CacheDuration: staticCacheDuration(appEnv),
		MaxAge:        staticMaxAge(appEnv),
	}
}

func staticCacheDuration(appEnv appenv.Environment) time.Duration {
	if appenv.IsDevelopment(appEnv) {
		return 0
	}

	return 300 * time.Second
}

func staticMaxAge(appEnv appenv.Environment) int {
	if appenv.IsDevelopment(appEnv) {
		return 0
	}

	return 3600
}

func sessionConfig(path string) sqlite3.Config {
	sessPath := path
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
		MaxOpenConns:    10,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
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
		CrossOriginResourcePolicy: "cross-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "none",
	}
}

func loadErrorPages() {
	errorPagesOnce.Do(func() {
		if page, err := os.ReadFile("./public/404.html"); err == nil {
			errorPages[fiber.StatusNotFound] = string(page)
		}

		if page, err := os.ReadFile("./public/500.html"); err == nil {
			errorPages[fiber.StatusInternalServerError] = string(page)
		}
	})
}

func sendErrorPage(ctx fiber.Ctx, code int) error {
	loadErrorPages()

	if page, ok := errorPages[code]; ok && page != "" {
		return ctx.Status(code).Type("html", "utf-8").SendString(page)
	}

	if page, ok := errorPages[fiber.StatusInternalServerError]; ok && page != "" {
		return ctx.Status(code).Type("html", "utf-8").SendString(page)
	}

	if code >= fiber.StatusInternalServerError {
		return ctx.Status(code).SendString("Internal Server Error")
	}

	return ctx.Status(code).SendString("Not Found")
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

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	if sendErr := sendErrorPage(ctx, code); sendErr != nil {
		return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	return nil
}
