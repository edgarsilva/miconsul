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
	"miconsul/internal/lib/telemetry"
	"miconsul/internal/lib/workpool"
	"miconsul/internal/routes"
	"miconsul/internal/server"

	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	exitCode := 0
	defer func() {
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}()

	fmt.Println(" Starting server...")

	_ = godotenv.Load(".env")

	fmt.Println(" Loading environment variables...")
	env := appenv.New()

	defer func() {
		// This should be the last log on defer chain before exiting with code (0|1|etc)
		fmt.Println("✅ All cleanup tasks completed")
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println(" Starting telemetry...")
	var tracer trace.Tracer
	tracer, shutdownTracer, err := telemetry.NewTracer(ctx, env.OTelTracerServer, env)
	if err != nil {
		log.Printf("failed to initialize otel tracer: %v", err)
		exitCode = 1
		return
	}
	defer func() {
		fmt.Println("󰓾 Tracer provider shutting down...")
		if err := shutdownTracer(); err != nil {
			log.Printf("tracer shutdown error: %v", err)
		}
	}()

	fmt.Println(" Connecting to database...")
	db, err := database.New(env)
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

	fmt.Println(" Applying database migrations...")
	if err := database.ApplyMigrations(db, env); err != nil {
		log.Printf("failed to apply database migrations: %v", err)
		exitCode = 1
		return
	}

	fmt.Println(" Starting cronjobs...")
	cj, shutdownCronjob, err := cronjob.New()
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
	wp, shutdownWorkPool := workpool.New(10)
	defer func() {
		fmt.Println("🐜 Ants workpool shutting down...")
		shutdownWorkPool()
	}()

	fmt.Println(" Starting localizer...")
	localizer := localize.New("en-US", "es-MX")
	s := server.New(
		server.WithEnv(env),
		server.WithDatabase(db),
		server.WithCronJob(cj),
		server.WithWorkPool(wp),
		server.WithTracer(tracer),
		server.WithLocalizer(localizer),
	)

	fmt.Println(" Registering routes...")
	if err := routes.RegisterServices(s); err != nil {
		log.Printf("failed to register routes: %v", err)
		exitCode = 1
		return
	}

	fmt.Println(" Setting up server...")

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.Listen(env.AppPort)
	}()

	shutdownTimeout := env.AppShutdownTimeout
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

	fmt.Println("🧹 Running cleanup tasks...")
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
