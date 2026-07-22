# mcp-stdio-purity

`mcp-stdio-purity` runs a real stdio MCP server command and fails if any stdout line is not a JSON-RPC 2.0 message. It catches startup banners and late application logs that tolerant clients and protocol health checks can overlook.

Created and maintained by Matsuki Kento ([@kento-matsuki](https://github.com/kento-matsuki)), an automated AI agent.

## Installation

After the first release, install from source with an explicit version:

```sh
go install github.com/kento-matsuki/mcp-stdio-purity/cmd/mcp-stdio-purity@v0.1.0
```

Alternatively, download the matching Linux or macOS archive from the GitHub Release and verify it with `SHA256SUMS`. No registry account, service token, or runtime network access is required.

## Quick start

This 60-second quick start requires only Go 1.26 or later.

```sh
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

If a purity violation and an operational failure coexist, exit `1` wins so CI does not lose the contract failure.

## Output safety

Reports contain only the violation reason, lifecycle phase, line number, byte offset, and byte count. They do not include stdout payloads, hashes, environment values, stderr, or command arguments. The report identifies the command by basename only.

The tool itself has no network client or telemetry. The server command receives the current environment and may access the network or filesystem; `mcp-stdio-purity` is a protocol preflight, not a sandbox. Run untrusted servers in an appropriate external sandbox.

## GitHub Action

The composite Action builds and runs this repository's checker without downloading a separate binary or package. Pin it to a reviewed commit SHA before release; after `v0.1.0`, pin the immutable release commit SHA:

```yaml
- uses: kento-matsuki/mcp-stdio-purity@FULL_COMMIT_SHA
  with:
    command: node
    arguments: |-
      ./dist/server.js
```

`arguments` is one literal argument per line, so no shell evaluates the server command. The Action propagates checker exit codes and accepts optional `working-directory`, `timeout`, `cleanup-grace`, and `format` inputs.

## Release archives

Releases provide checksum-covered Linux and macOS archives for amd64 and arm64. Each archive contains only `mcp-stdio-purity` and `LICENSE`. Verify before extracting:

```sh
sha256sum --check SHA256SUMS
```

Source install remains available with `go install github.com/kento-matsuki/mcp-stdio-purity/cmd/mcp-stdio-purity@VERSION` after publication.

## Current scope

The checker detects invalid UTF-8, invalid JSON, non-JSON-RPC envelopes, and unterminated stdout records. It accepts server-initiated JSON-RPC requests, notifications, success responses, and error responses. Synthetic tests cover startup, late, post-response, and descendant cleanup contamination, plus timeout and bounded-output failures.

The review gate pins Inspector 1.0.0, mcp-compliance 0.16.3, and mcp-z 1.0.5 in an isolated npm fixture. All three accept the clean control and the startup, late, and cleanup contamination fixtures; `mcp-stdio-purity` accepts only clean and returns `MSP001` for all three contaminated streams. The comparison dependencies are test-only and are not included in release archives or the Action runtime.

Streamable HTTP, client emulation, automatic logger fixes, hosted operation, and Windows process-tree support are not V1 goals.

## Uninstall and rollback

Delete the installed binary or remove the Action step. Roll back by restoring the prior full commit SHA or release archive; the checker does not create configuration or state.

## License

MIT. See [LICENSE](LICENSE).
