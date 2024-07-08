package dashboard

import (
	"miconsul/internal/lib/libtime"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"miconsul/internal/view"
	"time"
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
	s.DB.Model(modelIface).Where("created_at <= ?", libtime.BoM(time.Now())).Count(&lastMonth)

	var diff int64 = cnt - lastMonth

	return view.DashboardStat{
		Total: int(cnt),
		Diff:  int(diff),
	}
}
