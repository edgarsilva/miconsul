package main

import (
	"fiber-blueprint/internal/counter"
	"fiber-blueprint/internal/database"
	"fiber-blueprint/internal/routes"
	"fiber-blueprint/internal/server"
	"fiber-blueprint/internal/todos"
	"os"
)

func main() {
	db := database.New()
	app := server.New(db)

	appRouter := routes.NewRouter()
	app.RegisterRouter(&appRouter)

	todosRouter := todos.NewRouter()
	app.RegisterRouter(&todosRouter)

	counterRouter := counter.NewRouter()
	app.RegisterRouter(&counterRouter)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	err := app.Listen(port) // <-- this is a blocking call
	if err != nil {
		panic("cannot start server")
	}
}
