-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS clinics_idx_user_id_favorite_created_at ON clinics(user_id, favorite, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS clinics_idx_user_id_favorite_created_at;
-- +goose StatementEnd
