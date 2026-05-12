package twilio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultAPIBaseURL = "https://api.twilio.com"

type Config struct {
	AccountSID   string
	AuthToken    string
	WhatsAppFrom string
	APIBaseURL   string
	Client       *http.Client
}

type Sender struct {
	accountSID   string
	authToken    string
	whatsAppFrom string
	apiBaseURL   string
	client       *http.Client
}

func New(cfg Config) *Sender {
	apiBaseURL := strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = defaultAPIBaseURL
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &Sender{
		accountSID:   strings.TrimSpace(cfg.AccountSID),
		authToken:    strings.TrimSpace(cfg.AuthToken),
		whatsAppFrom: strings.TrimSpace(cfg.WhatsAppFrom),
		apiBaseURL:   apiBaseURL,
		client:       client,
	}
}

func (s *Sender) Send(ctx context.Context, to string, text string) error {
	if s == nil {
		return fmt.Errorf("twilio sender is nil")
	}
	if s.accountSID == "" {
		return fmt.Errorf("twilio account sid is required")
	}
	if s.authToken == "" {
		return fmt.Errorf("twilio auth token is required")
	}
	if s.whatsAppFrom == "" {
		return fmt.Errorf("twilio whatsapp from is required")
	}

	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("whatsapp recipient is required")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("whatsapp message text is required")
	}

	form := url.Values{}
	form.Set("To", withWhatsAppPrefix(to))
	form.Set("From", withWhatsAppPrefix(s.whatsAppFrom))
	form.Set("Body", text)

	endpoint := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Messages.json", s.apiBaseURL, s.accountSID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create twilio request: %w", err)
	}
	req.SetBasicAuth(s.accountSID, s.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send twilio request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read twilio response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("twilio send failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func withWhatsAppPrefix(value string) string {
	v := strings.TrimSpace(value)
	if strings.HasPrefix(v, "whatsapp:") {
		return v
	}
	return "whatsapp:" + v
}
