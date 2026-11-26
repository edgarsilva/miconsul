# Appointment session

Covers: internal/services/appointment/, internal/views/appointmentstartpage.templ, internal/views/appointmentspage.templ, internal/routes/router.go (appointment routes)
Starts: signed in as the seed admin. Source `drive.sh` first. `<uid>` is a confirmed appointment: `db "select uid from appointments where status='confirmed' order by booked_at limit 1"`

## 1. Open the session page
Do: open /appointments/<uid>/open
Expect: a link "Start session" and buttons "Save" and "End session"

## 2. Start the session
Do: click "Start session"
Expect: request POST /appointments/<uid>/start-session → 200; a "Pause session" link replaces "Start session"

## 3. Pause and resume
Do: click "Pause session", then "Start session" again
Expect: requests POST …/pause-session → 200 and POST …/start-session → 200; the link swaps each time

## 4. Save a note
Do: fill the first text box (Observations) with `Saved note` and click "Save"
Expect: request POST /appointments/<uid>/save-session → 204; `db "select observations from appointments where uid='<uid>'"` returns `Saved note`

## 5. End the session with an unsaved note
Do: fill the first text box with `Unsaved note` (no Save) and click "End session"
Expect: request POST /appointments/<uid>/complete → 204, then the path /appointments; the database row has status `done` and observations `Unsaved note`
