package twilio

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"
)

func ValidateWebhookSignature(authToken, requestURL string, form url.Values, signature string) bool {
	authToken = strings.TrimSpace(authToken)
	requestURL = strings.TrimSpace(requestURL)
	signature = strings.TrimSpace(signature)
	if authToken == "" || requestURL == "" || signature == "" {
		return false
	}

	mac := hmac.New(sha1.New, []byte(authToken))
	_, _ = mac.Write([]byte(webhookSignaturePayload(requestURL, form)))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

func webhookSignaturePayload(requestURL string, form url.Values) string {
	if len(form) == 0 {
		return requestURL
	}

	b := strings.Builder{}
	b.WriteString(requestURL)

	keys := make([]string, 0, len(form))
	for key := range form {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		values := form[key]
		for _, value := range values {
			b.WriteString(key)
			b.WriteString(value)
		}
	}

	return b.String()
}
