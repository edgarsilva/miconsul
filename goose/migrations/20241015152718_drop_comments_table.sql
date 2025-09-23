-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS comments;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'NO RECOVER FROM DROP TABLE';
-- +goose StatementEnd
