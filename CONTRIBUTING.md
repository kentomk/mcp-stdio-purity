# Contributing

Thank you for helping make stdio MCP failures predictable.

Keep contributions limited to raw stdout purity for an explicitly supplied server command. Do not add client emulation, hosted telemetry, framework-specific auto-fixes, or real production logs and credentials.

Before submitting a change, add or update a synthetic fixture and run:

```sh
gofmt -w .
scripts/quality-gate.sh
```

Reports and fixtures must never reproduce secret values or raw invalid stdout. By contributing, you agree that your contribution is licensed under the MIT License.
