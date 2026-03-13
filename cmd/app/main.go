package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/cronjob"
	"miconsul/internal/lib/localize"
	"miconsul/internal/lib/workpool"
	"miconsul/internal/observability/logging"
	"miconsul/internal/observability/metrics"
	"miconsul/internal/observability/tracing"
	"miconsul/internal/routes"
	"miconsul/internal/server"

	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel/trace"
)

type telemetryRuntime struct {
	tracer          trace.Tracer
	httpMetrics     metrics.HTTPMetrics
	requestLogger   logging.Logger
	dbLogger        logging.Logger
	shutdownTracer  func() error
	shutdownMetrics func() error
	shutdownLogs    func() error
}

type serverRunner interface {
	Listen(portOverride ...int) error
	ShutdownWithContext(ctx context.Context) error
}

var (
	setupEnvForMain       = setupEnv
	setupTelemetryForMain = setupTelemetry
	setupDBForMain        = setupDB
	newCronjobForMain     = cronjob.New
	newWorkpoolForMain    = workpool.New
	setupServerForMain    = setupServer
	registerRoutesForMain = routes.RegisterServices
	runLifecycleForMain   = runServerLifecycle
	exitForMain           = os.Exit
	notifyContextForMain  = signal.NotifyContext
)

func main() {
	exitCode := 0
	defer func() {
		if exitCode != 0 {
			exitForMain(exitCode)
		}
	}()

	fmt.Println(" Starting server...")

	fmt.Println(" Loading environment variables...")
	env, err := setupEnvForMain()
	if err != nil {
		log.Printf("failed to load environment config: %v", err)
		exitCode = 1
		return
	}

	defer func() {
		// This should be the last log on defer chain before exiting with code (0|1|etc)
		fmt.Println("✅ All cleanup tasks completed")
	}()

	ctx, stop := notifyContextForMain(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println(" Starting telemetry...")
	telemetry, err := setupTelemetryForMain(ctx, env)
	if err != nil {
		log.Printf("failed to initialize telemetry: %v", err)
		exitCode = 1
		return
	}
	defer func() {
		fmt.Println("󰓾 Tracer provider shutting down...")
		if err := telemetry.shutdownTracer(); err != nil {
			log.Printf("tracer shutdown error: %v", err)
		}
	}()
	defer func() {
		fmt.Println("󰓾 Meter provider shutting down...")
		if err := telemetry.shutdownMetrics(); err != nil {
			log.Printf("meter provider shutdown error: %v", err)
		}
	}()
	defer func() {
		fmt.Println("󰓾 Logger provider shutting down...")
		if err := telemetry.shutdownLogs(); err != nil {
			log.Printf("logger provider shutdown error: %v", err)
		}
	}()

	fmt.Println(" Connecting to database...")
	db, err := setupDBForMain(env, telemetry.dbLogger)
	if err != nil {
		log.Printf("failed to initialize database: %v", err)
		exitCode = 1
		return
	}
	defer func() {
		fmt.Println("🔌 Closing database connections...")
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	fmt.Println(" Starting cronjobs...")
	cj, shutdownCronjob, err := newCronjobForMain()
	if err != nil {
		log.Printf("failed to initialize cronjobs: %v", err)
		exitCode = 1
		return
	}
	defer func() {
		fmt.Println("🕑 Cronjobs shutting down...")
		if err := shutdownCronjob(); err != nil {
			log.Printf("cronjob shutdown error: %v", err)
		}
	}()

	fmt.Println(" Starting workpool...")
	wp, shutdownWorkPool := newWorkpoolForMain(10)
	defer func() {
		fmt.Println("🐜 Ants workpool shutting down...")
		shutdownWorkPool()
	}()

	fmt.Println(" Starting localizer...")
	s := setupServerForMain(env, db, cj, wp, telemetry, localize.New("en-US", "es-MX"))

	fmt.Println(" Registering routes...")
	if err := registerRoutesForMain(s); err != nil {
		log.Printf("failed to register routes: %v", err)
		exitCode = 1
		return
	}

	fmt.Println(" Setting up server...")
	runLifecycleForMain(ctx, s, env.AppPort, env.AppShutdownTimeout)

	fmt.Println("🧹 Running cleanup tasks...")
}

func setupEnv() (*appenv.Env, error) {
	_ = godotenv.Load(".env")

	env, err := appenv.New()
	if err != nil {
		return nil, err
	}

	return env, nil
}

func setupTelemetry(ctx context.Context, env *appenv.Env) (telemetryRuntime, error) {
	if env == nil {
		return telemetryRuntime{}, fmt.Errorf("environment config is required")
	}

	tracer, shutdownTracer, err := tracing.NewTracer(ctx, env.OTelTracerServer, env)
	if err != nil {
		return telemetryRuntime{}, fmt.Errorf("initialize otel tracer: %w", err)
	}

	fmt.Println(" Starting metrics telemetry...")
	httpMetrics, shutdownMetrics, err := metrics.New(ctx, env)
	if err != nil {
		return telemetryRuntime{}, fmt.Errorf("initialize otel meter provider: %w", err)
	}

	fmt.Println(" Starting logs telemetry...")
	logProvider, shutdownLogs, err := logging.NewProvider(ctx, env)
	if err != nil {
		return telemetryRuntime{}, fmt.Errorf("initialize otel logger provider: %w", err)
	}

	return telemetryRuntime{
		tracer:          tracer,
		httpMetrics:     httpMetrics,
		requestLogger:   logging.NewLogger(logProvider, env.AppName+".http"),
		dbLogger:        logging.NewLogger(logProvider, env.AppName+".db"),
		shutdownTracer:  shutdownTracer,
		shutdownMetrics: shutdownMetrics,
		shutdownLogs:    shutdownLogs,
	}, nil
}

func setupDB(env *appenv.Env, dbLogger logging.Logger) (*database.Database, error) {
	db, err := database.New(env, dbLogger, nil)
	if err != nil {
		return nil, err
	}

	fmt.Println(" Applying database migrations...")
	if err := database.ApplyMigrations(db, dbLogger); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func setupServer(env *appenv.Env, db *database.Database, cj *cronjob.Sched, wp *workpool.Pool, telemetry telemetryRuntime, localizer *localize.Localizer) *server.Server {
	return server.New(
		server.WithEnv(env),
		server.WithDatabase(db),
		server.WithCronJob(cj),
		server.WithWorkPool(wp.AntsPool()),
		server.WithTracer(telemetry.tracer),
		server.WithMetrics(telemetry.httpMetrics),
		server.WithRequestLogger(telemetry.requestLogger),
		server.WithLocalizer(localizer),
	)
}

func runServerLifecycle(ctx context.Context, s serverRunner, port int, shutdownTimeout time.Duration) {
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.Listen(port)
	}()

	select {
	case err := <-serveErr:
		if shouldLogServerError(err) {
			log.Printf("server stopped with error: %v", err)
		}
	case <-ctx.Done():
		fmt.Println("🩰 Graceful shutdown requested...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		err := s.ShutdownWithContext(shutdownCtx)
		if shouldLogServerError(err) {
			log.Printf("graceful shutdown error: %v", err)
		} else {
			fmt.Println("✅ Server shutdown completed")
		}

		select {
		case err := <-serveErr:
			if shouldLogServerError(err) {
				log.Printf("server exit error after shutdown: %v", err)
			}
		case <-time.After(shutdownTimeout):
			log.Printf("listen exit timed out after %s", shutdownTimeout)
		}
	}
}

func isExpectedServerCloseError(err error) bool {
	if err == nil {
		return true
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "server closed") ||
		strings.Contains(message, "listener closed") ||
		strings.Contains(message, "use of closed network connection")
}

func shouldLogServerError(err error) bool {
	return err != nil && !isExpectedServerCloseError(err)
}
