package tests

import (
	"net/http"
	"testing"
	"time"

	"miconsul/internal/models"
)

// newPatientLinkAppointment seeds an appointment with a patient link token.
func newPatientLinkAppointment(t *testing.T, h *testHarness, bookedAt time.Time) models.Appointment {
	t.Helper()

	u := h.createUser(models.UserRoleUser)
	p := h.createPatient(u.ID, "Ana")
	c := h.createClinic(u.ID, "Clinic")
	a := h.createAppointment(u.ID, p.ID, c.ID)
	a.Token = "TESTTOKEN234567ABCDEFGHIJK"
	a.BookedAt = bookedAt
	if err := h.db.GormDB().Model(&models.Appointment{}).Where("id = ?", a.ID).
		Updates(map[string]any{"token": a.Token, "booked_at": a.BookedAt}).Error; err != nil {
		t.Fatalf("set appointment token: %v", err)
	}

	return a
}

func appointmentStatus(t *testing.T, h *testHarness, id uint) models.AppointmentStatus {
	t.Helper()

	var a models.Appointment
	if err := h.db.GormDB().Select("status").Where("id = ?", id).Take(&a).Error; err != nil {
		t.Fatalf("load appointment: %v", err)
	}

	return a.Status
}

func TestPatientLinksWorkWithoutLogin(t *testing.T) {
	t.Run("local auth", func(t *testing.T) { testPatientLinksWorkWithoutLogin(t, newTestHarness(t)) })
	t.Run("logto", func(t *testing.T) { testPatientLinksWorkWithoutLogin(t, newTestHarness(t, withLogto)) })
}

func testPatientLinksWorkWithoutLogin(t *testing.T, h *testHarness) {
	a := newPatientLinkAppointment(t, h, time.Now().Add(48*time.Hour))

	resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: a.ConfirmPath()})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous confirm page: status %d (location %q), want 200", resp.StatusCode, resp.Header.Get("Location"))
	}

	resp, _ = h.doRequest(requestOptions{method: http.MethodPost, path: a.ConfirmPath()})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous confirm: status %d, want 200", resp.StatusCode)
	}
	if got := appointmentStatus(t, h, a.ID); got != models.AppointmentConfirmed {
		t.Fatalf("status after confirm = %q, want confirmed", got)
	}

	resp, _ = h.doRequest(requestOptions{method: http.MethodGet, path: a.CancelPath()})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous cancel page: status %d, want 200", resp.StatusCode)
	}

	resp, _ = h.doRequest(requestOptions{method: http.MethodPost, path: a.CancelPath()})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous cancel: status %d, want 200", resp.StatusCode)
	}
	if got := appointmentStatus(t, h, a.ID); got != models.AppointmentCanceled {
		t.Fatalf("status after cancel = %q, want canceled", got)
	}
}

func TestDoctorAppointmentRoutesStillRequireLogin(t *testing.T) {
	h := newTestHarness(t)
	a := newPatientLinkAppointment(t, h, time.Now().Add(48*time.Hour))

	resp, _ := h.doRequest(requestOptions{method: http.MethodGet, path: "/appointments/" + a.UID})
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("anonymous doctor appointment page: status 200, want a redirect to sign in")
	}
}
