#!/usr/bin/env bash
set -euo pipefail

PORT="${MICONSUL_STACK_PORT:-3200}"
RUN="${TMPDIR:-/tmp}/miconsul-stack"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
	cat >&2 <<EOF
usage: scripts/stack.sh up [checkout]   build, seed and start an isolated app on port $PORT
       scripts/stack.sh doctor          read-only check that the running instance is ours and usable
       scripts/stack.sh logs            follow the app log
       scripts/stack.sh down            stop the instance and delete its databases and logs

env: MICONSUL_STACK_PORT (default 3200), TMPDIR (run folder: \$TMPDIR/miconsul-stack)
EOF
	exit 2
}

stack_env() {
	export APP_ENV=development
	export APP_PORT="$PORT"
	export APP_DOMAIN="localhost:$PORT"
	export APP_PROTOCOL=http
	export DB_PATH="$RUN/app.sqlite"
	export SESSION_DB_PATH="$RUN/session.sqlite"
	export CACHE_DB_PATH="$RUN/cache.sqlite"
	export GOOSE_DRIVER=sqlite3
	export GOOSE_DBSTRING="$RUN/app.sqlite"
	export GOOSE_MIGRATION_DIR=./goose/migrations
	export JOBS_ENABLED=false
	export JOBS_UI_ENABLED=false
	export RATE_LIMITER_ENABLED=false
	export EMAIL_SMTP_URL=127.0.0.1
	export EMAIL_SENDER=verify@seed.local
	export EMAIL_SECRET=disabled
	export EMAIL_FROM_ADDRESS=verify@seed.local
	export WHATSAPP_PROVIDER=none
	export TWILIO_ACCOUNT_SID=disabled
	export TWILIO_AUTH_TOKEN=disabled
	export TWILIO_SMS_FROM=disabled
	export TWILIO_WHATSAPP_FROM=disabled
	export TWILIO_WHATSAPP_CONTENT_SID=disabled
	export TWILIO_API_BASE_URL=http://127.0.0.1:9
	unset LOGTO_URL LOGTO_APP_ID LOGTO_APP_SECRET LOGTO_RESOURCE
}

own_pid() {
	[ -f "$RUN/app.pid" ] || return 1
	local pid
	pid="$(cat "$RUN/app.pid")"
	[ -n "$pid" ] && [ "$(readlink -f "/proc/$pid/exe" 2>/dev/null)" = "$RUN/app" ] || return 1
	echo "$pid"
}

up() {
	local checkout
	checkout="$(realpath "${1:-$ROOT}")"
	down quiet
	mkdir -p "$RUN"
	cd "$checkout"

	[ -d node_modules ] || npm ci --ignore-scripts --allow-remote=all --no-audit --no-fund --loglevel=error
	make templ/build >"$RUN/build.log" 2>&1
	go build -tags fts5 -o "$RUN/app" ./cmd/app >>"$RUN/build.log" 2>&1

	local dirty=clean
	[ -z "$(git status --porcelain)" ] || dirty=dirty
	echo "$(git rev-parse --short HEAD) $dirty $checkout" >"$RUN/build.info"

	stack_env
	go run -tags fts5 ./cmd/seed -bulk=false >"$RUN/seed.log" 2>&1

	nohup "$RUN/app" >"$RUN/app.log" 2>&1 &
	echo $! >"$RUN/app.pid"

	for _ in $(seq 1 60); do
		if curl -sf -o /dev/null "http://localhost:$PORT/signin"; then
			echo "miconsul ready on http://localhost:$PORT ($(cat "$RUN/build.info"))"
			return 0
		fi
		sleep 0.5
	done
	echo "miconsul did not become ready; see $RUN/app.log" >&2
	tail -20 "$RUN/app.log" >&2
	return 1
}

doctor() {
	local pid fail=0
	if pid="$(own_pid)"; then
		echo "ok   process $pid is $RUN/app"
	else
		echo "FAIL no running instance of ours (pid file $RUN/app.pid)"
		return 1
	fi

	if [ -f "$RUN/build.info" ]; then
		echo "ok   built from $(cat "$RUN/build.info")"
	else
		echo "FAIL no build.info"
		fail=1
	fi

	if ss -ltnp "sport = :$PORT" 2>/dev/null | grep -q "pid=$pid,"; then
		echo "ok   port $PORT is owned by pid $pid"
	else
		echo "FAIL port $PORT is not owned by pid $pid"
		fail=1
	fi

	local path code
	for path in /livez /readyz /signin; do
		code="$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT$path" || true)"
		if [ "$code" = 200 ]; then
			echo "ok   GET $path -> 200"
		else
			echo "FAIL GET $path -> $code"
			fail=1
		fi
	done

	return "$fail"
}

down() {
	local pid
	if pid="$(own_pid)"; then
		kill "$pid" 2>/dev/null || true
	fi
	rm -rf "$RUN"
	[ "${1:-}" = quiet ] || echo "miconsul stack stopped"
}

case "${1:-}" in
up) up "${2:-}" ;;
doctor) doctor ;;
logs) tail -f "$RUN/app.log" ;;
down) down ;;
*) usage ;;
esac
