# HTMX 4 Migration Checklist

## Goal

Upgrade from mixed HTMX versions to HTMX 4 with behavior parity first, then optional cleanup.

## Phase 0 - Baseline and safety

- [x] Keep current feature work isolated on `feat/search-modal-shortcut`.
- [x] Run HTMX upgrade checker including `.templ` files.
- [ ] Capture pre-migration behavior with smoke checks:
  - [ ] Patients index search (`/patients/search`)
  - [ ] Clinics index search (`/clinics/search`)
  - [ ] Appointments index search (`/appointments/search`)
  - [ ] Appointment form clinic search (`/appointments/search/clinics`)
  - [ ] Shared boosted navigation links

## Phase 1 - Script/version alignment

- [ ] Replace mixed HTMX includes with a single HTMX 4 include in `internal/views/layouts.templ`:
  - [ ] `HTMLPage` currently uses `htmx@2.0.1`
  - [ ] `HTMLPageWithApexCharts` currently uses `htmx@1.9.9`
- [ ] Keep Alpine include unchanged for now.

## Phase 2 - HTMX 4 compatibility config (parity mode)

- [ ] Update `meta[name=htmx-config]` in `internal/views/layouts.templ`:
  - [ ] Rename `includeIndicatorStyles` -> `includeIndicatorCSS`
  - [ ] Add compatibility settings to preserve 2.x behavior for:
    - [ ] explicit attribute inheritance defaults
    - [ ] non-swapping 4xx/5xx responses

## Phase 3 - Fix upgrade-check findings

### Inheritance findings (17 total issues across 9 files)

- [ ] `internal/views/appointmentspage.templ`
  - [ ] Add `:inherited` for `hx-boost` at line 16
  - [ ] Add `:inherited` for `hx-swap`, `hx-target`, `hx-select` at line 59
  - [ ] Add `:inherited` for `hx-target`, `hx-swap` at line 431
- [ ] `internal/views/appointmentstartpage.templ`
  - [ ] Add `:inherited` for `hx-boost` at line 21
- [ ] `internal/views/clinicspage.templ`
  - [ ] Add `:inherited` for `hx-boost` at lines 14 and 96
- [ ] `internal/views/cmpbtns.templ`
  - [ ] Add `:inherited` for `hx-boost` at line 4
- [ ] `internal/views/cmpfooter.templ`
  - [ ] Add `:inherited` for `hx-boost` at line 10
- [ ] `internal/views/cmpnavbar.templ`
  - [ ] Add `:inherited` for `hx-boost` at line 6
- [ ] `internal/views/patientspage.templ`
  - [ ] Add `:inherited` for `hx-boost` at lines 15 and 53
- [ ] `internal/views/userspage.templ`
  - [ ] Add `:inherited` for `hx-boost` at lines 14 and 120
- [ ] `internal/views/layouts.templ`
  - [ ] Rename config key `includeIndicatorStyles` -> `includeIndicatorCSS`

## Phase 4 - Regenerate and validate

- [ ] Regenerate templ output: `make templ`
- [ ] Run tests: `go test ./...`
- [ ] Re-run upgrade check:
  - [ ] `npx htmx.org@next upgrade-check /home/edgar/workspace/miconsul --ext .templ`
- [ ] Manual smoke pass for HTMX-heavy flows:
  - [ ] navigation still boosted where expected
  - [ ] search fragments swap correctly
  - [ ] 4xx/5xx server responses keep previous UX behavior

## Phase 5 - Optional hardening after stable rollout

- [ ] Remove temporary HTMX compatibility flags one by one.
- [ ] Re-test after each flag removal.
- [ ] Evaluate whether `unsafe-eval` can be removed in a separate security-focused PR.

## Upgrade checker command used

```bash
npx htmx.org@next upgrade-check /home/edgar/workspace/miconsul --ext .templ
```
