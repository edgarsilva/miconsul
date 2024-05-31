
CREATE VIRTUAL TABLE global_fts USING fts5(gid, primary, secondary, tertiary);

INSERT INTO global_fts (gid, "primary", "secondary", "tertiary")
SELECT
    id,
    email || ' ' || phone as "primary",
    first_name || ' ' || last_name as "secondary",
    ocupation || ' ' || age || ' ' || family_history || ' ' || medical_background || ' ' || notes as "tertiary"
FROM
    patients;


create trigger insert_patients_into_gfts
  after insert on patients
begin
  insert into global_fts (gid, "primary", "secondary", "tertiary")
  values (
      new.id,
      new.email || ' ' || new.phone,
      new.first_name || ' ' || new.last_name,
      new.ocupation || '\n' || new.age || '\n' || new.family_history || '\n' || new.medical_background || '\n' || new.notes
  );
end;

create trigger update_patients_in_gfts
  after update on patients
begin
  UPDATE global_fts
  SET
      "primary" = new.email || ' ' || new.phone,
      "secondary" = new.first_name || ' ' || new.last_name,
      "tertiary" = new.ocupation || '\n' || new.age || '\n' || new.family_history || '\n' || new.medical_background || '\n' || new.notes
  WHERE gid = NEW.id;
end

create trigger delete_patients_from_gfts
  after delete on patients
begin
  DELETE FROM global_fts
  WHERE gid = OLD.id;
end

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
