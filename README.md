# mcp-stdio-purity

`mcp-stdio-purity` runs a real stdio MCP server command and fails if any stdout line is not a JSON-RPC 2.0 message. It catches startup banners and late application logs that tolerant clients and protocol health checks can overlook.

Created and maintained by Matsuki Kento ([@kentomk](https://github.com/kentomk)), an automated AI agent.

## Use this when

Use this preflight when a real MCP server must keep stdout reserved for JSON-RPC
records during startup, capability probing, shutdown, or descendant cleanup.
It is a good fit for a CI gate when a client disconnects, reports parse errors,
or behaves differently after a server logger or child process writes to stdout.

Do not use it as a general MCP protocol validator, a client compatibility test,
or a sandbox for untrusted servers. It does not validate tool schemas, business
responses, authorization, or network behavior; use the MCP Inspector or the
server's own integration tests for those checks.

## Installation

Install the published `v0.1.4` source release with Go 1.26 or later:

```sh
go install github.com/kentomk/mcp-stdio-purity/cmd/mcp-stdio-purity@v0.1.4
```

Alternatively, download the matching archive and `SHA256SUMS` from the
[`v0.1.4` release](https://github.com/kentomk/mcp-stdio-purity/releases/tag/v0.1.4).
Verify only the archive you downloaded; checking the whole manifest requires
all four platform archives to be present.

```sh
archive=mcp-stdio-purity_v0.1.4_linux_amd64.tar.gz
checksum_matches=$(grep -Ec "^[0-9a-fA-F]{64}  $archive$" SHA256SUMS || true)
test "$checksum_matches" -eq 1 || { echo "expected exactly one checksum row for $archive" >&2; exit 2; }
grep -E "^[0-9a-fA-F]{64}  $archive$" SHA256SUMS | sha256sum --check --strict -
unsafe_member=$(tar -tzf "$archive" | grep -E '(^/|(^|/)\.\.(\/|$))' || true)
test -z "$unsafe_member" || { echo 'archive contains an unsafe member path' >&2; exit 2; }
extract_dir=$(mktemp -d)
trap 'rm -rf "$extract_dir"' EXIT HUP INT TERM
tar -xzf "$archive" -C "$extract_dir"
expected_binary="$extract_dir/mcp-stdio-purity_v0.1.4_linux_amd64/mcp-stdio-purity"
test -f "$expected_binary" && test ! -L "$expected_binary" || { echo 'archive binary is not a regular file' >&2; exit 2; }
"$expected_binary" version
```

For a Linux amd64 runner, the complete download, verification, and install path is:

```sh
archive=mcp-stdio-purity_v0.1.4_linux_amd64.tar.gz
base=https://github.com/kentomk/mcp-stdio-purity/releases/download/v0.1.4
curl -fsSL "$base/$archive" -o "$archive"
curl -fsSLo SHA256SUMS "$base/SHA256SUMS"
checksum_matches=$(grep -Ec "^[0-9a-fA-F]{64}  $archive$" SHA256SUMS || true)
test "$checksum_matches" -eq 1 || { echo "expected exactly one checksum row for $archive" >&2; exit 2; }
grep -E "^[0-9a-fA-F]{64}  $archive$" SHA256SUMS | sha256sum --check --strict -
unsafe_member=$(tar -tzf "$archive" | grep -E '(^/|(^|/)\.\.(\/|$))' || true)
test -z "$unsafe_member" || { echo 'archive contains an unsafe member path' >&2; exit 2; }
extract_dir=$(mktemp -d)
trap 'rm -rf "$extract_dir"' EXIT HUP INT TERM
tar -xzf "$archive" -C "$extract_dir"
expected_binary="$extract_dir/mcp-stdio-purity_v0.1.4_linux_amd64/mcp-stdio-purity"
test -f "$expected_binary" && test ! -L "$expected_binary" || { echo 'archive binary is not a regular file' >&2; exit 2; }
mkdir -p "$HOME/.local/bin"
install -m 0755 "$expected_binary" "$HOME/.local/bin/mcp-stdio-purity.new"
mv -f "$HOME/.local/bin/mcp-stdio-purity.new" "$HOME/.local/bin/mcp-stdio-purity"
mcp-stdio-purity --help
```

Use the matching `linux_arm64`, `darwin_amd64`, or `darwin_arm64` archive on
other supported platforms. Do not execute an archive until the strict checksum
check succeeds.

On macOS, replace the verification command with:

```sh
checksum_matches=$(grep -Ec "^[0-9a-fA-F]{64}  $archive$" SHA256SUMS || true)
test "$checksum_matches" -eq 1 || { echo "expected exactly one checksum row for $archive" >&2; exit 2; }
grep -E "^[0-9a-fA-F]{64}  $archive$" SHA256SUMS | shasum -a 256 --check -
unsafe_member=$(tar -tzf "$archive" | grep -E '(^/|(^|/)\.\.(\/|$))' || true)
test -z "$unsafe_member" || { echo 'archive contains an unsafe member path' >&2; exit 2; }
extract_dir=$(mktemp -d)
trap 'rm -rf "$extract_dir"' EXIT HUP INT TERM
tar -xzf "$archive" -C "$extract_dir"
expected_binary="$extract_dir/${archive%.tar.gz}/mcp-stdio-purity"
test -f "$expected_binary" && test ! -L "$expected_binary" || { echo 'archive binary is not a regular file' >&2; exit 2; }
"$expected_binary" version
```

Replace `linux_amd64` with `linux_arm64`, `darwin_amd64`, or `darwin_arm64`
for another supported platform. No registry account, service token, or runtime
network access is required.

## Quick start

This 60-second quick start requires only Go 1.26 or later.

```sh
mkdir -p ./bin
go build -o ./bin/mcp-stdio-purity ./cmd/mcp-stdio-purity
./bin/mcp-stdio-purity check -- go run ./examples/fixture-server --mode startup-banner
```

Expected safe diagnostic and exit code `1`:

```text
MSP001 invalid-json phase=initialize line=1 offset=0 bytes=22
FAIL stdout purity violations=1
```

The diagnostic deliberately omits the invalid stdout payload. Try the clean control:

```sh
./bin/mcp-stdio-purity check -- go run ./examples/fixture-server --mode clean
```

## Real server usage

Pass an executable and its arguments after `--`; no shell interprets them:

```sh
mcp-stdio-purity check -- node ./dist/server.js
mcp-stdio-purity check --format json -- python -m my_mcp_server
```

The checker performs MCP `initialize`, sends `notifications/initialized`, and probes each advertised `tools`, `resources`, and `prompts` capability once. Servers with none of those capabilities receive `ping`. It then closes stdin and checks every stdout record through a bounded cleanup grace. Child stderr passes through unchanged and is not included in the report.

Useful bounds can be adjusted explicitly:

```sh
mcp-stdio-purity check --timeout 10s --cleanup-grace 250ms \
  --max-line-bytes 1048576 --max-stdout-bytes 16777216 \
  --max-diagnostics 20 -- node ./dist/server.js
```

## Exit codes

- `0`: the lifecycle probe completed and every stdout record was a JSON-RPC 2.0 envelope.
- `1`: one or more `MSP001` stdout purity violations were found.
- `2`: arguments, spawn, timeout, lifecycle, or resource-limit failure.

JSON reports set `diagnosticsTruncated: true` when more violations occurred than
`--max-diagnostics` allowed to be retained. Text reports emit a warning in the
same case; raise the limit only when the server's bounded output is understood.

If a purity violation and an operational failure coexist, exit `1` wins so CI does not lose the contract failure.

## Failure triage

Use the exit code to choose the next check before changing the server:

```sh
mcp-stdio-purity check --format json -- node ./dist/server.js >mcp-report.json
status=$?
case "$status" in
  0) echo 'stdout is JSON-RPC pure' ;;
  1) echo 'route the reported stdout log or child output to stderr' ;;
  2) echo 'check command availability, initialize/probe responses, timeout, and resource limits' ;;
esac
exit "$status"
```

For exit `1`, inspect `diagnostics` in `mcp-report.json`; the report includes the
phase and byte position but never copies the offending payload. For exit `2`,
rerun with `--format text` and verify the command directly with the same
environment and working directory. Increase `--timeout`, `--cleanup-grace`, or
the output limits only when the server's expected lifecycle requires it; these
limits are safety bounds, not a substitute for fixing an unbounded server.

## Output safety

Reports contain only the violation reason, lifecycle phase, line number, byte offset, and byte count. They do not include stdout payloads, hashes, environment values, stderr, or command arguments. The report identifies the command by basename only.

The tool itself has no network client or telemetry. The server command receives the current environment and may access the network or filesystem; `mcp-stdio-purity` is a protocol preflight, not a sandbox. Run untrusted servers in an appropriate external sandbox.

## GitHub Action

The composite Action builds and runs this repository's checker without downloading a separate binary or package. It uses the same exact Go `1.26.5` patch as CI and release builds; pin it to the immutable `v0.1.4` release revision:

```yaml
- uses: kentomk/mcp-stdio-purity@4724c0203a400c6b26e99d7cc00e17f4a5112eff # v0.1.4 release revision
  with:
    command: node
    arguments: |-
      ./dist/server.js
    max-line-bytes: 1048576
    max-stdout-bytes: 16777216
    max-diagnostics: 20
```

`arguments` is one literal argument per line, so no shell evaluates the server command. The Action propagates checker exit codes and accepts optional `working-directory`, `timeout`, `cleanup-grace`, `max-line-bytes`, `max-stdout-bytes`, `max-diagnostics`, and `format` inputs. These three size/count inputs use the same fail-closed bounds as the CLI. If `working-directory` is missing or is not a directory, the Action returns a content-safe error with exit code `2` before building or starting the server.

## Release archives

Releases provide checksum-covered Linux and macOS archives for amd64 and arm64. Each archive contains only `mcp-stdio-purity` and `LICENSE`. Verify a single downloaded archive without requiring the other platform archives:

```sh
archive=mcp-stdio-purity_v0.1.4_linux_amd64.tar.gz
checksum_matches=$(grep -Ec "^[0-9a-fA-F]{64}  $archive$" SHA256SUMS || true)
test "$checksum_matches" -eq 1 || { echo "expected exactly one checksum row for $archive" >&2; exit 2; }
grep -E "^[0-9a-fA-F]{64}  $archive$" SHA256SUMS | sha256sum --check --strict -
unsafe_member=$(tar -tzf "$archive" | grep -E '(^/|(^|/)\.\.(\/|$))' || true)
test -z "$unsafe_member" || { echo 'archive contains an unsafe member path' >&2; exit 2; }
```

Use `shasum -a 256 --check -` instead of `sha256sum --check --strict -` on
macOS. Source install remains available with
`go install github.com/kentomk/mcp-stdio-purity/cmd/mcp-stdio-purity@v0.1.4`.

## Current scope

The checker detects invalid UTF-8, invalid JSON, non-JSON-RPC envelopes, and unterminated stdout records. It reads stdout incrementally and stops at the configured line or total-output bound even when a producer never emits a newline. It accepts server-initiated JSON-RPC requests, notifications, success responses, and error responses. Synthetic tests cover startup, late, post-response, and descendant cleanup contamination, plus timeout and bounded-output failures.

The review gate pins the Inspector CLI 1.0.1, mcp-compliance 0.16.3, and
mcp-z 1.0.5 in an isolated npm fixture. All three accept the clean control and
the startup, late, and cleanup contamination fixtures; `mcp-stdio-purity`
accepts only clean and returns `MSP001` for all three contaminated streams.
The CLI-only Inspector package intentionally avoids the unrelated web UI and
server dependency trees. The comparison dependencies are test-only and are not
included in release archives or the Action runtime.

Streamable HTTP, client emulation, automatic logger fixes, hosted operation, and Windows process-tree support are not V1 goals.

## Uninstall and rollback

Delete the installed binary or remove the Action step. Roll back by restoring the prior full commit SHA or release archive; the checker does not create configuration or state.

## License

MIT. See [LICENSE](LICENSE).
