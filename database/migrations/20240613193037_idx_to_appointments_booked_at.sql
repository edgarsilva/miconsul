-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS `idx_appointments_booked_at` ON `appointments`(`booked_at`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS `idx_appointments_booked_at` ON `appointments`(`booked_at`);
-- +goose StatementEnd
