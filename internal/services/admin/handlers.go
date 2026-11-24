package admin

import (
	"fmt"

	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

// HandleIndexPage renders the admin landing page.
// GET: /admin
func (s *service) HandleIndexPage(c fiber.Ctx) error {
	userCount, err := s.CountUsers(c.Context())
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}

	vc := view.NewCtx(c)
	return view.Render(c, view.AdminIndexPage(vc, s.RuntimeStats(), userCount, s.AppEnv().JobsUIEnabled))
}

// HandleHealthStatus renders the overall health badge, always with 200 so
// htmx swaps it when the app is degraded.
// GET: /admin/health/status
func (s *service) HandleHealthStatus(c fiber.Ctx) error {
	return view.Render(c, view.AdminHealthStatus(s.HealthReport(c)))
}

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
// GET: /api/admin/runtime
func (s *service) HandleAPIRuntime(c fiber.Ctx) error {
	return c.JSON(newRuntimeOutput(s.RuntimeStats()))
}

// HandleAPIHealth returns the probe and dependency health as JSON, with 503
// when any of them fails.
// GET: /api/admin/health
func (s *service) HandleAPIHealth(c fiber.Ctx) error {
	report := s.HealthReport(c)

	status := fiber.StatusOK
	if !report.OK() {
		status = fiber.StatusServiceUnavailable
	}

	return c.Status(status).JSON(newHealthOutput(report))
}
