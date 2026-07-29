# Changelog

All notable changes will be documented here.

## Unreleased

- Point source installs, release downloads, and the composite Action example to
  the current verified `v0.1.2` release and its immutable public main revision.
- Repair the published install path to use `v0.1.1`, an immutable Action
  revision, and single-archive checksum verification on Linux and macOS.
- Add release and publisher regressions for the copy-ready verification path.
- Replace the full Inspector test dependency with the pinned CLI-only package
  after a transitive high-severity denial-of-service advisory blocked the gate.
- Make top-level `--help`, `-h`, `help`, and check help return a stable usage contract on stdout with exit code `0`.
- Add the initial shell-free stdio checker.
- Add content-safe text and JSON reports with stable `MSP001` diagnostics.
- Add clean, startup-banner, and late-log synthetic fixture coverage.
- Add capability-derived probes and server-initiated envelope coverage.
- Bound timeout, cleanup grace, stdout size, line size, and diagnostic count.
- Capture descendant cleanup output and terminate lingering Unix process groups.
- Add the source-built composite Action with literal newline-delimited arguments.
- Add reproducible Linux/macOS amd64/arm64 archives and `SHA256SUMS`.
- Add race, license, secret-pattern, Action, and release workflow gates.
- Pin and reproduce the Inspector, mcp-compliance, and mcp-z false-green comparison.
- Add clean-checkout quickstart and publisher payload gates.
- Align the README quick-start heading with the publisher's machine contract and keep the 60-second promise explicit.
