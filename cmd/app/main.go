package main

import (
	"fiber-blueprint/internal/counter"
	"fiber-blueprint/internal/home"
	"fiber-blueprint/internal/server"
	"fiber-blueprint/internal/todos"
	"os"
)

func main() {
	app := server.New()

	homeRouter := home.NewRouter()
	app.RegisterRouter(&homeRouter)

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
