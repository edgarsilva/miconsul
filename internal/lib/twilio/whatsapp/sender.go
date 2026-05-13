// Package whatsapp provides twilio whatsapp sender utilities.
package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"miconsul/internal/lib/twilio"
)

type Sender struct {
	client       *twilio.Client
	whatsAppFrom string
	contentSID   string
}

type Config struct {
	WhatsAppFrom string
	ContentSID   string
	AccountSID   string
	AuthToken    string
	APIBaseURL   string
	Client       *http.Client
}

type TemplateMessage struct {
	To        string
	Variables map[string]string
}

var e164Pattern = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)

func NewSender(cfg Config) *Sender {
	client := twilio.New(twilio.Config{
		AccountSID: cfg.AccountSID,
		AuthToken:  cfg.AuthToken,
		APIBaseURL: cfg.APIBaseURL,
		Client:     cfg.Client,
	})

	return &Sender{
		client:       client,
		whatsAppFrom: strings.TrimSpace(cfg.WhatsAppFrom),
		contentSID:   strings.TrimSpace(cfg.ContentSID),
	}
}

func (s *Sender) SendTemplate(ctx context.Context, msg TemplateMessage) error {
	if s == nil {
		return fmt.Errorf("whatsapp sender is nil")
	}
	if s.client == nil {
		return fmt.Errorf("twilio client is required")
	}
	if s.whatsAppFrom == "" {
		return fmt.Errorf("twilio whatsapp from is required")
	}
	if s.contentSID == "" {
		return fmt.Errorf("twilio whatsapp content sid is required")
	}

	to := normalizePhone(msg.To)
	if to == "" {
		return fmt.Errorf("whatsapp recipient is required")
	}
	if !e164Pattern.MatchString(to) {
		return fmt.Errorf("whatsapp recipient must be E.164 (e.g. +5213121014574)")
	}

	contentVars, err := json.Marshal(msg.Variables)
	if err != nil {
		return fmt.Errorf("marshal twilio content variables: %w", err)
	}

	form := url.Values{}
	form.Set("To", withWhatsAppPrefix(to))
	form.Set("From", withWhatsAppPrefix(s.whatsAppFrom))
	form.Set("ContentSid", s.contentSID)
	form.Set("ContentVariables", string(contentVars))

	path := fmt.Sprintf("/2010-04-01/Accounts/%s/Messages.json", s.client.AccountSID())
	_, err = s.client.PostForm(ctx, path, form)
	if err != nil {
		return err
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

func normalizePhone(value string) string {
	v := strings.TrimSpace(value)
	v = strings.TrimPrefix(v, "whatsapp:")
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, "(", "")
	v = strings.ReplaceAll(v, ")", "")
	if v == "" {
		return v
	}

	if v[0] == '+' {
		return v
	}

	onlyDigits := v
	if len(onlyDigits) == 10 {
		return "+52" + onlyDigits
	}

	if strings.HasPrefix(onlyDigits, "52") {
		return "+" + onlyDigits
	}

	return "+" + onlyDigits
}
