package admin

import "github.com/gofiber/fiber/v3"

// HandleAPIRuntime returns the process runtime stats as JSON.
// GET: /api/admin/runtime
func (s *service) HandleAPIRuntime(c fiber.Ctx) error {
	return c.JSON(newRuntimeOutput(s.RuntimeStats()))
}

// HandleAPIRuntimeHistory returns the recorded runtime samples as JSON, oldest
// first, with the seconds between samples.
// GET: /api/admin/runtime/history
func (s *service) HandleAPIRuntimeHistory(c fiber.Ctx) error {
	return c.JSON(newRuntimeHistoryOutput(s.RuntimeSampleInterval(), s.RuntimeHistory()))
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
