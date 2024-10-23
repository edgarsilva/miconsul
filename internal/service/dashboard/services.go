package dashboard

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"miconsul/internal/lib/libtime"
	"miconsul/internal/model"
	"miconsul/internal/server"
	"miconsul/internal/view"
	"time"

	"github.com/gofiber/fiber/v2"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) service {
	return service{
		Server: s,
	}
}

func (s service) CalcDashboardStats(c *fiber.Ctx, cu model.User) view.DashboardStats {
	ctx, span := s.Tracer.Start(c.UserContext(), "dashboard/services:CalcDashboardStats")
	defer span.End()

	cacheKey := s.TagWithSessionID(c, "dashboard_monthlystats")
	if stats, ok := s.ReadStatsCache(cacheKey); ok {
		return stats
	}

	patStats := s.CalcMonthlyStats(ctx, cu, &model.Patient{UserID: cu.ID})
	apptStats := s.CalcMonthlyStats(ctx, cu, &model.Appointment{UserID: cu.ID})
	stats := view.DashboardStats{
		Patients:     patStats,
		Appointments: apptStats,
	}

	s.WriteStatsCache(cacheKey, stats)

	return stats
}

func (s service) WriteStatsCache(cachekey string, stats view.DashboardStats) error {
	statsBytes, err := Serialize(stats)
	if err != nil {
		return err
	}

	err = s.CacheWrite(cachekey, &statsBytes, 15*time.Minute)
	if err != nil {
		return err
	}

	return nil
}

func (s service) ReadStatsCache(cachekey string) (stats view.DashboardStats, ok bool) {
	cacheBytes := make([]byte, 32)
	err := s.CacheRead(cachekey, &cacheBytes)
	if err != nil {
		return stats, false
	}

	stats, err = Deserialize(cacheBytes)
	if err != nil {
		return stats, false
	}

	return stats, true
}

func (s service) CalcMonthlyStats(ctx context.Context, cu model.User, imodel interface{}) view.DashboardStat {
	localCtx, span := s.Tracer.Start(ctx, "dashboard/services:CalcMonthlyStats")
	defer span.End()

	var cnt int64
	s.DB.WithContext(localCtx).Model(imodel).Where(imodel).Count(&cnt)

	var lastMonth int64
	s.DB.WithContext(localCtx).Model(imodel).Where("user_id = ? AND created_at <= ?", cu.ID, libtime.BoM(time.Now())).Count(&lastMonth)

	diff := cnt - lastMonth

	return view.DashboardStat{
		Total: int(cnt),
		Diff:  int(diff),
	}
}

func Serialize(stats view.DashboardStats) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(stats)
	if err != nil {
		return []byte{}, err
	}

	fmt.Println("Serialized Gob:", buffer.Bytes())
	return buffer.Bytes(), nil
}

func Deserialize(statsB []byte) (view.DashboardStats, error) {
	buffer := bytes.NewBuffer(statsB)
	decoder := gob.NewDecoder(buffer)

	var stats view.DashboardStats
	err := decoder.Decode(&stats)
	if err != nil {
		fmt.Println("Error deserializing:", err)
		return view.DashboardStats{}, err
	}

	return stats, err
}
