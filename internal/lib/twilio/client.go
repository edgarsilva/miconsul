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
	AccountSID string
	AuthToken  string
	APIBaseURL string
	Client     *http.Client
}

type Client struct {
	accountSID string
	authToken  string
	apiBaseURL string
	httpClient *http.Client
}

func New(cfg Config) *Client {
	apiBaseURL := strings.TrimRight(strings.TrimSpace(cfg.APIBaseURL), "/")
	if apiBaseURL == "" {
		apiBaseURL = defaultAPIBaseURL
	}

	httpClient := cfg.Client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		accountSID: strings.TrimSpace(cfg.AccountSID),
		authToken:  strings.TrimSpace(cfg.AuthToken),
		apiBaseURL: apiBaseURL,
		httpClient: httpClient,
	}
}

func (c *Client) PostForm(ctx context.Context, path string, form url.Values) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("twilio client is nil")
	}
	if c.accountSID == "" {
		return nil, fmt.Errorf("twilio account sid is required")
	}
	if c.authToken == "" {
		return nil, fmt.Errorf("twilio auth token is required")
	}

	endpoint := c.apiBaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create twilio request: %w", err)
	}
	req.SetBasicAuth(c.accountSID, c.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send twilio request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read twilio response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("twilio send failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, nil
}

func (c *Client) AccountSID() string {
	if c == nil {
		return ""
	}
	return c.accountSID
}
