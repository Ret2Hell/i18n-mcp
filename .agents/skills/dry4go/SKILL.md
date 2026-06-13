---
name: dry4go
description: Use when the user asks for duplicate Go code detection, DRY analysis, structural similarity reports, or refactoring candidates caused by repeated Go functions or methods
---

# Using dry4go

## Overview

dry4go finds candidate duplicate Go code across files and directories. It compares Go functions and methods by normalized AST structure rather than raw text, so it can report similar code even when names, identifiers, selectors, and literal values differ.

Use it to find refactoring opportunities, repeated business logic, copied handlers, duplicated test helpers, or structurally similar functions that should be reviewed together.

## Setup

Run from the root of a Go module:

```bash
go run github.com/unclebob/dry4go/cmd/dry4go@latest .
```

Or install it:

```bash
go install github.com/unclebob/dry4go/cmd/dry4go@latest
dry4go .
```

## Usage

```bash
# Search the current module for structurally similar functions and methods
dry4go .

# Compare specific files and directories in one search set
dry4go internal/foo/foo.go internal/bar ./cmd

# Raise the similarity threshold for stricter matches
dry4go --threshold 0.9 ./internal ./cmd

# Lower minimum size filters to catch smaller duplicates
dry4go --min-lines 3 --min-nodes 12 .

# Emit machine-readable output for scripts or follow-up tooling
dry4go --json --threshold 0.9 ./internal
```

When an argument is a directory, dry4go recursively includes `.go` files under that directory and skips `.git`, `vendor`, and `target` directories.

## Options

| Option | Meaning |
|--------|---------|
| `--threshold N` | Minimum structural similarity score, default `0.82` |
| `--min-lines N` | Minimum source lines in a candidate function, default `4` |
| `--min-nodes N` | Minimum normalized syntax nodes, default `20` |
| `--format F` | Output format: `text` or `json`, default `text` |
| `--json` | Same as `--format json` |
| `--text` | Same as `--format text` |

## Output

Default text output is optimized for quick review:

```text
DUPLICATE score=0.89
  internal/billing/invoice.go:12-25
  internal/billing/receipt.go:30-44
```

JSON output is useful when another tool or script should consume the candidates:

```json
{
  "candidates": [
    {
      "score": 0.8909090909090909,
      "left": {"file": "internal/billing/invoice.go", "start_line": 12, "end_line": 25},
      "right": {"file": "internal/billing/receipt.go", "start_line": 30, "end_line": 44},
      "left_nodes": 88,
      "right_nodes": 91
    }
  ]
}
```

## Interpreting Results

| Signal | Meaning | Action |
|--------|---------|--------|
| High score near `1.0` | Very similar normalized structure | Review for extraction or consolidation |
| Medium score near threshold | Similar flow with meaningful differences | Compare intent before refactoring |
| Many matches in one package | Repeated local pattern | Consider shared helper or interface |
| Matches across packages | Repeated domain logic | Check whether duplication is intentional boundary separation |

## Workflow

1. Run `go test ./...` first so the project is in a known-good state.
2. Run `dry4go .` for a broad duplicate scan.
3. Increase strictness with `dry4go --threshold 0.9 .` when the first report is noisy.
4. Use `dry4go --json ...` if you need to summarize or post-process candidates.
5. For each candidate, compare intent before changing code; structural similarity is a review signal, not an automatic refactor instruction.
6. Refactor only when the duplicated code represents the same concept or policy.
7. Rerun `go test ./...` and the focused `dry4go` command after refactoring.

## Notes

dry4go normalizes identifiers, local names, selector names, and literal values. It preserves structural Go syntax such as function shape, statement order, control flow, assignments, calls, composite literals, and operators.

Do not treat every reported duplicate as bad. Some duplication is appropriate when packages intentionally avoid coupling or when similar code serves different domain concepts.
