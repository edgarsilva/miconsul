# Deployment Runbook

## Coolify Required Env Vars

Minimum runtime vars for production deploy:

- `APP_ENV=production`
- `APP_NAME`, `APP_PROTOCOL`, `APP_DOMAIN`, `APP_VERSION`, `APP_PORT`
- `COOKIE_SECRET` (must be exactly 16, 24, or 32 bytes)
- `JWT_SECRET`
- `DB_PATH`, `SESSION_DB_PATH`
- `EMAIL_SENDER`, `EMAIL_SECRET`, `EMAIL_FROM_ADDRESS`, `EMAIL_SMTP_URL`
- `GOOSE_DRIVER`, `GOOSE_DBSTRING`, `GOOSE_MIGRATION_DIR`
- `ASSETS_DIR`

Optional one-time admin bootstrap vars:

- `ADMIN_USER`
- `ADMIN_PASSWORD`

Behavior:

- After migrations, app checks if an admin exists.
- If an admin already exists, no action is taken.
- If no admin exists and both `ADMIN_USER` + `ADMIN_PASSWORD` are set, app creates one admin user.
- If no admin exists and vars are missing, app continues without auto-creating an admin.

## Healthcheck Expectations

App health endpoints:

- `/livez` process liveness
- `/readyz` readiness (includes DB readiness)
- `/startupz` startup lifecycle (optional monitor)

Runtime image includes `curl`/`wget` for Coolify checks.

Suggested Coolify check target:

- Primary: `/readyz`
- Secondary: `/livez` for process-up validation

## Seeded vs Non-Seeded Behavior

- Seeded flow (`cmd/seed`): creates deterministic seed data including `admin@seed.local`.
- Non-seeded production flow: no seed dependency; use signup + admin bootstrap env vars if needed.

Admin access path:

- Existing admin can access admin routes directly.
- For fresh production DB without seed data, set `ADMIN_USER` + `ADMIN_PASSWORD` for first boot, then remove/rotate those vars.

## Logto Tenant Checklist (Coolify)

- Provision a Logto tenant/environment for production.
- Configure app callback/signout URLs for the production domain.
- Set runtime vars: `LOGTO_URL`, `LOGTO_APP_ID`, `LOGTO_APP_SECRET`, `LOGTO_RESOURCE`.
- Verify sign-in callback and API token audience after deploy.
