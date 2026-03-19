// Package server provides a server for the application that can be extended with routers.
package server

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/jobs"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/localize"
	obslogging "miconsul/internal/observability/logging"
	obsmetrics "miconsul/internal/observability/metrics"

	otelfiber "github.com/gofiber/contrib/v3/otel"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type Cache interface {
	Read(key string, dst *[]byte) error
	Write(key string, src *[]byte, ttl time.Duration) error
}

type Server struct {
	Env               *appenv.Env
	DB                *database.Database
	wp                *ants.Pool // <- WorkPool - handles Background Goroutines or Async Jobs (emails) with Ants
	jobs              *jobs.Runtime
	Cache             Cache
	SessionStore      *session.Store
	Localizer         *localize.Localizer
	Tracer            trace.Tracer
	RequestLog        obslogging.Logger
	Metrics           obsmetrics.HTTPMetrics
	StartedAt         time.Time
	ReadyAt           time.Time
	BootstrapDuration time.Duration
	*fiber.App
}

type ServerOption func(*Server) error

// New constructs a Server with the provided options and core setup.
func New(serverOpts ...ServerOption) *Server {
	server := Server{
		StartedAt: time.Now(),
	}

	for _, fnOpt := range serverOpts {
		if err := fnOpt(&server); err != nil {
			log.Fatal("🔴 failed to start server: option setup error:", err)
		}
	}

	if err := validateCriticalDeps(&server); err != nil {
		log.Fatal("🔴 failed to start server:", err)
	}

	if err := validateRuntimeConfig(&server); err != nil {
		log.Fatal("🔴 failed to start server:", err)
	}

	setupSessionStore(&server)
	setupFiberApp(&server)
	server.ReadyAt = time.Now()
	server.BootstrapDuration = server.ReadyAt.Sub(server.StartedAt)
	emitStartupBootstrapLog(&server)

	return &server
}

func validateCriticalDeps(s *Server) error {
	if s == nil {
		return errors.New("server is required")
	}

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

func validateRuntimeConfig(s *Server) error {
	if s == nil {
		return errors.New("server is required")
	}

	if !s.Env.IsValidEnvironment() {
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

func setupSessionStore(s *Server) {
	sessionPath := s.Env.SessionDBPath
	storage := sqlite3.New(sessionConfig(sessionPath))
	cookieSecure := s.Env.IsProduction() || strings.EqualFold(s.Env.AppProtocol, "https")
	s.SessionStore = session.NewStore(session.Config{
		Storage:      storage,
		CookieSecure: cookieSecure,
	})
}

func setupFiberApp(s *Server) {
	fiberConfig := fiber.Config{ErrorHandler: fiberAppErrorHandler}
	s.App = fiber.New(fiberConfig)

	setupCoreMiddleware(s)
	setupSecurityMiddleware(s)
	setupObservability(s)
	setupStaticFiles(s)
	setupHealthcheckRoutes(s)
}

func setupCoreMiddleware(s *Server) {
	app := s.App

	app.Use(recover.New()) // Recover MW catches panics that might stop app execution
	app.Use(RequestMetricsMiddleware(s.Metrics))
	app.Use(otelfiber.Middleware(
		otelfiber.WithNext(func(c fiber.Ctx) bool {
			path := c.Path()
			return strings.HasPrefix(path, "/public/") ||
				strings.HasPrefix(path, "/.well-known/") ||
				path == "/favicon.ico"
		}),
	))
	app.Use(logger.New())
	app.Use(cors.New())
	app.Use(requestid.New())
	app.Use(RequestLoggerMiddleware(s.RequestLog))
}

func setupSecurityMiddleware(s *Server) {
	app := s.App
	cookieSecret := s.Env.CookieSecret

	app.Use(helmet.New(helmetConfig()))
	app.Use(encryptcookie.New(encryptcookie.Config{Key: cookieSecret}))
	app.Use(favicon.New(favicon.Config{File: "./public/favicon.ico", URL: "/favicon.ico"}))

	rateLimiterEnabled := s.Env.RateLimiterEnabled && !s.Env.IsDevelopment()
	if rateLimiterEnabled {
		app.Use(limiter.New(limiterConfig()))
	}
}

func setupObservability(s *Server) {
	app := s.App
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
}

func setupStaticFiles(s *Server) {
	app := s.App
	app.Use("/public", static.New("./public", staticConfig(s.Env)))
	app.Use("/.well-known", static.New("./public/.well-known", staticConfig(s.Env)))
}

func setupHealthcheckRoutes(s *Server) {
	app := s.App

	app.Get(healthcheck.LivenessEndpoint, healthcheck.New(healthcheck.Config{
		Probe: livenessProbe(s),
	}))
	app.Get(healthcheck.ReadinessEndpoint, healthcheck.New(healthcheck.Config{
		Probe: readinessProbe(s),
	}))
	app.Get(healthcheck.StartupEndpoint, healthcheck.New(healthcheck.Config{
		Probe: startupProbe(s),
	}))
}

func emitStartupBootstrapLog(s *Server) {
	if !s.RequestLog.Enabled() {
		return
	}

	rec := otellog.Record{}
	rec.SetTimestamp(time.Now())
	rec.SetObservedTimestamp(time.Now())
	rec.SetEventName("server_startup")
	rec.SetBody(otellog.StringValue("server_startup"))
	rec.SetSeverity(otellog.SeverityInfo)
	rec.SetSeverityText("INFO")
	rec.AddAttributes(
		otellog.String("event", "server_startup"),
		otellog.String("started_at", s.StartedAt.UTC().Format(time.RFC3339)),
		otellog.String("ready_at", s.ReadyAt.UTC().Format(time.RFC3339)),
		otellog.Int64("bootstrap_duration_ms", s.BootstrapDuration.Milliseconds()),
		otellog.String("version", s.Env.AppVersion),
		otellog.String("environment", string(s.Env.Environment)),
	)

	s.RequestLog.Emit(context.Background(), rec)
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

// WithJobs configures the optional jobs runtime.
func WithJobs(j *jobs.Runtime) ServerOption {
	return func(server *Server) error {
		if j == nil {
			log.Warn("🟡 failed to set up jobs runtime; background jobs will not run")
			return nil
		}

		server.jobs = j
		return nil
	}
}

// WithTracer configures the tracer implementation.
func WithTracer(tracer trace.Tracer) ServerOption {
	return func(server *Server) error {
		if tracer == nil {
			return nil
		}

		server.Tracer = tracer
		return nil
	}
}

// WithMetrics configures the HTTP metrics instruments.
func WithMetrics(meter obsmetrics.HTTPMetrics) ServerOption {
	return func(server *Server) error {
		server.Metrics = meter
		return nil
	}
}

// WithRequestLogger configures the optional OTLP request logger.
func WithRequestLogger(logger obslogging.Logger) ServerOption {
	return func(server *Server) error {
		server.RequestLog = logger
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
