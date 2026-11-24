# Admin pages

Covers: internal/services/admin/, internal/views/adminpage.templ, internal/server/diagnostics.go, internal/lib/ringbuf/, internal/models/diagnostics.go, public/js/charts.js, internal/routes/router.go (admin routes), internal/views/cmpnavbar.templ
Starts: signed in as the seed admin; `verify.user@seed.local` exists (created in step 4 and 5 of the sign-in plan). Source `drive.sh` first.

## 1. Admin index
Do: click "Admin" in the navbar
Expect: the path is /admin; headings "Users", "Components", "Runtime" and "Health" and no "Jobs" (the jobs UI is off); the text "All systems operational" appears; request GET /admin/health/status → 200

## 2. Runtime page
Do: from /admin click the "Runtime" card, wait 20 seconds
Expect: the path is /admin/runtime; the text "Uptime", "Goroutines", "GC cycles", "Live heap" and "Heap objects"; `[data-chart]` count 2 and `.apexcharts-canvas` count 2 (`pw eval "() => ..."`); requests GET /admin/runtime/stats → 200 every 5 seconds and GET /admin/runtime/charts → 200 every 15 seconds; the counts stay at 2 after several refreshes

## 3. Health page
Do: open /admin/health
Expect: "All systems operational", the probes "Liveness", "Readiness" and "Startup" each "Passing", and "No external dependencies are registered."

## 4. Components and Back
Do: open /admin/components, click "Back"
Expect: the components page renders; "Back" returns to /admin

## 5. Edit another user
Do: open /admin/users (two rows with "Edit" links), open /admin/users/<uid of verify.user@seed.local>, set "Name" to `Edited By Admin` and "Phone" to `5550001111`, click "Save"
Expect: request PATCH /admin/users/<uid> → 200; `db "select name, phone from users where email='verify.user@seed.local'"` returns the new values

## 6. Admin JSON
Do: sign in with curl (sign-in plan, step 9) and request /api/admin/runtime, /api/admin/runtime/history, /api/admin/health and /api/users
Expect: runtime has `live_heap_bytes` and `gc_cycles`; history has `interval_seconds` and a `samples` list oldest first; health has `status` "ok"; users lists both users. Without a session, `-H "Accept: application/json"` gets 401 and a browser request gets a 303; a regular user's session gets 403.

## 7. Console and requests
Do: after steps 1 to 5, run `pw console error` and `pw requests`
Expect: 0 console errors and no request with status 400 or more
