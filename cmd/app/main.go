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
	"miconsul/internal/routes"
	"miconsul/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	fmt.Println(" Starting server...")
	defer func() {
		fmt.Println("✅ All cleanup tasks completed")
	}()

	fmt.Println(" Loading environment variables...")
	env := appenv.New()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println(" Connecting to database...")
	db := database.New(env.DBPath)
	defer func() {
		fmt.Println("🔌 Closing database connections...")
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	fmt.Println(" Applying database migrations...")
	if err := database.ApplyMigrations(db); err != nil {
		log.Printf("failed to apply database migrations: %v", err)
		return
	}

	// ctx := context.Background()
	// tp, shutdownTracer := otel.NewUptraceTracerProvider(ctx, env)
	// defer func() {
	// 	fmt.Println("󰓾 Tracer provider shutting down...")
	// 	shutdownTracer()
	// }()

	fmt.Println(" Starting cronjobs...")
	cj, shutdownCronjob := cronjob.New()
	defer func() {
		fmt.Println("🕑 Cronjobs shutting down...")
		shutdownCronjob()
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
		server.WithAppEnv(env),
		server.WithDatabase(db),
		server.WithCronJob(cj),
		server.WithWorkPool(wp),
		// server.WithTracerProvider(tp),
		server.WithLocalizer(localizer),
	)

	fmt.Println(" Registering routes...")
	routes.RegisterServices(s)

	fmt.Println(" Setting up server...")

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.Listen(env.AppPort)
	}()

	shutdownTimeout := env.AppShutdownTimeout
	select {
	case err := <-serveErr:
		if err != nil && !isExpectedServerCloseError(err) {
			log.Printf("server stopped with error: %v", err)
		}
	case <-ctx.Done():
		fmt.Println("🩰 Graceful shutdown requested...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		err := s.ShutdownWithContext(shutdownCtx)
		if err != nil && !errors.Is(err, context.Canceled) && !isExpectedServerCloseError(err) {
			log.Printf("graceful shutdown error: %v", err)
		} else {
			fmt.Println("✅ Server shutdown completed")
		}

		select {
		case err := <-serveErr:
			if err != nil && !isExpectedServerCloseError(err) {
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
