#!/usr/bin/env bash
set -euo pipefail

# install.sh - One-line installer for i18n-mcp.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Ret2Hell/i18n-mcp/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/Ret2Hell/i18n-mcp/main/install.sh | bash -s -- --dir /path/to/bin
#   curl -fsSL https://raw.githubusercontent.com/Ret2Hell/i18n-mcp/main/install.sh | bash -s -- --version v0.9.0
#
# Environment:
#   I18N_MCP_VERSION       Version to install, such as v0.9.0. Default: latest.
#   I18N_MCP_INSTALL_DIR   Install directory. Default: ~/.local/bin.
#   I18N_MCP_DOWNLOAD_URL  Override release download base URL for testing.

# Wrap in main() so interrupted curl|bash downloads do not execute partial logic.
main() {

REPO="Ret2Hell/i18n-mcp"
BINARY="i18n-mcp"
INSTALL_DIR="${I18N_MCP_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${I18N_MCP_VERSION:-latest}"
DOWNLOAD_URL="${I18N_MCP_DOWNLOAD_URL:-}"

usage() {
    echo "Usage: install.sh [--dir PATH] [--version VERSION]"
    echo "  --dir PATH         Install directory (default: ~/.local/bin)"
    echo "  --version VERSION  Version to install, such as v0.9.0 (default: latest)"
    echo "  --help, -h         Show this help"
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --dir)
            [ "$#" -ge 2 ] || { echo "error: --dir requires a value" >&2; exit 1; }
            INSTALL_DIR="$2"
            shift 2
            ;;
        --dir=*)
            INSTALL_DIR="${1#--dir=}"
            shift
            ;;
        --version)
            [ "$#" -ge 2 ] || { echo "error: --version requires a value" >&2; exit 1; }
            VERSION="$2"
            shift 2
            ;;
        --version=*)
            VERSION="${1#--version=}"
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            echo "error: unknown argument: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

need_downloader() {
    if command -v curl >/dev/null 2>&1; then
        echo curl
    elif command -v wget >/dev/null 2>&1; then
        echo wget
    else
        echo "error: curl or wget is required" >&2
        exit 1
    fi
}

fetch_stdout() {
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$1"
    else
        wget -qO- "$1"
    fi
}

download_file() {
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fSL --progress-bar -o "$2" "$1"
    else
        wget -q --show-progress -O "$2" "$1"
    fi
}

detect_os() {
    case "$(uname -s)" in
        Darwin) echo "macos" ;;
        Linux) echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "error: unsupported OS: $(uname -s)" >&2; exit 1 ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64) echo "x64" ;;
        arm64|aarch64) echo "arm64" ;;
        armv7l|armv7) echo "armv7" ;;
        *) echo "error: unsupported architecture: $arch" >&2; exit 1 ;;
    esac
}

fallback_archive_name() {
    local ext
    if [ "$OS" = "windows" ]; then
        ext="zip"
    else
        ext="tar.gz"
    fi
    echo "${BINARY}-${TAG}-${OS}-${ARCH}.${ext}"
}

asset_matches_os() {
    local lower="$1"
    case "$OS" in
        linux) [[ "$lower" == *linux* ]] ;;
        macos) [[ "$lower" == *macos* || "$lower" == *darwin* ]] ;;
        windows) [[ "$lower" == *windows* ]] ;;
        *) return 1 ;;
    esac
}

asset_matches_arch() {
    local lower="$1"
    case "$ARCH" in
        x64) [[ "$lower" == *x64* || "$lower" == *amd64* || "$lower" == *x86_64* ]] ;;
        arm64) [[ "$lower" == *arm64* || "$lower" == *aarch64* ]] ;;
        armv7) [[ "$lower" == *armv7* ]] ;;
        *) return 1 ;;
    esac
}

pick_asset_url() {
    local urls="$1"
    local url lower
    while IFS= read -r url; do
        [ -n "$url" ] || continue
        lower="${url,,}"
        case "$lower" in
            *.tar.gz|*.zip) ;;
            *) continue ;;
        esac
        case "$lower" in
            *checksums*|*shasums*|*.sbom.*|*.sig|*.pem) continue ;;
        esac
        asset_matches_os "$lower" || continue
        asset_matches_arch "$lower" || continue
        echo "$url"
        return 0
    done <<EOF_ASSETS
$urls
EOF_ASSETS
    return 1
}

DOWNLOADER="$(need_downloader)"
OS="$(detect_os)"
ARCH="$(detect_arch)"
RELEASE_JSON=""

case "$VERSION" in
    latest) TAG="" ;;
    v*) TAG="$VERSION" ;;
    *) TAG="v$VERSION" ;;
esac

if [ -z "$DOWNLOAD_URL" ]; then
    if [ "$VERSION" = "latest" ]; then
        API_URL="https://api.github.com/repos/${REPO}/releases/latest"
    else
        API_URL="https://api.github.com/repos/${REPO}/releases/tags/${TAG}"
    fi
    RELEASE_JSON="$(fetch_stdout "$API_URL" 2>/dev/null || true)"
    if [ "$VERSION" = "latest" ]; then
        TAG="$(printf '%s\n' "$RELEASE_JSON" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
        [ -n "$TAG" ] || { echo "error: could not determine latest release version" >&2; exit 1; }
    fi
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}"
fi

case "$DOWNLOAD_URL" in
    https://*|http://localhost*|http://127.0.0.1*) ;;
    *) echo "error: refusing non-HTTPS download URL: $DOWNLOAD_URL" >&2; exit 1 ;;
esac

ASSET_URL=""
if [ -n "$RELEASE_JSON" ]; then
    URLS="$(printf '%s\n' "$RELEASE_JSON" | sed -n 's/.*"browser_download_url":[[:space:]]*"\([^"]*\)".*/\1/p')"
    ASSET_URL="$(pick_asset_url "$URLS" || true)"
fi

if [ -n "$ASSET_URL" ]; then
    ARCHIVE="${ASSET_URL##*/}"
    URL="$ASSET_URL"
else
    ARCHIVE="$(fallback_archive_name)"
    URL="${DOWNLOAD_URL}/${ARCHIVE}"
fi

printf 'i18n-mcp installer\n'
printf '  version: %s\n' "$TAG"
printf '  os:      %s\n' "$OS"
printf '  arch:    %s\n' "$ARCH"
printf '  target:  %s/%s\n\n' "$INSTALL_DIR" "$BINARY"

DLDIR="$(mktemp -d)"
trap 'rm -rf "$DLDIR"' EXIT

printf 'Downloading %s...\n' "$ARCHIVE"
download_file "$URL" "$DLDIR/$ARCHIVE"

CHECKSUM_URL="${DOWNLOAD_URL}/SHASUMS256.txt"
if fetch_stdout "$CHECKSUM_URL" > "$DLDIR/SHASUMS256.txt" 2>/dev/null; then
    EXPECTED="$(grep "[[:space:]]${ARCHIVE}$" "$DLDIR/SHASUMS256.txt" | awk '{print $1}' | head -n 1)"
    if [ -n "$EXPECTED" ]; then
        if command -v sha256sum >/dev/null 2>&1; then
            ACTUAL="$(sha256sum "$DLDIR/$ARCHIVE" | awk '{print $1}')"
        elif command -v shasum >/dev/null 2>&1; then
            ACTUAL="$(shasum -a 256 "$DLDIR/$ARCHIVE" | awk '{print $1}')"
        else
            ACTUAL=""
        fi
        if [ -n "$ACTUAL" ] && [ "$EXPECTED" != "$ACTUAL" ]; then
            echo "error: checksum mismatch" >&2
            echo "  expected: $EXPECTED" >&2
            echo "  actual:   $ACTUAL" >&2
            exit 1
        elif [ -n "$ACTUAL" ]; then
            echo "Checksum verified."
        fi
    fi
fi

printf 'Extracting...\n'
cd "$DLDIR"
case "$ARCHIVE" in
    *.zip)
        command -v unzip >/dev/null 2>&1 || { echo "error: unzip is required" >&2; exit 1; }
        unzip -q "$ARCHIVE"
        ;;
    *.tar.gz)
        tar -xzf "$ARCHIVE"
        ;;
    *)
        echo "error: unsupported archive type: $ARCHIVE" >&2
        exit 1
        ;;
esac

if [ -f "$DLDIR/$BINARY" ]; then
    DLBIN="$DLDIR/$BINARY"
elif [ -f "$DLDIR/$BINARY.exe" ]; then
    DLBIN="$DLDIR/$BINARY.exe"
else
    DLBIN="$(find "$DLDIR" -type f \( -name "$BINARY" -o -name "$BINARY.exe" \) | head -n 1)"
    [ -n "$DLBIN" ] || { echo "error: binary not found after extraction" >&2; exit 1; }
fi

if [ "$OS" = "macos" ]; then
    xattr -d com.apple.quarantine "$DLBIN" 2>/dev/null || true
    codesign --sign - --force "$DLBIN" 2>/dev/null || true
fi

mkdir -p "$INSTALL_DIR"
DEST="$INSTALL_DIR/$BINARY"
if [ "$OS" = "windows" ]; then
    DEST="$INSTALL_DIR/$BINARY.exe"
fi
rm -f "$DEST"
cp "$DLBIN" "$DEST"
chmod 755 "$DEST"

VERSION_OUT="$("$DEST" --version 2>&1)" || {
    echo "error: installed binary failed to run" >&2
    if [ "$OS" = "macos" ]; then
        echo "  try: xattr -cr $DEST && codesign --force --sign - $DEST" >&2
    fi
    exit 1
}
printf 'Installed: %s\n' "$VERSION_OUT"

if ! printf '%s' "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
    printf '\nNOTE: %s is not in your PATH.\n' "$INSTALL_DIR"
    printf 'Add it to your shell config, for example:\n\n'
    printf '  echo '\''export PATH="%s:$PATH"'\'' >> ~/.zshrc\n' "$INSTALL_DIR"
fi

cat <<EOF_DONE

Done.

Run the stdio server:
  $DEST serve stdio --project /path/to/next-app

MCP client command:
  $DEST

MCP client args:
  ["serve", "stdio", "--project", "/absolute/path/to/next-app"]
EOF_DONE

} # end main()

main "$@"
