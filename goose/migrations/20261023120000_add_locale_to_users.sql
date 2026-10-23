-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN locale TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN locale;
-- +goose StatementEnd
