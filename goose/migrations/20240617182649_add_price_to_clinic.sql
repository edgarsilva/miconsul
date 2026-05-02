-- +goose Up
-- +goose StatementBegin
SELECT 'price is part of baseline schema';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'no down';
-- +goose StatementEnd
-
