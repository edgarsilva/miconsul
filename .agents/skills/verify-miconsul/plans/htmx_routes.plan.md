# HTMX routes and forms

Covers: internal/routes/router.go, internal/server/response.go, internal/views/*.templ, internal/services/*/handlers.go
Starts: signed in as the seed admin, on /dashboard?timeframe=day. Source `drive.sh` first. `<uid>` values come from the database: `db "select uid from appointments order by booked_at"`, `db "select uid from clinics"`. Run the steps in order: step 4 ends an appointment that step 5 relies on, and step 12 deletes the clinic, so it is last.

## 1. Patient search
Do: open /patients, fill the search box (a `searchbox` named "ctrl k": `ref searchbox 'ctrl k'`) with `alma`
Expect: request GET /patients/search?searchTerm=alma → 200; the list shows "Alma Rivera" and not "Bruno Chavez"

## 2. Appointment filter pills
Do: open /appointments?timeframe=week, click "confirmed"
Expect: request GET /appointments/search?status=confirmed&timeframe=week → 200; the address bar shows /appointments?status=confirmed&timeframe=week. Only the first two seeded appointments fall in the current week; the third is a week out.

## 3. Theme toggle
Do: click the visible theme toggle. Its checkbox is not clickable by ref, so click its label: `pw eval "() => { Array.from(document.querySelectorAll('label.swap')).find(e => e.offsetParent !== null).click(); return 'clicked'; }"`
Expect: request POST /theme/toggle → 200; `<html>` gets a `data-theme` attribute (the first save reloads the page once)

## 4. Session start, pause, end
Do: open /appointments/<first uid>/open, click "Start session", "Pause session", "Start session" again, then "End session"
Expect: POST …/start-session → 200, POST …/pause-session → 200, POST …/complete → 204, then the path /appointments; the row has status `done`

## 5. Done pill
Do: open /appointments?timeframe=week, click "done"
Expect: request GET /appointments/search?status=done&timeframe=week → 200; the patient of the appointment ended in step 4 ("Alma Rivera") is listed, and with "confirmed" selected instead she is not

## 6. Save a clinic
Do: open /clinics/<clinic uid>, fill "Name" with `Seed Family Clinic 2` and the "Appointment price" spinbutton with `500`, click the first "Save"
Expect: request PATCH /clinics/<clinic uid> → 200; `db "select name, price from clinics"` returns `Seed Family Clinic 2|50000` (cents). The seed price is 0 and the input has a minimum, so the form will not submit without a price.

## 7. Plain request to a fragment route
Do: sign in with curl (see SKILL.md) and request /patients/search?searchTerm=alma
Expect: status 200 and a bare list fragment: no `<html>`, and it contains "Alma Rivera". Without a session the same request returns 303. Handlers have one htmx-shaped path for every client.

## 8. Create a patient
Do: open /patients/new, fill "Name" `Dora Verify`, "Phone" `5551234567`, "Email address" `dora@verify.local` and the "Age" spinbutton `33` (required), click the first "Save"
Expect: request POST /patients → 200; the path becomes /patients/<new uid>; `db "select name from patients"` includes `Dora Verify`

## 9. Removed fallback route
Do: request /clinics/<clinic uid>/patch
Expect: status 404

## 10. Edit an appointment
Do: open /appointments/<confirmed uid>, set "Duration" to `45` and "Price" to `600` (the seed price is 0 and the input has a minimum), click "Save"
Expect: request PATCH /appointments/<uid> → 204; the row has duration 45 and price 60000

## 11. Save the profile
Do: open /profile, set "Name" to `Verify Admin` and "Phone" to `5559876543` (required; the seed admin has none), click "Save"
Expect: request PATCH /profile → 200; `db "select name, phone from users where email='admin@seed.local'"` returns the new values

## 12. Delete the clinic (last: destroys data)
Do: open /clinics/<clinic uid>, click "Delete", then `pw dialog-accept` for the "Are you sure?" dialog
Expect: request DELETE /clinics/<clinic uid> → 204, then the path /clinics; the row has `deleted_at` set

## 13. Boosted navigation, back and reload
Do: on /patients run `pw eval "() => { window.__marker = 1; return 'set'; }"`, click the first "Edit" link (it sits inside hx-boost), then go back, then go forward and reload
Expect: after the click the path is /patients/<uid> and `window.__marker` is still `1` (no full reload); after back the path is /patients and the marker is still set; after reload the page has the full layout (`#header`) and the patient form, and the marker is gone
