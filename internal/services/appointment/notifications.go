package appointment

import (
	"context"
	"fmt"
	"strings"

	"miconsul/internal/lib/twilio/whatsapp"
	"miconsul/internal/mailer"
	"miconsul/internal/models"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (s *service) DispatchBookedAlert(appointment models.Appointment) error {
	if s.Env.JobsEnabled {
		payload := TaskAppointmentPayload{AppointmentID: appointment.UID}
		_, err := s.EnqueueTask(context.Background(), TaskBookedAlert, payload)
		return err
	}

	return s.SendToWorker(context.Background(), func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultWorkerContextTimeout)
		defer cancel()

		s.sendBookedNow(ctx, appointment)
	})
}

func (s *service) SendReminder(appointment models.Appointment) error {
	return s.SendToWorker(context.Background(), func() {
		ctx, cancel := context.WithTimeout(context.Background(), defaultWorkerContextTimeout)
		defer cancel()

		s.sendReminderNow(ctx, appointment)
	})
}

func (s *service) SendReminderAlert(appointment models.Appointment) error {
	return s.SendReminder(appointment)
}

func (s *service) sendReminderNow(ctx context.Context, appointment models.Appointment) {
	ctx, span := s.Trace(ctx, "appointment/sendReminderNow")
	defer span.End()

	span.SetAttributes(attribute.String("appointment.uid", appointment.UID))

	enabled := appointment.EnableNotifications || appointment.Patient.EnableNotifications
	if !enabled {
		span.SetAttributes(attribute.Bool("notification.enabled", false))
		return
	}
	span.SetAttributes(attribute.Bool("notification.enabled", true))

	hadErrors := false

	if err := s.sendEmailNotification(ctx, span, appointment, "appointment_reminder", func() error {
		return mailer.SendAppointmentReminderEmail(s.Env, appointment)
	}); err != nil {
		hadErrors = true
	}

	bookedAt := appointment.BookedAtInLocalTime()
	vars := map[string]string{
		"1": bookedAt.Format("1/2"),
		"2": bookedAt.Format("3:04 PM"),
	}
	if err := s.sendWhatsAppNotification(ctx, span, appointment, "appointment_reminder", vars); err != nil {
		hadErrors = true
	}

	if span.SpanContext().IsValid() {
		if hadErrors {
			span.SetStatus(codes.Error, "notifications processed with errors")
		} else {
			span.SetStatus(codes.Ok, "notifications processed")
		}
	}
}

func (s *service) sendBookedNow(ctx context.Context, appointment models.Appointment) {
	ctx, span := s.Trace(ctx, "appointment/sendBookedNow")
	defer span.End()

	span.SetAttributes(attribute.String("appointment.uid", appointment.UID))

	enabled := appointment.EnableNotifications || appointment.Patient.EnableNotifications
	if !enabled {
		span.SetAttributes(attribute.Bool("notification.enabled", false))
		return
	}
	span.SetAttributes(attribute.Bool("notification.enabled", true))

	hadErrors := false

	if err := s.sendEmailNotification(ctx, span, appointment, "appointment_booked", func() error {
		return mailer.SendAppointmentBookedEmail(s.Env, appointment)
	}); err != nil {
		hadErrors = true
	}

	bookedAt := appointment.BookedAtInLocalTime()
	vars := map[string]string{
		"1": bookedAt.Format("1/2"),
		"2": bookedAt.Format("3:04 PM"),
	}
	if err := s.sendWhatsAppNotification(ctx, span, appointment, "appointment_booked", vars); err != nil {
		hadErrors = true
	}

	if span.SpanContext().IsValid() {
		if hadErrors {
			span.SetStatus(codes.Error, "notifications processed with errors")
		} else {
			span.SetStatus(codes.Ok, "notifications processed")
		}
	}
}

func (s *service) sendEmailNotification(ctx context.Context, span trace.Span, appointment models.Appointment, eventName string, send func() error) error {
	viaEmail := appointment.ViaEmail || appointment.Patient.ViaEmail
	to := strings.TrimSpace(appointment.Patient.Email)
	span.SetAttributes(attribute.Bool("notification.email.enabled", viaEmail), attribute.Bool("notification.email.to_present", to != ""))
	if !viaEmail || to == "" {
		return nil
	}

	status := models.NotificationSent
	var sendErr error
	if err := send(); err != nil {
		status = models.NotificationFailed
		sendErr = err
		span.RecordError(err)
		span.SetAttributes(attribute.String("notification.email.error", err.Error()))
	}

	s.appendNotification(ctx, appointment, models.Notification{
		Medium: models.NotificationMediumEmail,
		Name:   eventName,
		Status: status,
		To:     to,
	})

	return sendErr
}

func (s *service) sendWhatsAppNotification(ctx context.Context, span trace.Span, appointment models.Appointment, eventName string, vars map[string]string) error {
	viaWhatsApp := appointment.ViaWhatsapp || appointment.Patient.ViaWhatsapp
	to := strings.TrimSpace(appointment.Patient.Phone)
	span.SetAttributes(attribute.Bool("notification.whatsapp.enabled", viaWhatsApp), attribute.Bool("notification.whatsapp.to_present", to != ""))
	if !viaWhatsApp || to == "" {
		return nil
	}

	sender := s.newWhatsAppSender()
	if sender == nil {
		err := fmt.Errorf("whatsapp sender unavailable: missing twilio config")
		span.RecordError(err)
		span.SetAttributes(attribute.String("notification.whatsapp.error", err.Error()))
		s.appendNotification(ctx, appointment, models.Notification{Medium: models.NotificationMediumWhatsapp, Name: eventName, Status: models.NotificationFailed, To: to})
		return err
	}

	status := models.NotificationSent
	var sendErr error
	msg := whatsapp.TemplateMessage{To: to, Variables: vars}
	if err := sender.SendTemplate(ctx, msg); err != nil {
		status = models.NotificationFailed
		sendErr = err
		span.RecordError(err)
		span.SetAttributes(attribute.String("notification.whatsapp.error", err.Error()))
	}

	s.appendNotification(ctx, appointment, models.Notification{
		Medium: models.NotificationMediumWhatsapp,
		Name:   eventName,
		Status: status,
		To:     to,
	})

	return sendErr
}

func (s *service) newWhatsAppSender() *whatsapp.Sender {
	if s == nil || s.Env == nil {
		return nil
	}
	if strings.ToLower(strings.TrimSpace(s.Env.WhatsAppProvider)) != "twilio" {
		return nil
	}
	if strings.TrimSpace(s.Env.TwilioAccountSID) == "" || strings.TrimSpace(s.Env.TwilioAuthToken) == "" || strings.TrimSpace(s.Env.TwilioWhatsAppFrom) == "" || strings.TrimSpace(s.Env.TwilioWhatsAppContentSID) == "" {
		return nil
	}

	return whatsapp.NewSender(whatsapp.Config{
		WhatsAppFrom: s.Env.TwilioWhatsAppFrom,
		ContentSID:   s.Env.TwilioWhatsAppContentSID,
		AccountSID:   s.Env.TwilioAccountSID,
		AuthToken:    s.Env.TwilioAuthToken,
		APIBaseURL:   s.Env.TwilioAPIBaseURL,
	})
}

func (s *service) appendNotification(ctx context.Context, appointment models.Appointment, notification models.Notification) {
	appendErr := s.DB.WithContext(ctx).Model(&appointment).Association("Notifications").Append(&notification)
	if appendErr != nil {
		fmt.Println("failed to append notification:", appendErr.Error())
	}
}
