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

func (s service) CalcDashboardStats(cu model.User) view.DashboardStats {
	patStats := s.CalcMonthlyStats(cu, &model.Patient{UserID: cu.ID})
	apptStats := s.CalcMonthlyStats(cu, &model.Appointment{UserID: cu.ID})

	return view.DashboardStats{
		Patients:     patStats,
		Appointments: apptStats,
	}
}

func (s service) CalcMonthlyStats(cu model.User, modelIface interface{}) view.DashboardStat {
	var cnt int64
	s.DB.Where(modelIface).Count(&cnt)

	var lastMonth int64
	s.DB.Where("user_id = ? AND created_at <= ?", cu.ID, libtime.BoM(time.Now())).Count(&lastMonth)

	diff := cnt - lastMonth

	return view.DashboardStat{
		Total: int(cnt),
		Diff:  int(diff),
	}
}
