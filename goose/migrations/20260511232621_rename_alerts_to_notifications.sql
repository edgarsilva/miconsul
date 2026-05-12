-- +goose Up
-- +goose StatementBegin
ALTER TABLE alerts RENAME TO notifications;
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_name ON notifications(name);
CREATE INDEX IF NOT EXISTS idx_notifications_medium ON notifications(medium);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notifications RENAME TO alerts;
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_name ON alerts(name);
CREATE INDEX IF NOT EXISTS idx_alerts_medium ON alerts(medium);
-- +goose StatementEnd
