#!/bin/sh
set -eu

version_file=${VERSION_FILE:-version}
output=${OUTPUT:-bin/i18n-mcp}
package=${PACKAGE:-./cmd/i18n-mcp/}
module_path=${MODULE_PATH:-github.com/Ret2Hell/i18n-mcp}

if [ -z "${VERSION:-}" ]; then
  VERSION=$(tr -d '[:space:]' < "$version_file")
fi

case "$VERSION" in
  *[!0-9.]*|.*|*..*|*.)
    echo "invalid version: $VERSION" >&2
    exit 1
    ;;
esac
if ! printf '%s' "$VERSION" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "invalid version: $VERSION" >&2
  exit 1
fi

if [ -z "${COMMIT:-}" ]; then
  COMMIT=$(git rev-parse --short=12 HEAD 2>/dev/null || echo none)
fi

if [ -z "${DATE:-}" ]; then
  if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
    DATE=$(date -u -d "@$SOURCE_DATE_EPOCH" '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -r "$SOURCE_DATE_EPOCH" '+%Y-%m-%dT%H:%M:%SZ')
  else
    DATE=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  fi
fi

mkdir -p "$(dirname "$output")"

go build \
  -ldflags="-s -w -X ${module_path}/internal/version.Version=${VERSION} -X ${module_path}/internal/version.Commit=${COMMIT} -X ${module_path}/internal/version.Date=${DATE}" \
  -o "$output" \
  "$package"
