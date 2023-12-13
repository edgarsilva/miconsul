package main

import (
	"fiber-blueprint/internal/app"
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	app := app.New()

	app.RegisterFiberRoutes()
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	err := app.Listen(port)

	if err != nil {
		panic("cannot start server")
	}
}
