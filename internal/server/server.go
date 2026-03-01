// Package server provides a server for the application that can be extended with routers.
package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/cronjob"
	"miconsul/internal/lib/localize"
	"miconsul/internal/model"

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
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type Cache interface {
	Read(key string, dst *[]byte) error
	Write(key string, src *[]byte, ttl time.Duration) error
}

type Server struct {
	AppEnv       *appenv.Env
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

func New(serverOpts ...ServerOption) *Server {
	server := Server{}
	for _, fnOpt := range serverOpts {
		err := fnOpt(&server)
		if err != nil {
			log.Fatal("🔴 Failed to start server... exiting")
		}
	}

	sessionPath := ""
	if server.AppEnv != nil {
		sessionPath = server.AppEnv.SessionDBPath
	}
	storage := sqlite3.New(sessionConfig(sessionPath))
	server.SessionStore = session.NewStore(session.Config{
		Storage:      storage,
		CookieSecure: true,
	})

	tracer := server.Tracer
	if tracer == nil {
		tracer = otel.Tracer("fiberapp-server")
	}
	server.Tracer = tracer

	fiberConfig := fiber.Config{ErrorHandler: fiberAppErrorHandler}
	fiberApp := fiber.New(fiberConfig)
	fiberApp.Use(recover.New()) // Recover MW catches panics that might stop app execution
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
	startedAt := time.Now()
	fiberApp.Get("/metrics", func(c fiber.Ctx) error {
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
	fiberApp.Use("/public", static.New("./public", staticConfig()))

	server.App = fiberApp

	return &server
}

func WithAppEnv(env *appenv.Env) ServerOption {
	return func(server *Server) error {
		if env == nil {
			return errors.New("failed to start server without AppEnv")
		}

		server.AppEnv = env
		return nil
	}
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

func WithWorkPool(wp *ants.Pool) ServerOption {
	return func(server *Server) error {
		if wp == nil {
			fmt.Println("failed to setup workpool for sending emails and async work")
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

func WithTracer(tracer trace.Tracer) ServerOption {
	return func(server *Server) error {
		if tracer == nil {
			return nil
		}

		server.Tracer = tracer
		return nil
	}
}

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
		log.Warn("failed to add fn to run as job in worker pool, server.wp might be nil, running sinchronously")
		fn()
		return nil
	}

	err := s.wp.Submit(fn)
	return err
}

// CurrentUser returns currently logged-in(or anon) user by User.ID from fiber.Locals("id")
func (s *Server) CurrentUser(c fiber.Ctx) (model.User, error) {
	userI := c.Locals("current_user")
	cu, ok := userI.(model.User)
	if !ok {
		return model.User{}, nil
	}

	return cu, nil
}

func (s *Server) GormDB() *gorm.DB {
	if s == nil || s.DB == nil {
		return nil
	}

	return s.DB.GormDB()
}

func (s *Server) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if s == nil {
		panic("server.Trace called with nil server")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	tracer := s.Tracer
	if tracer == nil {
		tracer = otel.Tracer("fiberapp-server")
	}

	ctx, span := tracer.Start(ctx, spanName, opts...)
	return ctx, span
}

// Listen starts the fiberapp server (fiperapp.Listen()) on the specified port.
func (s *Server) Listen(portOverride ...int) error {
	port := 0
	if s.AppEnv != nil && s.AppEnv.AppPort > 0 {
		port = s.AppEnv.AppPort
	}

	if len(portOverride) > 0 {
		port = portOverride[0]
	}

	if port <= 0 {
		port = 3000
	}

	return s.App.Listen(":" + strconv.Itoa(port))
}

func (s *Server) Session(c fiber.Ctx) (*session.Session, error) {
	if s == nil {
		err := errors.New("failed to retrieve session: server is nil")
		log.Warn(err.Error())
		return nil, err
	}

	if s.SessionStore == nil {
		err := errors.New("failed to retrieve session: session store is nil")
		log.Warn(err.Error())
		return nil, err
	}

	sess, err := s.SessionStore.Get(c)
	if err != nil {
		log.Warn("Failed to retrieve session from req ctx:", err)
		return nil, err
	}

	return sess, nil
}

func (s *Server) SessionDestroy(c fiber.Ctx) {
	sess, err := s.Session(c)
	if err != nil {
		return
	}

	err = sess.Destroy()
	if err != nil {
		log.Info("Failed to destroy session:", err)
	}
}

// SessionWrite sets a session value.
func (s *Server) SessionWrite(c fiber.Ctx, k string, v any) (err error) {
	sess, err := s.Session(c)
	if err != nil {
		return err
	}

	sess.Set(k, v)
	return sess.Save()
}

// SessionRead gets a session string value by key, or returns the default value.
func (s *Server) SessionRead(c fiber.Ctx, key string, defaultVal string) string {
	sess, err := s.Session(c)
	if err != nil {
		return defaultVal
	}

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

// SessionUITheme returns the user UI theme (light|dark) from the session or query
// url param
func (s *Server) SessionUITheme(c fiber.Ctx) string {
	theme, ok := c.Locals("theme").(string)
	if !ok || theme == "" {
		theme = "light"
	}

	return theme
}

// CurrentLocale returns the user locale resolved for the current request.
func (s *Server) CurrentLocale(c fiber.Ctx) string {
	sess, err := s.Session(c)
	if err != nil {
		lang, ok := c.Locals("locale").(string)
		if !ok || lang == "" {
			lang = "es-MX"
		}

		return lang
	}

	lang, ok := sess.Get("lang").(string)
	if ok && lang != "" {
		return lang
	}

	lang, ok = c.Locals("locale").(string)
	if !ok || lang == "" {
		lang = "es-MX"
	}

	sess.Set("lang", lang)
	if err := sess.Save(); err != nil {
		log.Warn("Failed to save session language:", err)
	}

	return lang
}

// SessionID returns the user UI theme (light|dark) from the session or query
// url param
func (s *Server) SessionID(c fiber.Ctx) string {
	sessionID := c.Cookies("session_id", "")
	return sessionID
}

// TagWithSessionID returns tags the passed tag with the SessionID
// url param
func (s *Server) TagWithSessionID(c fiber.Ctx, tag string) string {
	return s.SessionID(c) + ":" + tag
}

// IsHTMX returns true if the request was initiated by HTMX
func (s *Server) IsHTMX(c fiber.Ctx) bool {
	isHTMX := c.Get("HX-Request", "") // will be a string 'true' for HTMX requests
	return isHTMX == "true"
}

// IsHTMX returns true if the request was initiated by HTMX
func (s *Server) NotHTMX(c fiber.Ctx) bool {
	return !s.IsHTMX(c)
}

func (s *Server) L(c fiber.Ctx, key string) (translation string) {
	if s.Localizer == nil {
		return ""
	}

	return s.Localizer.GetWithLocale(s.CurrentLocale(c), key)
}
