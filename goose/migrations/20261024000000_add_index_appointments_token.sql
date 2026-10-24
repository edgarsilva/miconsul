-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS idx_appointments_token ON appointments(token) WHERE token IS NOT NULL AND token != '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_appointments_token;
-- +goose StatementEnd
