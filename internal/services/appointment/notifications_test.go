package appointment

import (
	"errors"
	"testing"
)

func TestClassifyWhatsAppError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantKind   string
		wantStatus int
	}{
		{
			name:       "rate limited by status",
			err:        errors.New("twilio send failed with status 429: {\"code\":63038}"),
			wantKind:   "rate_limited",
			wantStatus: 429,
		},
		{
			name:       "invalid recipient",
			err:        errors.New("whatsapp recipient must be E.164"),
			wantKind:   "invalid_recipient",
			wantStatus: 0,
		},
		{
			name:       "auth config",
			err:        errors.New("whatsapp sender unavailable: missing twilio config"),
			wantKind:   "auth_or_config",
			wantStatus: 0,
		},
		{
			name:       "fallback provider error",
			err:        errors.New("unexpected provider boom"),
			wantKind:   "provider_error",
			wantStatus: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			kind, status := classifyWhatsAppError(tt.err)
			if kind != tt.wantKind {
				t.Fatalf("expected kind %q, got %q", tt.wantKind, kind)
			}
			if status != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, status)
			}
		})
	}
}
