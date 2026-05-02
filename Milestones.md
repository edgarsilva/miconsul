# Release Milestones

## Ongoing

- [x] README and Go Env Setup/Updates (2026/Mar/27)
- [x] Auth docs drift cleanup (`docs/testing.md` migration status) (2026/Mar/27)
- [ ] Uptime Kuma monitors + notifications + SLO-style alerts
- [ ] Upload images to S3-compatible object storage (RustFS)
- [ ] Move frontend + templ generated artifacts to image-build pipeline (keep runtime image slim)

  - [ ] Generate Tailwind CSS during Docker build from `styles/global.css` (instead of relying on prebuilt committed `public/global.css`)
  - [ ] Define templ generation policy for CI/image builds vs committed artifacts and enforce one canonical source of truth

## Nice-to-Have (Low Priority)

- [ ] Optional Valkey-backed HTTP sessions

## V0 (7day proof of concept)

[2024/Apr/21]

- [x] Basic project setup

[2024/Apr/25] - [2024/May/01]

- [x] Login page
- [x] Email/Password endpoints
- [x] Reset Password Functionality

[2024/May/02] - [2024/May/07]

- [x] Working Login Page
- [x] Reset Password Functionality
- [x] Signup page

[2024/May/08]

- [x] Confirm email Functionality

[2024/May/09]

- [x] Fix Schema Ids to use XID

**note**: v0 took more than 7ds narrow scope

[2024/May/10 - 2024/May/17]

- [x] Add ERD for Miconsul sample DB
- [x] Patients index page
- [x] Patients Create/Update Form
- [x] HTMX integration

[2024/May/18 - 2024/May/22]

- [x] Clinics views and endpoints
- [x] Appointments views and endpoints
- [x] Basic Toast and notifications

[2024/May/23 - 2024/May/31]

- [x] Added patient actions and emails

[2024/Jun/01 - 2024/Jun/06]

- [x] Running Somewhere other than the dev machine
- [x] Initial Deployment using coolify
- [x] Make updates and Docker file

[2024/Jun/10 - 2024/Jun/11]

- [x] Setup servers DNS and Coolify container/service manager
- [x] Minio install for Litestream
- [x] Hostinger backup for Sqlite DB backup streams with Litestream/Minio
- [x] Sep up Goose migrations

## V1 (We can consider v1 Alpha started here)

[2024/Jun/12 - 2024/Jun/17]

- [x] Appointment select day filtering by default
- [x] Appointment filtering by clinic
- [x] Clinic index page search
- [x] Patient index page search

[2024/Jun/18 - 2024/Jun/24]

- [x] Appointment add clinic icon
- [x] Overall architecture design
- [x] Optional authentication integration with Logto identity manage/provider

[2024/Jun/25 - 2024/Jun/30]

- [x] Fix navigation buttons on mobile
- [x] Accept file uploads for avatars

- [x] Dashboard updates

  - [x] Show favorite clinic in Dashboard

[2024/Jul/01 - 2024/Jul/07]

- [x] Upload patients profile pic to disk under authentication
- [x] Accept profile pic from Logto, from google identity.
- [ ] Upload images to S3-compatible object storage (RustFS)

- [ ] Notifications rollout (email + chat channels)

  - [ ] Keep email notification templates/actions aligned with appointment + professional info
  - [ ] Add Telegram delivery channel
  - [ ] Add WhatsApp delivery channel
  - [ ] Add Facebook Messenger delivery channel

- [ ] Release to beta testers after notification channels baseline is live
