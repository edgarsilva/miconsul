-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS `idx_users_ext_id` ON `users`(`ext_id`);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'no down';
-- +goose StatementEnd
