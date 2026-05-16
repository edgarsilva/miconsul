-- +goose Up
ALTER TABLE notifications RENAME COLUMN alertable_id TO notificationable_id;
ALTER TABLE notifications RENAME COLUMN alertable_type TO notificationable_type;

-- +goose Down
ALTER TABLE notifications RENAME COLUMN notificationable_id TO alertable_id;
ALTER TABLE notifications RENAME COLUMN notificationable_type TO alertable_type;
