package appointment

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3/log"
	"gorm.io/gorm"
)

func TestRecordFeedEventLogsFailures(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)
	ctx := context.Background()

	apnt := models.Appointment{UserID: user.ID, ClinicID: clinic.ID, PatientID: patient.ID, BookedAt: time.Now().Add(time.Hour), Token: "tok_feed"}
	if err := svc.CreateAppointment(ctx, &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	err := svc.DB.GormDB().Callback().Create().Before("gorm:create").Register("test_force_feed_events_create_error", func(db *gorm.DB) {
		if db.Statement != nil && db.Statement.Table == "feed_events" {
			db.AddError(errors.New("forced feed event error"))
		}
	})
	if err != nil {
		t.Fatalf("register create callback: %v", err)
	}

	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	if err := svc.StartAppointmentSessionByID(ctx, user.ID, apnt.UID); err != nil {
		t.Fatalf("expected the appointment change to succeed despite the feed event, got %v", err)
	}
	if !strings.Contains(logs.String(), "record updated feed event for appointment "+apnt.UID) {
		t.Fatalf("expected the feed event failure to be logged, got %q", logs.String())
	}

	logs.Reset()
	svc.recordFeedEvent(ctx, user.ID, "missing_uid", models.EventActionDeleted)
	if logs.Len() != 0 {
		t.Fatalf("expected no log for an unknown appointment, got %q", logs.String())
	}
}
