package admin

import (
	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

// HandleAdminModelsPage renders a list of available model types.
// GET: /admin/models
func (s *service) HandleAdminModelsPage(c fiber.Ctx) error {
	cu := s.CurrentUser(c)

	models, err := s.modelNameProvider.ListModelNames()
	if err != nil {
		log.Error("failed to list model names:", err)
		models = []string{}
	}

	theme := s.SessionUITheme(c)
	vc, _ := view.NewCtx(c, view.WithTheme(theme), view.WithCurrentUser(cu))
	return view.Render(c, view.AdminModelsPage(vc, models))
}
