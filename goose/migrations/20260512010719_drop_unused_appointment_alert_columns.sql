-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_appointments_old_booked_at;
ALTER TABLE appointments DROP COLUMN old_booked_at;
ALTER TABLE appointments DROP COLUMN booked_alert_sent_at;
ALTER TABLE appointments DROP COLUMN reminder_alert_sent_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE appointments ADD COLUMN old_booked_at DATETIME;
ALTER TABLE appointments ADD COLUMN booked_alert_sent_at DATETIME;
ALTER TABLE appointments ADD COLUMN reminder_alert_sent_at DATETIME;
CREATE INDEX IF NOT EXISTS idx_appointments_old_booked_at ON appointments(old_booked_at);
-- +goose StatementEnd
