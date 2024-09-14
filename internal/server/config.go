package server

import (
	"errors"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/storage/sqlite3/v2"
	logto "github.com/logto-io/go/client"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const csp = "default-src 'self';base-uri 'self';font-src 'self' https: data:;" +
	"form-action 'self';" +
	"frame-ancestors 'self';" +
	"img-src 'self' data: *.dicebear.com *.pravatar.cc images.unsplash.com *.googleusercontent.com *.gravatar.com;" +
	"object-src 'none';" +
	"script-src 'self' 'unsafe-eval' 'unsafe-inline' unpkg.com *.unpkg.com *.jsdelivr.net;" +
	"script-src-attr 'none';" +
	"style-src 'self' https: 'unsafe-inline';" +
	"upgrade-insecure-requests"

func limiterConfig() limiter.Config {
	return limiter.Config{
		Max:               100,
		Expiration:        60 * time.Second,
		LimiterMiddleware: limiter.SlidingWindow{},
	}
}

func staticConfig() fiber.Static {
	return fiber.Static{
		Compress:      true,
		ByteRange:     true,
		Browse:        false,
		Index:         "",
		CacheDuration: 300 * time.Second,
		MaxAge:        3600,
	}
}

func sessionConfig() sqlite3.Config {
	sessPath := os.Getenv("SESSION_PATH")
	if sessPath == "" {
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
		XPermittedCrossDomain:     "*.dicebear.com,*.googleusercontent.com",
	}
}

func LogtoConfig() *logto.LogtoConfig {
	endpoint := os.Getenv("LOGTO_URL")
	appid := os.Getenv("LOGTO_APP_ID")
	appsecret := os.Getenv("LOGTO_APP_SECRET")

	c := logto.LogtoConfig{
		Endpoint:  endpoint,
		AppId:     appid,
		AppSecret: appsecret,
		Resources: []string{"https://app.miconsul.xyz/api"},
		Scopes:    []string{"email", "phone", "picture", "custom_data", "app:read", "app:write"},
	}

	return &c
}

// recoverConfig returns recover middleware config object with the yield func
// executed with the request ctx to trace the panic error
func fiberAppErrorHandler(ctx *fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := fiber.StatusInternalServerError

	// Retrieve the custom status code if it's a *fiber.Error
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	// Send custom error to intrumentation logs
	span := trace.SpanFromContext(ctx.UserContext())
	defer span.End()

	if err != nil {
		// Record the error.
		span.RecordError(err)

		// Update the span status.
		span.SetStatus(codes.Error, err.Error())
	}

	// Return from handler
	return ctx.Status(code).SendString("Internal Server Error")
}
