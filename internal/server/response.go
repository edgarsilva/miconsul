package server

import (
	"os"

	"github.com/gofiber/fiber/v3"
)

// Redirect performs a 303 See Other redirect.
func (s *Server) Redirect(c fiber.Ctx, path string) error {
	return c.Redirect().Status(fiber.StatusSeeOther).To(path)
}

func (s *Server) SendFile(c fiber.Ctx, path string) error {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.Send(fileBytes)
}
