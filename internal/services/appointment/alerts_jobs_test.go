package appointment

import (
	"context"
	"encoding/json/v2"
	"slices"
	"strconv"
	"testing"
	"time"

	"miconsul/internal/jobs"
	"miconsul/internal/models"

	"gorm.io/gorm"
)

func TestSendReminderAlertExecutesWorkerPath(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_reminder",
		Patient:   patient,
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	if err := svc.SendReminderAlert(apnt); err != nil {
		t.Fatalf("expected reminder worker enqueue success, got %v", err)
	}
}

func TestFindAppointmentsDueForReminderComparesInUTC(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	mx, err := time.LoadLocation(models.DefaultTimezone)
	if err != nil {
		t.Fatalf("load %s: %v", models.DefaultTimezone, err)
	}
	now := time.Now().In(mx)

	bookings := map[string]time.Duration{
		"tok_soon":  time.Hour,
		"tok_later": 3 * time.Hour,
		"tok_past":  -time.Hour,
	}
	for token, in := range bookings {
		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  now.Add(in),
			Token:     token,
		}
		if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
			t.Fatalf("create appointment %s: %v", token, err)
		}
	}

	due, err := svc.findAppointmentsDueForReminder(t.Context(), now)
	if err != nil {
		t.Fatalf("find due appointments: %v", err)
	}
	if len(due) != 1 || due[0].Token != "tok_soon" {
		t.Fatalf("due = %d appointments %v, want only tok_soon", len(due), dueTokens(due))
	}
}

func TestHandleBookedAlertJobSkipsWhenAlreadySent(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_booked",
		Patient:   patient,
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	notification := models.Notification{
		Medium: models.NotificationMediumEmail,
		Name:   "appointment_booked",
		Status: models.NotificationSent,
		To:     patient.Email,
	}
	if err := svc.DB.WithContext(t.Context()).Model(&apnt).Association("Notifications").Append(&notification); err != nil {
		t.Fatalf("append booked notification: %v", err)
	}

	payload, err := json.Marshal(JobPayload{AppointmentID: apnt.UID})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	if err := svc.handleBookedAlertJob(context.Background(), jobs.Job{Payload: payload}); err != nil {
		t.Fatalf("expected idempotent skip, got %v", err)
	}
}

func TestAppendNotificationPersistsPrimaryKeyAlertableID(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_uid_persist",
		Patient:   patient,
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	svc.appendNotification(t.Context(), apnt, models.Notification{
		Medium: models.NotificationMediumEmail,
		Name:   "appointment_booked",
		Status: models.NotificationSent,
		To:     patient.Email,
	})

	notification, err := gorm.G[models.Notification](svc.DB.GormDB()).
		Where("name = ?", "appointment_booked").
		Where("`to` = ?", patient.Email).
		Order("id desc").
		First(t.Context())
	if err != nil {
		t.Fatalf("load notification: %v", err)
	}

	if notification.NotificationableType != "appointments" {
		t.Fatalf("expected notificationable_type appointments, got %q", notification.NotificationableType)
	}
	wantID := strconv.FormatUint(uint64(apnt.ID), 10)
	if notification.NotificationableID != wantID {
		t.Fatalf("expected notificationable_id %q, got %q", wantID, notification.NotificationableID)
	}
}

func TestNotificationSentAnyMediumUsesPrimaryKeyAlertableID(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_legacy_numeric",
		Patient:   patient,
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}

	legacy := models.Notification{
		Medium:               models.NotificationMediumEmail,
		Name:                 "appointment_reminder",
		Status:               models.NotificationSent,
		To:                   patient.Email,
		NotificationableID:   strconv.FormatUint(uint64(apnt.ID), 10),
		NotificationableType: "appointments",
	}
	if err := svc.DB.WithContext(t.Context()).Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy notification: %v", err)
	}

	sent, err := svc.notificationSentAnyMedium(t.Context(), apnt, "appointment_reminder")
	if err != nil {
		t.Fatalf("notificationSentAnyMedium returned error: %v", err)
	}
	if !sent {
		t.Fatalf("expected reminder to be treated as already sent")
	}
}

func dueTokens(appointments []models.Appointment) []string {
	tokens := make([]string, 0, len(appointments))
	for _, a := range appointments {
		tokens = append(tokens, a.Token)
	}
	return tokens
}

func TestFindAppointmentsDueForReminderSkipsVisitsNoLongerAwaited(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	statuses := []models.AppointmentStatus{
		models.AppointmentPending,
		models.AppointmentConfirmed,
		models.AppointmentInProgress,
		models.AppointmentDone,
		models.AppointmentCanceled,
	}
	for _, status := range statuses {
		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(time.Hour),
			Token:     "tok_" + string(status),
			Status:    status,
		}
		if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
			t.Fatalf("create %s appointment: %v", status, err)
		}
	}

	due, err := svc.findAppointmentsDueForReminder(t.Context(), time.Now())
	if err != nil {
		t.Fatalf("find due appointments: %v", err)
	}
	got := dueTokens(due)
	if len(got) != 2 || !slices.Contains(got, "tok_pending") || !slices.Contains(got, "tok_confirmed") {
		t.Fatalf("due = %v, want only tok_pending and tok_confirmed", got)
	}
}

func TestAlertJobsSendNothingForCanceledAppointments(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	patient.EnableNotifications = true
	patient.ViaEmail = true
	if err := svc.DB.GormDB().Save(&patient).Error; err != nil {
		t.Fatalf("enable patient notifications: %v", err)
	}

	runJobs := func(t *testing.T, status models.AppointmentStatus) int64 {
		t.Helper()

		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(time.Hour),
			Token:     "tok_jobs_" + string(status),
			Status:    status,
		}
		if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
			t.Fatalf("create appointment: %v", err)
		}

		payload, err := json.Marshal(JobPayload{AppointmentID: apnt.UID})
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		job := jobs.Job{Payload: payload}
		if err := svc.handleReminderJob(t.Context(), job); err != nil {
			t.Fatalf("reminder job: %v", err)
		}
		if err := svc.handleBookedAlertJob(t.Context(), job); err != nil {
			t.Fatalf("booked alert job: %v", err)
		}

		var attempts int64
		err = svc.DB.GormDB().Model(&models.Notification{}).
			Where("notificationable_type = ? AND notificationable_id = ?", apnt.AlertableType(), apnt.AlertableID()).
			Count(&attempts).Error
		if err != nil {
			t.Fatalf("count notifications: %v", err)
		}
		return attempts
	}

	if attempts := runJobs(t, models.AppointmentConfirmed); attempts == 0 {
		t.Fatalf("expected the jobs to try sending for a confirmed appointment")
	}
	if attempts := runJobs(t, models.AppointmentCanceled); attempts != 0 {
		t.Fatalf("expected no send attempts for a canceled appointment, got %d", attempts)
	}
}

func TestAppointmentsToRemindCountsOnlyRemindersSinceReschedule(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	create := func(token string) models.Appointment {
		t.Helper()
		apnt := models.Appointment{
			UserID:    user.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  time.Now().Add(time.Hour),
			Token:     token,
		}
		if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
			t.Fatalf("create %s: %v", token, err)
		}
		return apnt
	}
	remind := func(apnt models.Appointment) {
		t.Helper()
		svc.appendNotification(t.Context(), apnt, models.Notification{
			Medium: models.NotificationMediumEmail,
			Name:   "appointment_reminder",
			Status: models.NotificationSent,
			To:     patient.Email,
		})
	}

	neverMoved := create("tok_never_moved")
	remind(neverMoved)

	movedAfterReminder := create("tok_moved_after_reminder")
	remind(movedAfterReminder)
	rescheduleForTest(t, svc, movedAfterReminder)

	remindedAfterMove := create("tok_reminded_after_move")
	rescheduleForTest(t, svc, remindedAfterMove)
	remind(remindedAfterMove)

	neverReminded := create("tok_never_reminded")

	var appointments []models.Appointment
	for _, apnt := range []models.Appointment{neverMoved, movedAfterReminder, remindedAfterMove, neverReminded} {
		loaded, err := svc.TakeAppointmentByUID(t.Context(), user.ID, apnt.UID)
		if err != nil {
			t.Fatalf("load %s: %v", apnt.Token, err)
		}
		appointments = append(appointments, loaded)
	}

	unreminded, err := svc.appointmentsToRemind(t.Context(), appointments)
	if err != nil {
		t.Fatalf("appointmentsToRemind: %v", err)
	}
	got := dueTokens(unreminded)
	slices.Sort(got)
	want := []string{"tok_moved_after_reminder", "tok_never_reminded"}
	if !slices.Equal(got, want) {
		t.Fatalf("to remind = %v, want %v", got, want)
	}
}

func TestReminderJobRemindsAgainAfterReschedule(t *testing.T) {
	svc, user, clinic, patient := newAppointmentServiceForTests(t)

	patient.EnableNotifications = true
	patient.ViaEmail = true
	if err := svc.DB.GormDB().Save(&patient).Error; err != nil {
		t.Fatalf("enable patient notifications: %v", err)
	}

	apnt := models.Appointment{
		UserID:    user.ID,
		ClinicID:  clinic.ID,
		PatientID: patient.ID,
		BookedAt:  time.Now().Add(time.Hour),
		Token:     "tok_reschedule_job",
	}
	if err := svc.CreateAppointment(t.Context(), &apnt); err != nil {
		t.Fatalf("create appointment: %v", err)
	}
	svc.appendNotification(t.Context(), apnt, models.Notification{
		Medium: models.NotificationMediumEmail,
		Name:   "appointment_reminder",
		Status: models.NotificationSent,
		To:     patient.Email,
	})

	payload, err := json.Marshal(JobPayload{AppointmentID: apnt.UID})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	job := jobs.Job{Payload: payload}
	reminderAttempts := func() int64 {
		t.Helper()
		var n int64
		err := svc.DB.GormDB().Model(&models.Notification{}).
			Where("notificationable_type = ? AND notificationable_id = ?", apnt.AlertableType(), apnt.AlertableID()).
			Where("name = ?", "appointment_reminder").
			Count(&n).Error
		if err != nil {
			t.Fatalf("count reminders: %v", err)
		}
		return n
	}

	if err := svc.handleReminderJob(t.Context(), job); err != nil {
		t.Fatalf("reminder job: %v", err)
	}
	if n := reminderAttempts(); n != 1 {
		t.Fatalf("expected no new reminder before a reschedule, got %d reminders", n)
	}

	rescheduleForTest(t, svc, apnt)
	if err := svc.handleReminderJob(t.Context(), job); err != nil {
		t.Fatalf("reminder job after reschedule: %v", err)
	}
	if n := reminderAttempts(); n < 2 {
		t.Fatalf("expected a new reminder attempt after the reschedule, got %d reminders", n)
	}
}

// rescheduleForTest moves apnt 30 minutes later through the update service,
// stamping RescheduledAt as HandleUpdate does.
func rescheduleForTest(t *testing.T, svc *service, apnt models.Appointment) {
	t.Helper()

	bookedAt := apnt.BookedAt.Add(30 * time.Minute).UTC()
	updates := appointmentPatchUpdates{
		BookedAt:      bookedAt,
		RescheduledAt: time.Now().UTC(),
		BookedYear:    bookedAt.Year(),
		BookedMonth:   int(bookedAt.Month()),
		BookedDay:     bookedAt.Day(),
		BookedHour:    bookedAt.Hour(),
		BookedMinute:  bookedAt.Minute(),
		ClinicID:      apnt.ClinicID,
		PatientID:     apnt.PatientID,
	}
	if err := svc.UpdateAppointmentByID(t.Context(), apnt.UserID, apnt.UID, updates); err != nil {
		t.Fatalf("reschedule %s: %v", apnt.Token, err)
	}
}
