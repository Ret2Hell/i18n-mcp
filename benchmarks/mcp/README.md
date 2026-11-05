# MCP request-path JSON backend benchmark

This benchmark compares the effect of the **application's JSON wrapper** on real MCP tool calls. The old isolated JSON microbenchmarks were removed because their results were misleading as a proxy for MCP performance.

## What actually runs

`benchmarks/mcp/mcp_benchmark_test.go` uses:

- the production `httpserver.NewHandler`, stateless Streamable HTTP/SSE configuration, and `mcpserver.New`;
- a real SDK client talking to a loopback HTTP server, including SDK request/result serialization;
- actual configuration loading, locale discovery/reading, validation, diff analysis, translation planning, signed cursors/batch handles, pagination, and response-budget middleware;
- the production one-expensive-operation-at-a-time admission limit.

A generated project has **1,000 source keys and 500 translated keys across two namespaces**, in four locale JSON files. Messages include `{name}` placeholders and Unicode. It exercises `i18n.health`, `i18n.locales.list`, `i18n.keys.diff`, and `i18n.translation.plan`, with requested page sizes 50 and 200. The byte budget limits the 200-item translation plan to **195 returned items**; correctness assertions require this exact count under every backend. Other pages return the requested count, with continuation cursors.

Three separately built binaries use these application backends:

| Backend | Application wrapper |
| --- | --- |
| `sonic` | Unmodified production Sonic `ConfigStd`, v1.15.5 |
| `stdlib_v1` | Existing standard-library fallback wrapper, enabled on this architecture |
| `stdlib_v2_compat` | Benchmark-only v2 adapter with `DefaultOptionsV1()`, including exact `json.Number` streaming decoding |

The runner uses Go's `-overlay` to substitute **only** `internal/jsonutil/json_sonic.go`. No production source files are edited. Direct `encoding/json` imports in our application and the SDK remain unchanged, as they would when migrating just this wrapper. All application tests, plus the benchmark's page-count/budget checks, must pass under every backend before measurement. Native v2 defaults are deliberately **not** benchmarked here: they change the application contract and require a separate migration, not a drop-in backend switch. The v2 adapter is a compatibility implementation, not a fully optimized refactor to native v2 call sites.

## Measurement modes

- **`warm`**: repeated successful calls over an established client connection and initialized app. Fixture creation, connection setup, and an initial correctness call are excluded.
- **`fresh_lifecycle`**: each timed iteration constructs a fresh App/server/client connection, calls the tool once, and closes everything. This includes schema registration, SDK connection/discovery, and teardown. It is **not a new OS process**, cold JIT/type-cache measurement, or cold-disk measurement.
- **`saturated`**: four concurrent callers share the same server and SDK client session (with an HTTP connection pool). The production admission limiter rejects overlapping expensive work. Report `success/s`, `busy-%`, and successful-call p50/p95; **do not treat `ns/op` or `B/op` here as successful-request cost**, because those include rejected attempts. Health is exempt from the limiter.

The selected inventory/diff services reread files rather than retaining a full analysis cache. “Warm” means process/connection/OS-page-cache reuse, not an invented application-cache hit. This benchmark does not flush the OS page cache.

`ns/op` is mean elapsed operation cost (or amortized attempt cost under saturation); p50/p95 are per-sample observed successful-call latency percentiles. Allocation counts cover **both server and client** in the same process. The latency sample buffer is allocated outside the timer; timestamping and per-call success checks are included. Per-sample p95 estimates, especially fresh-lifecycle/saturation samples with fewer successful calls, are descriptive, not production SLO estimates.

## Reproduce

On an amd64/arm64 machine with Go 1.27+, uv, and an appropriate CPU affinity:

```sh
uv run --no-project python benchmarks/mcp/run.py \
  --count 6 --benchtime 1s --cpu 4 --affinity 8,9,10,11 \
  --output /tmp/mcp-json-results

go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e \
  /tmp/mcp-json-results/{sonic,stdlib_v1,stdlib_v2_compat}.txt
```

Omit `--affinity` or choose valid logical CPU IDs on another machine. `--validate-only` runs/builds the three backends without timing. `--bench` accepts a Go benchmark regex for focused runs. Production and benchmark files are not rewritten by the runner; overlays and compiled binaries live in a temporary directory.
