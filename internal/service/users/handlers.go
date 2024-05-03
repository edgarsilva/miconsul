package users

import (
	"strconv"

	"github.com/edgarsilva/go-scaffold/internal/database"

	"github.com/gofiber/fiber/v2"
	"syreclabs.com/go/faker"
)

// handleUsers listr all users in a table *the index*
//
// GET: /todos
func (s *service) HandleUsersPage(c *fiber.Ctx) error {
	// cu, err := s.CurrentUser(c)
	// if err != nil {
	// 	return c.Redirect("/login")
	// }

	theme := s.SessionGet(c, "theme", "light")
	// layoutProps, err := view.NewLayoutProps(view.WithCurrentUser(cu), view.WithTheme(theme))
	// if err != nil {
	// 	return c.Redirect("/login")
	// }

	if theme == "light" {
		s.SessionSet(c, "theme", "light")
	} else {
		s.SessionSet(c, "theme", "dark")
	}

	return c.SendString("Users Index Page")
}

// handleApiUsers returns all users as JSON
//
// GET: /api/todos - Get all todos
func (s *service) HandleGetUsers(c *fiber.Ctx) error {
	var users []database.User

	s.DB.
		Model(&database.User{}).
		Limit(10).
		Find(&users)

	res := struct{ Users []database.User }{
		Users: users,
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (s *service) HandleMakeUsers(c *fiber.Ctx) error {
	n, err := strconv.Atoi(c.Params("n"))
	if err != nil {
		n = 10
	}

	var users []database.User
	for i := 0; i <= n; i++ {
		users = append(users, database.User{
			Name:  faker.Name().Name(),
			Email: faker.Internet().Email(),
		})
	}

	res := s.DB.Create(&users)
	if err := res.Error; err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).SendString("Unprocessable entity")
	}

	return c.SendStatus(fiber.StatusOK)
}

// API
//
// handleAPIUsers returns all users as JSON
// GET: /api/todos - Get all todos
func (s *service) HandleAPIUsers(c *fiber.Ctx) error {
	var users []database.User

	s.DB.
		Model(&database.User{}).
		Limit(10).
		Find(&users)

	res := struct{ Users []database.User }{
		Users: users,
	}
	return c.Status(fiber.StatusOK).JSON(res)
}
