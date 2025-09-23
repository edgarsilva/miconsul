-- +goose Up
-- +goose StatementBegin
INSERT INTO global_fts (gid, "primary", "secondary", "tertiary")
SELECT
    id,
    email || ' ' || phone as "primary",
    first_name || ' ' || last_name as "secondary",
    ocupation || ' ' || age || ' ' || family_history || ' ' || medical_background || ' ' || notes as "tertiary"
FROM
    patients;

CREATE TRIGGER IF NOT EXISTS trgr_insert_patients_on_gfts
  AFTER INSERT on patients
BEGIN
  INSERT INTO global_fts (gid, "primary", "secondary", "tertiary")
  VALUES (
      new.id,
      new.email || ' ' || new.phone,
      new.first_name || ' ' || new.last_name,
      new.ocupation || '\n' || new.age || '\n' || new.family_history || '\n' || new.medical_background || '\n' || new.notes
  );
END;

CREATE TRIGGER IF NOT EXISTS trgr_update_patients_on_gfts
  AFTER UPDATE on patients
BEGIN
  UPDATE global_fts
  SET
      "primary" = new.email || ' ' || new.phone,
      "secondary" = new.first_name || ' ' || new.last_name,
      "tertiary" = new.ocupation || '\n' || new.age || '\n' || new.family_history || '\n' || new.medical_background || '\n' || new.notes
  WHERE gid = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trgr_delete_patients_on_gfts
  AFTER DELETE on patients
BEGIN
  DELETE FROM global_fts
  WHERE gid = OLD.id;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trgr_insert_patients_on_gfts;

DROP TRIGGER IF EXISTS trgr_update_patients_on_gfts;

DROP TRIGGER IF EXISTS trgr_delete_patients_on_gfts;
-- +goose StatementEnd


