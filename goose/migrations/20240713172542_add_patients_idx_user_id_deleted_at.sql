-- +goose Up
-- +goose StatementBegin
CREATE INDEX patients_idx_user_id_deleted_at ON patients(user_id, deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX patients_idx_user_id_deleted_at;
-- +goose StatementEnd
