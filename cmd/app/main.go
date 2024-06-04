package main

import (
	"log"
	"os"

	"github.com/edgarsilva/go-scaffold/internal/backgroundjob"
	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/localize"
	"github.com/edgarsilva/go-scaffold/internal/routes"
	"github.com/edgarsilva/go-scaffold/internal/server"
	"github.com/edgarsilva/go-scaffold/internal/workerpool"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")

	locales := localize.New("en-US", "es-MX")
	db := database.New(os.Getenv("DB_PATH"))
	wp, err := workerpool.New(10)
	if err != nil {
		log.Panic("Failed to start workerpool", err.Error())
	}

	bgj, shutdown := backgroundjob.New()
	defer shutdown()

	s := server.New(db, locales, wp, bgj)

	appRoutes := routes.New()
	s.RegisterRoutes(&appRoutes)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	err = s.Listen(port) // <-- this is a blocking call
	if err != nil {
		log.Panic("Failed to start server:", err.Error())
	}
}
