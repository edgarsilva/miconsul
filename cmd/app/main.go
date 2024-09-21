package main

import (
	"context"
	"fmt"
	"log"
	"miconsul/internal/database"
	"miconsul/internal/lib/bgjob"
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

	ctx := context.Background()
	tp, shutdownTP := otel.NewUptraceTracerProvider(ctx)

	defer func() {
		fmt.Println("Tracer is shutting down...")
		shutdownTP()
	}()

	bgj, shutdownBGJ := bgjob.New()
	defer func() {
		fmt.Println("BGJ is shutting down...")
		shutdownBGJ()
	}()

	wp, err := workerpool.New(10)
	if err != nil {
		log.Panic("Failed to start workerpool", err.Error())
	}

	localizer := localize.New("en-US", "es-MX")
	db := database.New(os.Getenv("DB_PATH"))
	s := server.New(
		server.WithDatabase(db),
		server.WithBGJob(bgj),
		server.WithWorkerPool(wp),
		server.WithTracerProvider(tp),
		server.WithLocalizer(localizer),
	)
	routes.RegisterServices(s)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	serverShutdown := make(chan struct{})

	go func() {
		<-c
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
