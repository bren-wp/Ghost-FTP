#!/bin/zsh
set -eu
APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$APP_DIR/Resources/byftp"
[[ -x "$BIN" ]] || { echo 'ByFTP runtime is missing or not executable.' >&2; exit 1; }
/usr/bin/osascript <<APPLESCRIPT
set cmd to quoted form of "$BIN"
tell application "Terminal"
  activate
  do script cmd
end tell
APPLESCRIPT
