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

xvfb_log="${RUNNER_TEMP:-/tmp}/xvfb.log"
app_log="${RUNNER_TEMP:-/tmp}/ghostftp-linux.log"
app_pid=''
Xvfb "$DISPLAY" -screen 0 1440x1000x24 -nolisten tcp >"$xvfb_log" 2>&1 &
xvfb_pid=$!
cleanup() {
  if [[ -n "$app_pid" ]]; then
    kill "$app_pid" 2>/dev/null || true
  fi
  kill "$xvfb_pid" 2>/dev/null || true
  rm -rf "$XDG_DATA_HOME"
}
trap cleanup EXIT

# Ghost FTP intentionally connects directly to the local Unix X11 socket. Wait
# for Xvfb to publish that socket instead of racing application startup.
display_number="${DISPLAY#:}"
display_number="${display_number%%.*}"
x11_socket="/tmp/.X11-unix/X${display_number}"
for _ in $(seq 1 100); do
  if [[ -S "$x11_socket" ]]; then
    break
  fi
  if ! kill -0 "$xvfb_pid" 2>/dev/null; then
    cat "$xvfb_log" >&2 || true
    echo 'Xvfb terminated before its local X11 socket became ready.' >&2
    exit 1
  fi
  sleep 0.1
done
[[ -S "$x11_socket" ]] || {
  cat "$xvfb_log" >&2 || true
  echo "Xvfb local X11 socket did not become ready: $x11_socket" >&2
  exit 1
}

"$exe" >"$app_log" 2>&1 &
app_pid=$!

win=''
for _ in $(seq 1 100); do
  win="$(xdotool search --onlyvisible --pid "$app_pid" --name '.*' 2>/dev/null | head -n1 || true)"
  if [[ -n "$win" ]]; then
    break
  fi
  if ! kill -0 "$app_pid" 2>/dev/null; then
    cat "$app_log" >&2 || true
    exit 1
  fi
  sleep 0.2
done
[[ -n "$win" ]] || {
  cat "$app_log" >&2 || true
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

main_png="$OUTPUT_DIR/ghost-ftp-linux-main-workspace.png"
bookmarks_png="$OUTPUT_DIR/ghost-ftp-linux-bookmarks.png"
settings_png="$OUTPUT_DIR/ghost-ftp-linux-settings.png"
if cmp -s "$main_png" "$bookmarks_png"; then
  echo 'Bookmarks evidence is identical to the main workspace; the real overlay did not open.' >&2
  exit 1
fi
if cmp -s "$main_png" "$settings_png"; then
  echo 'Settings evidence is identical to the main workspace; the real overlay did not open.' >&2
  exit 1
fi
if cmp -s "$bookmarks_png" "$settings_png"; then
  echo 'Bookmarks and Settings evidence are identical; distinct real overlays were not captured.' >&2
  exit 1
fi

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
