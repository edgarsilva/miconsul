package dashboard

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"time"

	"miconsul/internal/lib/libtime"
	"miconsul/internal/models"
	"miconsul/internal/server"
	view "miconsul/internal/views"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type service struct {
	*server.Server
}

func NewService(s *server.Server) (service, error) {
	if s == nil {
		return service{}, errors.New("dashboard service requires a non-nil server")
	}

	return service{
		Server: s,
	}, nil
}

func (s service) FavoriteClinic(c fiber.Ctx, userID uint) (models.Clinic, error) {
	clinics, err := gorm.G[models.Clinic](s.DB.GormDB()).
		Where("user_id = ?", userID).
		Order("favorite DESC, created_at").
		Limit(1).
		Find(c.Context())
	if err != nil {
		return models.Clinic{}, err
	}

	if len(clinics) == 0 {
		return models.Clinic{}, nil
	}

	return clinics[0], nil
}

func (s service) FindFeedEventsByUserID(ctx context.Context, userID uint, limit int) ([]models.FeedEvent, error) {
	var events []models.FeedEvent
	since := time.Now().AddDate(0, 0, -3)
	err := s.DB.WithContext(ctx).
		Model(&models.FeedEvent{}).
		Joins("JOIN appointments ON appointments.uid = feed_events.feed_eventable_id AND feed_events.feed_eventable_type = ?", "appointments").
		Where("appointments.user_id = ?", userID).
		Where("feed_events.ocurred_at > ?", since).
		Order("feed_events.ocurred_at DESC").
		Limit(limit).
		Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (s service) CalcDashboardStats(ctx context.Context, cu models.User) view.DashboardStats {
	ctx, span := s.Trace(ctx, "dashboard/services.CalcDashboardStats")
	defer span.End()

	cacheKey := cu.UID + ".dashboard.stats"
	if stats, ok := s.ReadStatsCache(cacheKey); ok {
		return stats
	}

	patStats := s.CalcMonthlyStats(ctx, cu, &models.Patient{UserID: cu.ID})
	apptStats := s.CalcMonthlyStats(ctx, cu, &models.Appointment{UserID: cu.ID})
	stats := view.DashboardStats{
		Patients:     patStats,
		Appointments: apptStats,
	}

	_ = s.WriteStatsCache(cacheKey, stats)

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

func (s service) CalcMonthlyStats(ctx context.Context, cu models.User, imodel interface{}) view.DashboardStat {
	localCtx, span := s.Trace(ctx, "dashboard/services:CalcMonthlyStats")
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
