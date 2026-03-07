package server

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	healthProbeTimeout = 750 * time.Millisecond
	startupGracePeriod = 2 * time.Second
)

func livenessProbe(srv *Server) func(c fiber.Ctx) bool {
	return func(c fiber.Ctx) bool {
		return srv != nil && srv.App != nil
	}
}

func readinessProbe(srv *Server) func(c fiber.Ctx) bool {
	return func(c fiber.Ctx) bool {
		if srv == nil || srv.DB == nil {
			return false
		}

		ctx, cancel := context.WithTimeout(c.Context(), healthProbeTimeout)
		defer cancel()

		var probeResult int
		err := srv.DB.WithContext(ctx).Raw("SELECT 1").Scan(&probeResult).Error
		if err != nil {
			return false
		}

		return probeResult == 1
	}
}

func startupProbe(srv *Server) func(c fiber.Ctx) bool {
	readyCheck := readinessProbe(srv)

	return func(c fiber.Ctx) bool {
		if srv == nil {
			return false
		}

		if time.Since(srv.StartedAt) < startupGracePeriod {
			return false
		}

		if srv.ReadyAt.IsZero() || srv.BootstrapDuration <= 0 {
			return false
		}

		return readyCheck(c)
	}
}
