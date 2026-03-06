#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:3000}"

# Defaults target ~39 RPM total across all routes:
# (3 public * 8 RPM) + (3 protected * 5 RPM) = 39 RPM
PUBLIC_RPM="${PUBLIC_RPM:-8}"
PROTECTED_RPM="${PROTECTED_RPM:-5}"

AUTH_EMAIL="${AUTH_EMAIL:-admin@seed.local}"
AUTH_PASSWORD="${AUTH_PASSWORD:-Admin123!}"

PUBLIC_ROUTES=(
  "/"
  "/signin"
  "/livez"
)

PROTECTED_ROUTES=(
  "/appointments/"
  "/patients/"
  "/clinics/"
)

rpm_to_qps() {
  local rpm="$1"
  awk "BEGIN { if ($rpm <= 0) { print \"0\"; exit } printf \"%.3f\", $rpm / 60 }"
}

rpm_to_interval_seconds() {
  local rpm="$1"
  awk "BEGIN { if ($rpm <= 0) { print \"0\"; exit } printf \"%.3f\", 60 / $rpm }"
}

PUBLIC_QPS="$(rpm_to_qps "$PUBLIC_RPM")"
PROTECTED_QPS="$(rpm_to_qps "$PROTECTED_RPM")"
PUBLIC_INTERVAL_SECONDS="$(rpm_to_interval_seconds "$PUBLIC_RPM")"
PROTECTED_INTERVAL_SECONDS="$(rpm_to_interval_seconds "$PROTECTED_RPM")"

get_auth_cookie() {
  curl -sS -X POST "$BASE_URL/api/auth/signin" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$AUTH_EMAIL\",\"password\":\"$AUTH_PASSWORD\"}" \
    -D - -o /dev/null \
    | tr -d '\r' \
    | awk '/^Set-Cookie: Auth=/{print $2}' \
    | cut -d';' -f1
}

run_public_route() {
  local route="$1"
  local interval_seconds="$2"
  while true; do
    oha --no-tui -n 1 "$BASE_URL$route" > /dev/null || true
    sleep "$interval_seconds"
  done
}

run_protected_route() {
  local route="$1"
  local interval_seconds="$2"
  local auth_cookie=""

  while [[ -z "$auth_cookie" ]]; do
    auth_cookie="$(get_auth_cookie || true)"
    if [[ -z "$auth_cookie" ]]; then
      echo "failed to get auth cookie for $route; retrying in 2s"
      sleep 2
    fi
  done

  while true; do
    if ! oha --no-tui -n 1 -H "Cookie: $auth_cookie" "$BASE_URL$route" > /dev/null; then
      auth_cookie="$(get_auth_cookie || true)"
    fi
    sleep "$interval_seconds"
  done
}

echo "Starting synthetic observability load against $BASE_URL"
echo "Public routes: ${#PUBLIC_ROUTES[@]} at ${PUBLIC_RPM} RPM each (~${PUBLIC_QPS} RPS each, one req every ${PUBLIC_INTERVAL_SECONDS}s)"
echo "Protected routes: ${#PROTECTED_ROUTES[@]} at ${PROTECTED_RPM} RPM each (~${PROTECTED_QPS} RPS each, one req every ${PROTECTED_INTERVAL_SECONDS}s)"

for route in "${PUBLIC_ROUTES[@]}"; do
  run_public_route "$route" "$PUBLIC_INTERVAL_SECONDS" &
done

for route in "${PROTECTED_ROUTES[@]}"; do
  run_protected_route "$route" "$PROTECTED_INTERVAL_SECONDS" &
done

wait
