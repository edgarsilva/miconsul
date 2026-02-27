#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AI_DIR="${ROOT_DIR}/docs/ai"
SOURCE_URL="https://templ.guide/llms.md"
UPSTREAM_FILE="${AI_DIR}/templ-llms.upstream.md"
META_FILE="${AI_DIR}/templ-llms.meta.txt"

mkdir -p "${AI_DIR}"

tmp_file="$(mktemp)"
trap 'rm -f "${tmp_file}"' EXIT

curl -fsSL "${SOURCE_URL}" -o "${tmp_file}"
mv "${tmp_file}" "${UPSTREAM_FILE}"

bytes="$(wc -c < "${UPSTREAM_FILE}" | tr -d ' ')"
lines="$(wc -l < "${UPSTREAM_FILE}" | tr -d ' ')"
sha256="$(sha256sum "${UPSTREAM_FILE}" | awk '{print $1}')"
fetched_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

cat > "${META_FILE}" <<EOF
source_url=${SOURCE_URL}
fetched_at_utc=${fetched_at}
upstream_file=docs/ai/templ-llms.upstream.md
bytes=${bytes}
lines=${lines}
sha256=${sha256}
EOF

printf "Updated %s\n" "${UPSTREAM_FILE}"
printf "Updated %s\n" "${META_FILE}"
printf "Note: keep docs/ai/templ-llms.compact.md aligned with upstream rules.\n"
