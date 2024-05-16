package clinics

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
func (s *service) HandleClinicsPage(c *fiber.Ctx) error {
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

// HandleCreateClinic inserts a new clinic record for the given user
//
// POST: /clinics
func (s *service) HandleCreateClinic(c *fiber.Ctx) error {
	// cu, err := s.CurrentUser(c)
	// if err != nil {
	// 	c.Redirect("/login")
	// }
	cu := model.User{}
	s.DB.Model(&model.User{}).First(&cu)

	clinic := model.Clinic{
		ExtID:  "ext-01",
		Name:   "ollimarket",
		Email:  "ollimarket@gmail.com",
		Phone:  "312-110-12345",
		UserID: cu.ID,
		Address: model.Address{
			Line1:   "Ave. De La Paz 123",
			Line2:   "",
			City:    "Colima",
			State:   "Colima",
			Country: "Mexico",
			Zip:     "28500",
		},
	}

	res := s.DB.Create(&clinic)
	if err := res.Error; err != nil {
		return c.SendString("Errors found" + err.Error())
	}

	patients := []model.Patient{}
	s.DB.Model(&model.Patient{}).Find(&patients)

	return c.JSON(clinic)
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
