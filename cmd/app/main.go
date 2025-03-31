package main

import (
	"context"
	"fmt"
	"log"
	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/cache"
	"miconsul/internal/lib/cronjob"
	"miconsul/internal/lib/localize"
	"miconsul/internal/lib/otel"
	"miconsul/internal/lib/workerpool"
	"miconsul/internal/routes"
	"miconsul/internal/server"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	defer func() {
		fmt.Println("✅ All cleanup tasks completed")
	}()

	env := appenv.New()

	ctx := context.Background()
	tp, shutdownTracer := otel.NewUptraceTracerProvider(ctx, env)
	defer func() {
		fmt.Println("󰓾 Tracer provider shutting down...")
		shutdownTracer()
	}()

	cj, shutdownCronjob := cronjob.New()
	defer func() {
		fmt.Println("🕑 Cronjobs shutting down...")
		shutdownCronjob()
	}()

	wp, shutdownWorkerPool := workerpool.New(10)
	defer func() {
		fmt.Println("🐜 Ants workerpool shutting down...")
		shutdownWorkerPool()
	}()

	cache, shutdownCache := cache.New()
	defer func() {
		fmt.Println("🦡 Badger Cache shutting down...")
		shutdownCache()
	}()

	localizer := localize.New("en-US", "es-MX")
	db := database.New(os.Getenv("DB_PATH"))
	s := server.New(
		server.WithAppEnv(env),
		server.WithDatabase(db),
		server.WithCronJob(cj),
		server.WithWorkerPool(wp),
		server.WithTracerProvider(tp),
		server.WithLocalizer(localizer),
		server.WithCache(cache),
	)
	routes.RegisterServices(s)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	osExitSignal := make(chan os.Signal, 1)
	signal.Notify(osExitSignal, os.Interrupt)

	serverShutdown := make(chan struct{})

	go func() {
		<-osExitSignal
		fmt.Println("Gracefully shutting down...")

		if err := s.Shutdown(); err != nil {
			log.Panic("Failed to gracefully shutdowm fiber app server:", err.Error())
		}
		serverShutdown <- struct{}{}
	}()

	if err := s.Listen(port); err != nil {
		log.Panic("Failed to start server on port:", port, err.Error())
	}

	<-serverShutdown
	fmt.Println("🧹 Running cleanup tasks...")
	// Your cleanup tasks go here
}
