#!/bin/sh
set -eu

project_root=$(unset CDPATH; cd -- "$(dirname -- "$0")/.." && pwd)
clean_root=$(mktemp -d)
cleanup() {
  chmod -R u+w "$clean_root" 2>/dev/null || true
  rm -rf "$clean_root"
}
trap cleanup EXIT HUP INT TERM

git -C "$project_root" archive --format=tar HEAD | tar -xf - -C "$clean_root"
mkdir -p "$clean_root/cache" "$clean_root/modcache" "$clean_root/gopath" "$clean_root/bin"

started=$(date +%s)
set +e
quickstart_output=$(cd "$clean_root" && \
  GOCACHE="$clean_root/cache" \
  GOMODCACHE="$clean_root/modcache" \
  GOPATH="$clean_root/gopath" \
  GOTOOLCHAIN=local \
  timeout 300 sh -c 'go build -o ./bin/mcp-stdio-purity ./cmd/mcp-stdio-purity && ./bin/mcp-stdio-purity check -- go run ./examples/fixture-server --mode startup-banner')
quickstart_status=$?
set -e
elapsed=$(( $(date +%s) - started ))

[ "$quickstart_status" -eq 1 ]
[ "$elapsed" -le 60 ]
printf '%s' "$quickstart_output" | grep -q '^MSP001 invalid-json '
if printf '%s' "$quickstart_output" | grep -q 'fixture startup banner'; then
  echo 'quickstart leaked raw contamination' >&2
  exit 1
fi
printf 'clean quickstart: %ss\n' "$elapsed"
