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
# Only actual application workspaces are capturable. All views are routed by
# the same Tauri initialization script and React components as Windows QA.
# Never inject synthetic server data, success states, or authentication.
qa_view="${GHOSTFTP_QA_VIEW:-main}"
case "$qa_view" in
  splash) qa_image="Linux-Splash-1290x852.png" ;;
  main) qa_image="Linux-Files-1290x852.png" ;;
  siteManager) qa_image="Linux-Sites-1290x852.png" ;;
  newConnection) qa_image="Linux-New-Connection-1290x852.png" ;;
  transferCenter) qa_image="Linux-Transfers-1290x852.png" ;;
  settings) qa_image="Linux-Settings-1290x852.png" ;;
  sync) qa_image="Linux-Sync-1290x852.png" ;;
  about) qa_image="Linux-About-1290x852.png" ;;
  *) echo "Unsupported native Linux QA screen: $qa_view" >&2; exit 2 ;;
esac
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
# A VM without /dev/dri must use a software renderer; do not disable the
# WebKit process sandbox or TLS/security checks to obtain screenshots.
export LIBGL_ALWAYS_SOFTWARE=1
# Do not set GDK_GL=disable: GTK/WebKit may fail to map the Tauri window.
# Use the ordinary GTK X11 path with Mesa's CPU renderer instead.
mkdir -p "$XDG_CONFIG_HOME" "$XDG_CACHE_HOME" "$XDG_DATA_HOME"

# xvfb-run starts its X server asynchronously. Some GitHub-hosted runners
# schedule this child after the capture script has already reached Openbox.
# Probe a real authenticated X11 connection before starting the window
# manager; a populated DISPLAY alone is not proof of a usable X server.
if [ -z "${DISPLAY:-}" ]; then
  echo "Linux capture requires an X11 DISPLAY from xvfb-run." >&2
  exit 1
fi
display_ready=0
for attempt in $(seq 1 80); do
  if xdpyinfo -display "$DISPLAY" > "$output/display-state.txt" 2> "$output/display-errors.log"; then
    display_ready=1
    break
  fi
  sleep 0.25
done
if [ "$display_ready" -ne 1 ]; then
  echo "Linux capture could not authenticate to the Xvfb display within 20 seconds." >&2
  cat "$output/display-errors.log" >&2 || true
  exit 1
fi

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

# GTK/WebKit can create a real top-level window before Openbox has
# registered it in _NET_CLIENT_LIST, or before X11 reports IsViewable.
# A post-merge Linux runner hit precisely this race: xwininfo -root -tree
# listed a 1258x820 "Ghost FTP" window while wmctrl and xdotool
# --onlyvisible returned nothing. Discover the *real* application window,
# validating its identity and size; never fall back to a screenshot of
# the desktop, a tiny helper process, or an illustration.
find_real_ghostftp_window() {
  local candidate title klass owner geom width height
  while IFS= read -r candidate; do
    [[ "$candidate" =~ ^(0x[[:xdigit:]]+|[0-9]+)$ ]] || continue
    title="$(xdotool getwindowname "$candidate" 2>/dev/null || true)"
    klass="$(xprop -id "$candidate" WM_CLASS 2>/dev/null || true)"
    owner="$(xdotool getwindowpid "$candidate" 2>/dev/null || true)"
    geom="$(xdotool getwindowgeometry --shell "$candidate" 2>/dev/null || true)"
    width="$(printf '%s\n' "$geom" | sed -n 's/^WIDTH=//p' | head -n1)"
    height="$(printf '%s\n' "$geom" | sed -n 's/^HEIGHT=//p' | head -n1)"
    # The GTK helper may have the same PID and WM_CLASS, but is only 10x10.
    [[ "$width" =~ ^[0-9]+$ && "$height" =~ ^[0-9]+$ ]] || continue
    (( width >= 500 && height >= 350 )) || continue
    # Accept exact owning process, or a distinctive Ghost FTP title+class
    # (some WebKit subprocesses own the window instead of the parent PID).
    if [[ "$owner" != "$app_pid" ]]; then
      [[ "$title" =~ [Gg]host[[:space:]]*[Ff][Tt][Pp] ]] || continue
      [[ "$klass" =~ [Gg]host[fF][tT][pP] ]] || continue
    fi
    printf '%s\n' "$candidate"
    return 0
  done < <(
    {
      wmctrl -lp 2>/dev/null | awk -v pid="$app_pid" '$3 == pid {print $1}' || true
      xdotool search --onlyvisible --pid "$app_pid" 2>/dev/null || true
      xdotool search --onlyvisible --name '[Gg]host.?[Ff][Tt][Pp]' 2>/dev/null || true
      xdotool search --pid "$app_pid" 2>/dev/null || true
      # These searches deliberately omit --onlyvisible; the verified
      # window may not be mapped yet even though it exists in the X tree.
      xdotool search --name '[Gg]host.?[Ff][Tt][Pp]' 2>/dev/null || true
      xdotool search --class '[Gg]host.?[Ff][Tt][Pp]' 2>/dev/null || true
      xwininfo -root -tree 2>/dev/null | awk '/"Ghost FTP":/ {print $1}' || true
    } | awk '!seen[$0]++'
  )
  return 1
}

window=""
for attempt in $(seq 1 80); do
  if ! kill -0 "$app_pid" 2>/dev/null; then
    record_windows
    echo "Ghost FTP Linux executable quit before opening a real GUI window." >&2
    tail -n 35 "$output/launch.log" >&2 || true
    exit 1
  fi
  window="$(find_real_ghostftp_window || true)"
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
# If Gtk produced the window before WM visibility registration, map and
# raise THIS verified window inside the disposable Xvfb session. The later
# real-pixel validator still rejects blank and structureless WebKit captures.
xdotool windowmap "$window" >/dev/null 2>&1 || true
xdotool windowraise "$window" >/dev/null 2>&1 || true
xdotool getwindowname "$window" > "$output/window-title.txt" 2>&1 || true
# The user-approved Linux screens are 1290 x 852 pixels. Tauri is
# undecorated; move the real top-level window to (0, 0) before capture.
xdotool windowsize "$window" 1290 852
xdotool windowmove "$window" 0 0

# The old test accepted a 1290x852 PNG that contained ONLY an Openbox
# titlebar around a fully black WebKit view. Size and byte count are
# insufficient. Keep waiting for genuine rendered controls and fail closed.
validator="$(dirname "$(realpath "${BASH_SOURCE[0]}")")/check-linux-premium-png.py"
test -s "$validator"
valid_capture=0
for attempt in $(seq 1 6); do
  sleep 5
  scrot --silent --overwrite "$output/$qa_image"
  echo "Capture attempt $attempt at 1290x852:" >> "$output/capture-attempts.log"
  if /usr/bin/python3 "$validator" "$output/$qa_image" >> "$output/capture-attempts.log" 2>&1; then
    valid_capture=1
    echo "Validated real Linux app content on attempt $attempt"
    break
  fi
done
if [ "$valid_capture" -ne 1 ]; then
  echo "Ghost FTP X11 window exists but its WebKit contents are still blank/structureless." >&2
  cat "$output/capture-attempts.log" >&2 || true
  echo "Openbox window evidence and the invalid PNG are preserved in the QA artifact." >&2
  exit 1
fi
(cd "$output" && sha256sum "$qa_image" > SHA256SUMS.txt)
cat > "$output/README.txt" <<TEXT
Ghost FTP actual Linux Tauri/WebKit native capture: $qa_view
Source: the running application; isolated GitHub Actions Xvfb session.
Image: $qa_image (1290x852). No user profiles, credentials, fabricated
server data or mocked transfer states are inserted.
This is visual evidence, NOT a claim of pixel-perfect acceptance.
TEXT
