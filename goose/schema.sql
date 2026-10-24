CREATE TABLE goose_db_version (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER NOT NULL,
		is_applied INTEGER NOT NULL,
		tstamp TIMESTAMP DEFAULT (datetime('now'))
	);
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  ext_id TEXT,
  profile_pic TEXT,
  name TEXT,
  email TEXT NOT NULL UNIQUE,
  password TEXT,
  theme TEXT,
  reset_token TEXT,
  confirm_email_token TEXT,
  confirm_email_expires_at DATETIME,
  reset_token_expires_at DATETIME,
  phone TEXT,
  timezone TEXT,
  role TEXT NOT NULL,
  created_at DATETIME,
  updated_at DATETIME
, locale TEXT);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_ext_id ON users(ext_id);
CREATE TABLE clinics (
  id INTEGER PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  ext_id TEXT,
  user_id INTEGER NOT NULL,
  cover_pic TEXT,
  profile_pic TEXT,
  name TEXT NOT NULL,
  email TEXT,
  phone TEXT NOT NULL,
  line1 TEXT,
  line2 TEXT,
  city TEXT,
  state TEXT,
  country TEXT,
  zip TEXT,
  whatsapp TEXT,
  telegram TEXT,
  messenger TEXT,
  instagram TEXT,
  facebook TEXT,
  favorite NUMERIC,
  price INTEGER,
  deleted_at DATETIME,
  created_at DATETIME,
  updated_at DATETIME,
  FOREIGN KEY(user_id) REFERENCES users(id)
);
CREATE INDEX idx_clinics_user_id ON clinics(user_id);
CREATE TABLE patients (
  id INTEGER PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  ext_id TEXT,
  user_id INTEGER NOT NULL,
  email TEXT,
  phone TEXT NOT NULL,
  ocupation TEXT,
  name TEXT NOT NULL,
  profile_pic TEXT,
  family_history TEXT,
  medical_background TEXT,
  notes TEXT,
  line1 TEXT,
  line2 TEXT,
  city TEXT,
  state TEXT,
  country TEXT,
  zip TEXT,
  whatsapp TEXT,
  telegram TEXT,
  messenger TEXT,
  instagram TEXT,
  facebook TEXT,
  age INTEGER,
  enable_notifications NUMERIC,
  via_email NUMERIC,
  via_whatsapp NUMERIC,
  via_messenger NUMERIC,
  via_telegram NUMERIC,
  deleted_at DATETIME,
  created_at DATETIME,
  updated_at DATETIME, via_sms NUMERIC,
  FOREIGN KEY(user_id) REFERENCES users(id)
);
CREATE INDEX idx_patients_user_id ON patients(user_id);
CREATE TABLE appointments (
  id INTEGER PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  ext_id TEXT,
  token TEXT,
  summary TEXT,
  observations TEXT,
  conclusions TEXT,
  notes TEXT,
  hashtags TEXT,
  timezone TEXT,
  user_id INTEGER NOT NULL,
  clinic_id INTEGER NOT NULL,
  patient_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  duration INTEGER,
  price INTEGER,
  booked_year INTEGER,
  booked_month INTEGER,
  booked_day INTEGER,
  booked_hour INTEGER,
  booked_minute INTEGER,
  no_show NUMERIC,
  enable_notifications NUMERIC,
  via_email NUMERIC,
  via_whatsapp NUMERIC,
  via_messenger NUMERIC,
  via_telegram NUMERIC,
  booked_at DATETIME NOT NULL,
  viewed_at DATETIME,
  confirmed_at DATETIME,
  done_at DATETIME,
  canceled_at DATETIME,
  pending_at DATETIME,
  rescheduled_at DATETIME,
  deleted_at DATETIME,
  created_at DATETIME,
  updated_at DATETIME, via_sms NUMERIC,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(clinic_id) REFERENCES clinics(id),
  FOREIGN KEY(patient_id) REFERENCES patients(id)
);
CREATE INDEX idx_appointments_user_id ON appointments(user_id);
CREATE INDEX idx_appointments_status ON appointments(status);
CREATE INDEX idx_appointments_clinic_id ON appointments(clinic_id);
CREATE INDEX idx_appointments_patient_id ON appointments(patient_id);
CREATE INDEX idx_appointments_booked_at ON appointments(booked_at);
CREATE TABLE "notifications" (
  id INTEGER PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  medium TEXT NOT NULL,
  name TEXT NOT NULL,
  title TEXT,
  sub TEXT,
  message TEXT,
  "from" TEXT,
  "to" TEXT,
  status TEXT NOT NULL DEFAULT 'pending',
  notificationable_id TEXT,
  notificationable_type TEXT,
  created_at DATETIME,
  updated_at DATETIME
);
CREATE INDEX poly_fevnt_idx ON "notifications"(notificationable_id, notificationable_type);
CREATE INDEX idx_alerts_status ON "notifications"(status);
CREATE INDEX idx_alerts_name ON "notifications"(name);
CREATE INDEX idx_alerts_medium ON "notifications"(medium);
CREATE TABLE feed_events (
  id INTEGER PRIMARY KEY,
  uid TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  subject TEXT,
  subject_id TEXT NOT NULL,
  subject_type TEXT NOT NULL,
  subject_url TEXT,
  action TEXT,
  target TEXT NOT NULL,
  target_id TEXT NOT NULL,
  target_type TEXT,
  target_url TEXT,
  ocurred_at DATETIME,
  extra1 TEXT,
  extra2 TEXT,
  extra3 TEXT,
  feed_eventable_id TEXT,
  feed_eventable_type TEXT,
  created_at DATETIME,
  updated_at DATETIME
, ext_id TEXT NOT NULL DEFAULT '', actor TEXT, actor_id TEXT, actor_url TEXT);
CREATE INDEX fe_target_idx ON feed_events(target, target_id);
CREATE INDEX idx_feed_events_ocurred_at ON feed_events(ocurred_at);
CREATE INDEX fe_poly_idx ON feed_events(feed_eventable_id, feed_eventable_type);
CREATE INDEX idx_feed_events_name ON feed_events(name);
CREATE INDEX fe_subject_idx ON feed_events(subject_id, subject_type);
CREATE INDEX idx_feed_events_action ON feed_events(action);
CREATE VIRTUAL TABLE global_fts USING fts5(gid, primary, secondary, tertiary);
/* global_fts(gid,"primary",secondary,tertiary) */
CREATE TABLE 'global_fts_data'(id INTEGER PRIMARY KEY, block BLOB);
CREATE TABLE 'global_fts_idx'(segid, term, pgno, PRIMARY KEY(segid, term)) WITHOUT ROWID;
CREATE TABLE 'global_fts_content'(id INTEGER PRIMARY KEY, c0, c1, c2, c3);
CREATE TABLE 'global_fts_docsize'(id INTEGER PRIMARY KEY, sz BLOB);
CREATE TABLE 'global_fts_config'(k PRIMARY KEY, v) WITHOUT ROWID;
CREATE TRIGGER trgr_insert_clinics_on_gfts
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
CREATE TRIGGER trgr_update_clinics_on_gfts
  AFTER UPDATE on clinics
BEGIN
  UPDATE global_fts
  SET
      "primary" = new.name,
      "secondary" = new.phone || ' ' || new.email,
      "tertiary" = new.ext_id
  WHERE gid = NEW.id;
END;
CREATE TRIGGER trgr_delete_clinics_on_gfts
  AFTER DELETE on clinics
BEGIN
  DELETE FROM global_fts
  WHERE gid = OLD.id;
END;
CREATE TRIGGER trgr_insert_patients_on_gfts
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
CREATE TRIGGER trgr_update_patients_on_gfts
  AFTER UPDATE on patients
BEGIN
  UPDATE global_fts
  SET
      "primary" = new.name,
      "secondary" = new.email || ' ' || new.phone,
      "tertiary" = new.ocupation || '\n' || new.family_history || '\n' || new.medical_background || '\n' || new.notes
  WHERE gid = NEW.id;
END;
CREATE TRIGGER trgr_delete_patients_on_gfts
  AFTER DELETE on patients
BEGIN
  DELETE FROM global_fts
  WHERE gid = OLD.id;
END;
CREATE INDEX `idx_patients_user_id_created_at` ON `patients`(`user_id`, `created_at` DESC);
CREATE INDEX clinics_idx_user_id_favorite_created_at ON clinics(user_id, favorite, created_at);
CREATE INDEX patients_idx_user_id_deleted_at ON patients(user_id, deleted_at);
CREATE INDEX idx_feed_events_ext_id ON feed_events(ext_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_name ON notifications(name);
CREATE INDEX idx_notifications_medium ON notifications(medium);
CREATE UNIQUE INDEX idx_appointments_token ON appointments(token) WHERE token IS NOT NULL AND token != '';
