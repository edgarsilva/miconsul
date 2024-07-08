-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN timezone text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN timezone;
-- +goose StatementEnd
