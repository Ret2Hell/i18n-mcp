# JSON v2 migration

Application code and tests use Go 1.27's `encoding/json/v2` and `encoding/json/jsontext` directly. Sonic, its architecture-specific backend/fallback, and `internal/jsonutil` have been removed. There are no v1 compatibility options or experimental build flags in application code.

## Application behavior

- JSON input rejects duplicate member names, including equivalent escaped names, invalid UTF-8, and trailing values or junk.
- Struct field matching is case-sensitive. Configuration and authenticated request-state payloads also reject unknown members, including incorrectly capitalized field names.
- Nil maps and slices serialize as `{}` and `[]`, rather than `null`. The report golden fixture records this intentional change.
- `omitempty` follows v2's JSON-value rules. Fields whose zero Go value must be omitted use `omitzero`; in particular, signed request-state data retains explicit empty objects/arrays rather than dropping them.
- Raw JSON values use `jsontext.Value`. Locale parsing preserves numeric spellings exactly, even nested large integers, decimal literals, and exponents, without converting them through `float64`.
- Ordered locale editing still preserves member order, raw numeric spellings, and configured formatting. Its parser/renderer now enforce v2 syntax and propagate invalid-string errors instead of silently replacing content.

Deterministic ordering and HTML/JavaScript escaping are explicit options at application encoding call sites. Determinism is important for digests, cursors, saved files, and reproducible output. Escaping also keeps response-budget estimates aligned with the MCP SDK's wire serialization. There is no wrapper package hiding these policies.

Full report exports use a single `jsontext.Encoder`, writing structural tokens and marshaling each row as it is produced. They do not build a complete JSON result in memory. Writer errors and cancellation remain observable; partial output must not be consumed after a failed export.

Preview digest encoding failures are returned before snapshots are published or verified. A partial encoding must never become an approval digest.

## Upgrade considerations

Valid existing configuration/state files remain readable without a state-format version bump. Fix duplicate keys, invalid text, incorrectly capitalized configuration names, and trailing data if previously tolerated files are rejected.

Restart the service after upgrading and regenerate active cursors, translation plans, and patch previews. Serialization changes may invalidate handles that bind serialized inputs; do not rely on approval tokens carrying across versions.

## Dependency boundary

This migration covers code owned by this repository. The MCP SDK, Viper, and other third-party dependencies retain their own JSON implementations and compatibility behavior; they were not forked or rewritten. In particular, JSON handled entirely by the SDK is not guaranteed to have the application's v2 defaults. There is no Sonic module or code dependency remaining, but `encoding/json` and other JSON libraries can still appear in the transitive dependency graph.

The [archived benchmark](../benchmarks/mcp/README.md) compared the old Sonic wrapper with a v1-compatible v2 adapter. Those measurements are historical, not measurements of this native-v2 implementation. The current benchmark runner measures the migrated application without backend overlays.

## Verification

```sh
go test -tags=mcpbench ./... -count=1
CGO_ENABLED=1 go test -race -tags=mcpbench ./... -count=1
golangci-lint run --build-tags=mcpbench ./...
uv run --no-project python benchmarks/mcp/run.py --validate-only
```

Regression tests cover strict locale/configuration decoding, exact numeric literals, authenticated malformed request-state payloads, empty raw signed data, deterministic preview digests, encoding failures, ordered editing, streamed export errors/cancellation, and real MCP tool calls. Cross-builds for 386 and arm64 also verify that the old architecture fallback is no longer needed.
