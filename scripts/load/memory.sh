#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   scripts/load/memory.sh sample OUT.csv   poll /api/admin/runtime until stopped (or SAMPLE_SECONDS)
#   scripts/load/memory.sh report OUT.csv   print peak, end and whether the live-heap floor rises
#
# env: BASE_URL, AUTH_EMAIL, AUTH_PASSWORD, INTERVAL (seconds, default 2), SAMPLE_SECONDS (default: until stopped)

BASE_URL="${BASE_URL:-http://localhost:3200}"
AUTH_EMAIL="${AUTH_EMAIL:-admin@seed.local}"
AUTH_PASSWORD="${AUTH_PASSWORD:-Admin123!}"
INTERVAL="${INTERVAL:-2}"
SAMPLE_SECONDS="${SAMPLE_SECONDS:-0}"

# shellcheck source=scripts/load/signin.sh
source "$(dirname "${BASH_SOURCE[0]}")/signin.sh"

sample() {
	local out="$1" cookie start now
	if ! command -v jq >/dev/null 2>&1; then
		echo "error: jq is required but was not found in PATH" >&2
		exit 1
	fi
	cookie="$(signin_cookie "$BASE_URL" "$AUTH_EMAIL" "$AUTH_PASSWORD")"
	if [[ -z "$cookie" ]]; then
		echo "error: sign-in failed for $AUTH_EMAIL at $BASE_URL" >&2
		exit 1
	fi

	echo "elapsed_s,goroutines,live_heap_bytes,alloc_bytes,sys_bytes,gc_cycles" >"$out"
	start="$(date +%s)"
	while true; do
		now="$(date +%s)"
		curl -sf -m 5 -H "Cookie: $cookie" "$BASE_URL/api/admin/runtime" |
			jq -r --arg t "$((now - start))" '[($t | tonumber), .goroutines, .live_heap_bytes, .alloc_bytes, .sys_bytes, .gc_cycles] | @csv' >>"$out" || true
		if [[ "$SAMPLE_SECONDS" -gt 0 && $((now - start)) -ge "$SAMPLE_SECONDS" ]]; then
			break
		fi
		sleep "$INTERVAL"
	done
}

report() {
	awk -F, '
		NR == 1 { next }
		{
			n++
			heap[n] = $3
			gor[n] = $2
			gc[n] = $6
			if ($3 > peak) peak = $3
			if ($2 > peakg) peakg = $2
		}
		END {
			if (n < 6) { print "error: need at least 6 samples, got " n; exit 1 }
			third = int(n / 3)
			first = heap[1]; last = heap[1]
			for (i = 1; i <= third; i++) if (heap[i] < first) first = heap[i]
			for (i = n - third + 1; i <= n; i++) if (heap[i] < last) last = heap[i]
			printf "samples            %d\n", n
			printf "peak live heap     %.1f MiB\n", peak / 1048576
			printf "end live heap      %.1f MiB\n", heap[n] / 1048576
			printf "floor, first third %.1f MiB\n", first / 1048576
			printf "floor, last third  %.1f MiB\n", last / 1048576
			printf "floor change       %+.1f%%\n", (last - first) * 100 / first
			printf "peak goroutines    %d (end %d)\n", peakg, gor[n]
			printf "gc cycles          %d\n", gc[n] - gc[1]
		}' "$1"
}

case "${1:-}" in
sample) sample "${2:?usage: memory.sh sample OUT.csv}" ;;
report) report "${2:?usage: memory.sh report OUT.csv}" ;;
*)
	echo "usage: $0 sample|report OUT.csv" >&2
	exit 2
	;;
esac
