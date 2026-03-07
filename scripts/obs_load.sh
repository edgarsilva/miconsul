#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:3000}"
CURL_TIMEOUT_SECONDS="${CURL_TIMEOUT_SECONDS:-5}"

# Defaults target ~39 RPM total across all routes.
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

rpm_to_interval_seconds() {
  local rpm="$1"
  awk "BEGIN { if ($rpm <= 0) { print \"3600\"; exit } printf \"%.3f\", 60 / $rpm }"
}

PUBLIC_INTERVAL_SECONDS="$(rpm_to_interval_seconds "$PUBLIC_RPM")"
PROTECTED_INTERVAL_SECONDS="$(rpm_to_interval_seconds "$PROTECTED_RPM")"
TOTAL_RPM=$((PUBLIC_RPM + PROTECTED_RPM))
if [[ "$TOTAL_RPM" -le 0 ]]; then
  echo "PUBLIC_RPM and PROTECTED_RPM cannot both be 0"
  exit 1
fi
MAIN_INTERVAL_SECONDS="$(rpm_to_interval_seconds "$TOTAL_RPM")"

get_auth_cookie() {
  curl -sS -X POST "$BASE_URL/api/auth/signin" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$AUTH_EMAIL\",\"password\":\"$AUTH_PASSWORD\"}" \
    -D - -o /dev/null \
    | tr -d '\r' \
    | awk '/^Set-Cookie: Auth=/{print $2}' \
    | cut -d';' -f1
}

random_index() {
  local length="$1"
  echo $((RANDOM % length))
}

pick_route_type() {
  local draw=$((RANDOM % TOTAL_RPM))
  if [[ "$draw" -lt "$PUBLIC_RPM" ]]; then
    echo "public"
  else
    echo "protected"
  fi
}

echo "Starting synthetic observability load against $BASE_URL"
echo "Public RPM weight: ${PUBLIC_RPM}"
echo "Protected RPM weight: ${PROTECTED_RPM}"
echo "Main loop interval: ${MAIN_INTERVAL_SECONDS}s (single process)"

auth_cookie=""

request_public() {
  local route="$1"
  curl -sS -o /dev/null -m "$CURL_TIMEOUT_SECONDS" "$BASE_URL$route" || true
}

request_protected() {
  local route="$1"

  if [[ -z "$auth_cookie" ]]; then
    auth_cookie="$(get_auth_cookie || true)"
    if [[ -z "$auth_cookie" ]]; then
      echo "failed to get auth cookie for protected route $route; retrying in 2s"
      sleep 2
      return
    fi
  fi

  local status_code
  status_code=$(curl -sS -o /dev/null -m "$CURL_TIMEOUT_SECONDS" -w "%{http_code}" -H "Cookie: $auth_cookie" "$BASE_URL$route" || echo "000")
  if [[ "$status_code" == "401" || "$status_code" == "403" || "$status_code" == "303" ]]; then
    auth_cookie="$(get_auth_cookie || true)"
  fi
}

while true; do
  route_type="$(pick_route_type)"

  if [[ "$route_type" == "public" ]]; then
    idx="$(random_index "${#PUBLIC_ROUTES[@]}")"
    route="${PUBLIC_ROUTES[$idx]}"
    request_public "$route"
  else
    idx="$(random_index "${#PROTECTED_ROUTES[@]}")"
    route="${PROTECTED_ROUTES[$idx]}"
    request_protected "$route"
  fi

  sleep "$MAIN_INTERVAL_SECONDS"
done
