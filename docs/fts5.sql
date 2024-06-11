CREATE VIRTUAL TABLE global_fts USING fts5(gid, primary, secondary, tertiary);

INSERT INTO global_fts (gid, "primary", "secondary", "tertiary")
SELECT
    id,
    email || ' ' || phone as "primary",
    first_name || ' ' || last_name as "secondary",
    ocupation || ' ' || age || ' ' || family_history || ' ' || medical_background || ' ' || notes as "tertiary"
FROM
    patients;

CREATE TRIGGER trgr_insert_patients_on_gfts
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

CREATE TRIGGER trgr_update_patients_on_gfts
  AFTER UPDATE on patients
BEGIN
  UPDATE global_fts
  SET
      "primary" = new.email || ' ' || new.phone,
      "secondary" = new.first_name || ' ' || new.last_name,
      "tertiary" = new.ocupation || '\n' || new.age || '\n' || new.family_history || '\n' || new.medical_background || '\n' || new.notes
  WHERE gid = NEW.id;
END

CREATE TRIGGER trgr_delete_patients_from_gfts
  AFTER DELETE on patients
BEGIN
  DELETE FROM global_fts
  WHERE gid = OLD.id;
END

-- db.Model(&User{}).Select("users.name, emails.email").Joins("left join emails on emails.user_id = users.id").Scan(&result{})
SELECT patients.*
FROM patients
INNER JOIN global_fts ON gid = id
WHERE
  global_fts MATCH '{primary secondary tertiary}: ed'
ORDER BY bm25(global_fts, 0, 1, 2, 3);

SELECT *
FROM `patients`
INNER JOIN global_fts ON gid = id
WHERE global_fts MATCH 'edg'
AND `patients`.`deleted_at` IS NULL LIMIT 5;
