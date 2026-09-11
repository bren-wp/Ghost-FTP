#!/usr/bin/env bash
set -euo pipefail

OUTPUT_DIR="${1:-ui-screenshots/linux}"
version="$(tr -d '\r\n' < VERSION)"
archive="dist/Ghost-FTP-${version}-Linux-amd64.tar.gz"
extract_dir="${RUNNER_TEMP:-/tmp}/ghostftp-linux-ui"

[[ -s "$archive" ]] || {
  echo "Missing Linux portable package: $archive" >&2
  exit 1
}

rm -rf "$extract_dir"
mkdir -p "$OUTPUT_DIR" "$extract_dir"
tar -xzf "$archive" -C "$extract_dir"
exe="$extract_dir/Ghost-FTP-${version}-Linux-amd64/ghostftp"
[[ -x "$exe" ]] || {
  echo "Missing packaged Linux executable: $exe" >&2
  exit 1
}

export DISPLAY="${GHOSTFTP_UI_DISPLAY:-:99}"
# Keep the runner's real HOME so Ghost FTP's filesystem-identity hardening sees
# an ordinary per-user path rather than GitHub's redirected RUNNER_TEMP tree.
# XDG_DATA_HOME still isolates evidence state on this ephemeral runner.
export XDG_DATA_HOME="$HOME/.ghostftp-ui-evidence-data"
rm -rf "$XDG_DATA_HOME"
install -d -m 700 "$XDG_DATA_HOME"

Xvfb "$DISPLAY" -screen 0 1440x1000x24 -nolisten tcp >"${RUNNER_TEMP:-/tmp}/xvfb.log" 2>&1 &
xvfb_pid=$!
"$exe" >"${RUNNER_TEMP:-/tmp}/ghostftp-linux.log" 2>&1 &
app_pid=$!
cleanup() {
  kill "$app_pid" 2>/dev/null || true
  kill "$xvfb_pid" 2>/dev/null || true
  rm -rf "$XDG_DATA_HOME"
}
trap cleanup EXIT

win=''
for _ in $(seq 1 100); do
  win="$(xdotool search --onlyvisible --pid "$app_pid" --name '.*' 2>/dev/null | head -n1 || true)"
  if [[ -n "$win" ]]; then
    break
  fi
  if ! kill -0 "$app_pid" 2>/dev/null; then
    cat "${RUNNER_TEMP:-/tmp}/ghostftp-linux.log" >&2 || true
    exit 1
  fi
  sleep 0.2
done
[[ -n "$win" ]] || {
  echo 'Unable to locate Ghost FTP Linux window.' >&2
  exit 1
}

xdotool windowsize "$win" 1280 900
sleep 0.8

capture() {
  local output="$1"
  import -window "$win" "$OUTPUT_DIR/$output"
  [[ -s "$OUTPUT_DIR/$output" ]] || {
    echo "Empty Linux screenshot: $output" >&2
    exit 1
  }
}

capture 'ghost-ftp-linux-main-workspace.png'

# Ghost FTP Linux uses one X11 window and application-owned overlays. These
# coordinates target the maintained top-row Bookmarks and Settings controls at
# the deterministic 1280x900 evidence geometry.
xdotool mousemove --window "$win" 1080 35 click 1
sleep 0.5
capture 'ghost-ftp-linux-bookmarks.png'
xdotool key --window "$win" Escape
sleep 0.3

xdotool mousemove --window "$win" 1210 35 click 1
sleep 0.5
capture 'ghost-ftp-linux-settings.png'
xdotool key --window "$win" Escape
sleep 0.3

for png in "$OUTPUT_DIR"/*.png; do
  dims="$(identify -format '%w %h %k' "$png")"
  read -r width height colors <<<"$dims"
  if (( width < 320 || height < 200 || colors < 8 )); then
    echo "Implausible Linux screenshot: $png ($dims)" >&2
    exit 1
  fi
  printf 'LINUX_UI=%s SHA256=%s\n' "$(basename "$png")" "$(sha256sum "$png" | awk '{print $1}')"
done

printf 'LINUX_UI_SCREENSHOTS=PASS\n'
