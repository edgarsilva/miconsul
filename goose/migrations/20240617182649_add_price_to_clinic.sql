-- +goose Up
-- +goose StatementBegin
ALTER TABLE clinics ADD COLUMN price integer;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE clinics DROP COLUMN price;
-- +goose StatementEnd
-
