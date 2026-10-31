-- +goose Up
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trgr_insert_clinics_on_gfts;
DROP TRIGGER IF EXISTS trgr_update_clinics_on_gfts;
DROP TRIGGER IF EXISTS trgr_delete_clinics_on_gfts;
DROP TRIGGER IF EXISTS trgr_insert_patients_on_gfts;
DROP TRIGGER IF EXISTS trgr_update_patients_on_gfts;
DROP TRIGGER IF EXISTS trgr_delete_patients_on_gfts;
DROP TABLE IF EXISTS global_fts;

CREATE TABLE global_search_docs (
  id INTEGER PRIMARY KEY,
  gid TEXT NOT NULL UNIQUE,
  kind TEXT NOT NULL,
  "primary" TEXT NOT NULL DEFAULT '',
  "secondary" TEXT NOT NULL DEFAULT '',
  "tertiary" TEXT NOT NULL DEFAULT ''
);

CREATE VIRTUAL TABLE global_fts USING fts5(
  "primary", "secondary", "tertiary",
  content='global_search_docs', content_rowid='id'
);

CREATE TRIGGER trgr_insert_global_search_docs_on_gfts
  AFTER INSERT ON global_search_docs
BEGIN
  INSERT INTO global_fts (rowid, "primary", "secondary", "tertiary")
  VALUES (NEW.id, NEW."primary", NEW."secondary", NEW."tertiary");
END;

CREATE TRIGGER trgr_update_global_search_docs_on_gfts
  AFTER UPDATE ON global_search_docs
BEGIN
  INSERT INTO global_fts (global_fts, rowid, "primary", "secondary", "tertiary")
  VALUES ('delete', OLD.id, OLD."primary", OLD."secondary", OLD."tertiary");
  INSERT INTO global_fts (rowid, "primary", "secondary", "tertiary")
  VALUES (NEW.id, NEW."primary", NEW."secondary", NEW."tertiary");
END;

CREATE TRIGGER trgr_delete_global_search_docs_on_gfts
  AFTER DELETE ON global_search_docs
BEGIN
  INSERT INTO global_fts (global_fts, rowid, "primary", "secondary", "tertiary")
  VALUES ('delete', OLD.id, OLD."primary", OLD."secondary", OLD."tertiary");
END;

INSERT INTO global_search_docs (gid, kind, "primary", "secondary", "tertiary")
SELECT
    uid,
    'clinics',
    name,
    phone || ' ' || ifnull(email, ''),
    ifnull(ext_id, '')
FROM
    clinics;

CREATE TRIGGER trgr_insert_clinics_on_gfts
  AFTER INSERT ON clinics
BEGIN
  INSERT INTO global_search_docs (gid, kind, "primary", "secondary", "tertiary")
  VALUES (
      NEW.uid,
      'clinics',
      NEW.name,
      NEW.phone || ' ' || ifnull(NEW.email, ''),
      ifnull(NEW.ext_id, '')
  );
END;

CREATE TRIGGER trgr_update_clinics_on_gfts
  AFTER UPDATE OF uid, name, phone, email, ext_id ON clinics
BEGIN
  UPDATE global_search_docs
  SET
      gid = NEW.uid,
      "primary" = NEW.name,
      "secondary" = NEW.phone || ' ' || ifnull(NEW.email, ''),
      "tertiary" = ifnull(NEW.ext_id, '')
  WHERE gid = OLD.uid;
END;

CREATE TRIGGER trgr_delete_clinics_on_gfts
  AFTER DELETE ON clinics
BEGIN
  DELETE FROM global_search_docs
  WHERE gid = OLD.uid;
END;

INSERT INTO global_search_docs (gid, kind, "primary", "secondary", "tertiary")
SELECT
    uid,
    'patients',
    name,
    ifnull(email, '') || ' ' || phone,
    ifnull(ocupation, '') || ' ' || ifnull(family_history, '') || ' ' || ifnull(medical_background, '') || ' ' || ifnull(notes, '')
FROM
    patients;

CREATE TRIGGER trgr_insert_patients_on_gfts
  AFTER INSERT ON patients
BEGIN
  INSERT INTO global_search_docs (gid, kind, "primary", "secondary", "tertiary")
  VALUES (
      NEW.uid,
      'patients',
      NEW.name,
      ifnull(NEW.email, '') || ' ' || NEW.phone,
      ifnull(NEW.ocupation, '') || ' ' || ifnull(NEW.family_history, '') || ' ' || ifnull(NEW.medical_background, '') || ' ' || ifnull(NEW.notes, '')
  );
END;

CREATE TRIGGER trgr_update_patients_on_gfts
  AFTER UPDATE OF uid, name, email, phone, ocupation, family_history, medical_background, notes ON patients
BEGIN
  UPDATE global_search_docs
  SET
      gid = NEW.uid,
      "primary" = NEW.name,
      "secondary" = ifnull(NEW.email, '') || ' ' || NEW.phone,
      "tertiary" = ifnull(NEW.ocupation, '') || ' ' || ifnull(NEW.family_history, '') || ' ' || ifnull(NEW.medical_background, '') || ' ' || ifnull(NEW.notes, '')
  WHERE gid = OLD.uid;
END;

CREATE TRIGGER trgr_delete_patients_on_gfts
  AFTER DELETE ON patients
BEGIN
  DELETE FROM global_search_docs
  WHERE gid = OLD.uid;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trgr_insert_clinics_on_gfts;
DROP TRIGGER IF EXISTS trgr_update_clinics_on_gfts;
DROP TRIGGER IF EXISTS trgr_delete_clinics_on_gfts;
DROP TRIGGER IF EXISTS trgr_insert_patients_on_gfts;
DROP TRIGGER IF EXISTS trgr_update_patients_on_gfts;
DROP TRIGGER IF EXISTS trgr_delete_patients_on_gfts;
DROP TRIGGER IF EXISTS trgr_insert_global_search_docs_on_gfts;
DROP TRIGGER IF EXISTS trgr_update_global_search_docs_on_gfts;
DROP TRIGGER IF EXISTS trgr_delete_global_search_docs_on_gfts;
DROP TABLE IF EXISTS global_fts;
DROP TABLE IF EXISTS global_search_docs;

CREATE VIRTUAL TABLE IF NOT EXISTS global_fts USING fts5(gid, primary, secondary, tertiary);

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

INSERT INTO global_fts (gid, "primary", "secondary", "tertiary")
SELECT
    id,
    name,
    email || ' ' || phone,
    ocupation || '\n' || family_history || '\n' || medical_background || '\n' || notes
FROM
    patients;

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

CREATE TRIGGER IF NOT EXISTS trgr_delete_patients_on_gfts
  AFTER DELETE on patients
BEGIN
  DELETE FROM global_fts
  WHERE gid = OLD.id;
END;
-- +goose StatementEnd
