# Benchmarks

Application benchmarks live here, separate from production packages.

- [MCP request-path benchmark](mcp/README.md): actual HTTP tool calls against the current production implementation, plus an archived backend comparison.

Benchmarks require the `mcpbench` build tag and are excluded from normal tests. Use the runner documented above to validate and measure the current application. The old isolated JSON microbenchmarks have been removed; JSON-only throughput was not representative of MCP latency.
