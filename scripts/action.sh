#!/bin/sh
set -eu

action_path=${GITHUB_ACTION_PATH:?GITHUB_ACTION_PATH is required}
command=${MSP_INPUT_COMMAND:?command input is required}
working_directory=${MSP_INPUT_WORKING_DIRECTORY:-.}
format=${MSP_INPUT_FORMAT:-text}
timeout=${MSP_INPUT_TIMEOUT:-10s}
cleanup_grace=${MSP_INPUT_CLEANUP_GRACE:-250ms}
max_line_bytes=${MSP_INPUT_MAX_LINE_BYTES:-1048576}
max_stdout_bytes=${MSP_INPUT_MAX_STDOUT_BYTES:-16777216}
max_diagnostics=${MSP_INPUT_MAX_DIAGNOSTICS:-20}

temp_root=${RUNNER_TEMP:-${TMPDIR:-/tmp}}
mkdir -p "$temp_root"
build_root=$(mktemp -d "$temp_root/mcp-stdio-purity-source.XXXXXX")
cleanup() {
  rm -rf "$build_root"
}
trap cleanup EXIT HUP INT TERM

if [ ! -d "$working_directory" ]; then
  if [ "$format" = json ]; then
    printf '%s\n' '{"schemaVersion":1,"status":"error","error":"working directory is not a directory"}'
  else
    printf '%s\n' 'error: working directory is not a directory'
  fi
  exit 2
fi

binary=$build_root/mcp-stdio-purity
arguments_file=$build_root/arguments
(cd "$action_path" && go build -trimpath -o "$binary" ./cmd/mcp-stdio-purity)
printf '%s' "${MSP_INPUT_ARGUMENTS:-}" > "$arguments_file"

set -- check --format "$format" --timeout "$timeout" --cleanup-grace "$cleanup_grace" \
  --max-line-bytes "$max_line_bytes" --max-stdout-bytes "$max_stdout_bytes" \
  --max-diagnostics "$max_diagnostics" -- "$command"
if [ -s "$arguments_file" ]; then
  while IFS= read -r argument || [ -n "$argument" ]; do
    set -- "$@" "$argument"
  done < "$arguments_file"
fi

cd "$working_directory"
"$binary" "$@"
