-- +goose Up
-- +goose StatementBegin
ALTER TABLE feed_events ADD COLUMN actor TEXT;
ALTER TABLE feed_events ADD COLUMN actor_id TEXT;
ALTER TABLE feed_events ADD COLUMN actor_url TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE feed_events DROP COLUMN actor;
ALTER TABLE feed_events DROP COLUMN actor_id;
ALTER TABLE feed_events DROP COLUMN actor_url;
-- +goose StatementEnd
