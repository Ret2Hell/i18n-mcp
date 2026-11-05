# MCP request-path JSON backend benchmark

This benchmark measures real MCP tool calls against the current production implementation, now using JSON v2 directly. The old JSON wrapper and Sonic backend have been removed. The comparison below is historical; it must not be read as a benchmark of the native-v2 implementation. The old isolated JSON microbenchmarks were removed because their results were misleading as a proxy for MCP performance.

## What actually runs

`benchmarks/mcp/mcp_benchmark_test.go` uses:

- the production `httpserver.NewHandler`, stateless Streamable HTTP/SSE configuration, and `mcpserver.New`;
- a real SDK client talking to a loopback HTTP server, including SDK request/result serialization;
- actual configuration loading, locale discovery/reading, validation, diff analysis, translation planning, signed cursors/batch handles, pagination, and response-budget middleware;
- the production one-expensive-operation-at-a-time admission limit.

A generated project has **1,000 source keys and 500 translated keys across two namespaces**, in four locale JSON files. Messages include `{name}` placeholders and Unicode. It exercises `i18n.health`, `i18n.locales.list`, `i18n.keys.diff`, and `i18n.translation.plan`, with requested page sizes 50 and 200. The byte budget limits the 200-item translation plan to **195 returned items**; correctness assertions require this exact count under every backend. Other pages return the requested count, with continuation cursors.

## Historical backend comparison

The comparison below was measured before the native-v2 migration and documented at commit `4cff2a0`. It is an archive, **not a measurement of the current application**. At that revision, three separately built binaries used these application backends:

| Backend | Application wrapper |
| --- | --- |
| `sonic` | Unmodified production Sonic `ConfigStd`, v1.15.5 |
| `stdlib_v1` | Existing standard-library fallback wrapper, enabled on this architecture |
| `stdlib_v2_compat` | Benchmark-only v2 adapter with `DefaultOptionsV1()`, including exact `json.Number` streaming decoding |

The historical runner used Go's `-overlay` to substitute **only** `internal/jsonutil/json_sonic.go`, without editing production sources. At that revision, direct `encoding/json` imports in the application and SDK were left unchanged. The subsequent migration removed the wrapper and migrated all first-party call sites; third-party dependencies retain their own serializers. All application tests, plus the benchmark's page-count/budget checks, passed under every historical backend before measurement. Native v2 defaults are deliberately **not** benchmarked here: they change the application contract and require a separate migration, not a drop-in backend switch. The v2 adapter is a compatibility implementation, not a fully optimized refactor to native v2 call sites.

## Measurement modes

- **`warm`**: repeated successful calls over an established client connection and initialized app. Fixture creation, connection setup, and an initial correctness call are excluded.
- **`fresh_lifecycle`**: each timed iteration constructs a fresh App/server/client connection, calls the tool once, and closes everything. This includes schema registration, SDK connection/discovery, and teardown. It is **not a new OS process**, cold JIT/type-cache measurement, or cold-disk measurement.
- **`saturated`**: four concurrent callers share the same server and SDK client session (with an HTTP connection pool). The production admission limiter rejects overlapping expensive work. Report `success/s`, `busy-%`, and successful-call p50/p95; **do not treat `ns/op` or `B/op` here as successful-request cost**, because those include rejected attempts. Health is exempt from the limiter.

The selected inventory/diff services reread files rather than retaining a full analysis cache. “Warm” means process/connection/OS-page-cache reuse, not an invented application-cache hit. This benchmark does not flush the OS page cache.

`ns/op` is mean elapsed operation cost (or amortized attempt cost under saturation); p50/p95 are per-sample observed successful-call latency percentiles. Allocation counts cover **both server and client** in the same process. The latency sample buffer is allocated outside the timer; timestamping and per-call success checks are included. Per-sample p95 estimates, especially fresh-lifecycle/saturation samples with fewer successful calls, are descriptive, not production SLO estimates.

## Results: Go 1.27.3 / Sonic v1.15.5

Linux/amd64, AMD Ryzen AI 9 365. Six independent 1-second samples per case/backend, `GOMAXPROCS=4`, affinity to logical CPUs 8–11. Backend order rotates across repetitions to reduce thermal/order bias. Generated environments, test logs, raw samples, and comparison output are intentionally not tracked in Git; only the historical summary below is retained. Comparisons were generated with `golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e`.

**The end-to-end difference is much smaller than the isolated JSON speedups.** Sonic retains a modest advantage in several measured tool paths, but many differences are within run-to-run noise. These results do not justify claims of a 2–3x faster MCP server.

### Established connection: mean request cost

Medians of six `ns/op` samples, expressed in **milliseconds per request**:

| Tool | Requested / returned items | Sonic | JSON v1 | v2 compatible | v2 vs Sonic |
| --- | ---: | ---: | ---: | ---: | --- |
| Locales | 50 / 50 | 6.99 | 7.76 | 8.17 | +16.9%, inconclusive |
| Diff | 50 / 50 | 19.27 | 20.06 | 20.31 | +5.4%, inconclusive |
| Plan | 50 / 50 | 20.03 | 20.91 | 21.40 | **+6.8%**, p=0.004 |
| Locales | 200 / 200 | 11.99 | 13.32 | 13.40 | **+11.7%**, p=0.009 |
| Diff | 200 / 200 | 25.60 | 27.28 | 27.19 | +6.2%, inconclusive |
| Plan | 200 / 195 | 27.93 | 30.52 | 30.95 | **+10.8%**, p=0.002 |

“Inconclusive” means benchstat did not detect a mean-cost difference at p<0.05 with these six samples; it does **not** establish equivalence. Comparisons are exploratory and not corrected for multiple testing. Health cost was 0.85 / 0.91 / 1.02 ms respectively; its absolute difference is small and it is not an analysis workload.

Successful-call latency, **median of per-sample p50 / p95 in milliseconds**:

| Tool / requested limit | Sonic p50 / p95 | v2 compatible p50 / p95 |
| --- | ---: | ---: |
| Locales / 50 | 6.96 / 9.04 | 7.89 / 10.30 |
| Diff / 50 | 19.12 / 22.21 | 20.10 / 23.30 |
| Plan / 50 | 19.87 / 23.04 | 21.27 / 24.64 |
| Locales / 200 | 11.69 / 15.05 | 13.16 / 16.02 |
| Diff / 200 | 25.51 / 28.68 | 27.27 / 30.33 |
| Plan / 200 | 27.91 / 31.89 | 30.45 / 36.01 |

At requested limit 200, total client+server allocation was **5.11 vs 5.02 MiB** (locales), **10.70 vs 10.56 MiB** (diff), and **12.96 vs 12.48 MiB** (plan), Sonic vs v2-compatible. V2 saved some bytes, but made roughly 1,600 more allocations per request in this fixture. Neither backend changes the SDK's substantial common overhead.

### Fresh application/connection lifecycle

At requested limit 200: Sonic vs v2-compatible mean costs were **38.44 vs 39.23 ms** for locales, **50.69 vs 54.04 ms** for diff, and **53.52 vs 56.78 ms** for planning. Only the diff difference was statistically detected in mean cost (+6.6%, p=0.002); most lifecycle comparisons were inconclusive.

This must not be presented as a cold-process startup comparison: process-global JSON caches/JIT and OS caches are already warm.

### Saturated single-project server

At requested limit 200, successful calls/second were:

| Tool | Sonic | v2 compatible | Busy rejection fraction (both) |
| --- | ---: | ---: | ---: |
| Locales | 51.26 | 44.81 | about 95% |
| Diff | 22.12 | 20.66 | about 98% |
| Plan | 18.78 | 18.71 | about 98% |

This is intentionally aggressive, retry-without-backoff traffic, **not normal client behavior**. It demonstrates the existing backpressure policy; reject traffic consumes resources and reduces successful throughput. The plan result is effectively indistinguishable here. A tiny amortized `ns/op` under overload is not evidence of fast tool execution.

## Historical conclusion and limits

These measurements supported trading a modest amount of latency for dependency simplicity and native v2 semantics. This fixture shows **roughly 7–12% higher warm mean cost for v2 in the statistically detected analysis cases**, not multi-fold differences. Diff and several lifecycle comparisons remain inconclusive. A native-v2 refactor or a change to SDK serialization is a different experiment.

This is substantially closer to actual MCP usage than JSON microbenchmarks, but it still uses a synthetic project and localhost transport. The fixture has no saved translation-state database, so it does not represent state-heavy projects. It does not measure remote network latency, TLS/OAuth, stdio, write/apply workflows, resource reads, continuation-page requests, multi-project traffic, real cold disk/process startup, or all project sizes. Authentication is disabled; the rest of the production HTTP handler is retained. Filesystem cache, machine load, and CPU frequency still introduce noise. Profile actual projects before treating small differences as universal.

## Run the current application

The runner now builds and measures the current production backend only; there are no source overlays or benchmark-only JSON implementations. Historical Sonic dependencies are not needed. On a machine with Go 1.27+, uv, and an appropriate CPU affinity:

```sh
uv run --no-project python benchmarks/mcp/run.py \
  --count 6 --benchtime 1s --cpu 4 --affinity 8,9,10,11 \
  --output /tmp/mcp-json-results

go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e \
  /tmp/mcp-json-results/application.txt
```

Omit `--affinity` or choose valid logical CPU IDs on another machine. `--validate-only` runs the application/workload tests without timing. `--bench` accepts a Go benchmark regex for focused runs. The runner records the Git revision and dirty state; compiled binaries live in a temporary directory.

To reproduce the **historical** three-backend experiment, check out `4cff2a0` in a separate worktree and follow its README. Do not interpret the archived v1-compatible v2 results as timings for the subsequent native-v2 migration.
