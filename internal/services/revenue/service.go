package revenue

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"miconsul/internal/lib/libtime"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"gorm.io/gorm"
)

// reportMonths is how many months the revenue page covers, current included.
const reportMonths = 12

// averageMonths is how many complete months before the current one the
// monthly average covers.
const averageMonths = 12

type service struct {
	*server.Server
}

func NewService(s *server.Server) (*service, error) {
	if s == nil {
		return nil, errors.New("revenue service requires a non-nil server")
	}

	return &service{Server: s}, nil
}

// Report returns the user's revenue for the last 12 months up to now, newest
// first, the selected month's revenue by clinic, and the monthly average of
// the 12 complete months before the current one. Months follow the clinic
// timezone; a selected month outside the report means the current month.
func (s *service) Report(ctx context.Context, userID uint, now time.Time, selected time.Time) (models.RevenueReport, error) {
	ctx, span := s.Trace(ctx, "revenue/service:Report")
	defer span.End()

	monthStarts := lastMonthStarts(now, max(reportMonths, averageMonths+1))
	if !slices.ContainsFunc(monthStarts[:reportMonths], selected.Equal) {
		selected = monthStarts[0]
	}

	done, err := s.doneAppointments(ctx, userID, monthStarts[len(monthStarts)-1], now)
	if err != nil {
		return models.RevenueReport{}, fmt.Errorf("load done appointments: %w", err)
	}

	months := make([]models.RevenueMonth, len(monthStarts))
	for i, start := range monthStarts {
		months[i].Start = start
	}

	byClinic := map[uint]*models.RevenueClinic{}
	clinicIDs := []uint{}
	for _, a := range done {
		start := libtime.BoM(libtime.InTimezone(a.BookedAt, models.DefaultTimezone))
		i := slices.IndexFunc(monthStarts, start.Equal)
		if i < 0 {
			continue
		}
		months[i].Total += a.Price
		months[i].Appointments++

		if !start.Equal(selected) {
			continue
		}
		clinic, ok := byClinic[a.ClinicID]
		if !ok {
			clinic = &models.RevenueClinic{}
			byClinic[a.ClinicID] = clinic
			clinicIDs = append(clinicIDs, a.ClinicID)
		}
		clinic.Total += a.Price
		clinic.Appointments++
	}

	if err := s.nameClinics(ctx, userID, byClinic, clinicIDs); err != nil {
		return models.RevenueReport{}, fmt.Errorf("load clinic names: %w", err)
	}

	clinics := make([]models.RevenueClinic, 0, len(byClinic))
	for _, id := range clinicIDs {
		clinics = append(clinics, *byClinic[id])
	}
	slices.SortStableFunc(clinics, func(a, b models.RevenueClinic) int {
		return cmp.Compare(b.Total, a.Total)
	})

	completed := 0
	for _, m := range months[1 : averageMonths+1] {
		completed += m.Total
	}

	report := models.RevenueReport{
		Months:         months[:reportMonths],
		Selected:       months[slices.IndexFunc(monthStarts, selected.Equal)],
		Clinics:        clinics,
		MonthlyAverage: completed / averageMonths,
	}

	return report, nil
}

// ParseMonth parses a YYYY-MM month as its first day in the clinic timezone;
// ok is false when value is not a month.
func ParseMonth(value string) (start time.Time, ok bool) {
	loc, err := time.LoadLocation(models.DefaultTimezone)
	if err != nil {
		return time.Time{}, false
	}

	start, err = time.ParseInLocation("2006-01", value, loc)
	if err != nil {
		return time.Time{}, false
	}

	return start, true
}

// doneAppointments returns the price, clinic and booking time of the user's
// done appointments booked in [start, end).
func (s *service) doneAppointments(ctx context.Context, userID uint, start, end time.Time) ([]models.Appointment, error) {
	return gorm.G[models.Appointment](s.DB.GormDB()).
		Select("booked_at", "price", "clinic_id").
		Where("user_id = ? AND status = ?", userID, models.AppointmentDone).
		Where("booked_at >= ? AND booked_at < ?", start.UTC(), end.UTC()).
		Find(ctx)
}

// nameClinics fills in the names of the user's clinics, deleted ones included.
func (s *service) nameClinics(ctx context.Context, userID uint, byClinic map[uint]*models.RevenueClinic, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	var clinics []models.Clinic
	err := s.DB.WithContext(ctx).Unscoped().
		Select("id", "name", "deleted_at").
		Where("user_id = ? AND id IN ?", userID, ids).
		Find(&clinics).Error
	if err != nil {
		return err
	}

	for _, clinic := range clinics {
		byClinic[clinic.ID].Name = clinic.Name
		byClinic[clinic.ID].Deleted = clinic.DeletedAt.Valid
	}

	return nil
}

// lastMonthStarts returns the first day, in the clinic timezone, of the month
// that contains now and of the n-1 months before it, newest first.
func lastMonthStarts(now time.Time, n int) []time.Time {
	current, _ := libtime.TimeframeBounds(now, models.DefaultTimezone, libtime.TimeframeMonth)

	starts := make([]time.Time, n)
	for i := range n {
		starts[i] = current.AddDate(0, -i, 0)
	}

	return starts
}
