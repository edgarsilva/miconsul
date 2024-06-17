-- +goose Up
-- +goose StatementBegin
ALTER TABLE appointments ADD COLUMN price integer;
UPDATE appointments
SET price = (
    SELECT cost
    FROM appointments a2
);
ALTER TABLE appointments DROP COLUMN cost;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'theres no down for this operation';
-- +goose StatementEnd
--
