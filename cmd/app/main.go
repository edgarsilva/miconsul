package main

import (
	"os"

	"github.com/edgarsilva/go-scaffold/internal/db"
	"github.com/edgarsilva/go-scaffold/internal/routes"
	"github.com/edgarsilva/go-scaffold/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")

	DB := db.New(os.Getenv("DB_PATH"))
	s := server.New(DB)

	appRoutes := routes.New()
	s.RegisterRoutes(&appRoutes)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	err := s.Listen(port) // <-- this is a blocking call
	if err != nil {
		panic("cannot start server")
	}
}
