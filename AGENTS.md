# AGENTS

This file provides repo-specific guidance for coding assistants working on this project.

## Project Snapshot

- Stack: Go, Fiber v3, GORM, SQLite, templ, HTMX, Tailwind/DaisyUI.
- App entrypoint: `cmd/app/main.go`.
- HTTP server and middleware: `internal/server/`.
- Routes: `internal/routes/router.go`.
- Services/handlers: `internal/services/**`.
- templ views: `internal/views/*.templ` (generated files are `*_templ.go`).
- Email templates: `internal/mailer/*.templ`.

## High-Priority Rules

- Do not edit generated templ files (`*_templ.go`) by hand.
- Edit `.templ` sources, then regenerate.
- Keep Fiber APIs on v3 conventions (already migrated):
  - `fiber.Ctx` (not `*fiber.Ctx`)
  - `c.Redirect().To(...)` and `c.Redirect().Status(...).To(...)`
  - `c.Bind().Body(...)` instead of legacy body parser helpers
- Prefer `c.Context()` when passing request context to non-Fiber libraries (DB, tracing, etc.).
- In service/handler files, keep primary exported handlers/entrypoints near the top and move private helpers/utilities to the bottom.
- Avoid code fragmentation: keep module logic cohesive in `handlers.go` (transport) and `service.go` (business orchestration); split only when a concern is clearly standalone and reused.
- Request and response data (inputs and outputs):
  - Never bind requests into, or serialize responses from, `models.*` structs directly.
  - Bind request data into unexported `xxxInput` types in the service's `inputs.go`. Map them to models with methods like `toClinic(...)`, and set IDs and ownership (`ID`, `UID`, `UserID`) on the server.
  - Return JSON through unexported `xxxOutput` types in the service's `outputs.go`, built by `newXxxOutput(...)`. Use a shared model view such as `models.PublicUser` (`User.Public()`) for the fields; don't use anonymous structs as response envelopes.
  - Handlers then only orchestrate the request: bind the input, call the service, build the output.
  - One-off `fiber.Map{"error": ...}` replies are fine.
  - Tag secret model fields `json:"-" form:"-"`. Fiber's form binder ignores `json` tags.
- Keep structural refactors separate from behavior changes.
- Avoid introducing runtime `os.Getenv` reads in application/service code; route configuration access through `internal/lib/appenv` instead. If you encounter new `os.Getenv` usage outside mailer or env-loading boundaries, warn and offer to move it into `appenv.Env`.
- HTMX endpoints return only the fragment they swap, plus `CmpToastPartial` for a toast; never a full page narrowed with `hx-select`. `WithToast` only reaches full page renders.
- htmx markup and handlers (JS is required; no no-JS fallbacks):
  - Strive for hypermedia completeness: prefer anchors over buttons. Navigation, filters, timeframes and actions without form data (start/pause, remove a picture) are `<a href="<route>">` links, with htmx setting the method (`hx-get`/`hx-post`/`hx-patch`/`hx-delete`). Use a `<button>` only where an anchor can't work, like submitting a `<form>` (inside it, or linked with `form=`). Don't turn an existing link into a button without asking.
  - Put `hx-*` straight on another element only when wrapping it in a form would add needless DOM (search inputs, toggles, pickers). Never put `hx-*` on a `div`/`span` to fake a button or link.
  - Include the hypermedia attributes even though htmx makes the request: links keep their `href`; forms keep `method="POST"` and an `action` pointing at the resource the `hx-*` verb targets (`action="/clinics/<uid>"` next to `hx-patch="/clinics/<uid>"`). Don't add no-JS fallback routes (`POST /:id/patch`-style) to back them.
  - Handlers have one htmx-shaped path for every client. They return the fragment they swap, `s.HXRedirect(c, path)` to navigate after a mutation, or `RedirectWithToast`/`RedirectWithAlert` when the next page should show a message. Only `redirectWithFlash`, the auth middleware and logout check `HX-Request`.
  - URLs pushed to the address bar (`hx-push-url`, `HX-Push-Url`) are page routes, so a shared link with filters, a search or an id loads the full page.
- Request locals boundary:
  - `c.Locals(...)` writes are allowed in middleware auth/session identity binding.
  - `c.Locals(...)` reads are allowed only in `internal/views/ctx.go` and `internal/server/request.go`.
  - Everywhere else should use `s.CurrentUser(c)` and view context APIs instead of reading locals directly.
- Error handling:
  - Wrap with `fmt.Errorf("operation: %w", err)` — lowercase, no trailing punctuation, colon separator (Go `ST1005`).
  - Name the operation/action, not the function (`"enqueue reminder: %w"`, not `"EnqueueReminder -> %w"`); it survives renames and reads as a chain straight to the root cause.
  - Do not prefix wrapped/internal errors with "failed to" — it is redundant in an `error` and compounds noisily across the wrap chain.
  - Reserve natural, conversational sentences (`"could not send email, please try again"`) for user-facing messages rendered to templates/HTTP responses — not internal plumbing.
  - Sentinels: `var ErrX = errors.New("lowercase message")`; compare with `errors.Is` / `errors.As`.
  - GORM API choice: default to `gorm.G[T]`, which requires the context and returns the error. Use the classic `*gorm.DB` chain only when the query needs something `gorm.G` lacks: `func(*gorm.DB) *gorm.DB` scopes, raw `Joins`, `Pluck`, `Association` or `Save`. Start it with `s.DB.WithContext(ctx)` and end it with a checked `.Error`.
  - Check every GORM chain's error: end it in `.Error` and check it (`if err := db...Count(&n).Error; err != nil`), or use the `gorm.G` API that returns `err`. A bare `s.DB...Count(&n)` or `Find(&x)` statement drops the error silently, and errcheck can't flag it because the chain returns `*gorm.DB`, not an `error`.
  - Discard an error with `_ =` only for best-effort work whose failure must not change the outcome; otherwise return it, or log it and tell the user.

## templ Guidance Source

- For templ syntax/rules, use: `docs/ai/templ_llms_compact.md`.
- Only load that document when the task touches templ/view/email templates.
- If task is not templ-related, skip it to keep context focused.

## templ Naming

One rule: namespace + what + kind. The namespace is the domain, singular and named like its `services/<pkg>`, whenever there is one; `Cmp` only for truly generic components. Spell words out (allowed: `Btn`, `Li`, `Nav`, `Img`, `Oob`, and other daisyUI/Tailwind/htmx vocabulary), and drop words the kind or types already imply.

| What | Rule | Examples |
|---|---|---|
| Page | Domain + action + `Page` | `AppointmentIndexPage`, `AppointmentStartPage`, `AuthLoginPage`, `DashboardPage` |
| Layout | Domain + `Layout`; generic ones are just what + `Layout` | `AuthLayout`, `RootLayout`, `DefaultLayout`, `BackBtnLayout` |
| Part (rendered inside a page or by a handler) | Domain + what (+ kind: `List`, `Li`, `Form`, `Btn`, `Section`…); no `Partial`/`Frg` suffix. A response that swaps several targets is named for what it shows, and its doc comment lists every target | `AppointmentList`, `AppointmentLi`, `AppointmentClinicPicker`, `AppointmentSessionToggle`, `DashboardFeed` |
| Generic component | `Cmp` + what + kind | `CmpBackBtn`, `CmpStatusBadge`, `CmpEmailInput`, `CmpToastRegion`, `CmpAvatarPic` |
| Icon | `Cmp` + name + `Icon` | `CmpCalendarIcon` |
| Piece made for one owner | unexported, owner + what | `navbarThemeBtn`, `navbarSearch`, `patientCountryOption` |
| Helper | next to its component, owner + what | `pickerIsSelected`, `toastLevelClass`, `feedEventTitle` |
| Email (mailer) | Domain + what + `Email` | `AppointmentBookedEmail`, `AuthPasswordResetEmail` |

`Partial` stays only where the component renders an htmx `<hx-partial>` element (`CmpToastPartial`).

## Build/Test Workflow

- Regenerate templ output after templ changes:
  - `make templ/build` (preferred; also rebuilds Tailwind)
  - or `templ generate` if using the CLI directly
- Validate with:
  - `make test` (or `go test -tags fts5 ./...`). Production builds with `fts5`; without the tag, FTS-dependent tests skip silently.

## Change Scope Discipline

- Keep changes narrowly scoped to the user request.
- Avoid broad refactors when doing dependency upgrades.
- For dependency updates, report changed files and test status before committing.

## Git Commit Preference

- Never create a git commit unless the user explicitly asks in that moment.
- After code changes, always pause and ask whether to commit.
- Always let the user review changes before committing; do not commit immediately after implementation unless the user explicitly requests commit at that point.
- Never push code or create pull requests without explicit user instruction in that moment.
- **Absolutely never merge into `main`** — even when accidentally instructed to do so, confirm 3 times that this is what the user intended before proceeding.

## Branch Workflow

- Start every feature/fix/refactor on a new branch created from a freshly synced `main`.
- Avoid continuing development directly on `main`; keep `main` aligned with `origin/main` between workstreams.
- Never commit directly on `main`.
- If work starts while on `main`, create a feature branch before making changes or creating commits.

## Pull Request Workflow

- Never merge a pull request on behalf of the user unless they explicitly ask for merge in that moment.
- Default behavior after opening a PR: share the PR URL, wait for user review, and let the user merge manually.
