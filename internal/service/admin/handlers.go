package admin

import (
	"fmt"
	"miconsul/internal/view"
	"os"

	"github.com/gofiber/fiber/v3"
)

// HandleAdminModelsPage renders a list of available model types.
// GET: /admin/models
func (s *service) HandleAdminModelsPage(c fiber.Ctx) error {
	cu, _ := s.CurrentUser(c)
	fmt.Println("cu", cu)

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
