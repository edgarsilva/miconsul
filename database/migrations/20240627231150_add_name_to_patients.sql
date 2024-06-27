-- +goose Up
-- +goose StatementBegin
ALTER TABLE patients ADD COLUMN name TEXT;

UPDATE patients
SET name = (
    SELECT first_name || ' ' || last_name
    FROM patients p2
);

DROP TRIGGER IF EXISTS trgr_insert_patients_on_gfts;
DROP TRIGGER IF EXISTS trgr_update_patients_on_gfts;

ALTER TABLE patients DROP COLUMN first_name;
ALTER TABLE patients DROP COLUMN last_name;

CREATE TRIGGER IF NOT EXISTS trgr_insert_patients_on_gfts
  AFTER INSERT on patients
BEGIN
  INSERT INTO global_fts (gid, "primary", "secondary", "tertiary")
  VALUES (
      new.id,
      new.name,
      new.email || ' ' || new.phone,
      new.ocupation || '\n' || new.family_history || '\n' || new.medical_background || '\n' || new.notes
  );
END;

CREATE TRIGGER IF NOT EXISTS trgr_update_patients_on_gfts
  AFTER UPDATE on patients
BEGIN
  UPDATE global_fts
  SET
      "primary" = new.name,
      "secondary" = new.email || ' ' || new.phone,
      "tertiary" = new.ocupation || '\n' || new.family_history || '\n' || new.medical_background || '\n' || new.notes
  WHERE gid = NEW.id;
END;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'No down migration';
-- +goose StatementEnd
