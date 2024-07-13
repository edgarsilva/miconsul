-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS `idx_clinics_id_created_at_desc`;
DROP INDEX IF EXISTS `idx_patients_id_created_at_desc`;
CREATE INDEX IF NOT EXISTS `idx_patients_user_id_created_at` ON `patients`(`user_id`, `created_at` DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS `idx_patients_user_id_created_at`;
-- +goose StatementEnd
