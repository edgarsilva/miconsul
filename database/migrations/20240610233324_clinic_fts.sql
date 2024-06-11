-- +goose Up
-- +goose StatementBegin
INSERT INTO global_fts (gid, "primary", "secondary", "tertiary")
SELECT
    id,
    name as "primary",
    phone || ' ' || email as "secondary",
    ext_id as "tertiary"
FROM
    clinics;

CREATE TRIGGER IF NOT EXISTS trgr_insert_clinics_on_gfts
  AFTER INSERT on clinics
BEGIN
  INSERT INTO global_fts (gid, "primary", "secondary", "tertiary")
  VALUES (
      new.id,
      new.name,
      new.phone || ' ' || new.email,
      new.ext_id
  );
END;

CREATE TRIGGER IF NOT EXISTS trgr_update_clinics_on_gfts
  AFTER UPDATE on clinics
BEGIN
  UPDATE global_fts
  SET
      "primary" = new.name,
      "secondary" = new.phone || ' ' || new.email,
      "tertiary" = new.ext_id
  WHERE gid = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trgr_delete_clinics_on_gfts
  AFTER DELETE on clinics
BEGIN
  DELETE FROM global_fts
  WHERE gid = OLD.id;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trgr_insert_clinics_on_gfts;

DROP TRIGGER IF EXISTS trgr_update_clinics_on_gfts;

DROP TRIGGER IF EXISTS trgr_delete_clinics_on_gfts;
-- +goose StatementEnd
