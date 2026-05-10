#!/usr/bin/env sh
# notebooklm-mcp installer
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.sh | sh
#   curl -sSL https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.sh | sh -s -- v0.2.0
#
# Re-running this script upgrades to the latest release.
#
# Environment overrides:
#   NOTEBOOKLM_MCP_INSTALL_DIR  (default: $HOME/.notebooklm-mcp/bin)
#   NOTEBOOKLM_MCP_VERSION      (default: latest)

set -eu

REPO="vankcdhv/notebook-mcp"
INSTALL_DIR="${NOTEBOOKLM_MCP_INSTALL_DIR:-$HOME/.notebooklm-mcp/bin}"
BIN_NAME="notebooklm-mcp"

err() { printf "error: %s\n" "$1" >&2; exit 1; }

# --- detect OS ---
case "$(uname -s)" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  MINGW*|MSYS*|CYGWIN*)
    err "Windows is not supported by this installer; download the .zip from https://github.com/$REPO/releases" ;;
  *) err "unsupported OS: $(uname -s)" ;;
esac

# --- detect arch ---
case "$(uname -m)" in
  x86_64|amd64)  ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) err "unsupported architecture: $(uname -m)" ;;
esac

# --- resolve version ---
VERSION="${1:-${NOTEBOOKLM_MCP_VERSION:-latest}}"
if [ "$VERSION" = "latest" ]; then
  printf "==> Resolving latest release\n"
  VERSION=$(
    curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
      | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1
  )
  [ -z "$VERSION" ] && err "could not resolve latest version (rate-limited?)"
fi
VERSION="${VERSION#v}"

case "$OS" in
  linux|darwin) ARCHIVE="${BIN_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz" ;;
  *) err "unsupported OS: $OS" ;;
esac
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/${ARCHIVE}"
CHECKSUMS_URL="https://github.com/$REPO/releases/download/v${VERSION}/checksums.txt"

# --- download + verify ---
TMP="$(mktemp -d 2>/dev/null || mktemp -d -t nbmcp)"
trap 'rm -rf "$TMP"' EXIT INT TERM

printf "==> Downloading %s\n" "$ARCHIVE"
curl -fL --progress-bar -o "$TMP/$ARCHIVE" "$DOWNLOAD_URL" \
  || err "download failed: $DOWNLOAD_URL"

printf "==> Verifying checksum\n"
curl -fsSL -o "$TMP/checksums.txt" "$CHECKSUMS_URL" \
  || err "could not fetch checksums.txt"

EXPECTED=$(grep "  $ARCHIVE\$" "$TMP/checksums.txt" | awk '{print $1}')
[ -z "$EXPECTED" ] && err "no checksum entry for $ARCHIVE"

if command -v shasum >/dev/null 2>&1; then
  ACTUAL=$(shasum -a 256 "$TMP/$ARCHIVE" | awk '{print $1}')
elif command -v sha256sum >/dev/null 2>&1; then
  ACTUAL=$(sha256sum "$TMP/$ARCHIVE" | awk '{print $1}')
else
  err "neither shasum nor sha256sum is available"
fi
[ "$ACTUAL" = "$EXPECTED" ] || err "checksum mismatch (expected $EXPECTED, got $ACTUAL)"

# --- extract + install ---
printf "==> Installing to %s\n" "$INSTALL_DIR/$BIN_NAME"
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
SRC="$TMP/$BIN_NAME"
[ -f "$SRC" ] || SRC="$TMP/${BIN_NAME}_${VERSION}_${OS}_${ARCH}/$BIN_NAME"
[ -f "$SRC" ] || err "binary not found in archive"

mkdir -p "$INSTALL_DIR"
if [ "${INSTALL_DIR#"$HOME/.notebooklm-mcp"}" != "$INSTALL_DIR" ]; then
  chmod 700 "$HOME/.notebooklm-mcp" 2>/dev/null || true
fi
install -m 755 "$SRC" "$INSTALL_DIR/$BIN_NAME"

# --- strip macOS quarantine ---
if [ "$OS" = "darwin" ]; then
  xattr -dr com.apple.quarantine "$INSTALL_DIR/$BIN_NAME" 2>/dev/null || true
fi

BIN_PATH="$INSTALL_DIR/$BIN_NAME"

printf "\n\033[32m✓\033[0m Installed notebooklm-mcp \033[1mv%s\033[0m → %s\n\n" "$VERSION" "$BIN_PATH"

# --- next-step instructions ---
if [ -f "$HOME/.notebooklm-mcp/profile.json" ]; then
  printf "Existing login profile detected — no setup needed.\n\n"
  printf "If you have not registered the MCP yet, run:\n"
  printf "  \033[1mclaude mcp add -s user notebooklm-mcp %s\033[0m\n\n" "$BIN_PATH"
  printf "Then restart your Claude Code session.\n"
else
  printf "Next steps:\n"
  printf "  \033[1m1.\033[0m First-time setup (installs Playwright browser, opens Google login):\n"
  printf "       \033[1m%s setup\033[0m\n\n" "$BIN_PATH"
  printf "  \033[1m2.\033[0m Register with Claude Code (user scope = global):\n"
  printf "       \033[1mclaude mcp add -s user notebooklm-mcp %s\033[0m\n\n" "$BIN_PATH"
  printf "  \033[1m3.\033[0m Restart your Claude Code session.\n"
fi
