package patients

import (
	"strconv"

	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/view"
	"github.com/gofiber/fiber/v2"
	"syreclabs.com/go/faker"
)

// handlePatientsPage renders the patients page HTML
//
// GET: /patients
func (s *service) HandlePatientsPage(c *fiber.Ctx) error {
	// cu, err := s.CurrentUser(c)
	// if err != nil {
	// 	return c.Redirect("/login")
	// }

	theme := s.SessionUITheme(c)
	layoutProps, err := view.NewLayoutProps(view.WithTheme(theme))
	if err != nil {
		return c.Redirect("/login")
	}

	id := c.Params("id", "")
	patientProfile := model.Patient{}
	patientProfile.ID = id
	if id != "" {
		s.DB.Model(&model.Patient{}).First(&patientProfile)
	}
	patients := []model.Patient{}
	s.DB.Model(&model.Patient{}).Find(&patients)

	return view.Render(c, view.PatientsPage(patients, patientProfile, layoutProps))
}

func (s *service) HandleCreatePatient(c *fiber.Ctx) error {
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

func (s *service) HandleMockPatient(c *fiber.Ctx) error {
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
