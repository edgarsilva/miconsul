package appointment

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHandleTwilioDeliveryStatusWebhook(t *testing.T) {
	t.Run("returns unauthorized when signature is invalid", func(t *testing.T) {
		svc, _, _, _ := newAppointmentServiceForTests(t)
		svc.Env.AppProtocol = "https"
		svc.Env.AppDomain = "miconsul.link"
		svc.Env.TwilioAuthToken = "twilio-token"

		app := fiber.New()
		app.Post("/api/webhooks/twilio_delivery_status", svc.HandleTwilioDeliveryStatusWebhook)

		form := url.Values{"MessageSid": {"SM1"}, "MessageStatus": {"delivered"}}
		req := httptest.NewRequest(http.MethodPost, "/api/webhooks/twilio_delivery_status", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Twilio-Signature", "bad-signature")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("accepts valid twilio signature", func(t *testing.T) {
		svc, _, _, _ := newAppointmentServiceForTests(t)
		svc.Env.AppProtocol = "https"
		svc.Env.AppDomain = "miconsul.link"
		svc.Env.TwilioAuthToken = "twilio-token"

		app := fiber.New()
		app.Post("/api/webhooks/twilio_delivery_status", svc.HandleTwilioDeliveryStatusWebhook)

		form := url.Values{"MessageSid": {"SM1"}, "MessageStatus": {"delivered"}, "To": {"+523121014574"}}
		requestURL := "https://miconsul.link/api/webhooks/twilio_delivery_status"
		sig := twilioSignatureForTest("twilio-token", requestURL, form)

		req := httptest.NewRequest(http.MethodPost, "/api/webhooks/twilio_delivery_status", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Twilio-Signature", sig)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})
}

func twilioSignatureForTest(authToken, requestURL string, form url.Values) string {
	b := strings.Builder{}
	b.WriteString(requestURL)

	keys := make([]string, 0, len(form))
	for key := range form {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, value := range form[key] {
			b.WriteString(key)
			b.WriteString(value)
		}
	}

	mac := hmac.New(sha1.New, []byte(authToken))
	_, _ = mac.Write([]byte(b.String()))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
