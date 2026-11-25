#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=scripts/load/signin.sh
source "$(dirname "${BASH_SOURCE[0]}")/load/signin.sh"

BASE_URL="${BASE_URL:-http://localhost:3000}"
TARGET_PATH="${TARGET_PATH:-/appointments/}"
DURATION="${DURATION:-30s}"
CONCURRENCY="${CONCURRENCY:-30}"

AUTH_EMAIL="${AUTH_EMAIL:-admin@seed.local}"
AUTH_PASSWORD="${AUTH_PASSWORD:-Admin123!}"

if ! command -v oha >/dev/null 2>&1; then
  echo "error: oha is required but was not found in PATH"
  echo "install from https://github.com/hatoo/oha or your package manager"
  exit 1
fi

get_auth_cookie() {
  signin_cookie "$BASE_URL" "$AUTH_EMAIL" "$AUTH_PASSWORD"
}

echo "Starting authenticated load test"
echo "Base URL:     $BASE_URL"
echo "Target path:  $TARGET_PATH"
echo "Duration:     $DURATION"
echo "Concurrency:  $CONCURRENCY"

AUTH_COOKIE="$(get_auth_cookie || true)"
if [[ -z "$AUTH_COOKIE" ]]; then
  echo "error: failed to authenticate and retrieve the session cookie"
  exit 1
fi

oha \
  -z "$DURATION" \
  -c "$CONCURRENCY" \
  -H "Cookie: $AUTH_COOKIE" \
  "$BASE_URL$TARGET_PATH"
