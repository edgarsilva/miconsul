#!/usr/bin/env bash

# signin_cookie prints the session cookie ("session_id=...") for a local-form
# sign-in. Usage: signin_cookie BASE_URL EMAIL PASSWORD
signin_cookie() {
	local base="$1" email="$2" password="$3"
	curl -sS -o /dev/null -D - -X POST "$base/signin" \
		--data-urlencode "email=$email" \
		--data-urlencode "password=$password" |
		tr -d '\r' |
		awk -F'[ ;]+' 'tolower($1) == "set-cookie:" && $2 ~ /^session_id=/ { print $2; exit }'
}
