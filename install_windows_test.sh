#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
INSTALL_PS1="$SCRIPT_DIR/install.ps1"
README="$SCRIPT_DIR/README.md"

assert_file() {
  [ -f "$1" ] || { printf 'missing file: %s\n' "$1" >&2; exit 1; }
}

assert_contains() {
  file=$1
  needle=$2
  if ! grep -Fq "$needle" "$file"; then
    printf 'missing expected text in %s: %s\n' "$file" "$needle" >&2
    exit 1
  fi
}

assert_file "$INSTALL_PS1"
assert_contains "$INSTALL_PS1" 'notebooklm-mcp_${Version}_windows_${Arch}.zip'
assert_contains "$INSTALL_PS1" 'Get-FileHash'
assert_contains "$INSTALL_PS1" 'Expand-Archive'
assert_contains "$INSTALL_PS1" '$env:LOCALAPPDATA'
assert_contains "$INSTALL_PS1" 'NOTEBOOKLM_MCP_INSTALL_DIR'
assert_contains "$INSTALL_PS1" 'NOTEBOOKLM_MCP_VERSION'
assert_contains "$INSTALL_PS1" 'claude mcp add -s user notebooklm-mcp'
assert_contains "$README" 'install.ps1'
assert_contains "$README" 'iwr -useb https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.ps1 | iex'

printf 'install.ps1 checks passed\n'
