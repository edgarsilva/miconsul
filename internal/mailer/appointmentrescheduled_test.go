package mailer

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"miconsul/internal/lib/preferences"
	"miconsul/internal/models"
)

func TestAppointmentRescheduledEmailRenders(t *testing.T) {
	appointment := models.Appointment{
		BookedAt: time.Date(2026, 12, 1, 16, 30, 0, 0, time.UTC),
		Patient:  models.Patient{Name: "Ana Perez"},
		Clinic:   models.Clinic{Name: "Clinica Centro", Line1: "Calle 1", City: "CDMX", State: "CDMX"},
	}

	for _, locale := range []string{string(preferences.DefaultLocale), "en-US"} {
		var out bytes.Buffer
		if err := AppointmentRescheduledEmail(locale, appointment).Render(t.Context(), &out); err != nil {
			t.Fatalf("render %s: %v", locale, err)
		}
		html := out.String()
		for _, want := range []string{"Ana Perez", "Clinica Centro", "Calle 1", l(locale, "email.rescheduled_body"), l(locale, "email.rescheduled_closing")} {
			if !strings.Contains(html, want) && !strings.Contains(html, strings.ReplaceAll(want, "'", "&#39;")) {
				t.Errorf("%s email is missing %q", locale, want)
			}
		}
	}
}
