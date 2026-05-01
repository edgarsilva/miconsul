package appointment

import (
	"context"
	"fmt"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
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
		if apnt.Status != models.ApntStatusPending {
			t.Fatalf("expected default pending status, got %q", apnt.Status)
		}

		got, err := svc.TakeAppointmentByID(ctx, user.ID, apnt.ID)
		if err != nil {
			t.Fatalf("take appointment: %v", err)
		}
		if got.ID != apnt.ID || got.Clinic.ID == "" || got.Patient.ID == "" {
			t.Fatalf("expected preloaded appointment entities")
		}
	})

	t.Run("lookups by ids work", func(t *testing.T) {
		if _, err := svc.TakePatientByID(ctx, user.ID, patient.ID); err != nil {
			t.Fatalf("take patient by id: %v", err)
		}
		if _, err := svc.TakeClinicByID(ctx, user.ID, clinic.ID); err != nil {
			t.Fatalf("take clinic by id: %v", err)
		}
	})

	t.Run("token update wrappers execute", func(t *testing.T) {
		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(2 * time.Hour),
			Token:     "tok_flow",
			Status:    models.ApntStatusPending,
		}
		if err := svc.CreateAppointment(ctx, &apnt); err != nil {
			t.Fatalf("create appointment: %v", err)
		}

		if err := svc.ConfirmAppointmentByIDAndToken(ctx, apnt.ID, apnt.Token); err != nil {
			t.Fatalf("confirm by token: %v", err)
		}
		if err := svc.CancelAppointmentByIDAndToken(ctx, apnt.ID, apnt.Token); err != nil {
			t.Fatalf("cancel by token: %v", err)
		}
		if err := svc.RequestAppointmentDateChangeByIDAndToken(ctx, apnt.ID, apnt.Token); err != nil {
			t.Fatalf("request date change by token: %v", err)
		}

		got, err := svc.TakeAppointmentByIDAndToken(ctx, apnt.ID, apnt.Token)
		if err != nil {
			t.Fatalf("take by id and token: %v", err)
		}
		if got.Status != models.ApntStatusPending {
			t.Fatalf("expected pending status after date change, got %q", got.Status)
		}
	})

	t.Run("listing helpers return data", func(t *testing.T) {
		appointments, err := svc.FindAppointmentsBy(ctx, user.ID, patient.ID, clinic.ID, "day")
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

		_, err = svc.FindClinicsBySearchTerm(ctx, user.ID, "clinic")
		if err == nil {
			t.Fatalf("expected FTS-backed clinic search error without global_fts table")
		}
	})

	t.Run("patient with last done appointment preloads done appointment", func(t *testing.T) {
		doneApnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(-time.Hour),
			Status:    models.ApntStatusDone,
			Token:     "tok_done",
		}
		if err := svc.CreateAppointment(ctx, &doneApnt); err != nil {
			t.Fatalf("create done appointment: %v", err)
		}

		got, err := svc.TakePatientByIDWithLastDoneAppointment(ctx, user.ID, patient.ID)
		if err != nil {
			t.Fatalf("take patient with done appointment: %v", err)
		}
		if len(got.Appointments) == 0 || got.Appointments[0].Status != models.ApntStatusDone {
			t.Fatalf("expected preloaded done appointment")
		}
	})

	t.Run("update and delete by id branches", func(t *testing.T) {
		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(3 * time.Hour),
			Token:     "tok_upd_del",
			Status:    models.ApntStatusPending,
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
		if err := svc.UpdateAppointmentByID(ctx, user.ID, apnt.ID, patch); err != nil {
			t.Fatalf("update appointment by id: %v", err)
		}

		if err := svc.UpdateAppointmentByID(ctx, user.ID, "missing", patch); err != gorm.ErrRecordNotFound {
			t.Fatalf("expected record not found for missing update, got %v", err)
		}

		if err := svc.DeleteAppointmentByID(ctx, user.ID, apnt.ID); err != nil {
			t.Fatalf("delete appointment by id: %v", err)
		}
		if err := svc.DeleteAppointmentByID(ctx, user.ID, apnt.ID); err != gorm.ErrRecordNotFound {
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

	if err := gdb.AutoMigrate(&models.User{}, &models.Clinic{}, &models.Patient{}, &models.Appointment{}); err != nil {
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
