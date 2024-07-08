-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN phone string;
CREATE INDEX IF NOT EXISTS `idx_users_ext_id` ON `users`(`ext_id`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN phone;
DROP INDEX IF EXISTS `idx_users_ext_id` ON `users`(`ext_id`);
-- +goose StatementEnd
