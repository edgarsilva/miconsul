package main

import (
	"os"

	"github.com/edgarsilva/go-scaffold/internal/database"
	"github.com/edgarsilva/go-scaffold/internal/routes"
	"github.com/edgarsilva/go-scaffold/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")

	db := database.New(os.Getenv("DB_PATH"))
	s := server.New(db)

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
