-- +goose Up
-- +goose StatementBegin
ALTER TABLE appointments ADD COLUMN timezone text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE appointments DROP COLUMN timezone;
-- +goose StatementEnd
