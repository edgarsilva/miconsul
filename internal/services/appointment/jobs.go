package appointment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"miconsul/internal/jobs"
	"miconsul/internal/models"

	"gorm.io/gorm"
)

func (s *service) bootstrapJobs() error {
	if s == nil {
		return nil
	}

	if err := s.registerReminderSweepHandler(); err != nil {
		if errors.Is(err, jobs.ErrRuntimeUnavailable) {
			return nil
		}
		return err
	}

	if err := s.registerBookedAlertHandler(); err != nil {
		if errors.Is(err, jobs.ErrRuntimeUnavailable) {
			return nil
		}
		return err
	}

	if err := s.registerReminderHandler(); err != nil {
		if errors.Is(err, jobs.ErrRuntimeUnavailable) {
			return nil
		}
		return err
	}

	if _, err := s.RegisterScheduledTask(ReminderSweepSchedule, TaskReminderSweep, TaskReminderSweepPayload{}); err != nil {
		if errors.Is(err, jobs.ErrRuntimeUnavailable) {
			return nil
		}
		return err
	}

	return nil
}

func (s *service) registerReminderSweepHandler() error {
	return s.RegisterTaskHandler(TaskReminderSweep, s.handleReminderSweepTask)
}

func (s *service) registerBookedAlertHandler() error {
	return s.RegisterTaskHandler(TaskBookedAlert, s.handleBookedAlertTask)
}

func (s *service) registerReminderHandler() error {
	return s.RegisterTaskHandler(TaskReminder, s.handleReminderTask)
}

func (s *service) handleReminderSweepTask(ctx context.Context, _ jobs.Task) error {
	st := time.Now()
	year, month, day := st.Date()
	et := time.Date(year, month, day, st.Hour(), st.Minute(), 0, 0, st.Location()).Add(2 * time.Hour)

	appointments, err := gorm.G[models.Appointment](s.DB.GormDB()).
		Where("booked_at > ?", st).
		Where("booked_at <= ?", et).
		Where("NOT EXISTS (SELECT 1 FROM notifications WHERE notifications.alertable_id = appointments.uid AND notifications.alertable_type = ? AND notifications.name = ? AND notifications.medium = ? AND notifications.status IN ?)", "appointments", "appointment_reminder", models.NotificationMediumEmail, []models.NotificationStatus{models.NotificationSent, models.NotificationSuccess}).
		Find(ctx)
	if err != nil {
		return fmt.Errorf("load appointments for reminder sweep: %w", err)
	}

	for _, appointment := range appointments {
		if _, err := s.EnqueueTask(ctx, TaskReminder, TaskAppointmentPayload{AppointmentID: appointment.UID}); err != nil {
			log.Printf("appointment jobs: enqueue reminder failed for %s: %v", appointment.UID, err)
		}
	}

	return nil
}

func (s *service) handleReminderTask(ctx context.Context, task jobs.Task) error {
	payload := TaskAppointmentPayload{}
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return fmt.Errorf("decode reminder payload: %w", err)
	}
	if payload.AppointmentID == "" {
		return errors.New("appointment_id is required")
	}

	appointment, err := gorm.G[models.Appointment](s.DB.GormDB()).
		Preload("Patient", nil).
		Preload("Clinic", nil).
		Where("uid = ?", payload.AppointmentID).
		First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load appointment for reminder: %w", err)
	}

	sent, err := s.notificationSent(ctx, appointment.UID, "appointment_reminder", models.NotificationMediumEmail)
	if err != nil {
		return fmt.Errorf("check reminder notification status: %w", err)
	}
	if sent {
		return nil
	}

	s.sendReminderNow(ctx, appointment)
	return nil
}

func (s *service) handleBookedAlertTask(ctx context.Context, task jobs.Task) error {
	payload := TaskAppointmentPayload{}
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return fmt.Errorf("decode booked alert payload: %w", err)
	}
	if payload.AppointmentID == "" {
		return errors.New("appointment_id is required")
	}

	appointment, err := gorm.G[models.Appointment](s.DB.GormDB()).
		Preload("Patient", nil).
		Preload("Clinic", nil).
		Where("uid = ?", payload.AppointmentID).
		First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load appointment for booked alert: %w", err)
	}

	sent, err := s.notificationSent(ctx, appointment.UID, "appointment_booked", models.NotificationMediumEmail)
	if err != nil {
		return fmt.Errorf("check booked notification status: %w", err)
	}
	if sent {
		return nil
	}

	s.sendBookedNow(ctx, appointment)
	return nil
}

func (s *service) notificationSent(ctx context.Context, appointmentUID, name string, medium models.NotificationMedium) (bool, error) {
	var count int64
	err := s.DB.WithContext(ctx).
		Model(&models.Notification{}).
		Where("alertable_id = ?", appointmentUID).
		Where("alertable_type = ?", "appointments").
		Where("name = ?", name).
		Where("medium = ?", medium).
		Where("status IN ?", []models.NotificationStatus{models.NotificationSent, models.NotificationSuccess}).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
