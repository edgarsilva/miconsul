-- +goose Up
-- +goose StatementBegin
INSERT INTO global_search_docs (gid, kind, "primary", "secondary", "tertiary")
SELECT
    uid,
    'users',
    ifnull(name, ''),
    email,
    ''
FROM
    users;

CREATE TRIGGER trgr_insert_users_on_gfts
  AFTER INSERT ON users
BEGIN
  INSERT INTO global_search_docs (gid, kind, "primary", "secondary", "tertiary")
  VALUES (
      NEW.uid,
      'users',
      ifnull(NEW.name, ''),
      NEW.email,
      ''
  );
END;

CREATE TRIGGER trgr_update_users_on_gfts
  AFTER UPDATE OF uid, name, email ON users
BEGIN
  UPDATE global_search_docs
  SET
      gid = NEW.uid,
      "primary" = ifnull(NEW.name, ''),
      "secondary" = NEW.email
  WHERE gid = OLD.uid;
END;

CREATE TRIGGER trgr_delete_users_on_gfts
  AFTER DELETE ON users
BEGIN
  DELETE FROM global_search_docs
  WHERE gid = OLD.uid;
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trgr_insert_users_on_gfts;
DROP TRIGGER IF EXISTS trgr_update_users_on_gfts;
DROP TRIGGER IF EXISTS trgr_delete_users_on_gfts;
DELETE FROM global_search_docs WHERE kind = 'users';
-- +goose StatementEnd
