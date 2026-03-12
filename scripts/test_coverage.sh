#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "$SCRIPT_DIR/.." && pwd)"

cd "$ROOT_DIR"

run_quiet() {
  local label="$1"
  shift

  local log_file
  log_file="$(mktemp)"

  if ! "$@" >"$log_file" 2>&1; then
    printf "\033[31m❌ %s failed\033[0m\n" "$label"
    cat "$log_file"
    rm -f "$log_file"
    exit 1
  fi

  rm -f "$log_file"
}

mkdir -p coverage
rm -f \
  coverage/all.out \
  coverage/all.filtered.out \
  coverage/summary.filtered.txt \
  coverage/pkg_coverage.txt \
  coverage/service_pkg_coverage.txt \
  coverage/cmd_app.out \
  coverage/cmd_seed.out \
  coverage/cmd_coverage.txt \
  coverage/internal_unit.out \
  coverage/internal_unit.filtered.out \
  coverage/integration.out \
  coverage/integration.filtered.out \
  coverage/split_coverage.txt

printf "\033[36m🧪 Running canonical coverage (internal + integration tests)...\033[0m\n"
run_quiet "canonical coverage" go test ./internal/... ./tests/... -covermode=atomic -coverpkg=./internal/... -coverprofile=coverage/all.out

printf "\033[36m🧪 Running internal unit coverage...\033[0m\n"
run_quiet "internal unit coverage" go test ./internal/... -covermode=atomic -coverpkg=./internal/... -coverprofile=coverage/internal_unit.out

printf "\033[36m🧪 Running integration coverage...\033[0m\n"
run_quiet "integration coverage" go test ./tests/... -covermode=atomic -coverpkg=./internal/... -coverprofile=coverage/integration.out

printf "\033[36m🧪 Running cmd package-local coverage...\033[0m\n"
run_quiet "cmd/app coverage" go test ./cmd/app -covermode=atomic -coverprofile=coverage/cmd_app.out
run_quiet "cmd/seed coverage" go test ./cmd/seed -covermode=atomic -coverprofile=coverage/cmd_seed.out

awk 'NR==1 || ($0 !~ /internal\/lib\/localize\// && $0 !~ /_templ\.go:/)' coverage/internal_unit.out > coverage/internal_unit.filtered.out
awk 'NR==1 || ($0 !~ /internal\/lib\/localize\// && $0 !~ /_templ\.go:/)' coverage/integration.out > coverage/integration.filtered.out

unit_pct="$(go tool cover -func=coverage/internal_unit.filtered.out | awk '/^total:/{print $NF}')"
integration_pct="$(go tool cover -func=coverage/integration.filtered.out | awk '/^total:/{print $NF}')"
printf "internal/unit %s\nintegration/tests %s\n" "$unit_pct" "$integration_pct" > coverage/split_coverage.txt

cmd_app_pct="$(go tool cover -func=coverage/cmd_app.out | awk '/^total:/{print $NF}')"
cmd_seed_pct="$(go tool cover -func=coverage/cmd_seed.out | awk '/^total:/{print $NF}')"
printf "cmd/app %s\ncmd/seed %s\n" "$cmd_app_pct" "$cmd_seed_pct" > coverage/cmd_coverage.txt

printf "\033[35m🧹 Filtering generated files from coverage profile...\033[0m\n"
awk 'NR==1 || ($0 !~ /internal\/lib\/localize\// && $0 !~ /_templ\.go:/)' coverage/all.out > coverage/all.filtered.out
go tool cover -func=coverage/all.filtered.out > coverage/summary.filtered.txt

awk '
  NR>1 {
    split($1, a, ":")
    file=a[1]
    block=$1
    stmts=$2+0
    cnt=$3+0
    if (!(block in seen)) {
      seen[block]=1
      blockStmts[block]=stmts
      blockFile[block]=file
    }
    if (cnt>0) blockCovered[block]=1
  }
  END {
    for (b in seen) {
      pkg=blockFile[b]
      sub(/\/[^\/]+$/, "", pkg)
      total[pkg]+=blockStmts[b]
      if (blockCovered[b]) covered[pkg]+=blockStmts[b]
    }
    for (p in total) {
      pct=(total[p]>0)?(100*covered[p]/total[p]):0
      printf "%06.2f%% %4d/%-4d %s\n", pct, covered[p], total[p], p
    }
  }
' coverage/all.filtered.out | sort -n > coverage/pkg_coverage.txt

awk '$3 ~ /^miconsul\/internal\/service\// {print}' coverage/pkg_coverage.txt > coverage/service_pkg_coverage.txt

printf "\033[33m📉 Lowest covered packages (filtered):\033[0m\n"
awk 'NR<=12 {printf "  • [%02d] %-8s %-7s %s\n", NR, $1, $2, $3}' coverage/pkg_coverage.txt

printf "\033[34m📚 Coverage lanes (filtered):\033[0m\n"
awk '{printf "  %s: %s\n", $1, $2}' coverage/split_coverage.txt

printf "\033[34m📦 Cmd coverage (package-local):\033[0m\n"
awk '{printf "  %s: %s\n", $1, $2}' coverage/cmd_coverage.txt

kpi_total="$(awk '/^total:/{print $NF}' coverage/summary.filtered.txt)"
printf "\n\033[1;32m✅ Coverage Total: %s\033[0m \033[2m(generated files excluded; internal KPI)\033[0m\n" "$kpi_total"
