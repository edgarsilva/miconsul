package admin

import (
	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

// HandleComponentsPage renders an example of every component, as a style
// reference for admins.
// GET: /admin/components
func (s *service) HandleComponentsPage(c fiber.Ctx) error {
	vc := view.NewCtx(c)
	return view.Render(c, view.AdminComponentsPage(vc))
}

// HandleAdminModelsPage renders a list of available model types.
// GET: /admin/models
func (s *service) HandleAdminModelsPage(c fiber.Ctx) error {
	models, err := s.modelNameProvider.ListModelNames()
	if err != nil {
		log.Error("failed to list model names:", err)
		models = []string{}
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.AdminModelsPage(vc, models))
}

// HandleAPIRuntime returns the process runtime stats as JSON.
// GET: /debug/runtime
func (s *service) HandleAPIRuntime(c fiber.Ctx) error {
	return c.JSON(newRuntimeOutput(s.RuntimeStats()))
}

// HandleAPIHealth returns the probe and dependency health as JSON, with 503
// when any of them fails.
// GET: /debug/health
func (s *service) HandleAPIHealth(c fiber.Ctx) error {
	report := s.HealthReport(c)

	status := fiber.StatusOK
	if !report.OK() {
		status = fiber.StatusServiceUnavailable
	}

	return c.Status(status).JSON(newHealthOutput(report))
}
