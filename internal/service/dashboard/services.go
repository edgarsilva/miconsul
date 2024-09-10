package dashboard

import (
	"context"
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

func (s service) CalcDashboardStats(ctx context.Context, cu model.User) view.DashboardStats {
	sctx, span := s.Tracer.Start(ctx, "dashboard/services:CalcDashboardStats")
	defer span.End()

	patStats := s.CalcMonthlyStats(sctx, cu, &model.Patient{UserID: cu.ID})
	apptStats := s.CalcMonthlyStats(sctx, cu, &model.Appointment{UserID: cu.ID})

	return view.DashboardStats{
		Patients:     patStats,
		Appointments: apptStats,
	}
}

func (s service) CalcMonthlyStats(ctx context.Context, cu model.User, imodel interface{}) view.DashboardStat {
	ctxa, span := s.Tracer.Start(ctx, "dashboard/services:CalcMonthlyStats")
	defer span.End()

	var cnt int64
	s.DB.WithContext(ctxa).Model(imodel).Where(imodel).Count(&cnt)

	var lastMonth int64
	s.DB.WithContext(ctxa).Model(imodel).Where("user_id = ? AND created_at <= ?", cu.ID, libtime.BoM(time.Now())).Count(&lastMonth)

	diff := cnt - lastMonth

	return view.DashboardStat{
		Total: int(cnt),
		Diff:  int(diff),
	}
}
