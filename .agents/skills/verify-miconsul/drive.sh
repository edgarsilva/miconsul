PW_SESSION="${PW_SESSION:-verify}"
BASE="${BASE:-http://localhost:${MICONSUL_STACK_PORT:-3200}}"
STACK="${TMPDIR:-/tmp}/miconsul-stack"

if command -v mise >/dev/null 2>&1; then
	playwright_dir="$(mise where npm:@playwright/cli 2>/dev/null || true)"
	[ -z "$playwright_dir" ] || PATH="$playwright_dir/node_modules/.bin:$PATH"
fi

pw() { playwright-cli -s="$PW_SESSION" "$@"; }

snap() {
	pw snapshot >/dev/null 2>&1
	ls -t .playwright-cli/*.yml | head -1
}

ref() { grep -o "$1 \"$2\"[^[]*\[ref=[a-z0-9]*\]" "$(snap)" | head -1 | sed 's/.*ref=\(.*\)\]/\1/'; }

textboxes() { grep -o 'textbox[^[]*\[ref=[a-z0-9]*\]' "$(snap)" | sed 's/.*ref=\(.*\)\]/\1/'; }

signin() {
	pw open "$BASE/signin" >/dev/null 2>&1
	set -- $(textboxes)
	pw fill "$1" admin@seed.local >/dev/null
	pw fill "$2" 'Admin123!' >/dev/null
	pw click "$(ref button 'Sign in')" 2>&1 | grep "Page URL"
}

db() { sqlite3 "$STACK/app.sqlite" "$@"; }

path() { pw eval "() => location.pathname + location.search" 2>&1 | grep -o '"[^"]*"' | head -1 | tr -d '"'; }

first() { grep -o "$1[^[]*\[ref=[a-z0-9]*\]" "$(snap)" | head -1 | sed 's/.*ref=\(.*\)\]/\1/'; }

has() { pw eval "() => document.body.innerText.includes($(printf '%s' "$1" | python3 -c 'import json,sys;print(json.dumps(sys.stdin.read()))'))" 2>&1 | grep -o "true\|false" | head -1; }
