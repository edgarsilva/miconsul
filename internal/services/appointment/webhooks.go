package appointment

import (
	"fmt"
	"net/url"
	"strings"

	"miconsul/internal/lib/twilio"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// HandleTwilioDeliveryStatusWebhook receives Twilio message delivery status callbacks.
// POST: /api/webhooks/twilio_delivery_status
func (s *service) HandleTwilioDeliveryStatusWebhook(c fiber.Ctx) error {
	_, span := s.Trace(c.Context(), "appointment/webhooks:twilio_delivery_status")
	defer span.End()

	if strings.TrimSpace(s.Env.TwilioAuthToken) == "" {
		span.SetStatus(codes.Error, "twilio auth token missing")
		return c.SendStatus(fiber.StatusServiceUnavailable)
	}

	requestURL := twilioWebhookURL(s, c)
	if requestURL == "" {
		span.SetStatus(codes.Error, "webhook url unavailable")
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	form := c.Request().PostArgs().QueryString()
	parsedForm, _ := url.ParseQuery(string(form))
	signature := c.Get("X-Twilio-Signature", "")
	if !twilio.ValidateWebhookSignature(s.Env.TwilioAuthToken, requestURL, parsedForm, signature) {
		span.SetStatus(codes.Error, "invalid twilio signature")
		span.SetAttributes(attribute.String("webhook.provider", "twilio"))
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	messageSID := strings.TrimSpace(c.FormValue("MessageSid", ""))
	messageStatus := strings.TrimSpace(c.FormValue("MessageStatus", ""))
	to := strings.TrimSpace(c.FormValue("To", ""))
	errorCode := strings.TrimSpace(c.FormValue("ErrorCode", ""))
	errorMessage := strings.TrimSpace(c.FormValue("ErrorMessage", ""))

	span.SetAttributes(
		attribute.String("webhook.provider", "twilio"),
		attribute.String("notification.twilio.message_sid", messageSID),
		attribute.String("notification.twilio.message_status", messageStatus),
		attribute.String("notification.twilio.to", to),
	)
	if errorCode != "" {
		span.SetAttributes(attribute.String("notification.twilio.error_code", errorCode))
	}
	if errorMessage != "" {
		span.SetAttributes(attribute.String("notification.twilio.error_message", errorMessage))
	}

	log.Infof("twilio delivery status webhook sid=%s status=%s to=%s error_code=%s", messageSID, messageStatus, to, errorCode)
	return c.SendStatus(fiber.StatusOK)
}

func twilioWebhookURL(s *service, c fiber.Ctx) string {
	if s == nil || s.Env == nil {
		return ""
	}

	protocol := strings.TrimSpace(s.Env.AppProtocol)
	host := strings.TrimSpace(s.Env.AppDomain)
	path := strings.TrimSpace(c.OriginalURL())
	if protocol == "" || host == "" || path == "" {
		return ""
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return fmt.Sprintf("%s://%s%s", protocol, host, path)
}
