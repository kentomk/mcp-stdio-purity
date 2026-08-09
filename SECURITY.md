# Security policy

## Supported versions

The published `v0.1.4` release is supported. Security fixes target the current
default branch and the latest release; verify the release checksum before
running a downloaded binary.

## Reporting

Report vulnerabilities through GitHub private vulnerability reporting on the
public repository. Do not put tokens, environment values, raw MCP payloads,
server logs, or production command arguments in a public report. If private
reporting is unavailable, use a minimal synthetic reproducer and redact all
sensitive values.

## Security boundary

`mcp-stdio-purity` does not use a shell, network client, telemetry, or credential store. Reports omit raw stdout, stderr, environment values, and command arguments. The explicitly launched server inherits the caller environment and may access the network and filesystem; this tool does not sandbox it.

Treat payload disclosure, command invocation through a shell, unbounded output, hung child processes, or orphan descendants as security bugs. Linux and macOS commands run in a dedicated process group; timeout and cleanup-grace paths terminate that group and close the private stdout pipe.
