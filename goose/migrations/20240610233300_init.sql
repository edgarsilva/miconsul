-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
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
);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_ext_id ON users(ext_id);

CREATE TABLE IF NOT EXISTS clinics (
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
CREATE INDEX IF NOT EXISTS idx_clinics_user_id ON clinics(user_id);

CREATE TABLE IF NOT EXISTS patients (
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
  updated_at DATETIME,
  FOREIGN KEY(user_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_patients_user_id ON patients(user_id);

CREATE TABLE IF NOT EXISTS appointments (
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
  old_booked_at DATETIME,
  booked_alert_sent_at DATETIME,
  reminder_alert_sent_at DATETIME,
  viewed_at DATETIME,
  confirmed_at DATETIME,
  done_at DATETIME,
  canceled_at DATETIME,
  pending_at DATETIME,
  rescheduled_at DATETIME,
  deleted_at DATETIME,
  created_at DATETIME,
  updated_at DATETIME,
  FOREIGN KEY(user_id) REFERENCES users(id),
  FOREIGN KEY(clinic_id) REFERENCES clinics(id),
  FOREIGN KEY(patient_id) REFERENCES patients(id)
);
CREATE INDEX IF NOT EXISTS idx_appointments_user_id ON appointments(user_id);
CREATE INDEX IF NOT EXISTS idx_appointments_status ON appointments(status);
CREATE INDEX IF NOT EXISTS idx_appointments_clinic_id ON appointments(clinic_id);
CREATE INDEX IF NOT EXISTS idx_appointments_patient_id ON appointments(patient_id);
CREATE INDEX IF NOT EXISTS idx_appointments_booked_at ON appointments(booked_at);
CREATE INDEX IF NOT EXISTS idx_appointments_old_booked_at ON appointments(old_booked_at);

CREATE TABLE IF NOT EXISTS alerts (
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
  alertable_id TEXT,
  alertable_type TEXT,
  created_at DATETIME,
  updated_at DATETIME
);
CREATE INDEX IF NOT EXISTS poly_fevnt_idx ON alerts(alertable_id, alertable_type);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_name ON alerts(name);
CREATE INDEX IF NOT EXISTS idx_alerts_medium ON alerts(medium);

CREATE TABLE IF NOT EXISTS feed_events (
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
);
CREATE INDEX IF NOT EXISTS fe_target_idx ON feed_events(target, target_id);
CREATE INDEX IF NOT EXISTS idx_feed_events_ocurred_at ON feed_events(ocurred_at);
CREATE INDEX IF NOT EXISTS fe_poly_idx ON feed_events(feed_eventable_id, feed_eventable_type);
CREATE INDEX IF NOT EXISTS idx_feed_events_name ON feed_events(name);
CREATE INDEX IF NOT EXISTS fe_subject_idx ON feed_events(subject_id, subject_type);
CREATE INDEX IF NOT EXISTS idx_feed_events_action ON feed_events(action);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'No goose down action for initial migration.';
-- +goose StatementEnd
