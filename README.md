# i18n-mcp

[![GitHub Release](https://img.shields.io/github/v/release/Ret2Hell/i18n-mcp?style=flat&color=blue)](https://github.com/Ret2Hell/i18n-mcp/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/Ret2Hell/i18n-mcp/ci.yml?label=CI)](https://github.com/Ret2Hell/i18n-mcp/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-blue)](go.mod)
[![MCP](https://img.shields.io/badge/MCP-stdio_%7C_streamable_HTTP-purple)](#mcp-client-configuration)
[![Tools](https://img.shields.io/badge/MCP_tools-16-orange)](#mcp-tools)
[![Docker](https://img.shields.io/badge/docker-ret2hell%2Fi18n--mcp-blue)](https://hub.docker.com/r/ret2hell/i18n-mcp)
[![Platform](https://img.shields.io/badge/macOS_%7C_Linux_%7C_Windows-supported-lightgrey)](https://github.com/Ret2Hell/i18n-mcp/releases/latest)
[![License](https://img.shields.io/badge/license-see_LICENSE-green)](LICENSE)

**An MCP server for inspecting, validating, translating, and safely updating JSON i18n locale files in frontend projects.** Detect locale layouts, bootstrap `.i18n-mcp.json`, find missing and stale translations, plan agent translation batches, validate placeholders and ICU shape, scan application source for dead keys, preview patches, and write only after explicit approval.

`i18n-mcp` supports both local stdio and MCP Streamable HTTP transports. It runs with the same filesystem permissions as the user or container process that starts it. Write tools are dry-run by default and require `apply: true` before changing files.

> **Security & Trust** - This tool reads project files and can write locale JSON, config, and state files when explicitly asked. The default path is preview-first: config writes, state rebuilds, translation applies, key prune, and key rename all return patch previews unless `apply: true` is provided. The HTTP server binds to loopback by default and refuses non-loopback binds unless authentication is enabled. The built-in static bearer-token verifier is intended only for local development; place production deployments behind a properly authenticated and authorized gateway. The server itself does not send locale content to translation APIs.

## Why i18n-mcp

- **Safe locale writes** - every write workflow is dry-run first, returns changed-file metadata and unified diffs, and writes only with `apply: true`.
- **Framework-aware i18n bootstrap** - detects project hints, JSON locale layouts, locales, namespaces, and a proposed `.i18n-mcp.json` config.
- **Value-aware stale detection** - tracks source hashes in `.i18n-mcp/state.json` instead of storing metadata inside locale files.
- **Translation planning, not blind rewriting** - builds deterministic batches for missing and stale keys, then validates proposed translations before apply.
- **Validation built for UI strings** - checks placeholders, HTML-like tags, ICU arguments, markdown-sensitive patterns, and empty targets.
- **Dead-key review workflow** - scans TS/TSX/JS/JSX/MJS/CJS usage, reports confidence, respects dynamic hints, and refuses unsafe prune by default.
- **CI-ready audits** - emits Markdown or JSON reports and exits according to configurable failure policy.
- **Flexible MCP transport** - use stdio for local coding agents or Streamable HTTP for network-capable MCP clients.

## Quick Start

**One-line install** (macOS / Linux shell):

```bash
curl -fsSL https://raw.githubusercontent.com/Ret2Hell/i18n-mcp/main/install.sh | bash
```

Open OpenCode, Codex, or Claude Code from your frontend project directory and start working. The installer configures supported agents automatically.

Run either transport manually if needed:

```bash
# Local stdio
i18n-mcp serve stdio --project /path/to/frontend-app

# Streamable HTTP at http://127.0.0.1:7339/mcp
i18n-mcp serve http --project /path/to/frontend-app
```

Ask your agent to detect the i18n project. The first MCP tool call is:

```json
{
  "name": "i18n.project.detect",
  "arguments": {}
}
```

## Coding Agent Support

The one-line installer configures these agents by default:

| Agent | Config written | Server project root |
| --- | --- | --- |
| OpenCode | `~/.config/opencode/opencode.json` | `.` from the OpenCode workspace |
| Codex CLI | `~/.codex/config.toml` | `.` from the Codex workspace |
| Claude Code | `~/.claude/.mcp.json` and `~/.claude.json` | `.` from the Claude workspace |

After install, restart or open your coding agent from the frontend project directory. The MCP server is registered as `i18n-mcp` and starts as:

```bash
i18n-mcp serve stdio --project .
```

Use `--skip-config` if you want to install only the binary and configure agents yourself.

## Docker

Pull the image:

```bash
docker pull ret2hell/i18n-mcp:latest
```

Run the stdio server:

```bash
docker run --rm -i ret2hell/i18n-mcp:latest serve stdio
```

For real project access, mount the project directory:

```bash
docker run --rm -i -v "$PWD:/workspace" -w /workspace ret2hell/i18n-mcp:latest serve stdio --project /workspace
```

Run Streamable HTTP on port 7339:

```bash
export I18N_MCP_DEV_TOKEN='replace-with-a-random-secret'
docker run --rm -p 127.0.0.1:7339:7339 \
  -e I18N_MCP_DEV_TOKEN \
  -v "$PWD:/workspace" -w /workspace \
  ret2hell/i18n-mcp:latest serve http \
  --project /workspace --addr 0.0.0.0:7339 --auth-required \
  --auth-resource http://127.0.0.1:7339/mcp \
  --dev-static-token-env I18N_MCP_DEV_TOKEN
```

This is a development-only bearer-token setup. Authentication is required because the process binds to a non-loopback address inside the container. Configure the client to send `Authorization: Bearer <token>`.

## Installation

### One-Line Installer

```bash
curl -fsSL https://raw.githubusercontent.com/Ret2Hell/i18n-mcp/main/install.sh | bash
```

### Go Run Without Installing

Run directly from the module:

```bash
go run github.com/Ret2Hell/i18n-mcp/cmd/i18n-mcp@latest --help
go run github.com/Ret2Hell/i18n-mcp/cmd/i18n-mcp@latest serve stdio --project /path/to/frontend-app
```

### Go Install

```bash
go install github.com/Ret2Hell/i18n-mcp/cmd/i18n-mcp@latest
```

Then run:

```bash
i18n-mcp --help
i18n-mcp serve stdio --project /path/to/frontend-app
```

### Build From Source

```bash
git clone https://github.com/Ret2Hell/i18n-mcp.git
cd i18n-mcp
go mod download
go test ./...
go build -o bin/i18n-mcp ./cmd/i18n-mcp
```

### Keeping Up to Date

Re-run the installer to replace the binary with the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/Ret2Hell/i18n-mcp/main/install.sh | bash
```

### Uninstall

Remove the installed binary and any project state you no longer want:

```bash
rm -f "$(command -v i18n-mcp)"
rm -rf /path/to/frontend-app/.i18n-mcp
```

Do not remove `.i18n-mcp.json` unless you also want to remove project configuration.

## MCP Client Configuration

The installer configures OpenCode, Codex CLI, and Claude Code automatically using stdio. For other MCP clients, choose either stdio or Streamable HTTP.

```json
{
  "mcpServers": {
    "i18n-mcp": {
      "command": "/absolute/path/to/i18n-mcp",
      "args": ["serve", "stdio", "--project", "/absolute/path/to/frontend-app"]
    }
  }
}
```

Use absolute paths so the client does not depend on its current working directory.

### Streamable HTTP

Start the HTTP server (the default endpoint is `http://127.0.0.1:7339/mcp`):

```bash
i18n-mcp serve http --project /absolute/path/to/frontend-app
```

Configure a Streamable HTTP-capable client with that URL. The exact configuration keys vary by client; a typical configuration looks like:

```json
{
  "mcpServers": {
    "i18n-mcp": {
      "type": "streamable-http",
      "url": "http://127.0.0.1:7339/mcp"
    }
  }
}
```

The server also exposes `GET /healthz`. Use `--addr` and `--path` to change the listen address and MCP endpoint. Loopback HTTP is unauthenticated by default. See [HTTP Deployment](#http-deployment) before binding to another interface.

### Docker-based stdio

For Docker-based clients using stdio, use `docker` as the command and pass the run arguments explicitly. Ensure the project is mounted into the container.

```json
{
  "mcpServers": {
    "i18n-mcp": {
      "command": "docker",
      "args": [
        "run",
        "--rm",
        "-i",
        "-v",
        "/absolute/path/to/frontend-app:/workspace",
        "-w",
        "/workspace",
        "ret2hell/i18n-mcp:latest",
        "serve",
        "stdio",
        "--project",
        "/workspace"
      ]
    }
  }
}
```

## First Project Workflow

1. Call `i18n.project.detect`.
2. Review the returned `proposedConfig`.
3. Call `i18n.config.write` with the proposed config and no `apply` flag to preview `.i18n-mcp.json`.
4. Review the patch preview.
5. Call `i18n.config.write` again with `apply: true` to write the config.
6. Call `i18n.config.validate`.
7. Call `i18n.state.rebuild` without `apply` to preview state bootstrap.
8. Call `i18n.state.rebuild` with `apply: true` after review.
9. Call `i18n.keys.diff` to inspect missing, stale, invalid, extra, and unknown keys.

## Features

### Project Detection & Config

- Detect framework hints, i18n libraries, JSON locale file layouts, locales, and namespaces.
- Generate a proposed `.i18n-mcp.json` config for review.
- Validate config through MCP or CLI schema generation.
- Preview config writes before applying them.

### Locale Inventory & Diff

- List locale files, namespaces, key counts, flattened keys, and duplicate namespace warnings.
- Compare source and target locales.
- Report `current`, `missing`, `stale`, `invalid`, `unknown`, and `extra` keys.

### Translation Workflow

- Build deterministic translation batches for missing and stale keys.
- Include configured style guide and glossary references in plans.
- Validate proposed translations against current source values.
- Reject source drift by default.
- Preview locale JSON and state patches before writing.

### Dead-Key Workflow

- Scan source usage in `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, and `.cjs` files.
- Detect literal calls, JSX `i18nKey`, and namespace-bound translation calls.
- Classify keys as `used`, `probably_unused`, `maybe_dynamic`, `ignored`, or `kept`.
- Respect `ignoredKeyPatterns`, `keptKeyPatterns`, and `dynamicKeyHints`.
- Prune exact keys only; unsafe statuses are refused unless explicitly overridden.

### CI & Reports

- Run non-interactive audits with Markdown or JSON output.
- Configure failure policy for missing, stale, invalid, and dead-key statuses.
- Generate reports through MCP and read the latest report resource.

## MCP Tools

| Tool | Description |
| --- | --- |
| `i18n.health` | Return server version, project root, and health status. |
| `i18n.project.detect` | Detect project hints, locale layouts, and proposed config. |
| `i18n.config.get` | Return resolved config with defaults and origin. |
| `i18n.config.validate` | Validate resolved config. |
| `i18n.config.write` | Preview or write `.i18n-mcp.json`; dry-run by default. |
| `i18n.locales.list` | List locale files, namespaces, key counts, and warnings. |
| `i18n.keys.diff` | Compare source and target locale keys. |
| `i18n.keys.usage_scan` | Scan source files for translation key usage evidence. |
| `i18n.keys.dead_report` | Classify likely dead keys. |
| `i18n.keys.prune` | Preview or remove exact locale keys; dry-run by default. |
| `i18n.keys.rename` | Preview or rename exact locale keys and state; dry-run by default. |
| `i18n.translation.plan` | Build a translation batch for missing and stale keys. |
| `i18n.translation.validate` | Validate proposed translations. |
| `i18n.translation.apply` | Preview or apply translations and update state; dry-run by default. |
| `i18n.state.rebuild` | Preview or rebuild `.i18n-mcp/state.json`; dry-run by default. |
| `i18n.report.generate` | Generate deterministic JSON or Markdown audit reports. |

## MCP Prompts

| Prompt | Purpose |
| --- | --- |
| `i18n_project_bootstrap` | Guide detection, config creation, and state bootstrap. |
| `i18n_translate_batch` | Translate a planned batch while preserving validation-sensitive syntax. |
| `i18n_review_translations` | Review proposed translations before apply. |
| `i18n_audit_dead_keys` | Review dead-key candidates and dynamic hints. |
| `i18n_ci_report_summary` | Summarize an audit report for CI or PR review. |
| `i18n_add_feature_keys` | Plan source keys and translations for a feature. |

## MCP Resources

| Resource | Description |
| --- | --- |
| `i18n://locales` | Locale inventory summary. |
| `i18n://locales/{locale}/{namespace}` | Raw JSON and flattened units for one locale namespace. |
| `i18n://analysis/diff` | Latest key diff report. |
| `i18n://analysis/usage` | Latest usage scan. |
| `i18n://analysis/dead-keys` | Latest dead-key report. |
| `i18n://translation/plan/latest` | Latest translation plan. |
| `i18n://reports/latest` | Latest generated audit report. |

## Configuration

`i18n-mcp` reads `.i18n-mcp.json` from the project root by default. You can pass a different config path with `--config` or `I18N_MCP_CONFIG`.

Generate the JSON Schema:

```bash
i18n-mcp schema > i18n-mcp.schema.json
```

### Complete Example

```json
{
  "$schema": "https://example.com/i18n-mcp.schema.json",
  "sourceLocale": "en",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "translationFunctions": ["t", "i18n.t"],
  "namespaceFunctions": ["useTranslations", "getTranslations"],
  "ignoredKeyPatterns": ["debug.*"],
  "keptKeyPatterns": ["legal.*"],
  "dynamicKeyHints": ["routes.*"],
  "format": {
    "sortKeys": false,
    "indent": 2,
    "trailingNewline": true
  },
  "translation": {
    "mode": "agent",
    "styleGuidePath": "docs/i18n-style.md",
    "glossaryPath": "docs/glossary.md"
  },
  "ci": {
    "failOnMissing": true,
    "failOnStale": false,
    "failOnInvalid": true,
    "failOnDeadKeys": false
  }
}
```

Replace the example `$schema` URL with your generated schema path or a published schema URL.

### Config Fields

| Field | Required | Description |
| --- | --- | --- |
| `$schema` | No | Optional schema URL for editor validation. |
| `sourceLocale` | Yes | Locale used as the source for translation and source hashes. |
| `targetLocales` | No | Target locales to maintain. Empty means infer from discovered locale files. |
| `localeFiles` | Yes | Project-relative JSON file patterns. Must include `{locale}` and may include `{namespace}`. |
| `defaultNamespace` | No | Namespace used when a file pattern has no `{namespace}` segment. |
| `translationFunctions` | No | Function names used by static usage scanning, such as `t` and `i18n.t`. |
| `namespaceFunctions` | No | Functions that bind namespaces, such as `useTranslations` and `getTranslations`. |
| `ignoredKeyPatterns` | No | Dead-key patterns to exclude from pruning recommendations. |
| `keptKeyPatterns` | No | Dead-key patterns that should always be retained. |
| `dynamicKeyHints` | No | Patterns that may be used dynamically and should be treated as unsafe to prune. |
| `format.sortKeys` | No | Sort JSON object keys when writing locale files. Default is `false`. |
| `format.indent` | No | Number of spaces for JSON indentation. Default is `2`. |
| `format.trailingNewline` | No | Write a trailing newline. Default is `true`. |
| `translation.mode` | No | Translation mode. Default is `agent`. Future values may include `provider` and `sampling`. |
| `translation.provider` | No | Provider name for future provider mode. Do not store credentials here. |
| `translation.styleGuidePath` | No | Project-relative style guide file used in translation plans. |
| `translation.glossaryPath` | No | Project-relative glossary file used in translation plans. |
| `ci.failOnMissing` | No | Fail `i18n-mcp audit` when missing translations exist. |
| `ci.failOnStale` | No | Fail `i18n-mcp audit` when stale translations exist. |
| `ci.failOnInvalid` | No | Fail `i18n-mcp audit` when invalid translations exist. |
| `ci.failOnDeadKeys` | No | Fail `i18n-mcp audit` when probably unused keys exist. |

### Common Locale Layouts

Single JSON file per locale:

```json
{
  "sourceLocale": "en",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["messages/{locale}.json"],
  "defaultNamespace": "common"
}
```

Namespace files per locale:

```json
{
  "sourceLocale": "en",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["messages/{locale}/{namespace}.json"]
}
```

Public locales layout:

```json
{
  "sourceLocale": "en",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["public/locales/{locale}/{namespace}.json"]
}
```

Source directory layout:

```json
{
  "sourceLocale": "en",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["src/locales/{locale}/{namespace}.json"]
}
```

### Validate And Write Config

Validate config through MCP:

```json
{
  "name": "i18n.config.validate",
  "arguments": {}
}
```

Preview writing config:

```json
{
  "name": "i18n.config.write",
  "arguments": {
    "config": {
      "sourceLocale": "en",
      "targetLocales": ["fr"],
      "localeFiles": ["messages/{locale}.json"],
      "defaultNamespace": "common"
    }
  }
}
```

`i18n.config.write` is dry-run by default. Review the returned patch, then call it again with `apply: true` to write `.i18n-mcp.json`.

## Stale Translation Workflow

Stale translation detection is value-aware. A target translation becomes stale when the source string changes after the target was translated.

### State File

`i18n-mcp` stores translation state in `.i18n-mcp/state.json`. Locale JSON files are not modified to store metadata.

Each state entry records:

- Locale.
- Namespace.
- Key.
- Current source hash.
- Hash the target translation was translated from.
- Target hash.
- Status and timestamps.

### Bootstrap State

After writing and validating config, rebuild state from existing translations.

Preview first:

```json
{
  "name": "i18n.state.rebuild",
  "arguments": {}
}
```

Apply after review:

```json
{
  "name": "i18n.state.rebuild",
  "arguments": {"apply": true}
}
```

### Detect Stale Translations

Run the diff tool:

```json
{
  "name": "i18n.keys.diff",
  "arguments": {}
}
```

Diff statuses include:

- `current`: target exists and matches current source state.
- `missing`: source key is absent from the target locale.
- `stale`: target exists, but the source value hash changed since translation.
- `invalid`: target fails placeholder, tag, ICU, or other validation rules.
- `unknown`: target exists but state is missing.
- `extra`: target key does not exist in the source locale.

### Plan Translation Work

Create a batch for missing and stale keys:

```json
{
  "name": "i18n.translation.plan",
  "arguments": {
    "statuses": ["missing", "stale"],
    "includeContext": true
  }
}
```

Use the `i18n_translate_batch` prompt or your agent workflow to generate proposals. Proposals should have this shape:

```json
[
  {
    "locale": "fr",
    "namespace": "common",
    "key": "hello",
    "sourceValue": "Hello",
    "value": "Bonjour"
  }
]
```

### Validate Proposals

```json
{
  "name": "i18n.translation.validate",
  "arguments": {
    "translations": [
      {
        "locale": "fr",
        "namespace": "common",
        "key": "hello",
        "sourceValue": "Hello",
        "value": "Bonjour"
      }
    ]
  }
}
```

Validation rejects source drift by default. If the source string changed after proposals were generated, regenerate the proposal from the latest source value.

### Preview Translation Apply

```json
{
  "name": "i18n.translation.apply",
  "arguments": {
    "translations": [
      {
        "locale": "fr",
        "namespace": "common",
        "key": "hello",
        "sourceValue": "Hello",
        "value": "Bonjour"
      }
    ]
  }
}
```

This returns patch previews and does not write files.

### Apply Translations After Review

```json
{
  "name": "i18n.translation.apply",
  "arguments": {
    "apply": true,
    "translations": [
      {
        "locale": "fr",
        "namespace": "common",
        "key": "hello",
        "sourceValue": "Hello",
        "value": "Bonjour"
      }
    ]
  }
}
```

Locale files are written first. State is updated only after locale writes succeed.

### Stale Translation Troubleshooting

- `unknown` means state is missing. Run `i18n.state.rebuild` after reviewing existing translations.
- `stale` means the source text changed. Re-translate from the current source value.
- `invalid` means validation failed. Preserve placeholders, tags, ICU arguments, and non-empty target strings.

## Dead-Key Workflow

Dead-key detection is conservative. Static scanning can find many key usages, but dynamic key construction can hide usage from lexical scanners.

### Scan Usage

```json
{
  "name": "i18n.keys.usage_scan",
  "arguments": {}
}
```

The scanner looks at `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, and `.cjs` files and ignores common generated or dependency directories.

Supported patterns include:

- `t("key")`
- `t('key')`
- `i18n.t("key")`
- `<Trans i18nKey="key" />`
- `useTranslations("namespace")` with `t("key")`
- `getTranslations("namespace")` with `t("key")`

### Confidence

Usage evidence includes confidence:

- `exact`: config or explicit evidence proves the status.
- `high`: namespace-bound or JSX literal evidence.
- `medium`: unqualified literal call evidence.
- `low`: dynamic key hints or broad matches.

### Generate Dead-Key Report

```json
{
  "name": "i18n.keys.dead_report",
  "arguments": {
    "refreshUsage": true
  }
}
```

Statuses:

- `used`: statically observed usage.
- `probably_unused`: no static usage and no dynamic hint matched.
- `maybe_dynamic`: dynamic usage may cover the key.
- `ignored`: matched `ignoredKeyPatterns`.
- `kept`: matched `keptKeyPatterns`.

### Dynamic Hints

Add `dynamicKeyHints` when code builds keys dynamically:

```json
{
  "dynamicKeyHints": ["routes.*", "errors.*"]
}
```

Keys matching dynamic hints become `maybe_dynamic` and are unsafe to prune by default.

### Preview Prune

Prune exact keys only. Dry-run is the default:

```json
{
  "name": "i18n.keys.prune",
  "arguments": {
    "keys": [
      {"namespace": "common", "key": "unused.title"}
    ]
  }
}
```

The tool returns patch previews and does not write files.

### Apply Prune

Apply only after reviewing the patch:

```json
{
  "name": "i18n.keys.prune",
  "arguments": {
    "apply": true,
    "keys": [
      {"namespace": "common", "key": "unused.title"}
    ]
  }
}
```

By default, prune refuses `used`, `maybe_dynamic`, `ignored`, and `kept` keys. Only use `allowUnsafe` after human review.

### Recommended Dead-Key Review Process

1. Run `i18n.keys.usage_scan`.
2. Run `i18n.keys.dead_report`.
3. Review `probably_unused` candidates and dynamic hints.
4. Use the `i18n_audit_dead_keys` prompt if your MCP client supports prompts.
5. Preview `i18n.keys.prune` without `apply`.
6. Review patch output.
7. Apply exact keys with `apply: true`.

## CI Usage

`i18n-mcp audit` runs a non-interactive i18n audit and exits according to the configured CI failure policy.

### Run Locally

Markdown output:

```bash
i18n-mcp audit --project /path/to/frontend-app --output markdown
```

JSON output:

```bash
i18n-mcp audit --project /path/to/frontend-app --output json
```

Refresh usage scan explicitly:

```bash
i18n-mcp audit --project /path/to/frontend-app --output json --refresh-usage
```

### Failure Policy

Configure failure behavior in `.i18n-mcp.json`:

```json
{
  "ci": {
    "failOnMissing": true,
    "failOnStale": false,
    "failOnInvalid": true,
    "failOnDeadKeys": false
  }
}
```

Defaults:

- Missing translations fail CI.
- Invalid translations fail CI.
- Stale translations are reported but do not fail CI.
- Dead-key candidates are reported but do not fail CI.

### Exit Codes

- `0`: audit completed and no configured failure condition matched.
- `1`: runtime error or configured failure condition matched.

Failure diagnostics are written to stderr. Report content is written to stdout.

### GitHub Actions Example

```yaml
name: i18n-audit

on:
  pull_request:
  push:
    branches: [main]

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"
      - run: go build -o bin/i18n-mcp ./cmd/i18n-mcp
      - run: bin/i18n-mcp audit --project . --output markdown > i18n-audit.md
      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: i18n-audit
          path: i18n-audit.md
```

### MCP Report Tool

Generate a report through MCP:

```json
{
  "name": "i18n.report.generate",
  "arguments": {
    "format": "markdown",
    "refreshUsage": true
  }
}
```

Read the latest generated report:

```text
i18n://reports/latest
```

## CLI Reference

```bash
i18n-mcp --help
i18n-mcp serve stdio --project /path/to/frontend-app
i18n-mcp serve stdio --project /path/to/frontend-app --config .i18n-mcp.json --log-level warn
i18n-mcp serve http --project /path/to/frontend-app
i18n-mcp serve http --project /path/to/frontend-app --addr 127.0.0.1:7339 --path /mcp
i18n-mcp schema > i18n-mcp.schema.json
i18n-mcp audit --project /path/to/frontend-app --output markdown
i18n-mcp audit --project /path/to/frontend-app --output json
```

## Security Model

`i18n-mcp` supports stdio and Streamable HTTP. With either transport, the server has the same filesystem access as the user running it.

### Project Root Guard

All project file access is constrained to the configured project root.

The root guard rejects:

- Path traversal outside the project root.
- Absolute paths outside the project root.
- Symlink escapes for guarded reads and writes.
- Writes through symlink path components.

### Dry-Run Defaults

Write tools default to preview mode.

| Tool | Writes Files | Default |
| --- | --- | --- |
| `i18n.config.write` | `.i18n-mcp.json` | Dry-run |
| `i18n.state.rebuild` | `.i18n-mcp/state.json` | Dry-run |
| `i18n.translation.apply` | Locale JSON and state | Dry-run |
| `i18n.keys.prune` | Locale JSON | Dry-run |
| `i18n.keys.rename` | Locale JSON and state | Dry-run |

Actual writes require explicit `apply: true`.

### Patch Preview

Write tools return structured changed-file output and unified diffs before writing. Review patches before applying.

### Atomic Writes

Locale, config, and state writes use temporary files plus atomic rename where possible. If a multi-file operation partially fails, the tool reports which files were written and which were not.

### Destructive Operations

Prune and rename are destructive because they remove or move keys.

Safety rules:

- Prune requires exact namespace and key selections.
- Prune refuses `used`, `maybe_dynamic`, `ignored`, and `kept` keys by default.
- Rename rejects destination conflicts by default.
- Overwrite behavior must be explicit.

### Credentials

Do not store credentials in `.i18n-mcp.json`, `.i18n-mcp/state.json`, reports, prompts, or tool arguments.

When provider mode exists, credentials should come from environment variables or secure storage and must not be logged or returned in MCP results.

### Reports

Reports include i18n-relevant config, locale summaries, diff status, usage evidence, and dead-key candidates. Reports must not include unrelated source files, environment variables, or provider credentials.

### Stdio Deployment

Stdio is intended for a single user running the server against a project checkout.

Recommendations:

- Run the server only for trusted projects.
- Use a project-specific root with `--project`.
- Review dry-run patches before apply.
- Do not expose stdio server processes to untrusted clients.

### HTTP Deployment

Streamable HTTP defaults to `127.0.0.1:7339`, serves MCP at `/mcp`, and exposes a `/healthz` endpoint. The server refuses to bind to a non-loopback address unless `--auth-required` is set.

The built-in bearer-token mode provides protected-resource metadata and scope checks, but its static token verifier is for development only. Clients must send the token as `Authorization: Bearer <token>`:

```bash
export I18N_MCP_DEV_TOKEN='replace-with-a-random-secret'
i18n-mcp serve http \
  --project /path/to/frontend-app \
  --auth-required \
  --auth-resource http://127.0.0.1:7339/mcp \
  --dev-static-token-env I18N_MCP_DEV_TOKEN
```

For production or remote access, keep the server on loopback or a private interface behind a gateway that provides TLS, token validation, authorization, rate limiting, and secret-safe audit logging. Do not expose project files over a network without authentication and authorization.

## Troubleshooting

| Problem | Fix |
| --- | --- |
| MCP client cannot find the server | For stdio, use an absolute `command` path and restart the client. For HTTP, verify the client supports Streamable HTTP and can reach the configured `/mcp` URL. |
| HTTP server rejects the listen address | Non-loopback binds require `--auth-required`; use loopback for local clients or configure authentication. |
| Server starts in the wrong project | Pass `--project /absolute/path/to/frontend-app`. |
| Config validation fails | Run `i18n.project.detect`, review `proposedConfig`, then preview `i18n.config.write`. |
| Diff shows `unknown` keys | Run `i18n.state.rebuild` without `apply`, review, then run with `apply: true`. |
| Stale translations remain | Re-plan with `i18n.translation.plan` and regenerate proposals from current source values. |
| Prune refuses a key | Check `i18n.keys.dead_report`; `used`, `maybe_dynamic`, `ignored`, and `kept` are refused by default. |
| Audit fails CI | Check `.i18n-mcp.json` `ci` policy and inspect the Markdown or JSON report. |
| Docker cannot see project files | Mount the project with `-v /absolute/project:/workspace` and pass `--project /workspace`. |
| Binary not found after install | Add the install directory to `PATH` or install with `--dir "$HOME/bin"`. |

## Distribution & Release Maintenance

This project uses GoReleaser for cross-platform binaries, checksums, SBOMs, GitHub Releases, and Docker images.

Release workflow outputs include:

- Linux, macOS, and Windows archives.
- `SHASUMS256.txt` checksums.
- SBOM artifacts.
- Docker image tags.

Create a release with:

```bash
DOCKER_IMAGE=ret2hell/i18n-mcp goreleaser release --clean
```

Docker image tags:

```text
ret2hell/i18n-mcp:VERSION
ret2hell/i18n-mcp:latest
```

## License

See [LICENSE](LICENSE).
