package appointment

const (
	TaskBookedAlert   = "appointment:booked_alert"
	TaskReminderAlert = "appointment:reminder_alert"
	TaskReminderSweep = "appointment:reminder_sweep"
)

type TaskAppointmentPayload struct {
	AppointmentID string `json:"appointment_id"`
}

type TaskReminderSweepPayload struct{}
