-- +goose Up
ALTER TABLE appointments ADD COLUMN via_sms NUMERIC;
ALTER TABLE patients ADD COLUMN via_sms NUMERIC;

-- +goose Down
