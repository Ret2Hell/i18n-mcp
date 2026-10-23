# Bounded-analysis validation

## Scope

Tested the current active MCP features against OteraX using the Inspector CLI over loopback Streamable HTTP. The checkout had 28 locale files, 27 target locales, and 235,872 diff records. Existing user edits to English were included. All project operations were read-only; state rebuild used its default dry-run mode. The project was not modified.

The original 16.6 GiB failure involved concurrent dead-key/report analysis. Dead-key tools remain disabled; this change does **not** claim to repair their dormant classifier. The measurements below validate bounded active tools and resources instead.

## Measurements

Wall times include starting a fresh `bunx @modelcontextprotocol/inspector` process for each call. Response sizes are CLI JSON-envelope bytes. These are single-run measurements, not statistical latency guarantees.

| Operation | Previous response size | New response size | New wall time |
| --- | ---: | ---: | ---: |
| Default diff | 191,460,500 B | 24,978 B | 3.166 s |
| Default translation plan | 13,186,970 B | 21,695 B | 2.386 s |
| State rebuild preview | 236,274,534 B | 7,029 B | 1.376 s |
| Compact JSON report | Previous full-project request timed out | 20,614 B | 2.390 s |
| Compact Markdown report | Previous full-project request timed out | 24,909 B | 2.297 s |
| Filtered French missing-key diff, 10 rows | — | 7,948 B | 0.729 s |
| Diff continuation resource | — | 27,869 B | 2.880 s |
| Latest compact report resource | — | 22,694 B | 0.537 s |
| French namespace resource | — | 21,116 B | 0.599 s |
| Locale inventory with unit page | — | 23,455 B | 1.522 s |

Peak server RSS, sampled from `/proc/<pid>/status` every 20 ms across sequential requests and a concurrency probe, was **352,052 KiB (~344 MiB)**. This is observed RSS, not a hard process-memory guarantee. The server still loads and flattens the selected project files; input size matters.

### Concurrent work and cancellation

Four concurrent full-project diff requests produced one successful bounded page and three explicit busy responses (~0.74 s including Inspector startup). Expensive operations are rejected rather than queued when another operation is active. Health and catalogs do not consume the work slot.

A raw HTTP diff request was disconnected after 100 ms. The next filtered diff request through Inspector succeeded in 0.656 s, with no busy error. Unit tests additionally cover cancellation before parsing/flattening and before state writes, plus capacity release.

### Full local export

`audit --output json --full` streamed **235,872 rows** into a 93,655,568-byte JSON file in **2.599 s**, with sampled peak RSS **258,180 KiB (~252 MiB)**. The output parsed successfully and contained the complete diff counts. Exit status was 1 because the existing project violated missing/invalid translation CI policy, not because export failed.

JSON exports retain the selected inventory and state needed for analysis, but do not accumulate the full diff result or cache the exported data. Full Markdown remains an explicit in-memory export.

## Reproduce

Build and start a server against a trusted project:

```bash
go build -o /tmp/i18n-mcp ./cmd/i18n-mcp
/tmp/i18n-mcp serve http --project /path/to/OteraX --addr 127.0.0.1:17359
```

In another terminal:

```bash
bunx @modelcontextprotocol/inspector --cli http://127.0.0.1:17359/mcp \
  --transport http --method tools/call --tool-name i18n.keys.diff \
  --format json > /tmp/diff-page.json

jq '.result.structuredContent.report | {summary, page}' /tmp/diff-page.json

bunx @modelcontextprotocol/inspector --cli http://127.0.0.1:17359/mcp \
  --transport http --method resources/read \
  --uri "$(jq -r '.result.structuredContent.report.page.nextResource' /tmp/diff-page.json)" \
  --format json > /tmp/diff-next-page.json

bunx @modelcontextprotocol/inspector --cli http://127.0.0.1:17359/mcp \
  --transport http --method tools/call --tool-name i18n.report.generate \
  --tool-arg format=json --format json > /tmp/report-summary.json

# Explicit full local export; inspect exit status for configured CI failures.
/tmp/i18n-mcp audit --project /path/to/OteraX --output json --full > /tmp/i18n-audit.json
```

Raw benchmark results from the development run were retained locally under `/tmp/i18n-bounded-benchmark/`; they are not repository fixtures.

## Regression coverage and limits

Tests cover page completeness without duplicate records, filter-scoped counts, stale/project/filter/page-size cursor rejection, serialized item budgets, oversize-item errors, default and hard planning limits, resource continuation, compact state/report serialization, full-export I/O failure, owner-isolated bounded latest-record stores, early file filtering, cancellation, and busy admission.

No full analysis cache was introduced. Pages recompute their filtered snapshot, trading some repeated CPU for bounded retention and reliable content-based cursor invalidation. Large individual inputs, unusually large metadata, very deep JSON, and writes with large patch previews are not made free by output pagination. The 128 KiB network budget specifically covers diff, locale, planning, report, and resource result envelopes; it is not a universal limit on every write-tool patch. Translation validation false positives from the earlier audit are separate work and remain unresolved.
