package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"miconsul/internal/lib/twilio"
)

type Sender struct {
	client *twilio.Client
	from   string
}

type Config struct {
	From       string
	AccountSID string
	AuthToken  string
	APIBaseURL string
	Client     *http.Client
}

type Message struct {
	To   string
	Body string
}

var e164Pattern = regexp.MustCompile(`^\+[1-9]\d{7,14}$`)

func NewSender(cfg Config) *Sender {
	client := twilio.New(twilio.Config{
		AccountSID: cfg.AccountSID,
		AuthToken:  cfg.AuthToken,
		APIBaseURL: cfg.APIBaseURL,
		Client:     cfg.Client,
	})

	return &Sender{client: client, from: strings.TrimSpace(cfg.From)}
}

func (s *Sender) Send(ctx context.Context, msg Message) error {
	if s == nil {
		return fmt.Errorf("sms sender is nil")
	}
	if s.client == nil {
		return fmt.Errorf("twilio client is required")
	}
	if s.from == "" {
		return fmt.Errorf("twilio sms from is required")
	}

	to := normalizePhone(msg.To)
	if to == "" {
		return fmt.Errorf("sms recipient is required")
	}
	if !e164Pattern.MatchString(to) {
		return fmt.Errorf("sms recipient must be E.164 (e.g. +5213121014574)")
	}

	body := strings.TrimSpace(msg.Body)
	if body == "" {
		return fmt.Errorf("sms body is required")
	}

	form := url.Values{}
	form.Set("To", to)
	form.Set("From", normalizePhone(s.from))
	form.Set("Body", body)

	path := fmt.Sprintf("/2010-04-01/Accounts/%s/Messages.json", s.client.AccountSID())
	_, err := s.client.PostForm(ctx, path, form)
	if err != nil {
		return err
	}

	return nil
}

func normalizePhone(value string) string {
	v := strings.TrimSpace(value)
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
