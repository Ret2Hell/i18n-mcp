# Benchmarks

Application benchmarks live here, separate from production packages.

- [MCP request-path benchmark](mcp/README.md): actual HTTP tool calls, comparing the application JSON backends without rewriting production sources.

Benchmarks require the `mcpbench` build tag and are excluded from normal tests. Use the runner documented above to build and validate all backend variants. The old isolated JSON microbenchmarks have been removed; JSON-only throughput was not representative of MCP latency.
