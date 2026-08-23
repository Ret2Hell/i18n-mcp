# JSON implementation benchmark

This opt-in benchmark compares `encoding/json` with ByteDance Sonic without changing production code. It uses the `jsonbench` build tag so Sonic and its runtime warning do not affect the normal test suite.

> Sonic v1.15.2 supports optimized execution on the project's Go 1.26.7 toolchain. The preflight test and benchmarks still verify that native Sonic is active so fallback measurements cannot be mistaken for real Sonic results.

It models the JSON work performed by i18n-mcp:

- typed translation-batch marshal, indented marshal, and unmarshal;
- dynamic locale-file decoding with `UseNumber` and rejection of trailing values;
- string quoting used by JSON locale editing;
- JSON validation;
- small (10), medium (100), and large (1,000) translation/key sets.

Two Sonic modes are included:

- `sonic_std`: `sonic.ConfigStd`, the appropriate replacement when standard-library behavior matters;
- `sonic_default`: faster defaults that may differ in HTML escaping, map-key ordering, and string handling.

`TestCompatibility` checks representative typed round trips and dynamic-number decoding before measurements. It is not an exhaustive compatibility proof.

## Go 1.26.7 results

Measured on Linux/amd64 with an AMD Ryzen AI 9 365. Each result is the median of six 750 ms samples. The table reports the throughput improvement of `sonic.ConfigStd` over `encoding/json`; this is the compatibility-oriented mode selected for production integration.

| Operation | 10 items | 100 items | 1,000 items |
| --- | ---: | ---: | ---: |
| Compact marshal | 1.31x | 1.65x | 1.84x |
| Indented marshal | 1.14x | 1.02x | 1.13x |
| Typed unmarshal | 2.19x | 2.76x | 2.93x |
| Dynamic locale decode | 2.13x | 2.16x | 1.99x |
| JSON validation | 9.11x | 8.91x | 9.52x |

The gains have memory trade-offs. For 1,000 typed items, `ConfigStd` unmarshal used 1.52 MB/op versus 1.09 MB/op for the standard library, while compact marshal used 518 KB/op versus 419 KB/op. Dynamic locale decoding used less memory with Sonic: 246 KB/op versus 316 KB/op for 1,000 keys.

Small standalone string marshaling should remain on `encoding/json`: short strings took 101 ns/op with `ConfigStd` versus 63 ns/op with the standard library, and escaped strings took 178 ns/op versus 139 ns/op. `ConfigDefault` can decode typed data faster, but its map ordering, HTML escaping, and string-handling differences are unsuitable for deterministic hashes and serialized output.

## Run

Keep the machine idle and disable power-saving or thermal throttling where practical. First verify compatibility:

```sh
go test -tags=jsonbench ./benchmarks/json \
  -run '^(TestSonicNativeImplementation|TestCompatibility)$' -count=1
```

Collect statistically useful samples for later comparison:

```sh
go test -tags=jsonbench ./benchmarks/json \
  -run '^$' -bench '.' -benchmem \
  -benchtime=750ms -count=6 \
  | tee /tmp/json-bench.txt
```

Summarize variance and relative performance:

```sh
go run golang.org/x/perf/cmd/benchstat@latest /tmp/json-bench.txt
```

For quick iteration, use `-benchtime=200ms -count=1`. To isolate a production pattern, use a regular expression such as:

```sh
go test -tags=jsonbench ./benchmarks/json -run '^$' \
  -bench '^BenchmarkDecodeDynamicLocale' -benchmem -benchtime=1s -count=10
```

Interpret `ns/op`, `B/op`, `allocs/op`, and the reported throughput together. Prefer `sonic_std` when estimating a low-risk migration. JSON speedups only affect the JSON portion of end-to-end operations; filesystem access, analysis, network calls, hashing, and MCP handling can dominate total latency.

Sonic uses architecture-specific optimized code. Run this on every deployment architecture and Go version that matters, and compare binary size/startup separately before deciding to migrate.
