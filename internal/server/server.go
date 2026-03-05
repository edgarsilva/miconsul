// Package server provides a server for the application that can be extended with routers.
package server

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/cronjob"
	"miconsul/internal/lib/localize"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/storage/sqlite3/v2"

	"github.com/panjf2000/ants/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type Cache interface {
	Read(key string, dst *[]byte) error
	Write(key string, src *[]byte, ttl time.Duration) error
}

type Server struct {
	Env          *appenv.Env
	DB           *database.Database
	wp           *ants.Pool     // <- WorkPool - handles Background Goroutines or Async Jobs (emails) with Ants
	cj           *cronjob.Sched // <- CronJob scheduler
	Cache        Cache
	SessionStore *session.Store
	Localizer    *localize.Localizer
	Tracer       trace.Tracer
	*fiber.App
}

type ServerOption func(*Server) error

// New constructs a Server with the provided options and core setup.
func New(serverOpts ...ServerOption) *Server {
	server := Server{}
	if err := server.applyServerOptions(serverOpts...); err != nil {
		log.Fatal("🔴 failed to start server: option setup error:", err)
	}

	if err := server.validateCriticalDeps(); err != nil {
		log.Fatal("🔴 failed to start server:", err)
	}

	if err := server.validateRuntimeConfig(); err != nil {
		log.Fatal("🔴 failed to start server:", err)
	}

	server.setupSessionStore()
	server.setupFiberApp()

	return &server
}

func (s *Server) applyServerOptions(serverOpts ...ServerOption) error {
	for _, fnOpt := range serverOpts {
		err := fnOpt(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) validateCriticalDeps() error {
	if s.Env == nil {
		return errors.New("environment config is required")
	}

	if s.DB == nil {
		return errors.New("Database is required")
	}

	if s.Tracer == nil {
		return errors.New("tracer is required; pass server.WithTracer(...) to server.New(...)")
	}

	return nil
}

func (s *Server) validateRuntimeConfig() error {
	environment := s.Env.Environment
	if !appenv.IsValidEnvironment(environment) {
		return errors.New("APP_ENV is invalid")
	}

	cookieSecret := s.Env.CookieSecret
	if cookieSecret == "" {
		return errors.New("COOKIE_SECRET is required")
	}

	if len(cookieSecret) < 32 {
		return errors.New("COOKIE_SECRET must be at least 32 characters")
	}

	return nil
}

func (s *Server) setupSessionStore() {
	sessionPath := s.Env.SessionDBPath
	storage := sqlite3.New(sessionConfig(sessionPath))
	cookieSecure := appenv.IsProduction(s.Env.Environment) || strings.EqualFold(s.Env.AppProtocol, "https")
	s.SessionStore = session.NewStore(session.Config{
		Storage:      storage,
		CookieSecure: cookieSecure,
	})
}

func (s *Server) setupFiberApp() {
	environment := s.Env.Environment
	cookieSecret := s.Env.CookieSecret
	fiberConfig := fiber.Config{ErrorHandler: fiberAppErrorHandler}
	fiberApp := fiber.New(fiberConfig)

	s.setupCoreMiddleware(fiberApp)
	s.setupSecurityMiddleware(fiberApp, cookieSecret)
	s.setupObservability(fiberApp)
	s.setupStaticFiles(fiberApp, environment)

	s.App = fiberApp
}

func (s *Server) setupCoreMiddleware(app *fiber.App) {
	app.Use(recover.New()) // Recover MW catches panics that might stop app execution
	app.Use(s.otelMiddleware())
	app.Use(logger.New())
	app.Use(cors.New())
	app.Use(requestid.New())
	app.Get(healthcheck.LivenessEndpoint, healthcheck.New())
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New())
	app.Get(healthcheck.StartupEndpoint, healthcheck.New())
}

func (s *Server) otelMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		carrier := propagation.MapCarrier{
			"traceparent": c.Get("traceparent"),
			"tracestate":  c.Get("tracestate"),
			"baggage":     c.Get("baggage"),
		}

		ctx := otel.GetTextMapPropagator().Extract(c.Context(), carrier)
		spanName := fmt.Sprintf("%s %s", c.Method(), c.Path())
		ctx, span := s.Tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Method()),
				attribute.String("url.path", c.Path()),
			),
		)

		c.SetContext(ctx)
		err := c.Next()

		statusCode := c.Response().StatusCode()
		span.SetAttributes(attribute.Int("http.status_code", statusCode))
		if route := c.Route(); route != nil && route.Path != "" {
			span.SetName(fmt.Sprintf("%s %s", c.Method(), route.Path))
			span.SetAttributes(attribute.String("http.route", route.Path))
		}

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return err
		}

		if statusCode >= 500 {
			span.SetStatus(codes.Error, httpStatusText(statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		span.End()
		return nil
	}
}

func httpStatusText(code int) string {
	switch code {
	case fiber.StatusInternalServerError:
		return "internal server error"
	case fiber.StatusBadGateway:
		return "bad gateway"
	case fiber.StatusServiceUnavailable:
		return "service unavailable"
	case fiber.StatusGatewayTimeout:
		return "gateway timeout"
	default:
		return "request failed"
	}
}

func (s *Server) setupSecurityMiddleware(app *fiber.App, cookieSecret string) {
	app.Use(helmet.New(helmetConfig()))
	app.Use(encryptcookie.New(encryptcookie.Config{Key: cookieSecret}))
	app.Use(favicon.New(favicon.Config{File: "./public/favicon.ico", URL: "/favicon.ico"}))
	app.Use(limiter.New(limiterConfig()))
}

func (s *Server) setupObservability(app *fiber.App) {
	if s.Env != nil {
		env := s.Env.Environment
		if !appenv.IsDevOrTest(env) {
			return
		}
	}

	startedAt := time.Now()
	app.Get("/metrics", func(c fiber.Ctx) error {
		mem := runtime.MemStats{}
		runtime.ReadMemStats(&mem)

		return c.JSON(fiber.Map{
			"uptime_seconds":    int(time.Since(startedAt).Seconds()),
			"goroutines":        runtime.NumGoroutine(),
			"alloc_bytes":       mem.Alloc,
			"total_alloc_bytes": mem.TotalAlloc,
			"sys_bytes":         mem.Sys,
			"heap_objects":      mem.HeapObjects,
		})
	})
}

func (s *Server) setupStaticFiles(app *fiber.App, environment appenv.Environment) {
	app.Use("/public", static.New("./public", staticConfig(environment)))
}

// WithEnv configures the server environment.
func WithEnv(env *appenv.Env) ServerOption {
	return func(server *Server) error {
		if env == nil {
			return errors.New("failed to start server without environment config")
		}

		server.Env = env
		return nil
	}
}

// WithDatabase configures the server database dependency.
func WithDatabase(db *database.Database) ServerOption {
	return func(server *Server) error {
		if db == nil {
			return errors.New("failed to start server without Database connection")
		}

		server.DB = db
		return nil
	}
}

// WithLocalizer configures the optional localization service.
func WithLocalizer(localizer *localize.Localizer) ServerOption {
	return func(server *Server) error {
		if localizer == nil {
			return nil
		}

		server.Localizer = localizer
		return nil
	}
}

// WithWorkPool configures the optional async worker pool.
func WithWorkPool(wp *ants.Pool) ServerOption {
	return func(server *Server) error {
		if wp == nil {
			log.Warn("🟡 failed to set up workpool for async jobs; running synchronous fallback")
			return nil
		}

		server.wp = wp
		return nil
	}
}

// WithCronJob configures the optional cron scheduler.
func WithCronJob(cj *cronjob.Sched) ServerOption {
	return func(server *Server) error {
		if cj == nil {
			log.Warn("🟡 failed to set up cron scheduler; cron jobs will not run")
			return nil
		}

		server.cj = cj
		return nil
	}
}

// WithTracer configures the optional tracer implementation.
func WithTracer(tracer trace.Tracer) ServerOption {
	return func(server *Server) error {
		if tracer == nil {
			return nil
		}

		server.Tracer = tracer
		return nil
	}
}

// WithCache configures the optional cache implementation.
func WithCache(cache Cache) ServerOption {
	return func(server *Server) error {
		if cache == nil {
			return nil
		}

		server.Cache = cache
		return nil
	}
}

// AddCronJob passes fn as a job(fn) to run at a cron interval
func (s *Server) AddCronJob(crontab string, fn func()) error {
	if s.cj == nil {
		return errors.New("failed to add new cron job, server.cj might be nil, cron job is not running")
	}

	_, err := s.cj.RunCron(crontab, false, fn)
	return err
}

// SendToWorker passes fn as a job for a worker in the workpool, to be executed as a go routine
// when the a worker is available
func (s *Server) SendToWorker(fn func()) error {
	if s.wp == nil {
		log.Warn("failed to add fn to run as job in worker pool, server.wp might be nil, running synchronously")
		fn()
		return nil
	}

	err := s.wp.Submit(fn)
	return err
}

// GormDB returns the active gorm DB handle when available.
func (s *Server) GormDB() *gorm.DB {
	if s.DB == nil {
		return nil
	}

	return s.DB.GormDB()
}

// AppEnv returns the active application environment configuration.
func (s *Server) AppEnv() *appenv.Env {
	return s.Env
}

// Trace starts a span with the configured tracer and returns updated context.
func (s *Server) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, span := s.Tracer.Start(ctx, spanName, opts...)
	return ctx, span
}

// Listen starts the fiberapp server (fiberApp.Listen()) on the specified port.
func (s *Server) Listen(portOverride ...int) error {
	port := 0
	if s.Env != nil && s.Env.AppPort > 0 {
		port = s.Env.AppPort
	}

	if len(portOverride) > 0 {
		port = portOverride[0]
	}

	if port <= 0 {
		port = 3000
	}

	return s.App.Listen(":" + strconv.Itoa(port))
}

// CacheWrite writes a value to the Cache
func (s *Server) CacheWrite(key string, src *[]byte, ttl time.Duration) error {
	if s.Cache == nil {
		return nil
	}
	err := s.Cache.Write(key, src, ttl)

	return err
}

// CacheRead reads a cache value by key
func (s *Server) CacheRead(key string, dst *[]byte) error {
	if s.Cache == nil {
		return nil
	}
	return s.Cache.Read(key, dst)
}
