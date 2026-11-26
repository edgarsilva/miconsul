-- +goose Up
ALTER TABLE notifications RENAME COLUMN notificationable_id TO notifiable_id;
ALTER TABLE notifications RENAME COLUMN notificationable_type TO notifiable_type;

-- +goose Down
ALTER TABLE notifications RENAME COLUMN notifiable_id TO notificationable_id;
ALTER TABLE notifications RENAME COLUMN notifiable_type TO notificationable_type;
