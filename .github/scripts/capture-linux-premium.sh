#!/usr/bin/env bash
# Real running Linux Tauri/WebKit screenshot evidence (not the approved concept).
# Run only on the isolated GitHub Actions Linux runner under xvfb-run.
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: capture-linux-premium.sh <real ELF> <output directory>" >&2
  exit 2
fi
binary="$(realpath "$1")"
output="$(realpath -m "$2")"
test -x "$binary"
mkdir -p "$output"
private_home="$(mktemp -d)"
chmod 700 "$private_home"
export HOME="$private_home"
export XDG_CONFIG_HOME="$private_home/.config"
export XDG_CACHE_HOME="$private_home/.cache"
export XDG_DATA_HOME="$private_home/.local/share"
export GDK_BACKEND=x11
# Prevent WebKit's DMABUF compositor from returning black screenshots under Xvfb.
export WEBKIT_DISABLE_DMABUF_RENDERER=1
export WEBKIT_DISABLE_COMPOSITING_MODE=1
mkdir -p "$XDG_CONFIG_HOME" "$XDG_CACHE_HOME" "$XDG_DATA_HOME"

# Xvfb exposes an X server but does not launch a window manager. A live
# Tauri/GTK window can remain unmapped or lack WM_VISIBLE state without one.
# Run a lightweight WM in this disposable display; don't hide a missing GUI.
openbox --sm-disable > "$output/window-manager.log" 2>&1 &
wm_pid=$!
app_pid=""
cleanup() {
  if [ -n "$app_pid" ]; then
    kill "$app_pid" >/dev/null 2>&1 || true
    wait "$app_pid" >/dev/null 2>&1 || true
  fi
  kill "$wm_pid" >/dev/null 2>&1 || true
  wait "$wm_pid" >/dev/null 2>&1 || true
  rm -rf "$private_home"
}
trap cleanup EXIT

manager_ready=0
for attempt in $(seq 1 40); do
  if wmctrl -m > "$output/window-manager-state.txt" 2>&1; then
    manager_ready=1
    break
  fi
  if ! kill -0 "$wm_pid" 2>/dev/null; then break; fi
  sleep 0.25
done
if [ "$manager_ready" -ne 1 ]; then
  echo "Linux reference capture could not start the isolated Openbox window manager." >&2
  cat "$output/window-manager.log" >&2 || true
  exit 1
fi

"$binary" > "$output/launch.log" 2>&1 &
app_pid=$!

record_windows() {
  # Preserve diagnostics in the QA artifact when the GUI cannot be located.
  wmctrl -lpx > "$output/window-list.txt" 2>&1 || true
  xwininfo -root -tree > "$output/window-tree.txt" 2>&1 || true
  xdotool search --onlyvisible --name '[Gg]host.?[Ff][Tt][Pp]' \
    > "$output/title-matches.txt" 2>&1 || true
}

window=""
for attempt in $(seq 1 80); do
  if ! kill -0 "$app_pid" 2>/dev/null; then
    record_windows
    echo "Ghost FTP Linux executable quit before opening a visible window." >&2
    tail -n 35 "$output/launch.log" >&2 || true
    exit 1
  fi
  # Prefer the X11 window that belongs to our exact process, rather than
  # assuming one fixed GTK title (some WebKit/Tauri versions name it ghostftp).
  window="$(wmctrl -lp 2>/dev/null | awk -v pid="$app_pid" '$3 == pid { print $1; exit }' || true)"
  if [ -z "$window" ]; then
    window="$(xdotool search --onlyvisible --pid "$app_pid" 2>/dev/null | head -n1 || true)"
  fi
  if [ -z "$window" ]; then
    window="$(xdotool search --onlyvisible --name '[Gg]host.?[Ff][Tt][Pp]' 2>/dev/null | head -n1 || true)"
  fi
  if [ -z "$window" ]; then
    window="$(xdotool search --onlyvisible --class '[Gg]host.?[Ff][Tt][Pp]' 2>/dev/null | head -n1 || true)"
  fi
  if [ -n "$window" ]; then break; fi
  sleep 0.3
done
record_windows
if [ -z "$window" ]; then
  echo "Ghost FTP process remained alive but no mapped GUI window was found by PID, title or WM_CLASS." >&2
  cat "$output/window-manager-state.txt" "$output/window-list.txt" >&2 || true
  tail -n 35 "$output/window-tree.txt" >&2 || true
  tail -n 35 "$output/launch.log" >&2 || true
  exit 1
fi
echo "Discovered actual Ghost FTP X11 window: $window (app PID $app_pid)"
xdotool getwindowname "$window" > "$output/window-title.txt" 2>&1 || true
# The user-approved Linux screens are 1290 x 852 pixels. Tauri is
# undecorated; move the real top-level window to (0, 0) before capture.
xdotool windowsize "$window" 1290 852
xdotool windowmove "$window" 0 0
sleep 4
scrot --silent --overwrite "$output/Linux-Files-1290x852.png"

python3 - "$output/Linux-Files-1290x852.png" <<'PYTHON'
import pathlib
import struct
import sys
p = pathlib.Path(sys.argv[1])
header = p.open("rb").read(24)
if len(header) != 24 or header[:8] != b"\x89PNG\r\n\x1a\n" or header[12:16] != b"IHDR":
    raise SystemExit("Linux native screenshot is not a valid PNG")
width, height = struct.unpack(">II", header[16:24])
if (width, height) != (1290, 852):
    raise SystemExit(f"Linux screenshot has unexpected size: {width}x{height}")
if p.stat().st_size < 18000:
    raise SystemExit("Linux UI screenshot appears empty or blank")
print(f"Captured genuine Linux Ghost FTP UI: {width}x{height}, {p.stat().st_size} bytes")
PYTHON
sha256sum "$output/Linux-Files-1290x852.png" > "$output/SHA256SUMS.txt"
cat > "$output/README.txt" <<'TEXT'
Ghost FTP real Linux native QA screenshot at 1290 x 852 from Tauri/WebKit
running in an isolated GitHub Actions Xvfb session with no user profiles.
This is evidence for comparison, not a claim of 1:1 pixel parity with the
user-supplied premium 29-screen Linux concept. No fabricated server files,
connections, transfer history, credentials or screenshots are injected.
TEXT
