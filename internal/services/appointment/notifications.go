package appointment

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"miconsul/internal/lib/twilio/whatsapp"
	"miconsul/internal/mailer"
	"miconsul/internal/models"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
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
		kind, providerStatus := classifyWhatsAppError(err)
		attrs := []attribute.KeyValue{
			attribute.String("notification.whatsapp.error", err.Error()),
			attribute.String("notification.whatsapp.error_kind", kind),
		}
		if providerStatus > 0 {
			attrs = append(attrs, attribute.Int("notification.whatsapp.provider_status_code", providerStatus))
		}
		span.SetAttributes(attrs...)
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
		kind, providerStatus := classifyWhatsAppError(err)
		attrs := []attribute.KeyValue{
			attribute.String("notification.whatsapp.error", err.Error()),
			attribute.String("notification.whatsapp.error_kind", kind),
		}
		if providerStatus > 0 {
			attrs = append(attrs, attribute.Int("notification.whatsapp.provider_status_code", providerStatus))
		}
		span.SetAttributes(attrs...)
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
	notification.NotificationableID = appointment.AlertableID()
	notification.NotificationableType = appointment.AlertableType()
	if notification.NotificationableID == "" {
		fmt.Println("failed to append notification: appointment primary key missing")
		return
	}

	if err := gorm.G[models.Notification](s.DB.GormDB()).Create(ctx, &notification); err != nil {
		fmt.Println("failed to append notification:", err.Error())
	}
}

func classifyWhatsAppError(err error) (string, int) {
	if err == nil {
		return "", 0
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	status := extractProviderStatusCode(msg)

	switch {
	case strings.Contains(msg, "63038") || status == 429:
		return "rate_limited", status
	case strings.Contains(msg, "e.164") || strings.Contains(msg, "recipient") || strings.Contains(msg, "invalid"):
		return "invalid_recipient", status
	case strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "auth") || strings.Contains(msg, "token") || strings.Contains(msg, "missing twilio config"):
		return "auth_or_config", status
	default:
		return "provider_error", status
	}
}

func extractProviderStatusCode(msg string) int {
	const marker = "status "
	idx := strings.Index(msg, marker)
	if idx == -1 {
		return 0
	}

	start := idx + len(marker)
	if start+3 > len(msg) {
		return 0
	}

	code, err := strconv.Atoi(msg[start : start+3])
	if err != nil {
		return 0
	}

	return code
}
