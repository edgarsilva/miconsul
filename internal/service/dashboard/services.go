package dashboard

import (
	"time"

	"github.com/edgarsilva/go-scaffold/internal/common"
	"github.com/edgarsilva/go-scaffold/internal/model"
	"github.com/edgarsilva/go-scaffold/internal/server"
	"github.com/edgarsilva/go-scaffold/internal/view"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	return service{
		Server: s,
	}
}

func (s service) CalcDashboardStats() view.DashboardStats {
	patStats := s.CalcMonthlyStats(&model.Patient{})
	apptStats := s.CalcMonthlyStats(&model.Appointment{})

	return view.DashboardStats{
		Patients:     patStats,
		Appointments: apptStats,
	}
}

func (s service) CalcMonthlyStats(modelIface interface{}) view.DashboardStat {
	var cnt int64
	s.DB.Model(modelIface).Count(&cnt)

	var lastMonth int64
	s.DB.Model(modelIface).Where("created_at <= ?", common.BoM(time.Now())).Count(&lastMonth)

	var diff int64 = cnt - lastMonth

	return view.DashboardStat{
		Total: int(cnt),
		Diff:  int(diff),
	}
}
