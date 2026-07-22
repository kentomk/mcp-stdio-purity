#!/bin/sh
set -eu

project_root=$(unset CDPATH; cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

test -z "$(gofmt -l .)"
command -v shellcheck >/dev/null 2>&1 || { echo 'shellcheck is required' >&2; exit 1; }
shellcheck scripts/*.sh tests/*.sh tests/alternatives/*.sh
go test ./...
if command -v zig >/dev/null 2>&1; then
  CGO_ENABLED=1 CC="zig cc" go test -race ./internal/checker -run TestValidateRecord
else
  CGO_ENABLED=1 go test -race ./internal/checker -run TestValidateRecord
fi
go vet ./...
tests/static-policy.sh
tests/action-smoke.sh
tests/release-package.sh
tests/release-workflow.sh
