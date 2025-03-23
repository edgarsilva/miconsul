// Package server provides a server for the application that can be extended with routers.
package server

import (
	"context"
	"errors"
	"fmt"
	"miconsul/internal/database"
	"miconsul/internal/lib/cache"
	"miconsul/internal/lib/cronjob"
	"miconsul/internal/lib/localize"
	"miconsul/internal/lib/sessionstorage"
	"miconsul/internal/model"
	"os"
	"time"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/encryptcookie"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/fiber/v2/middleware/session"

	logto "github.com/logto-io/go/client"
	"github.com/panjf2000/ants/v2"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Cache interface {
	Read(key string, dst *[]byte) error
	Write(key string, src *[]byte, ttl time.Duration) error
}

type Server struct {
	DB           *database.Database
	wp           *ants.Pool     // <- WorkerPool - handles Background Goroutines or Async Jobs (emails) with Ants
	cj           *cronjob.Sched // <- CronJob scheduler
	Cache        Cache
	SessionStore *session.Store
	Localizer    *localize.Localizer
	LogtoConfig  *logto.LogtoConfig
	TP           *sdktrace.TracerProvider
	Tracer       trace.Tracer
	*fiber.App
}

type ServerOption func(*Server) error

func New(serverOpts ...ServerOption) *Server {
	server := Server{}
	for _, fnOpt := range serverOpts {
		err := fnOpt(&server)
		if err != nil {
			log.Fatal("🔴 Failed to start server... exiting")
		}
	}

	storage := sessionstorage.New()
	server.SessionStore = session.New(session.Config{
		Storage:      storage,
		CookieSecure: true,
	})

	tracer := otel.Tracer("fiberapp-server")
	server.Tracer = tracer

	fiberConfig := fiber.Config{ErrorHandler: fiberAppErrorHandler}
	fiberApp := fiber.New(fiberConfig)
	fiberApp.Use(recover.New()) // Recover MW catches panics that might stop app execution
	fiberApp.Use(otelfiber.Middleware())
	fiberApp.Use(logger.New())

	fiberApp.Use(cors.New())

	fiberApp.Use(helmet.New(helmetConfig()))

	fiberApp.Use(requestid.New())
	fiberApp.Use(encryptcookie.New(encryptcookie.Config{
		Key: os.Getenv("COOKIE_SECRET"),
	}))
	fiberApp.Use(favicon.New(favicon.Config{
		File: "./public/favicon.ico",
		URL:  "/favicon.ico",
	}))
	fiberApp.Use(healthcheck.New()) // healthcheck endpoints /livez /readyz
	fiberApp.Use(limiter.New(limiterConfig()))
	fiberApp.Get("/metrics", monitor.New()) // app monitor @ /metrics
	fiberApp.Static("/public", "./public", staticConfig())

	server.App = fiberApp

	logtoConfig := LogtoConfig()
	server.LogtoConfig = logtoConfig

	return &server
}

func WithDatabase(db *database.Database) ServerOption {
	return func(server *Server) error {
		if db == nil {
			return errors.New("failed to start server without Database conection")
		}

		server.DB = db
		return nil
	}
}

func WithLocalizer(localizer *localize.Localizer) ServerOption {
	return func(server *Server) error {
		if localizer == nil {
			return nil
		}

		server.Localizer = localizer
		return nil
	}
}

func WithWorkerPool(wp *ants.Pool) ServerOption {
	return func(server *Server) error {
		if wp == nil {
			fmt.Println("failed to server worker pool for sending emails and async work")
			return nil
		}

		server.wp = wp
		return nil
	}
}

func WithCronJob(cj *cronjob.Sched) ServerOption {
	return func(server *Server) error {
		if cj == nil {
			fmt.Println("failed to start server cron job scheduler")
			return nil
		}

		server.cj = cj
		return nil
	}
}

func WithTracerProvider(tp *sdktrace.TracerProvider) ServerOption {
	return func(server *Server) error {
		if tp == nil {
			return nil
		}

		server.TP = tp
		return nil
	}
}

func WithCache(cache *cache.Cache) ServerOption {
	return func(server *Server) error {
		if cache == nil {
			return nil
		}

		server.Cache = cache
		return nil
	}
}

// SendToWorker passes fn as a job to the worker pool to be executed in a go routine
func (s *Server) AddCronJob(crontab string, fn func()) error {
	if s.wp == nil {
		return errors.New("failed to add new cron job, server.cj might be nil, cron job is not running")
	}

	_, err := s.cj.RunCron(crontab, false, fn)
	return err
}

// SendToWorker passes fn as a job to the worker pool to be executed in a go routine
func (s *Server) SendToWorker(fn func()) error {
	if s.wp == nil {
		fmt.Println("failed to add fn to run as job in worker pool, server.wp might be nil, running sinchronously")
		fn()
		return nil
	}

	err := s.wp.Submit(fn)
	return err
}

// CurrentUser returns currently logged-in(or anon) user by User.ID from fiber.Locals("id")
func (s *Server) CurrentUser(c *fiber.Ctx) (model.User, error) {
	userI := c.Locals("current_user")
	cu, ok := userI.(model.User)
	if !ok {
		return model.User{}, nil
	}

	return cu, nil
}

func (s *Server) DBClient() *database.Database {
	return s.DB
}

func (s *Server) TracerClient() trace.Tracer {
	return s.Tracer
}

func (s *Server) Trace(c *fiber.Ctx, spanName string) (context.Context, func()) {
	ctx, span := s.Tracer.Start(c.UserContext(), spanName)
	return ctx, func() {
		span.End()
	}
}

// Listen starts the fiberapp server (fiperapp.Listen()) on the specified port.
func (s *Server) Listen(port string) error {
	return s.App.Listen(fmt.Sprintf(":%v", port))
}

func (s *Server) LogtoEnabled() bool {
	logtourl := os.Getenv("LOGTO_URL")
	return logtourl != ""
}

func (s *Server) Session(c *fiber.Ctx) *session.Session {
	sess, err := s.SessionStore.Get(c)
	if err != nil {
		log.Info("Failed to retrieve session from req ctx:", err)
	}
	return sess
}

func (s *Server) SessionDestroy(c *fiber.Ctx) {
	sess := s.Session(c)
	err := sess.Destroy()
	if err != nil {
		log.Info("Failed to destroy session:", err)
	}
}

// SessionWrite sets a session value.
func (s *Server) SessionWrite(c *fiber.Ctx, k string, v interface{}) (err error) {
	sess := s.Session(c)
	sess.Set(k, v)
	defer func() {
		err = sess.Save()
	}()

	return nil
}

// SessionRead gets a session string value by key, or returns the default value.
func (s *Server) SessionRead(c *fiber.Ctx, key string, defaultVal string) string {
	sess := s.Session(c)
	val := sess.Get(key)
	if val == nil {
		return defaultVal
	}

	valStr, ok := val.(string)
	if !ok {
		return defaultVal
	}

	return valStr
}

// CacheWrite writes a value to the Cache
func (s *Server) CacheWrite(key string, src *[]byte, ttl time.Duration) error {
	err := s.Cache.Write(key, src, ttl)

	return err
}

// CacheRead reads a cache value by key
func (s *Server) CacheRead(key string, dst *[]byte) error {
	return s.Cache.Read(key, dst)
}

// SessionUITheme returns the user UI theme (light|dark) from the session or query
// url param
func (s *Server) SessionUITheme(c *fiber.Ctx) string {
	theme, ok := c.Locals("theme").(string)
	if !ok || theme == "" {
		theme = "light"
	}

	return theme
}

// SessionLang returns the user language from header Accepts-Language or session
func (s *Server) SessionLang(c *fiber.Ctx) string {
	sess := s.Session(c)
	lang, ok := sess.Get("lang").(string)
	if ok && lang != "" {
		return lang
	}

	lang, ok = c.Locals("lang").(string)
	if !ok || lang == "" {
		lang = "es-MX"
	}

	sess.Set("lang", lang)
	return lang
}

// SessionUITheme returns the user UI theme (light|dark) from the session or query
// url param
func (s *Server) SessionID(c *fiber.Ctx) string {
	sessionID := c.Cookies("session_id", "")
	return sessionID
}

// SessionUITheme returns the user UI theme (light|dark) from the session or query
// url param
func (s *Server) TagWithSessionID(c *fiber.Ctx, tag string) string {
	return s.SessionID(c) + ":" + tag
}

// IsHTMX returns true if the request was initiated by HTMX
func (s *Server) IsHTMX(c *fiber.Ctx) bool {
	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	return isHTMX == "true"
}

// IsHTMX returns true if the request was initiated by HTMX
func (s *Server) NotHTMX(c *fiber.Ctx) bool {
	return !s.IsHTMX(c)
}

func (s *Server) L(c *fiber.Ctx, key string) (translation string) {
	if s.Localizer == nil {
		return ""
	}

	return s.Localizer.GetWithLocale(s.SessionLang(c), key)
}
