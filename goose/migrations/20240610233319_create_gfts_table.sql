-- +goose Up
-- +goose StatementBegin
CREATE VIRTUAL TABLE IF NOT EXISTS global_fts USING fts5(gid, primary, secondary, tertiary);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIRTUAL TABLE IF EXISTS global_fts
-- +goose StatementEnd
