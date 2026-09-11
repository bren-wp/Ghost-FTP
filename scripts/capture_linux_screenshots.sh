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

# Ghost FTP is a direct X11 client and intentionally does not depend on EWMH
# process metadata. Locate the client by the WM_NAME written by x11_linux.go,
# then separately validate its WM_CLASS instead of relying on a combined
# xdotool search predicate that is unreliable on a bare Xvfb server.
win=''
for _ in $(seq 1 100); do
  win="$(xdotool search --onlyvisible --name '^Ghost FTP' 2>/dev/null | head -n1 || true)"
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
  echo 'Unable to locate the Ghost FTP WM_NAME Linux window.' >&2
  exit 1
}
window_name="$(xdotool getwindowname "$win" 2>/dev/null || true)"
window_class="$(xdotool getwindowclassname "$win" 2>/dev/null || true)"
[[ "$window_name" == Ghost\ FTP* ]] || {
  echo "Resolved X11 window has an unexpected title: $window_name" >&2
  exit 1
}
case "$window_class" in
  GhostFTP|ghostftp) ;;
  *)
    echo "Resolved Ghost FTP title has an unexpected WM_CLASS: $window_class" >&2
    exit 1
    ;;
esac
printf 'LINUX_UI_WINDOW=%s TITLE=%s CLASS=%s PID=%s\n' "$win" "$window_name" "$window_class" "$app_pid"

read_window_geometry() {
  local geometry
  geometry="$(xdotool getwindowgeometry --shell "$win")"
  window_x="$(printf '%s\n' "$geometry" | awk -F= '$1=="X" {print $2}')"
  window_y="$(printf '%s\n' "$geometry" | awk -F= '$1=="Y" {print $2}')"
  window_width="$(printf '%s\n' "$geometry" | awk -F= '$1=="WIDTH" {print $2}')"
  window_height="$(printf '%s\n' "$geometry" | awk -F= '$1=="HEIGHT" {print $2}')"
}

window_x=''
window_y=''
window_width=''
window_height=''
previous_geometry=''
for _ in $(seq 1 50); do
  read_window_geometry
  current_geometry="${window_x},${window_y}:${window_width}x${window_height}"
  if [[ "$window_x" =~ ^-?[0-9]+$ && "$window_y" =~ ^-?[0-9]+$ &&
        "$window_width" =~ ^[0-9]+$ && "$window_height" =~ ^[0-9]+$ ]] &&
     (( window_width >= 940 && window_height >= 680 )); then
    if [[ "$current_geometry" == "$previous_geometry" ]]; then
      break
    fi
    previous_geometry="$current_geometry"
  fi
  sleep 0.1
done
[[ "$window_x" =~ ^-?[0-9]+$ && "$window_y" =~ ^-?[0-9]+$ &&
   "$window_width" =~ ^[0-9]+$ && "$window_height" =~ ^[0-9]+$ ]] || {
  echo "Unable to read Linux UI geometry: ${window_x:-?},${window_y:-?}:${window_width:-?}x${window_height:-?}" >&2
  exit 1
}
(( window_width >= 940 && window_height >= 680 )) || {
  echo "Linux UI is below the supported minimum geometry: ${window_width}x${window_height}" >&2
  exit 1
}
printf 'LINUX_UI_GEOMETRY=X=%s Y=%s WIDTH=%s HEIGHT=%s WINDOW=%s\n' \
  "$window_x" "$window_y" "$window_width" "$window_height" "$win"

stable_a="${RUNNER_TEMP:-/tmp}/ghostftp-linux-stable-a.png"
stable_b="${RUNNER_TEMP:-/tmp}/ghostftp-linux-stable-b.png"
rm -f "$stable_a" "$stable_b"
for _ in $(seq 1 30); do
  import -window "$win" "$stable_a"
  sleep 0.25
  import -window "$win" "$stable_b"
  if cmp -s "$stable_a" "$stable_b"; then
    break
  fi
  mv -f "$stable_b" "$stable_a"
  sleep 0.25
done
cmp -s "$stable_a" "$stable_b" || {
  cat "$app_log" >&2 || true
  echo 'Linux UI did not reach a stable startup state before evidence capture.' >&2
  exit 1
}
cp "$stable_b" "$OUTPUT_DIR/ghost-ftp-linux-main-workspace.png"

read_window_geometry
bookmarks_client_x=$((window_width - 173))
settings_client_x=$((window_width - 68))
header_client_y=35

main_png="$OUTPUT_DIR/ghost-ftp-linux-main-workspace.png"
bookmarks_png="$OUTPUT_DIR/ghost-ftp-linux-bookmarks.png"
settings_png="$OUTPUT_DIR/ghost-ftp-linux-settings.png"

open_distinct_overlay() {
  local label="$1"
  local client_x="$2"
  local client_y="$3"
  local output="$4"
  local root_x=$((window_x + client_x))
  local root_y=$((window_y + client_y))

  printf 'LINUX_UI_CLICK=%s CLIENT=%s,%s ROOT=%s,%s\n' \
    "$label" "$client_x" "$client_y" "$root_x" "$root_y"

  for _ in $(seq 1 20); do
    xdotool windowfocus "$win" >/dev/null 2>&1 || true
    xdotool mousemove "$root_x" "$root_y"
    xdotool click 1
    sleep 0.3
    import -window "$win" "$output"
    if [[ -s "$output" ]] && ! cmp -s "$main_png" "$output"; then
      return 0
    fi
    sleep 0.25
  done

  cat "$app_log" >&2 || true
  echo "$label evidence is identical to the main workspace; the real overlay did not open." >&2
  return 1
}

open_distinct_overlay 'Bookmarks' "$bookmarks_client_x" "$header_client_y" "$bookmarks_png"
xdotool key --window "$win" Escape
sleep 0.3

open_distinct_overlay 'Settings' "$settings_client_x" "$header_client_y" "$settings_png"
xdotool key --window "$win" Escape
sleep 0.3

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
