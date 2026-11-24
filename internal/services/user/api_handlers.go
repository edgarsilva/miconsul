package user

import "github.com/gofiber/fiber/v3"

// HandleAPIUsers returns users as JSON for admin API clients.
// GET: /api/users
func (s *service) HandleAPIUsers(c fiber.Ctx) error {
	users, err := s.FindRecentUsers(c.Context(), 10)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	out := newAPIUsersOutput(users)
	return c.Status(fiber.StatusOK).JSON(out)
}
