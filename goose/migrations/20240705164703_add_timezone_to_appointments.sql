-- +goose Up
-- +goose StatementBegin
SELECT 'timezone is part of baseline schema';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'no down';
-- +goose StatementEnd
