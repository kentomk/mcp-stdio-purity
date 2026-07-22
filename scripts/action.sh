#!/bin/sh
set -eu

action_path=${GITHUB_ACTION_PATH:?GITHUB_ACTION_PATH is required}
command=${MSP_INPUT_COMMAND:?command input is required}
working_directory=${MSP_INPUT_WORKING_DIRECTORY:-.}
format=${MSP_INPUT_FORMAT:-text}
timeout=${MSP_INPUT_TIMEOUT:-10s}
cleanup_grace=${MSP_INPUT_CLEANUP_GRACE:-250ms}

temp_root=${RUNNER_TEMP:-${TMPDIR:-/tmp}}
mkdir -p "$temp_root"
build_root=$(mktemp -d "$temp_root/mcp-stdio-purity-source.XXXXXX")
cleanup() {
  rm -rf "$build_root"
}
trap cleanup EXIT HUP INT TERM

binary=$build_root/mcp-stdio-purity
arguments_file=$build_root/arguments
(cd "$action_path" && go build -trimpath -o "$binary" ./cmd/mcp-stdio-purity)
printf '%s' "${MSP_INPUT_ARGUMENTS:-}" > "$arguments_file"

set -- check --format "$format" --timeout "$timeout" --cleanup-grace "$cleanup_grace" -- "$command"
if [ -s "$arguments_file" ]; then
  while IFS= read -r argument || [ -n "$argument" ]; do
    set -- "$@" "$argument"
  done < "$arguments_file"
fi

cd "$working_directory"
"$binary" "$@"
