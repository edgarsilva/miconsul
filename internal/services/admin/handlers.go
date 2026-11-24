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
	userCount, err := s.countUsers(c.Context())
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

// HandleRuntimePage renders the runtime stats page.
// GET: /admin/runtime
func (s *service) HandleRuntimePage(c fiber.Ctx) error {
	env := s.AppEnv()

	vc := view.NewCtx(c)
	return view.Render(c, view.AdminRuntimePage(vc, s.RuntimeStats(), env.AppVersion, string(env.Environment)))
}

// HandleRuntimeStats renders the runtime stats fragment the runtime page polls.
// GET: /admin/runtime/stats
func (s *service) HandleRuntimeStats(c fiber.Ctx) error {
	return view.Render(c, view.AdminRuntimeStats(s.RuntimeStats()))
}

// HandleHealthPage renders the probes and dependency checks page.
// GET: /admin/health
func (s *service) HandleHealthPage(c fiber.Ctx) error {
	vc := view.NewCtx(c)
	return view.Render(c, view.AdminHealthPage(vc, s.HealthReport(c)))
}

// HandleHealthChecks renders the checks fragment the health page polls, always
// with 200 so htmx swaps it when the app is degraded.
// GET: /admin/health/checks
func (s *service) HandleHealthChecks(c fiber.Ctx) error {
	return view.Render(c, view.AdminHealthChecks(s.HealthReport(c)))
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
