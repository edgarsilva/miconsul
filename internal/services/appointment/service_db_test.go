package appointment

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/lib/libtime"
	"miconsul/internal/models"
	"miconsul/internal/server"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAppointmentServiceDBFlows(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)
	ctx := context.Background()

	t.Run("create and fetch appointment", func(t *testing.T) {
		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(time.Hour),
			Token:     "tok_1",
		}
		if err := svc.CreateAppointment(ctx, &apnt); err != nil {
			t.Fatalf("create appointment: %v", err)
		}
		if apnt.Status != models.AppointmentPending {
			t.Fatalf("expected default pending status, got %q", apnt.Status)
		}

		got, err := svc.TakeAppointmentByUID(ctx, user.ID, apnt.UID)
		if err != nil {
			t.Fatalf("take appointment: %v", err)
		}
		if got.ID != apnt.ID || got.Clinic.ID == 0 || got.Patient.ID == 0 {
			t.Fatalf("expected preloaded appointment entities")
		}
	})

	t.Run("lookups by ids work", func(t *testing.T) {
		if _, err := svc.TakePatientByUID(ctx, user.ID, patient.UID); err != nil {
			t.Fatalf("take patient by id: %v", err)
		}
		if _, err := svc.TakeClinicByUID(ctx, user.ID, clinic.UID); err != nil {
			t.Fatalf("take clinic by id: %v", err)
		}
	})

	t.Run("lookups trim the uid before querying", func(t *testing.T) {
		if _, err := svc.TakePatientByUID(ctx, user.ID, " "+patient.UID+" "); err != nil {
			t.Fatalf("take patient by padded uid: %v", err)
		}
		if _, err := svc.TakeClinicByUID(ctx, user.ID, " "+clinic.UID+" "); err != nil {
			t.Fatalf("take clinic by padded uid: %v", err)
		}

		apnt := models.Appointment{UserID: user.ID, ClinicID: clinic.ID, PatientID: patient.ID, BookedAt: time.Now().Add(time.Hour), Token: "tok_padded"}
		if err := svc.CreateAppointment(ctx, &apnt); err != nil {
			t.Fatalf("create appointment: %v", err)
		}
		if _, err := svc.TakeAppointmentByUID(ctx, user.ID, " "+apnt.UID+" "); err != nil {
			t.Fatalf("take appointment by padded uid: %v", err)
		}
		if _, err := svc.TakeAppointmentByToken(ctx, " "+apnt.Token+" "); err != nil {
			t.Fatalf("take appointment by padded token: %v", err)
		}
	})

	t.Run("patient link actions", func(t *testing.T) {
		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(2 * time.Hour),
			Token:     "tok_flow",
			Status:    models.AppointmentPending,
		}
		if err := svc.CreateAppointment(ctx, &apnt); err != nil {
			t.Fatalf("create appointment: %v", err)
		}

		loaded, err := svc.TakeAppointmentByToken(ctx, apnt.Token)
		if err != nil {
			t.Fatalf("take by token: %v", err)
		}
		if loaded, err = svc.ConfirmAppointment(ctx, loaded); err != nil {
			t.Fatalf("confirm: %v", err)
		}
		if loaded, err = svc.RequestAppointmentDateChange(ctx, loaded); err != nil {
			t.Fatalf("request date change: %v", err)
		}

		got, err := svc.TakeAppointmentByToken(ctx, apnt.Token)
		if err != nil {
			t.Fatalf("take by token: %v", err)
		}
		if got.Status != models.AppointmentPending || loaded.Status != models.AppointmentPending {
			t.Fatalf("expected pending status after date change, got stored %q, returned %q", got.Status, loaded.Status)
		}

		if loaded, err = svc.CancelAppointment(ctx, loaded); err != nil {
			t.Fatalf("cancel: %v", err)
		}
		if loaded.PatientCanConfirm() {
			t.Fatalf("a canceled appointment must not allow confirm or date change")
		}
	})

	t.Run("listing helpers return data", func(t *testing.T) {
		appointments, err := svc.FindAppointmentsBy(ctx, user.ID, appointmentFilter{Timeframe: libtime.TimeframeWeek})
		if err != nil {
			t.Fatalf("find appointments: %v", err)
		}
		if len(appointments) == 0 {
			t.Fatalf("expected at least one appointment")
		}

		clinics, err := svc.FindRecentClinicsByUserID(ctx, user.ID, 5)
		if err != nil {
			t.Fatalf("find recent clinics: %v", err)
		}
		if len(clinics) == 0 {
			t.Fatalf("expected at least one clinic")
		}

		patients, err := svc.FindRecentPatientsByUserID(ctx, user.ID, 5)
		if err != nil {
			t.Fatalf("find recent patients: %v", err)
		}
		if len(patients) == 0 {
			t.Fatalf("expected at least one patient")
		}

		clinicsBySearch, err := svc.FindClinicsBySearchTerm(ctx, user.ID, "")
		if err != nil {
			t.Fatalf("find clinics by empty search term: %v", err)
		}
		if len(clinicsBySearch) == 0 {
			t.Fatalf("expected clinics from empty search fallback")
		}

		_, err = svc.FindAppointmentsBy(ctx, user.ID, appointmentFilter{Timeframe: libtime.TimeframeWeek, SearchTerm: "patient"})
		if err == nil {
			t.Fatalf("expected FTS-backed appointment search error without global_fts table")
		}

		_, err = svc.FindClinicsBySearchTerm(ctx, user.ID, "clinic")
		if err == nil {
			t.Fatalf("expected FTS-backed clinic search error without global_fts table")
		}
	})

	t.Run("update and delete by id branches", func(t *testing.T) {
		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(3 * time.Hour),
			Token:     "tok_upd_del",
			Status:    models.AppointmentPending,
		}
		if err := svc.CreateAppointment(ctx, &apnt); err != nil {
			t.Fatalf("create appointment for update/delete: %v", err)
		}

		patch := appointmentPatchUpdates{
			BookedAt:     time.Now().Add(4 * time.Hour),
			BookedYear:   time.Now().Year(),
			BookedMonth:  int(time.Now().Month()),
			BookedDay:    time.Now().Day(),
			BookedHour:   12,
			BookedMinute: 30,
			Price:        15000,
			ClinicID:     clinic.ID,
			PatientID:    patient.ID,
			Duration:     60,
		}
		if err := svc.UpdateAppointmentByID(ctx, user.ID, apnt.UID, patch); err != nil {
			t.Fatalf("update appointment by id: %v", err)
		}

		if err := svc.UpdateAppointmentByID(ctx, user.ID, "missing", patch); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected record not found for missing update, got %v", err)
		}

		if err := svc.DeleteAppointmentByID(ctx, user.ID, apnt.UID); err != nil {
			t.Fatalf("delete appointment by id: %v", err)
		}
		if err := svc.DeleteAppointmentByID(ctx, user.ID, apnt.UID); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected record not found for repeated delete, got %v", err)
		}
	})
}

func newAppointmentServiceForTests(t *testing.T) (*service, models.User, models.Clinic, models.Patient) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := gdb.AutoMigrate(&models.User{}, &models.Clinic{}, &models.Patient{}, &models.Appointment{}, &models.Notification{}, &models.FeedEvent{}); err != nil {
		t.Fatalf("automigrate models: %v", err)
	}

	srv := &server.Server{
		Env: &appenv.Env{AppName: "miconsul", JWTSecret: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"},
		DB:  &database.Database{DB: gdb},
	}
	svc, err := New(srv)
	if err != nil {
		t.Fatalf("new appointment service: %v", err)
	}

	u := models.User{Email: "appt@example.com", Password: "hash", Role: models.UserRoleUser}
	if err := gorm.G[models.User](gdb).Create(context.Background(), &u); err != nil {
		t.Fatalf("create user: %v", err)
	}

	cl := models.Clinic{UserID: u.ID, Name: "Clinic", Email: "c@example.com", Phone: "123"}
	if err := gorm.G[models.Clinic](gdb).Create(context.Background(), &cl); err != nil {
		t.Fatalf("create clinic: %v", err)
	}

	pt := models.Patient{UserID: u.ID, Name: "Patient", Age: 30, Phone: "555", Email: "p@example.com"}
	if err := gorm.G[models.Patient](gdb).Create(context.Background(), &pt); err != nil {
		t.Fatalf("create patient: %v", err)
	}

	return svc, u, cl, pt
}

func TestPatientLinkActionsAreIdempotent(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)
	ctx := context.Background()

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(24 * time.Hour),
		Token:     "tok_idempotent",
	}
	if err := svc.CreateAppointment(ctx, &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	countEvents := func(action models.EventAction) int64 {
		var n int64
		if err := svc.DB.GormDB().Model(&models.FeedEvent{}).Where("action = ? AND feed_eventable_id = ?", string(action), apnt.UID).Count(&n).Error; err != nil {
			t.Fatalf("count feed events: %v", err)
		}
		return n
	}

	act := func(name string, fn func(context.Context, models.Appointment) (models.Appointment, error)) {
		t.Helper()
		loaded, err := svc.TakeAppointmentByToken(ctx, apnt.Token)
		if err != nil {
			t.Fatalf("%s: take: %v", name, err)
		}
		if _, err := fn(ctx, loaded); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	for range 2 {
		act("confirm", svc.ConfirmAppointment)
	}
	first, err := svc.TakeAppointmentByToken(ctx, apnt.Token)
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if n := countEvents(models.EventActionConfirmed); n != 1 {
		t.Fatalf("confirmed feed events after two confirms = %d, want 1", n)
	}

	act("third confirm", svc.ConfirmAppointment)
	again, _ := svc.TakeAppointmentByToken(ctx, apnt.Token)
	if !again.ConfirmedAt.Equal(first.ConfirmedAt) {
		t.Fatalf("a repeat confirm changed ConfirmedAt")
	}

	for range 2 {
		act("cancel", svc.CancelAppointment)
	}
	if n := countEvents(models.EventActionCanceled); n != 1 {
		t.Fatalf("canceled feed events after two cancels = %d, want 1", n)
	}
}

func TestPatientActionRefusedAfterConcurrentChange(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)
	ctx := context.Background()

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(24 * time.Hour),
		Token:     "tok_concurrent",
	}
	if err := svc.CreateAppointment(ctx, &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	loaded, err := svc.TakeAppointmentByToken(ctx, apnt.Token)
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if err := svc.DB.GormDB().Model(&models.Appointment{}).Where("id = ?", apnt.ID).Update("status", models.AppointmentCanceled).Error; err != nil {
		t.Fatalf("cancel underneath: %v", err)
	}

	if _, err := svc.ConfirmAppointment(ctx, loaded); !errors.Is(err, ErrAppointmentChanged) {
		t.Fatalf("confirm after a concurrent cancel: err = %v, want ErrAppointmentChanged", err)
	}
	got, err := svc.TakeAppointmentByToken(ctx, apnt.Token)
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if got.Status != models.AppointmentCanceled {
		t.Fatalf("status = %q, want still canceled", got.Status)
	}
}

func TestFindAppointmentsByStatus(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)
	ctx := context.Background()

	for i, status := range []models.AppointmentStatus{models.AppointmentConfirmed, models.AppointmentPending} {
		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now(),
			Status:    status,
			Token:     fmt.Sprintf("tok_status_%d", i),
		}
		if err := svc.CreateAppointment(ctx, &apnt); err != nil {
			t.Fatalf("create %s appointment: %v", status, err)
		}
	}

	filter := appointmentFilter{
		PatientUID: patient.UID,
		ClinicUID:  clinic.UID,
		Timeframe:  libtime.TimeframeDay,
		Statuses:   []models.AppointmentStatus{models.AppointmentConfirmed},
	}
	appointments, err := svc.FindAppointmentsBy(ctx, user.ID, filter)
	if err != nil {
		t.Fatalf("find appointments: %v", err)
	}
	if len(appointments) != 1 || appointments[0].Status != models.AppointmentConfirmed {
		t.Fatalf("expected only the confirmed appointment, got %+v", appointments)
	}

	filter.Statuses = nil
	appointments, err = svc.FindAppointmentsBy(ctx, user.ID, filter)
	if err != nil {
		t.Fatalf("find appointments: %v", err)
	}
	if len(appointments) != 2 {
		t.Fatalf("expected both appointments without a status filter, got %d", len(appointments))
	}
}
