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
	"miconsul/internal/lib/jobs"
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

type bootstrapResult struct {
	env     *appenv.Env
	server  *server.Server
	cleanup func()
}

func main() {
	fmt.Println(" Starting server...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runtime, err := bootstrapServer(ctx)
	if err != nil {
		runtime.cleanup()
		os.Exit(1)
	}
	defer runtime.cleanup()

	fmt.Println(" Setting up server...")
	runServerLifecycle(ctx, runtime.server, runtime.env.AppPort, runtime.env.AppShutdownTimeout)
}

func bootstrapServer(ctx context.Context) (bootstrapResult, error) {
	result := bootstrapResult{cleanup: func() {}}
	cleanupFns := []func(){}
	pushCleanup := func(fn func()) {
		cleanupFns = append(cleanupFns, fn)
	}
	result.cleanup = func() {
		fmt.Println("🧹 Running cleanup tasks...")
		for i := len(cleanupFns) - 1; i >= 0; i-- {
			cleanupFns[i]()
		}
		fmt.Println("✅ All cleanup tasks completed")
	}

	fmt.Println(" Loading environment variables...")
	env, err := setupEnv()
	if err != nil {
		log.Printf("failed to load environment config: %v", err)
		return result, err
	}
	result.env = env

	fmt.Println(" Starting telemetry...")
	telemetry, err := setupTelemetry(ctx, env)
	if err != nil {
		log.Printf("failed to initialize telemetry: %v", err)
		return result, err
	}
	pushCleanup(func() {
		fmt.Println("󰓾 Tracer provider shutting down...")
		if err := telemetry.shutdownTracer(); err != nil {
			log.Printf("tracer shutdown error: %v", err)
		}
	})
	pushCleanup(func() {
		fmt.Println("󰓾 Meter provider shutting down...")
		if err := telemetry.shutdownMetrics(); err != nil {
			log.Printf("meter provider shutdown error: %v", err)
		}
	})
	pushCleanup(func() {
		fmt.Println("󰓾 Logger provider shutting down...")
		if err := telemetry.shutdownLogs(); err != nil {
			log.Printf("logger provider shutdown error: %v", err)
		}
	})

	fmt.Println(" Connecting to database...")
	db, err := setupDB(env, telemetry.dbLogger)
	if err != nil {
		log.Printf("failed to initialize database: %v", err)
		return result, err
	}
	pushCleanup(func() {
		fmt.Println("🔌 Closing database connections...")
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	})

	fmt.Println(" Starting cronjobs...")
	cj, shutdownCronjob, err := setupCronjob()
	if err != nil {
		log.Printf("failed to initialize cronjobs: %v", err)
		return result, err
	}
	pushCleanup(func() {
		fmt.Println("🕑 Cronjobs shutting down...")
		if err := shutdownCronjob(); err != nil {
			log.Printf("cronjob shutdown error: %v", err)
		}
	})

	fmt.Println(" Starting workpool...")
	wp, shutdownWorkPool := setupWorkpool(10)
	pushCleanup(func() {
		fmt.Println("🐜 Ants workpool shutting down...")
		shutdownWorkPool()
	})

	fmt.Println(" Starting jobs runtime...")
	jobsRuntime, err := setupJobs(env)
	if err != nil {
		log.Printf("failed to initialize jobs runtime: %v", err)
		return result, err
	}
	pushCleanup(func() {
		fmt.Println("📬 Jobs runtime shutting down...")
		if err := jobsRuntime.Shutdown(); err != nil {
			log.Printf("jobs runtime shutdown error: %v", err)
		}
	})

	fmt.Println(" Starting localizer...")
	s := server.New(
		server.WithEnv(env),
		server.WithDatabase(db),
		server.WithCronJob(cj),
		server.WithWorkPool(wp.AntsPool()),
		server.WithJobs(jobsRuntime),
		server.WithTracer(telemetry.tracer),
		server.WithMetrics(telemetry.httpMetrics),
		server.WithRequestLogger(telemetry.requestLogger),
		server.WithLocalizer(localize.New("en-US", "es-MX")),
	)
	result.server = s

	fmt.Println(" Registering routes...")
	if err := setupRoutes(s); err != nil {
		log.Printf("failed to register routes: %v", err)
		return result, err
	}

	return result, nil
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

func setupCronjob() (*cronjob.Sched, func() error, error) {
	return cronjob.New()
}

func setupWorkpool(size int) (*workpool.Pool, func()) {
	return workpool.New(size)
}

func setupJobs(env *appenv.Env) (*jobs.Runtime, error) {
	return jobs.New(env)
}

func setupRoutes(s *server.Server) error {
	return routes.RegisterServices(s)
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
