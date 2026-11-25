# Features

What a person can do in Miconsul, how they reach it, and what proves it works. Agents use this map with the `verify-miconsul` skill (`.agents/skills/verify-miconsul/SKILL.md`): start the app with `scripts/stack.sh up`, sign in as the seed admin, and follow a row. Tests named under "Proof" exist today; the observable result is what to check in the running app.

Seed data from `scripts/stack.sh up`: the admin `admin@seed.local` / `Admin123!`, one clinic "Seed Family Clinic", patients "Alma Rivera", "Bruno Chavez" and "Carla Medina", and three confirmed appointments 48 hours, 72 hours and 7 days from the start. Nothing is booked for today, so "My day" is empty; "My week" and "My month" show the appointments that fall inside them. Email, SMS and WhatsApp are off, so a link that arrives by email (sign-up confirmation, password reset, patient links) is read from the database instead.

## Accounts

| Feature | How the user reaches it | Proof |
|---|---|---|
| Sign in | "Sign in" on the landing page or in the navbar; fill the email and password, press "Sign in" | Lands on the dashboard and the navbar now says "Sign out". Tests: `TestAuthHandlers`, `TestAuthHandlersSigninAndAPI` |
| Sign out | "Sign out" in the navbar | Back on the sign-in page; the old session no longer opens `/dashboard`. Tests: `TestAuthHandlersLogoutAndSessionEndpoints` |
| Sign up and confirm email | "Sign up" on the landing page or "Create an account" on the sign-in page; the confirmation link is `/signup/confirm/<token>` | A user row exists with a `confirm_email_token`; opening the link redirects to the sign-in page with "Email confirmed, you should be able to signin now", and the user can sign in. Tests: `TestAuthHandlersSignupPaths`, `TestAuthHandlersSignupConfirmEmailPaths` |
| Reset password | "Forgot password?" on the sign-in page, "Reset my password"; the link is `/resetpassword/change/<token>`, then "Change password" | The reset token appears on the user row; after "Change password" the new password signs in. Tests: `TestAuthHandlersResetPasswordPaths` |
| Sign in through Logto | With Logto configured, "Sign in" goes to the provider and sign-up and reset go there too | Local sign-up and reset routes hand off to Logto. Tests: `TestLocalAuthDisabledWithLogto` |

## Dashboard

| Feature | How the user reaches it | Proof |
|---|---|---|
| Landing page | `/` while signed out: "Sign up" and "Sign in" | Signed in, `/` redirects to the dashboard. Tests: `TestHandleHomePage` |
| Dashboard | "Dashboard" in the navbar | Cards "Patients", "Appointments" and "Revenue this month" with "View all" links; the appointments list with the tabs "My day", "My week", "My month" and the pills "confirmed", "pending", "done"; "My feed" lists recent activity or "No recent activity". Tests: `TestDashboardHandlers`, `TestHandleAppointmentsSection` |

## Appointments

| Feature | How the user reaches it | Proof |
|---|---|---|
| List and filter | "Appointments" in the navbar; the tabs and pills; type in "Search appointments" | The list narrows to the chosen timeframe, status and search; the address bar keeps the filters. Tests: `TestAppointmentHandlersFlows`, `TestIndexSectionFilterLinks` |
| Create | "Add new" on Appointments: Step 1 select the clinic, Step 2 select the patient, Step 3 "Booked at", "Duration", "Price", then "Save" | The appointment appears in the list. Tests: `TestAppointmentHandlersFlows` |
| Pick the clinic and patient | "Search clinics" and "Search patients" inside the form; choosing a clinic fills the price | The picker lists matches and keeps the selection. Tests: `TestHandleSearchClinicsKeepsSelection`, `TestHandleSearchPatientsKeepsSelection` |
| Edit and reschedule | "Change" on a row, edit, "Save" | The list shows the new details; moving the time marks the appointment rescheduled. Tests: `TestAppointmentEditPageSelectsItsClinicAndPatient`, `TestHandleUpdateStampsRescheduledAtOnlyWhenTheTimeMoves` |
| Run a session | "Open" on a row: "Start session", "Pause session", "Save" the notes, "End session" | Start and pause swap the button; "End session" returns to the list with the appointment "done" and the notes stored. Tests: `TestAppointmentStartEndSessionPostsComplete` |

## Clinics

| Feature | How the user reaches it | Proof |
|---|---|---|
| List and search | "Clinics" in the navbar; "Search clinics" | Matching clinics are listed. Tests: `TestClinicHandlers` |
| Create, edit, delete | "Add new"; "Edit" on a row, change the fields, "Save"; "Delete" and confirm "Are you sure?" | The row appears, changes or disappears; cleared fields stay cleared. Tests: `TestClinicHandlers`, `TestHandleUpdateSavesClearedFields` |

## Patients

| Feature | How the user reaches it | Proof |
|---|---|---|
| List and search | "Patients" in the navbar; "Search patients" | Matching patients are listed with their last appointment. Tests: `TestPatientHandlers` |
| Create, edit, delete | "Add new"; "Edit" on a row, change the fields, "Save"; "Delete" and confirm "Are you sure?"; "Remove" drops the picture | The row appears, changes or disappears. Tests: `TestPatientHandlers`, `TestPatientHTMXResponsesAreFragments` |

## Revenue and profile

| Feature | How the user reaches it | Proof |
|---|---|---|
| Revenue | "Revenue" in the navbar; a month row in "Last 12 months" selects that month | "Revenue", "Done appointments" and "Monthly average" for the month, plus "By patient", "By clinic" and the "Last 12 months" chart. Tests: `TestRevenueHandlers`, `TestReport` |
| Profile | The avatar in the navbar opens "My profile": name, phone, "Language", picture, "Save" | The change persists after a reload and the navbar reflects it. Tests: `TestUserHandlers`, `TestProfileUpdateRefreshesSignedInUser` |
| Theme | The sun and moon icon in the navbar | The page theme switches and the choice is remembered. Tests: `TestThemeHandlers`, `TestHandleToggle` |

## Patient links and notifications

| Feature | How the user reaches it | Proof |
|---|---|---|
| Confirm or cancel from the email | The patient opens `/p/a/<token>/confirm` from the email, no sign-in: "Confirm my appointment", or "Need changes?" then "Cancel my appointment" | "Your appointment has been confirmed." or "Your appointment has been canceled."; the status changes in the database; an expired or unknown link shows "Appointment not found!". Tests: `TestPatientLinksWorkWithoutLogin`, `TestPatientLinkExpiry`, `TestPatientLinkStatusRules` |
| Reschedule email | When a booked appointment's time moves, the patient gets an email once per reschedule | The email carries the new time and the subject "Your appointment has been rescheduled". Tests: `TestAppointmentRescheduledEmailRenders`, `TestRescheduledEmailStringsAreLocalized`, `TestRescheduledAlertJobNotifiesOncePerReschedule` |

## Admin

Admins see "Admin" in the navbar; everyone else is turned away.

| Feature | How the user reaches it | Proof |
|---|---|---|
| Admin index | "Admin" in the navbar | Cards "Users", "Components", "Runtime" and "Health" ("Jobs" only when the jobs UI is enabled); the health card loads "All systems operational". Tests: `TestHandleIndexPageRendersToolCards`, `TestNavbarShowsAdminLinkOnlyToAdmins` |
| Users | The "Users" card, then "Edit" on a row | The user's changes persist; a regular user gets no access. Tests: `TestUserHandlers`, `TestAdminRoutesRequireAdmin` |
| Runtime | The "Runtime" card | Tiles for "Uptime", "Goroutines", "GC cycles", "Live heap" and "Heap objects" refresh every 5 seconds; the "Live heap" and "Goroutines" charts draw once a sample exists, and the page keeps exactly two charts across refreshes. Tests: `TestHandleRuntimePages`, `TestHandleRuntimeChartsFragmentAndPageEmbedding`, `TestRuntimeSamplerRecordsSamplesUntilCancelled` |
| Health | The "Health" card | "All systems operational", the probes "Liveness", "Readiness" and "Startup" as "Passing", and the boot timings. Tests: `TestHandleHealthPages`, `TestHandleAPIHealthDependencies` |
| Admin JSON | Signed-in admin: `GET /api/admin/runtime`, `/api/admin/runtime/history`, `/api/admin/health`, `/api/users` | JSON with the documented keys; 401 without a session. Tests: `TestHandleAPIRuntimeKeys`, `TestHandleAPIRuntimeHistoryShape`, `TestHandleAPIHealthKeys` |
| API sign-in | `POST /api/auth/signin` with an email and password as JSON, when Logto is off | `{"ok":true}` and a `session_id` cookie that opens the admin JSON. Tests: `TestAuthHandlersSigninAndAPI` |
| Components | The "Components" card | Every UI component renders with sample data. Tests: `TestAdminHandlers` |
| Jobs monitor | The "Jobs" card, only with `JOBS_UI_ENABLED` | The queue monitor opens for an admin; without the setting `/admin/jobs` does not exist. Tests: `TestAdminHandlers` |

## Screens

The screens and actions a person meets, one row each: where it lives (area), its route, and the feature row above that proves it. `make features/map` draws them as the user flows.

| Screen | Area | Route | Feature |
|---|---|---|---|
| Landing page | Accounts | `/` | Landing page |
| Sign in | Accounts | `/signin` | Sign in |
| Sign up | Accounts | `/signup` | Sign up and confirm email |
| Confirm email | Accounts | `/signup/confirm/:token` | Sign up and confirm email |
| Reset password | Accounts | `/resetpassword` | Reset password |
| Change password | Accounts | `/resetpassword/change/:token` | Reset password |
| Dashboard | Dashboard | `/dashboard` | Dashboard |
| Dashboard timeframe | Dashboard | `/dashboard/appointments` | Dashboard |
| Appointments list | Appointments | `/appointments` | List and filter |
| Filter appointments | Appointments | `/appointments/search` | List and filter |
| New appointment | Appointments | `/appointments/new` | Create |
| Pick clinic and patient | Appointments | `/appointments/search/clinics` | Pick the clinic and patient |
| Save appointment | Appointments | `POST /appointments` | Create |
| Edit appointment | Appointments | `/appointments/:id` | Edit and reschedule |
| Save changes | Appointments | `PATCH /appointments/:id` | Edit and reschedule |
| Open appointment | Appointments | `/appointments/:id/open` | Run a session |
| Start session | Appointments | `POST .../start-session` | Run a session |
| Pause session | Appointments | `POST .../pause-session` | Run a session |
| Save notes | Appointments | `POST .../save-session` | Run a session |
| End session | Appointments | `POST .../complete` | Run a session |
| Clinics list | Clinics | `/clinics` | List and search |
| Search clinics | Clinics | `/clinics/search` | List and search |
| New clinic | Clinics | `/clinics/new` | Create, edit, delete |
| Edit clinic | Clinics | `/clinics/:id` | Create, edit, delete |
| Save clinic | Clinics | `POST` or `PATCH /clinics` | Create, edit, delete |
| Delete clinic | Clinics | `DELETE /clinics/:id` | Create, edit, delete |
| Patients list | Patients | `/patients` | List and search |
| Search patients | Patients | `/patients/search` | List and search |
| New patient | Patients | `/patients/new` | Create, edit, delete |
| Edit patient | Patients | `/patients/:id` | Create, edit, delete |
| Save patient | Patients | `POST` or `PATCH /patients` | Create, edit, delete |
| Delete patient | Patients | `DELETE /patients/:id` | Create, edit, delete |
| Revenue page | Revenue | `/revenue` | Revenue |
| Profile page | Profile | `/profile` | Profile |
| Save profile | Profile | `PATCH /profile` | Profile |
| Theme toggle | Theme | `POST /theme/toggle` | Theme |
| Reschedule email | Patient links and notifications | email | Reschedule email |
| Confirm page | Patient links and notifications | `/p/a/:token/confirm` | Confirm or cancel from the email |
| Cancel page | Patient links and notifications | `/p/a/:token/cancel` | Confirm or cancel from the email |
| Admin index | Admin | `/admin` | Admin index |
| Users | Admin | `/admin/users` | Users |
| Edit user | Admin | `/admin/users/:id` | Users |
| Runtime | Admin | `/admin/runtime` | Runtime |
| Health | Admin | `/admin/health` | Health |
| Components | Admin | `/admin/components` | Components |
| Jobs monitor | Admin | `/admin/jobs` | Jobs monitor |

## User flows

What the person does to get from one screen to the next. A flow that returns to an earlier screen is drawn as a loop under the screens.

| From | Does | To |
|---|---|---|
| Landing page | Sign in | Sign in |
| Landing page | Sign up | Sign up |
| Sign in | Create an account | Sign up |
| Sign in | Forgot password? | Reset password |
| Sign up | opens the email link | Confirm email |
| Reset password | opens the email link | Change password |
| Sign in | signs in | Dashboard |
| Dashboard | tabs and pills | Dashboard timeframe |
| Dashboard | navbar | Appointments list |
| Dashboard | navbar | Clinics list |
| Dashboard | navbar | Patients list |
| Dashboard | navbar | Revenue page |
| Dashboard | avatar | Profile page |
| Dashboard | sun and moon | Theme toggle |
| Dashboard | Sign out | Sign in |
| Dashboard | navbar, admins | Admin index |
| Appointments list | tabs, pills, search | Filter appointments |
| Appointments list | Add new | New appointment |
| New appointment | Step 1 and 2 | Pick clinic and patient |
| Pick clinic and patient | Save | Save appointment |
| Save appointment | lands on the list | Appointments list |
| Appointments list | Change | Edit appointment |
| Edit appointment | Save | Save changes |
| Save changes | lands on the list | Appointments list |
| Save changes | time moved | Reschedule email |
| Appointments list | Open | Open appointment |
| Open appointment | Start session | Start session |
| Start session | Pause session | Pause session |
| Open appointment | Save | Save notes |
| Open appointment | End session | End session |
| Reschedule email | opens the link | Confirm page |
| Confirm page | Need changes? | Cancel page |
| Clinics list | Search clinics | Search clinics |
| Clinics list | Add new | New clinic |
| Clinics list | Edit | Edit clinic |
| New clinic | Save | Save clinic |
| Save clinic | lands on the list | Clinics list |
| Edit clinic | Save | Save clinic |
| Edit clinic | Delete | Delete clinic |
| Delete clinic | lands on the list | Clinics list |
| Patients list | Search patients | Search patients |
| Patients list | Add new | New patient |
| Patients list | Edit | Edit patient |
| New patient | Save | Save patient |
| Save patient | lands on the list | Patients list |
| Edit patient | Save | Save patient |
| Edit patient | Delete | Delete patient |
| Delete patient | lands on the list | Patients list |
| Profile page | Save | Save profile |
| Admin index | Users | Users |
| Users | Edit | Edit user |
| Admin index | Runtime | Runtime |
| Admin index | Health | Health |
| Admin index | Components | Components |
| Admin index | Jobs | Jobs monitor |

## Where each feature lives

Which source paths belong to which feature, and which plan in `.agents/skills/verify-miconsul/plans/` drives it. Pick rows from `git diff --name-only`; a row with "none yet" has no plan, so drive it from the table above.

| Feature area | Source paths | Plan |
|---|---|---|
| Accounts | `internal/services/auth/`, `internal/views/auth*.templ`, `internal/views/landingpage.templ` | sign-in |
| Dashboard | `internal/services/dashboard/`, `internal/views/dashboardpage.templ` | none yet |
| Appointments | `internal/services/appointment/`, `internal/views/appointment*.templ` | appointment-session, htmx-routes |
| Clinics | `internal/services/clinic/`, `internal/views/clinicspage.templ` | htmx-routes |
| Patients | `internal/services/patient/`, `internal/views/patientspage.templ` | htmx-routes |
| Revenue | `internal/services/revenue/`, `internal/views/revenuepage.templ` | none yet |
| Profile | `internal/services/user/`, `internal/views/userspage.templ` | htmx-routes |
| Theme | `internal/services/theme/`, `internal/views/cmpthemetoggle.templ` | htmx-routes |
| Patient links and notifications | `internal/services/appointment/`, `internal/mailer/` | none yet |
| Admin | `internal/services/admin/`, `internal/views/adminpage.templ`, `internal/server/diagnostics.go`, `internal/lib/ringbuf/`, `public/js/charts.js` | admin |
| Shared: routes, layouts, htmx responses | `internal/routes/router.go`, `internal/server/`, `internal/views/layouts.templ`, `internal/views/cmpnavbar.templ` | htmx-routes, sign-in |
