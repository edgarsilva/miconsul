CREATE TABLE `todos` (`created_at` datetime,`content` text NOT NULL DEFAULT null,`user_id` text NOT NULL DEFAULT null,`updated_at` datetime,`id` text NOT NULL DEFAULT null,`completed` numeric,PRIMARY KEY (`id`),CONSTRAINT `fk_users_todos` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`));
CREATE INDEX `idx_todos_user_id` ON `todos`(`user_id`);
CREATE INDEX `idx_todos_created_at` ON `todos`(`created_at` desc);

CREATE TABLE IF NOT EXISTS "articles"  (`created_at` datetime,`updated_at` datetime,`user_id` text NOT NULL DEFAULT null,`title` text,`content` text,`id` text NOT NULL DEFAULT null,PRIMARY KEY (`id`),CONSTRAINT `fk_users_articles` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`),CONSTRAINT `fk_articles_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`));
CREATE INDEX `idx_articles_created_at` ON `articles`(`created_at` desc);
CREATE INDEX `idx_articles_user_id` ON `articles`(`user_id`);

CREATE TABLE IF NOT EXISTS "comments"  (`user_id` text NOT NULL DEFAULT null,`article_id` text NOT NULL DEFAULT null,`content` text,`created_at` datetime,`updated_at` datetime,`id` text NOT NULL DEFAULT null,PRIMARY KEY (`id`),CONSTRAINT `fk_users_comments` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`),CONSTRAINT `fk_articles_comments` FOREIGN KEY (`article_id`) REFERENCES `articles`(`id`),CONSTRAINT `fk_comments_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`));
CREATE INDEX `idx_comments_user_id` ON `comments`(`user_id`);
CREATE INDEX `idx_comments_article_id` ON `comments`(`article_id`);

CREATE TABLE IF NOT EXISTS "clinics"  (`ext_id` text,`name` text NOT NULL DEFAULT null,`line1` text,`line2` text,`city` text,`state` text,`country` text,`zip` text,`email` text,`phone` text NOT NULL DEFAULT null,`instagram_url` text,`facebook_url` text,`user_id` text NOT NULL DEFAULT null,`created_at` datetime,`updated_at` datetime,`id` text NOT NULL DEFAULT null,`whatsapp` text,`telegram` text,`instagram` text,`facebook` text,`messenger` text,`profile_pic` text, `cover_pic` text, `deleted_at` datetime, `favorite` numeric,PRIMARY KEY (`id`),CONSTRAINT `fk_clinics_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`),CONSTRAINT `fk_users_clinics` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`));
CREATE INDEX `idx_clinics_user_id` ON `clinics`(`user_id`);
CREATE INDEX `idx_clinics_deleted_at` ON `clinics`(`deleted_at`);

CREATE TABLE IF NOT EXISTS "patients"  (`ext_id` text,`email` text,`phone` text NOT NULL DEFAULT null,`facebook_url` text,`profile_url` text,`user_id` text NOT NULL DEFAULT null,`created_at` datetime,`updated_at` datetime,`id` text NOT NULL DEFAULT null,`line1` text,`line2` text,`city` text,`state` text,`country` text,`zip` text,`facebook_handle` text,`whatsapp_handle` text,`telegram_handle` text,`age` integer,`profile_pic` text,`whatsapp` text,`telegram` text,`messenger` text,`instagram` text,`facebook` text,`first_name` text NOT NULL DEFAULT null,`last_name` text NOT NULL DEFAULT null,`username` text,`pass` text, `ocupation` text, `enable_notifications` numeric, `family_history` text, `medical_background` text, `notes` text, `via_email` numeric, `via_whatsapp` numeric, `via_messenger` numeric, `via_telegram` numeric, `deleted_at` datetime,PRIMARY KEY (`id`),CONSTRAINT `fk_patients_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`),CONSTRAINT `fk_users_patients` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`));
CREATE INDEX `idx_patients_user_id` ON `patients`(`user_id`);
CREATE INDEX `idx_patients_deleted_at` ON `patients`(`deleted_at`);

CREATE TABLE IF NOT EXISTS "appointments"  (`booked_at` datetime NOT NULL DEFAULT null,`confirmed_at` datetime DEFAULT null,`canceled_at` datetime DEFAULT null,`rescheduled_at` datetime DEFAULT null,`accepted_at` datetime DEFAULT null,`no_show_at` datetime DEFAULT null,`deleted_at` datetime,`created_at` datetime,`updated_at` datetime,`id` text NOT NULL DEFAULT null,`summary` text,`observations` text,`conclusions` text,`notes` text,`ext_id` text,`hashtags` text,`user_id` text NOT NULL DEFAULT null,`clinic_id` text NOT NULL DEFAULT null,`patient_id` text NOT NULL DEFAULT null,`duration` integer,`booked_month` integer,`booked_minute` integer,`booked_hour` integer,`booked_day` integer,`booked_year` integer,`confirmed` numeric,`canceled` numeric,`rescheduled` numeric,`accepted` numeric,`no_show` numeric,`cost` integer,`viewed_at` datetime DEFAULT null,`sent_at` datetime DEFAULT null,`started_at` datetime DEFAULT null,`done_at` datetime DEFAULT null,`begin_at` datetime DEFAULT null,`status` text NOT NULL DEFAULT "draft", `notification_status` text NOT NULL DEFAULT "pending", `enable_notifications` numeric, `via_email` numeric, `via_whatsapp` numeric, `via_messenger` numeric, `via_telegram` numeric, `booked_alert_sent_at` datetime DEFAULT null, `reminder_alert_sent_at` datetime DEFAULT null, `pending_at` datetime DEFAULT null, `old_booked_at` datetime DEFAULT null, `token` text,PRIMARY KEY (`id`),CONSTRAINT `fk_appointments_clinic` FOREIGN KEY (`clinic_id`) REFERENCES `clinics`(`id`),CONSTRAINT `fk_appointments_patient` FOREIGN KEY (`patient_id`) REFERENCES `patients`(`id`),CONSTRAINT `fk_users_appointments` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`),CONSTRAINT `fk_patients_appoinments` FOREIGN KEY (`patient_id`) REFERENCES `patients`(`id`),CONSTRAINT `fk_patients_appointments` FOREIGN KEY (`patient_id`) REFERENCES `patients`(`id`));
CREATE INDEX `idx_appointments_user_id` ON `appointments`(`user_id`);
CREATE INDEX `idx_appointments_status` ON `appointments`(`status`);
CREATE INDEX `idx_appointments_clinic_id` ON `appointments`(`clinic_id`);
CREATE INDEX `idx_appointments_patient_id` ON `appointments`(`patient_id`);
CREATE INDEX `idx_appointments_deleted_at` ON `appointments`(`deleted_at`);
CREATE INDEX `idx_appointments_notification_status` ON `appointments`(`notification_status`);

CREATE VIRTUAL TABLE global_fts USING fts5(gid, primary, secondary, tertiary)
/* global_fts(gid,"primary",secondary,tertiary) */;
CREATE TABLE IF NOT EXISTS 'global_fts_data'(id INTEGER PRIMARY KEY, block BLOB);
CREATE TABLE IF NOT EXISTS 'global_fts_idx'(segid, term, pgno, PRIMARY KEY(segid, term)) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS 'global_fts_content'(id INTEGER PRIMARY KEY, c0, c1, c2, c3);
CREATE TABLE IF NOT EXISTS 'global_fts_docsize'(id INTEGER PRIMARY KEY, sz BLOB);
CREATE TABLE IF NOT EXISTS 'global_fts_config'(k PRIMARY KEY, v) WITHOUT ROWID;

CREATE TABLE `alerts` (`medium` text NOT NULL DEFAULT null,`name` text NOT NULL DEFAULT null,`title` text,`sub` text,`message` text,`sent_at` datetime DEFAULT null,`delivered_at` datetime DEFAULT null,`viewed_at` datetime DEFAULT null,`from` text,`to` text,`status` text NOT NULL DEFAULT "pending",`alertable_id` text,`alertable_type` text,`created_at` datetime,`updated_at` datetime,`id` text NOT NULL DEFAULT null,PRIMARY KEY (`id`));
CREATE INDEX `poly_fevnt_idx` ON `alerts`(`alertable_id`,`alertable_type`);
CREATE INDEX `idx_alerts_status` ON `alerts`(`status`);
CREATE INDEX `idx_alerts_name` ON `alerts`(`name`);
CREATE INDEX `idx_alerts_medium` ON `alerts`(`medium`);

CREATE TABLE IF NOT EXISTS "feed_events"  (`subject` text,`subject_id` text NOT NULL DEFAULT null,`subject_type` text NOT NULL DEFAULT null,`subject_url` text,`action` text,`target` text NOT NULL DEFAULT null,`target_id` text NOT NULL DEFAULT null,`target_type` text,`target_url` text,`ocurred_at` datetime DEFAULT null,`extra1` text,`extra2` text,`extra3` text,`feed_eventable_id` text,`feed_eventable_type` text,`created_at` datetime,`updated_at` datetime,`id` text NOT NULL DEFAULT null,`name` text NOT NULL DEFAULT null,PRIMARY KEY (`id`));
CREATE INDEX `fe_target_idx` ON `feed_events`(`target`,`target_id`);
CREATE INDEX `idx_feed_events_ocurred_at` ON `feed_events`(`ocurred_at`);
CREATE INDEX `fe_poly_idx` ON `feed_events`(`feed_eventable_id`,`feed_eventable_type`);
CREATE INDEX `idx_feed_events_name` ON `feed_events`(`name`);
CREATE INDEX `fe_subject_idx` ON `feed_events`(`subject_id`,`subject_type`);
CREATE INDEX `idx_feed_events_action` ON `feed_events`(`action`);
CREATE INDEX `idx_appointments_old_booked_at` ON `appointments`(`old_booked_at`);

CREATE TABLE IF NOT EXISTS "users"  (`confirm_email_expires_at` datetime,`reset_token_expires_at` datetime,`name` text,`email` text NOT NULL DEFAULT null,`role` text NOT NULL DEFAULT null,`password` text,`theme` text,`reset_token` text,`confirm_email_token` text,`created_at` datetime,`updated_at` datetime,`id` text NOT NULL DEFAULT null,`ext_id` text,`profile_pic` text,PRIMARY KEY (`id`));
CREATE UNIQUE INDEX `idx_users_email` ON `users`(`email`);
CREATE INDEX `idx_users_role` ON `users`(`role`);
