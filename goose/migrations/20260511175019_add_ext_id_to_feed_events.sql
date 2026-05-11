-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_events ADD COLUMN ext_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_feed_events_ext_id ON feed_events(ext_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_feed_events_ext_id;
ALTER TABLE feed_events DROP COLUMN ext_id;
-- +goose StatementEnd
