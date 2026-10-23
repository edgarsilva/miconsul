// Package server provides a server for the application that can be extended with routers.
package server

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/jobs"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/localize"

	"github.com/gofiber/contrib/v3/otel"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
	"github.com/gofiber/fiber/v3/middleware/favicon"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	fiberrecover "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/gofiber/storage/sqlite3/v2"

	"github.com/panjf2000/ants/v2"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
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
	TracerProvider    trace.TracerProvider
	Tracer            trace.Tracer
	MeterProvider     metric.MeterProvider
	LoggerProvider    otellog.LoggerProvider
	Logger            otellog.Logger
	StartedAt         time.Time
	ReadyAt           time.Time
	BootstrapDuration time.Duration
	*fiber.App
}

type ServerOption func(*Server) error

// New constructs a Server with the provided options and core setup. It
// returns an error when an option fails or a required dependency or setting
// is missing.
func New(serverOpts ...ServerOption) (*Server, error) {
	server := Server{
		StartedAt: time.Now(),
	}

	for _, fnOpt := range serverOpts {
		if err := fnOpt(&server); err != nil {
			return nil, fmt.Errorf("apply server option: %w", err)
		}
	}

	if err := validateCriticalDeps(&server); err != nil {
		return nil, fmt.Errorf("validate server dependencies: %w", err)
	}

	if err := validateRuntimeConfig(&server); err != nil {
		return nil, fmt.Errorf("validate server config: %w", err)
	}

	setupTracer(&server)
	setupLogger(&server)
	setupSessionStore(&server)
	setupFiberApp(&server)
	server.ReadyAt = time.Now()
	server.BootstrapDuration = server.ReadyAt.Sub(server.StartedAt)
	emitStartupBootstrapLog(&server)

	return &server, nil
}

func validateCriticalDeps(s *Server) error {
	if s == nil {
		return errors.New("server is required")
	}

	if s.Env == nil {
		return errors.New("environment config is required")
	}

	if s.DB == nil {
		return errors.New("database is required")
	}

	if s.TracerProvider == nil {
		return errors.New("tracer provider is required; pass server.WithTracerProvider(...) to server.New(...)")
	}

	if s.MeterProvider == nil {
		return errors.New("meter provider is required; pass server.WithMeterProvider(...) to server.New(...)")
	}

	if s.LoggerProvider == nil {
		return errors.New("logger provider is required; pass server.WithLoggerProvider(...) to server.New(...)")
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

	return nil
}

func setupTracer(s *Server) {
	s.Tracer = s.TracerProvider.Tracer(scopeName(s.Env))
}

func setupLogger(s *Server) {
	s.Logger = s.LoggerProvider.Logger(scopeName(s.Env))
}

// scopeName returns the instrumentation scope for the server's tracer and logger.
func scopeName(env *appenv.Env) string {
	if env.OTelTracerServer != "" {
		return env.OTelTracerServer
	}

	return env.AppName + ".server"
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
	fiberConfig := fiber.Config{
		ErrorHandler: fiberAppErrorHandler,
		JSONEncoder:  marshalJSON,
		JSONDecoder:  unmarshalJSON,
	}
	s.App = fiber.New(fiberConfig)

	setupCoreMiddleware(s)
	setupSecurityMiddleware(s)
	setupStaticFiles(s)
	setupHealthcheckRoutes(s)
}

func setupCoreMiddleware(s *Server) {
	app := s.App

	app.Use(fiberrecover.New()) // Recover MW catches panics that might stop app execution
	app.Use(otel.New(
		otel.WithTracerProvider(s.TracerProvider),
		otel.WithMeterProvider(s.MeterProvider),
		otel.WithNext(func(c fiber.Ctx) bool {
			path := c.Path()
			return path == "/readyz" || path == "/livez"
		}),
	))
	app.Use(logger.New())
	app.Use(cors.New())
	app.Use(requestid.New())
	app.Use(RequestLoggerMiddleware(s.Logger))
}

func setupSecurityMiddleware(s *Server) {
	app := s.App
	cookieSecret := s.Env.CookieSecret

	app.Use(helmet.New(helmetConfig()))
	app.Use(csrf.NewCrossOriginProtection())
	app.Use(encryptcookie.New(encryptcookie.Config{Key: cookieSecret}))
	app.Use(favicon.New(favicon.Config{File: "./public/favicon.ico", URL: "/favicon.ico"}))

	rateLimiterEnabled := s.Env.RateLimiterEnabled && !s.Env.IsDevelopment()
	if rateLimiterEnabled {
		app.Use(limiter.New(limiterConfig()))
	}
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

// WithEnv configures the server environment.
func WithEnv(env *appenv.Env) ServerOption {
	return func(server *Server) error {
		if env == nil {
			return errors.New("start server without environment config")
		}

		server.Env = env
		return nil
	}
}

// WithDatabase configures the server database dependency.
func WithDatabase(db *database.Database) ServerOption {
	return func(server *Server) error {
		if db == nil {
			return errors.New("start server without Database connection")
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

// WithTracerProvider sets the server's tracer provider.
func WithTracerProvider(provider trace.TracerProvider) ServerOption {
	return func(server *Server) error {
		if provider == nil {
			return nil
		}

		server.TracerProvider = provider
		return nil
	}
}

// WithMeterProvider sets the server's meter provider.
func WithMeterProvider(provider metric.MeterProvider) ServerOption {
	return func(server *Server) error {
		if provider == nil {
			return nil
		}

		server.MeterProvider = provider
		return nil
	}
}

// WithLoggerProvider sets the server's logger provider.
func WithLoggerProvider(provider otellog.LoggerProvider) ServerOption {
	return func(server *Server) error {
		if provider == nil {
			return nil
		}

		server.LoggerProvider = provider
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
