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
	"github.com/gofiber/fiber/v3/log"
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
		return models.Clinic{}, fmt.Errorf("find favorite clinic: %w", err)
	}

	if len(clinics) == 0 {
		return models.Clinic{}, nil
	}

	return clinics[0], nil
}

// FindAppointmentsBy returns up to 10 of the user's appointments booked
// in timeframe, filtered by statuses when there are any.
func (s service) FindAppointmentsBy(ctx context.Context, userID uint, timeframe libtime.Timeframe, statuses []models.AppointmentStatus) ([]models.Appointment, error) {
	appointments := []models.Appointment{}
	err := s.DB.WithContext(ctx).
		Model(models.Appointment{}).
		Preload("Patient").
		Preload("Clinic").
		Where("user_id = ?", userID).
		Scopes(models.AppointmentBookedIn(timeframe), models.AppointmentWithStatus(statuses...)).
		Limit(10).
		Find(&appointments).
		Error
	if err != nil {
		return appointments, fmt.Errorf("find dashboard appointments: %w", err)
	}

	return appointments, nil
}

// FindFeedEventsByUserID returns the user's newest appointment feed events,
// at most limit, however old they are.
func (s service) FindFeedEventsByUserID(ctx context.Context, userID uint, limit int) ([]models.FeedEvent, error) {
	var events []models.FeedEvent
	err := s.DB.WithContext(ctx).
		Model(&models.FeedEvent{}).
		Joins("JOIN appointments ON appointments.uid = feed_events.feed_eventable_id AND feed_events.feed_eventable_type = ?", "appointments").
		Where("appointments.user_id = ?", userID).
		Order("feed_events.ocurred_at DESC").
		Limit(limit).
		Find(&events).Error
	if err != nil {
		return nil, fmt.Errorf("find feed events: %w", err)
	}

	return events, nil
}

// CalcDashboardStats returns the user's patient and appointment stats, cached
// for 15 minutes. Stats are only cached when every count succeeds.
func (s service) CalcDashboardStats(ctx context.Context, cu models.User) (view.DashboardStats, error) {
	ctx, span := s.Trace(ctx, "dashboard/services.CalcDashboardStats")
	defer span.End()

	cacheKey := cu.UID + ".dashboard.stats"
	if stats, ok := s.ReadStatsCache(cacheKey); ok {
		return stats, nil
	}

	patStats, err := s.CalcMonthlyStats(ctx, cu, &models.Patient{UserID: cu.ID})
	if err != nil {
		return view.DashboardStats{}, fmt.Errorf("count patients: %w", err)
	}

	apptStats, err := s.CalcMonthlyStats(ctx, cu, &models.Appointment{UserID: cu.ID})
	if err != nil {
		return view.DashboardStats{}, fmt.Errorf("count appointments: %w", err)
	}

	revenue, err := s.CalcMonthRevenue(ctx, cu, time.Now())
	if err != nil {
		return view.DashboardStats{}, fmt.Errorf("sum revenue: %w", err)
	}

	stats := view.DashboardStats{
		Patients:     patStats,
		Appointments: apptStats,
		Revenue:      revenue,
	}

	if err := s.WriteStatsCache(cacheKey, stats); err != nil {
		log.Warn("cache dashboard stats:", err)
	}

	return stats, nil
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

func (s service) CalcMonthlyStats(ctx context.Context, cu models.User, imodel any) (view.DashboardStat, error) {
	localCtx, span := s.Trace(ctx, "dashboard/services:CalcMonthlyStats")
	defer span.End()

	var cnt int64
	if err := s.DB.WithContext(localCtx).Model(imodel).Where(imodel).Count(&cnt).Error; err != nil {
		return view.DashboardStat{}, err
	}

	bom := libtime.BoM(libtime.InTimezone(time.Now(), models.DefaultTimezone))

	var lastMonth int64
	if err := s.DB.WithContext(localCtx).Model(imodel).Where("user_id = ? AND created_at < ?", cu.ID, bom.UTC()).Count(&lastMonth).Error; err != nil {
		return view.DashboardStat{}, err
	}

	diff := cnt - lastMonth

	return view.DashboardStat{
		Total: int(cnt),
		Diff:  int(diff),
	}, nil
}

// CalcMonthRevenue sums the price of the user's done appointments booked this
// month up to now, and over the same span of last month, in the clinic
// timezone. Last month's span stops at its end when it is shorter.
func (s service) CalcMonthRevenue(ctx context.Context, cu models.User, now time.Time) (view.DashboardRevenue, error) {
	ctx, span := s.Trace(ctx, "dashboard/services:CalcMonthRevenue")
	defer span.End()

	monthStart, _ := libtime.TimeframeBounds(now, models.DefaultTimezone, libtime.TimeframeMonth)
	elapsed := now.Sub(monthStart)
	prevStart := monthStart.AddDate(0, -1, 0)
	prevEnd := prevStart.Add(elapsed)
	if prevEnd.After(monthStart) {
		prevEnd = monthStart
	}

	total, err := s.sumDoneRevenue(ctx, cu.ID, monthStart, now)
	if err != nil {
		return view.DashboardRevenue{}, fmt.Errorf("sum this month: %w", err)
	}

	previous, err := s.sumDoneRevenue(ctx, cu.ID, prevStart, prevEnd)
	if err != nil {
		return view.DashboardRevenue{}, fmt.Errorf("sum last month: %w", err)
	}

	return view.DashboardRevenue{Total: total, Previous: previous}, nil
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

// sumDoneRevenue sums the price of the user's done appointments booked in
// [start, end), in cents.
func (s service) sumDoneRevenue(ctx context.Context, userID uint, start, end time.Time) (int, error) {
	var total int
	err := gorm.G[models.Appointment](s.DB.GormDB()).
		Select("COALESCE(SUM(price), 0)").
		Where("user_id = ? AND status = ?", userID, models.AppointmentDone).
		Where("booked_at >= ? AND booked_at < ?", start.UTC(), end.UTC()).
		Scan(ctx, &total)
	if err != nil {
		return 0, err
	}

	return total, nil
}
