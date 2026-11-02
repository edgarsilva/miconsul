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
