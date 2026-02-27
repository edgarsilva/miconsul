package admin

import (
	"fmt"
	"miconsul/internal/model"
	"miconsul/internal/view"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"syreclabs.com/go/faker"
)

// handleUsers listr all users in a table *the index*
//
// GET: /todos
func (s *service) HandleAdminModelsPage(c fiber.Ctx) error {
	cu, err := s.CurrentUser(c)
	fmt.Println("cu", cu)
	if err != nil || cu.Role != model.UserRoleAdmin {
		return c.Redirect().To("/login")
	}

	dir, err := os.ReadDir("internal/model")
	if err != nil {
		fmt.Println("FS ERROR ->", err)
	}

	models := make([]string, 0, len(dir))
	fmt.Println("Listing subdir/parent")
	for _, entry := range dir {
		fmt.Println(" ", entry.Name(), entry.IsDir())

		mn, err := FindModelName(entry)
		if err != nil {
			fmt.Println(err)
			continue
		}
		models = append(models, mn)
	}

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.AdminModelsPage(vc, models))
}

// handleApiUsers returns all users as JSON
//
// GET: /api/todos - Get all todos
func (s *service) HandleGetUsers(c fiber.Ctx) error {
	var users []model.User

	s.DB.
		Model(&model.User{}).
		Limit(10).
		Find(&users)

	res := struct{ Users []model.User }{
		Users: users,
	}

	return c.Status(fiber.StatusOK).JSON(res)
}

func (s *service) HandleMakeUsers(c fiber.Ctx) error {
	n, err := strconv.Atoi(c.Params("n"))
	if err != nil {
		n = 10
	}

	var users []model.User
	for i := 0; i <= n; i++ {
		users = append(users, model.User{
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
func (s *service) HandleAPIUsers(c fiber.Ctx) error {
	var users []model.User

	s.DB.
		Model(&model.User{}).
		Limit(10).
		Find(&users)

	res := struct{ Users []model.User }{
		Users: users,
	}
	return c.Status(fiber.StatusOK).JSON(res)
}
