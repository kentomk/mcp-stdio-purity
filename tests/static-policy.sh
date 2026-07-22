#!/bin/sh
set -eu

project_root=$(unset CDPATH; cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

grep -q '^MIT License$' LICENSE
jq -e '
  .schemaVersion == 1 and
  .candidateId == "20260718T144541Z-7acd" and
  .owner == "kentomk" and
  .author == "@kentomk" and
  (.createdBy | test("Matsuki Kento") and test("@kentomk") and test("automated AI agent"; "i")) and
  .automatedAgent == true and
  .project == "mcp-stdio-purity"
' .kento-oss.json >/dev/null

[ -z "$(go list -m -f '{{if not .Main}}{{.Path}} {{.Version}}{{end}}' all | sed '/^$/d')" ]

if git grep -I -n -E \
  '(BEGIN [A-Z ]*PRIVATE KEY|github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|AKIA[A-Z0-9]{16})' \
  -- . ':!tests/static-policy.sh'; then
  echo 'secret-like material found in tracked files' >&2
  exit 1
fi

grep -Eq 'uses: actions/setup-go@[0-9a-f]{40}([[:space:]]|$)' action.yml
if grep -Eq 'uses: actions/setup-go@v[0-9]' action.yml; then
  echo 'mutable setup-go reference found in composite Action' >&2
  exit 1
fi
