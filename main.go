package main

import (
	"log"
	"miconsul/internal/backgroundjob"
	"miconsul/internal/database"
	"miconsul/internal/localize"
	"miconsul/internal/routes"
	"miconsul/internal/server"
	"miconsul/internal/workerpool"
	"os"

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
	routes.RegisterServices(s)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	err = s.Listen(port) // <-- this is a blocking call
	if err != nil {
		log.Panic("Failed to start server:", err.Error())
	}
}
