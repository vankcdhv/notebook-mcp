#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
INSTALL_SH="$SCRIPT_DIR/install.sh"

assert_contains() {
  needle=$1
  if ! grep -Fq "$needle" "$INSTALL_SH"; then
    printf 'missing expected text: %s\n' "$needle" >&2
    exit 1
  fi
}

assert_not_contains() {
  needle=$1
  if grep -Fq "$needle" "$INSTALL_SH"; then
    printf 'unexpected text: %s\n' "$needle" >&2
    exit 1
  fi
}

assert_contains 'OS=linux'
assert_contains 'OS=darwin'
assert_contains 'case "$OS" in'
assert_contains 'tar -xzf "$TMP/$ARCHIVE" -C "$TMP"'
assert_contains 'install -m 755 "$SRC" "$INSTALL_DIR/$BIN_NAME"'
assert_contains 'chmod 700 "$HOME/.notebooklm-mcp"'
assert_not_contains 'chmod 700 "$(dirname "$INSTALL_DIR")"'

printf 'install.sh portability checks passed\n'
