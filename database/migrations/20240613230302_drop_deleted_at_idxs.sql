-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS `idx_clinics_deleted_at`;
DROP INDEX IF EXISTS `idx_patients_deleted_at`;
DROP INDEX IF EXISTS `idx_appointments_deleted_at`;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'No down for this one';
-- +goose StatementEnd
