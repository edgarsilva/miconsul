---
name: verify-miconsul
description: Launch an isolated Miconsul (Go, Fiber, HTMX clinic app) with seed data, drive it in a browser or with curl the way a user does, and capture proof. Use to verify a change in the real app before a PR, or when a bug only shows in the browser.
---

# Verify Miconsul

Prove a change works in the running app, not only in tests. Read `docs/features.md` for the feature you touched: it says how a user reaches it and what proves it works.

## Launch

```
scripts/stack.sh up
```

Builds the current checkout, creates a throwaway SQLite database, seeds it, starts the app and waits until `/signin` answers. It prints `miconsul ready on http://localhost:3200 (<commit> clean|dirty <path>)`. `dirty` means uncommitted changes are included in the build.

- Base URL: `http://localhost:3200` (set `MICONSUL_STACK_PORT` to change it).
- Isolated: its own port, and databases in `${TMPDIR:-/tmp}/miconsul-stack`. The dev database and the dev server on port 3000 are never touched.
- Off: jobs, the rate limiter, Logto and all outbound email, SMS and WhatsApp.
- Sign in with the local form: `admin@seed.local` / `Admin123!` (an admin; it exists only in this throwaway database).
- Seed data: one clinic "Seed Family Clinic", patients "Alma Rivera", "Bruno Chavez" and "Carla Medina", and three confirmed appointments 2 days, 3 days and 1 week ahead. The Day view is empty because nothing is booked for today.
- If the port is taken or the build fails, `up` stops with the log tail. Do not start a second copy.

## Doctor

```
scripts/stack.sh doctor
```

Read-only. Checks that the process behind the pid file is our binary, which commit it was built from, that the port is owned by that process, and that `/livez`, `/readyz` and `/signin` answer 200. Run it first whenever a page looks wrong, and before trusting an instance you did not start.

## Plans

`plans/` holds step lists, each step a "Do" and an "Expect" with the request it should make, all run against the real app. Pick the plans whose `Covers:` paths match what changed:

```
git diff --name-only --relative $(git merge-base HEAD origin/main)
```

If none match, or for an upgrade, run all of them. Order when running several: `sign-in`, `appointment-session`, `admin`, `htmx-routes` (it deletes the seed clinic, so it goes last).

| Plan | Drives |
|---|---|
| `plans/sign-in.plan.md` | sign in and out, sign up and confirm, password reset, API sign-in, the admin boundary for a regular user |
| `plans/appointment-session.plan.md` | start, pause, save and end a session |
| `plans/admin.plan.md` | the admin index, runtime, health, components, editing another user, the admin JSON |
| `plans/htmx-routes.plan.md` | patient search, filter pills, theme toggle, creating and editing records, deletes, boosted navigation |

A step fails when its "Expect" is not on the page, when the page logs a console error or an uncaught page error, or when a request returns 400 or more, unless it is listed under known noise. Use `docs/features.md` for features no plan covers yet.

## Drive

Source the helpers from the evidence folder, then call them: `. <repo>/.agents/skills/verify-miconsul/drive.sh`. Underneath is `playwright-cli` (`@playwright/cli`, session flag `-s=<name>`):

```
playwright-cli -s=verify open http://localhost:3200/signin
playwright-cli -s=verify snapshot        # YAML of the page; elements carry [ref=...]
playwright-cli -s=verify fill <ref> "admin@seed.local"
playwright-cli -s=verify click <ref>
playwright-cli -s=verify goto http://localhost:3200/admin/runtime
playwright-cli -s=verify eval "() => document.querySelectorAll('[data-chart]').length"
playwright-cli -s=verify screenshot --filename=<absolute path>.png
playwright-cli -s=verify console error
playwright-cli -s=verify requests
playwright-cli -s=verify close
```

- Run it from the evidence folder: it writes `.playwright-cli/` into the current directory.
- Take refs from a fresh `snapshot` before each action; they change when the page changes.
- Sign-in page: the first text box is the email, the second the password, and the button is "Sign in". It lands on `/dashboard?timeframe=day`.
- Prefer link and button names and routes over positions. Navigation in the app is htmx: after a click, wait for the new content in a snapshot, not for a page load.
- If `-s=<name>` is rejected as an unknown option, another `playwright-cli` wrapper is first on PATH. Put the `@playwright/cli` install first (with mise: `$(mise where npm:@playwright/cli)/node_modules/.bin`).

HTTP and JSON, with curl (no browser needed for APIs). Form posts need an `Origin` header because of cross-origin protection:

```
curl -c jar.txt -H "Origin: http://localhost:3200" -H "Content-Type: application/json" \
  -d '{"email":"admin@seed.local","password":"Admin123!"}' http://localhost:3200/api/auth/signin
curl -b jar.txt http://localhost:3200/api/admin/runtime
```

Without a session, browsers get a 303 redirect and JSON clients a 401. `curl` does not follow redirects unless told to, so a bare 200 from a load tool may be a followed redirect to the sign-in page.

Database state, read-only:

```
sqlite3 "${TMPDIR:-/tmp}/miconsul-stack/app.sqlite" "select uid, status from appointments order by booked_at"
```

## Evidence

Create `${TMPDIR:-/tmp}/miconsul-evidence/<yyyymmdd-hhmmss>-<feature>/` and put everything there: screenshots, the `console error` and `requests` output, curl responses and a short `notes.md` saying what was done and what was seen. It is outside the run folder on purpose, so `down` never deletes it.

A proof means:
- Exercise the real user path: click and type as a user would, not internal setters or test-only routes.
- Capture the action and the resulting state, not only the last screen. Take one screenshot per page.
- Check side effects next to what is visible: the database row, the response status of the request the click made.
- After each page, read `console error` and `requests`. A console error, an uncaught page error or a request with status 400 or more fails the step unless it is listed under known noise.
- A precondition missing from the seed data is a failure with the reason written down, not a reason to weaken the journey.

## Cleanup

```
scripts/stack.sh down
playwright-cli -s=verify close
```

`down` stops only the process named in its own pid file, never by process name, and deletes the run folder (databases and logs). The evidence folder is kept. If `up` failed halfway, run `down` before trying again.

## Helpers

`scripts/stack.sh`: `up [checkout]` builds another checkout or worktree instead of the current one; `logs` follows the app log.

`.agents/skills/verify-miconsul/drive.sh`: source it from the evidence folder. It puts the mise-installed `playwright-cli` first on PATH when there is one and defines:

| Function | Does |
|---|---|
| `pw <args>` | `playwright-cli` in the session `$PW_SESSION` (default `verify`) |
| `ref <role> <name>` | the ref of the first element with that role and accessible name, e.g. `pw click "$(ref button 'Sign in')"` |
| `first <role>` | the ref of the first element with that role |
| `textboxes` | the refs of the page's text boxes, in order |
| `has <text>` | `true` or `false`: whether the page text contains it |
| `path` | the current path and query |
| `signin` | signs in as the seed admin |
| `db <sql>` | runs a read-only query on the stack's database |

`$BASE` is the base URL.

## Known noise

None recorded yet. Add an entry only for an error that appears on `main` and does not affect behavior.
