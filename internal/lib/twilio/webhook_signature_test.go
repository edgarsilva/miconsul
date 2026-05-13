package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"testing"
)

func TestValidateWebhookSignature(t *testing.T) {
	authToken := "token123"
	requestURL := "https://miconsul.link/api/webhooks/twilio_delivery_status"
	form := url.Values{
		"MessageSid":    {"SM123"},
		"MessageStatus": {"delivered"},
	}

	payload := webhookSignaturePayload(requestURL, form)
	mac := hmac.New(sha1.New, []byte(authToken))
	_, _ = mac.Write([]byte(payload))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !ValidateWebhookSignature(authToken, requestURL, form, signature) {
		t.Fatalf("expected signature to validate")
	}

	if ValidateWebhookSignature(authToken, requestURL, form, "bad-signature") {
		t.Fatalf("expected signature validation to fail")
	}
}
