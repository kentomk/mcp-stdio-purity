# Changelog

All notable changes will be documented here.

## Unreleased

- Pin the copy-ready composite Action to the immutable v0.1.4 release revision instead of moving public main, preventing routine maintenance commits from invalidating the documented integration.

- Align the copy-ready composite Action example and stale-pin regression with
  public main `bfb1959a09065decc4d060bbb04f34a2ce1054f0`.
- Reconcile the copy-ready composite Action example with the post-publication
  public main `34b74273667d360470794b0777a92284ece104a9`.
- Align source-install, archive, checksum, security, and composite Action examples with the published `v0.1.4` release and current public main.
- Align the copy-ready composite Action example and publisher regression with the latest broker-verified public main revision.
- Align the copy-ready composite Action example and publisher regression with the latest broker-verified public main revision.
- Align the copy-ready composite Action example and publisher regression with the latest broker-verified public main revision.
- Align the copy-ready composite Action example and publisher regression with the latest broker-verified public main revision.
- Align the copy-ready composite Action example and its publisher regression with the broker-verified public main revision used by the current release path.
- Give integration fixtures a bounded two-second cleanup window so the full lifecycle suite remains stable under race instrumentation without changing the CLI's 250 ms default.
- Count a long newline-delimited stdout record as one line even when the bounded reader receives it in multiple fragments.
- Refresh the immutable Action example to public main `b7721b9b` and reject the superseded `563d134` pin in the publisher contract.
- Fix the copy-ready archive install example to use the extracted directory and create the user binary directory in a fresh home.
- Expose the CLI's stdout line, total-output, and diagnostic-count safety limits through the composite Action, with smoke coverage for exact forwarding.
- Align source-install, archive, checksum, release URL, and composite Action examples with the published `v0.1.3` release and its current public main revision.
- Pin the copy-ready composite Action example to the current successful public main and reject the superseded revision in the publisher contract.
- Fail closed when a manual or broker-triggered release repair omits the required tag name before packaging assets.
- Pin the composite Action's source build to Go `1.26.5`, matching CI and
  release workflows, and add a regression against mutable `go-version-file`
  resolution.
- Return a content-safe JSON or text exit `2` before building when the Action's
  `working-directory` is missing or is not a directory, with regression coverage
  that prevents the path from leaking into output.
- Add a copy-ready failure-triage path that separates stdout purity violations
  (exit `1`) from operational failures (exit `2`) without exposing payloads.
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
# Unreleased

- Report when bounded diagnostic output omitted additional stdout purity violations, so a low `--max-diagnostics` limit cannot be mistaken for a complete count.
