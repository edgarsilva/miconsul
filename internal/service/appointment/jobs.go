package appointment

import (
	"fmt"

	"miconsul/internal/model"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s *service) RegisterCronJob() error {
	err := s.AddCronJobOnce("appointment:reminder", "0/1 * * * *", func() {
		jobCtx, cancel := s.newCronJobContext()
		defer cancel()

		ctx, span := s.Trace(jobCtx, "appointment.cron.reminder_job",
			trace.WithAttributes(
				attribute.String("grouping.fingerprint", "cronjob"),
			),
		)
		defer span.End()

		appointments := []model.Appointment{}
		err := s.DB.
			WithContext(ctx).
			Model(&model.Appointment{}).
			Preload("Patient").
			Preload("Clinic").
			Scopes(model.AppointmentWithPendingAlerts).
			Find(&appointments).
			Error
		if err != nil {
			fmt.Println("failed to load appointments for reminder job:", err.Error())
			return
		}
		for _, appointment := range appointments {
			s.SendReminderAlert(appointment)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to register appointment reminder cron job: %w", err)
	}

	return nil
}
