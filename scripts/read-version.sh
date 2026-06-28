#!/usr/bin/env bash
set -euo pipefail

version_file=${VERSION_FILE:-version}
mode=${1:-version}

if [[ ! -f "$version_file" ]]; then
  echo "version file not found: $version_file" >&2
  exit 1
fi

version=$(tr -d '[:space:]' < "$version_file")
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid version: $version" >&2
  exit 1
fi

case "$mode" in
  version)
    printf '%s\n' "$version"
    ;;
  tag)
    printf 'v%s\n' "$version"
    ;;
  github-output)
    {
      printf 'version=%s\n' "$version"
      printf 'tag=v%s\n' "$version"
      printf 'date=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    } >> "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required for github-output mode}"
    ;;
  *)
    echo "usage: $0 [version|tag|github-output]" >&2
    exit 2
    ;;
esac
