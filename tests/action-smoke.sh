#!/bin/sh
set -eu

project_root=$(unset CDPATH; cd -- "$(dirname -- "$0")/.." && pwd)
smoke_root=$(mktemp -d)
cleanup() {
  rm -rf "$smoke_root"
}
trap cleanup EXIT HUP INT TERM

clean_output=$(env \
  GITHUB_ACTION_PATH="$project_root" \
  RUNNER_TEMP="$smoke_root/run" \
  MSP_INPUT_COMMAND=go \
  MSP_INPUT_WORKING_DIRECTORY="$project_root" \
  MSP_INPUT_TIMEOUT=5s \
  MSP_INPUT_CLEANUP_GRACE=250ms \
  MSP_INPUT_ARGUMENTS='run
./examples/fixture-server
--mode
clean' \
  MSP_INPUT_FORMAT=json \
  "$project_root/scripts/action.sh")
printf '%s' "$clean_output" | grep -q '"status": "passed"'

set +e
dirty_output=$(env \
  GITHUB_ACTION_PATH="$project_root" \
  RUNNER_TEMP="$smoke_root/run" \
  MSP_INPUT_COMMAND=go \
  MSP_INPUT_WORKING_DIRECTORY="$project_root" \
  MSP_INPUT_TIMEOUT=5s \
  MSP_INPUT_CLEANUP_GRACE=250ms \
  MSP_INPUT_ARGUMENTS='run
./examples/fixture-server
--mode
startup-banner' \
  MSP_INPUT_FORMAT=text \
  "$project_root/scripts/action.sh" 2>&1)
dirty_status=$?
set -e
[ "$dirty_status" -eq 1 ]
printf '%s' "$dirty_output" | grep -q '^MSP001 invalid-json '
if printf '%s' "$dirty_output" | grep -q 'fixture startup'; then
  echo 'raw fixture output leaked from the Action' >&2
  exit 1
fi

set +e
invalid_output=$(env \
  GITHUB_ACTION_PATH="$project_root" \
  RUNNER_TEMP="$smoke_root/run" \
  MSP_INPUT_COMMAND=go \
  MSP_INPUT_WORKING_DIRECTORY="$project_root" \
  MSP_INPUT_TIMEOUT=0s \
  MSP_INPUT_CLEANUP_GRACE=250ms \
  MSP_INPUT_ARGUMENTS='run
./examples/fixture-server
--mode
clean' \
  MSP_INPUT_FORMAT=json \
  "$project_root/scripts/action.sh" 2>&1)
invalid_status=$?
set -e
[ "$invalid_status" -eq 2 ]
printf '%s' "$invalid_output" | grep -q '"status": "error"'
printf '%s' "$invalid_output" | grep -q '"error": "invalid timeout or output limits"'
[ -z "$(find "$smoke_root/run" -maxdepth 1 -name 'mcp-stdio-purity-source.*' -print -quit)" ]
