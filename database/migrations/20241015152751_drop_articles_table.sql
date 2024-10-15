-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS articles;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'NO RECOVER FROM DROP TABLE';
-- +goose StatementEnd
