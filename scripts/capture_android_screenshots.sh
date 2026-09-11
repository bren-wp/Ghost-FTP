#!/usr/bin/env bash
set -euo pipefail

APK_PATH="${1:-android/dist/Ghost-FTP-Android.apk}"
OUTPUT_DIR="${2:-ui-screenshots/android}"
AVD_NAME="${GHOSTFTP_UI_AVD_NAME:-ghostftp-ui}"
SYSTEM_IMAGE="${GHOSTFTP_UI_SYSTEM_IMAGE:-system-images;android-35;google_apis;x86_64}"
DEVICE="${GHOSTFTP_UI_DEVICE:-pixel_6}"
EMULATOR_LOG="${RUNNER_TEMP:-/tmp}/android-emulator.log"
UI_XML_DEVICE="/sdcard/ghostftp-window.xml"
UI_XML_LOCAL="${RUNNER_TEMP:-/tmp}/ghostftp-window.xml"

[[ -s "$APK_PATH" ]] || {
  echo "Missing Android APK: $APK_PATH" >&2
  exit 1
}

mkdir -p "$OUTPUT_DIR"
if [[ -e /dev/kvm ]]; then
  sudo chmod 666 /dev/kvm
fi

printf 'no\n' | avdmanager create avd --force --name "$AVD_NAME" --package "$SYSTEM_IMAGE" --device "$DEVICE"
emulator -avd "$AVD_NAME" -no-window -no-audio -no-boot-anim -no-snapshot -gpu swiftshader_indirect >"$EMULATOR_LOG" 2>&1 &
emulator_pid=$!
cleanup() {
  timeout 10s adb emu kill >/dev/null 2>&1 || true
  kill "$emulator_pid" 2>/dev/null || true
}
trap cleanup EXIT

# Never let a dead or unreachable emulator block this evidence workflow
# indefinitely. Poll both the adb transport and Android boot property while also
# proving that the emulator process is still alive.
timeout 10s adb start-server >/dev/null 2>&1 || true
booted=''
for _ in $(seq 1 180); do
  if ! kill -0 "$emulator_pid" 2>/dev/null; then
    cat "$EMULATOR_LOG" >&2 || true
    echo 'Android emulator terminated before becoming ready.' >&2
    exit 1
  fi

  state="$(timeout 5s adb get-state 2>/dev/null || true)"
  if [[ "$state" == 'device' ]]; then
    booted="$(timeout 5s adb shell getprop sys.boot_completed 2>/dev/null | tr -d '\r' || true)"
    [[ "$booted" == '1' ]] && break
  fi
  sleep 1
done
if [[ "$booted" != '1' ]]; then
  cat "$EMULATOR_LOG" >&2 || true
  echo 'Android emulator did not complete boot within the bounded capture window.' >&2
  exit 1
fi
printf 'ANDROID_EMULATOR_BOOT=PASS PID=%s\n' "$emulator_pid"

timeout 10s adb shell settings put global window_animation_scale 0
timeout 10s adb shell settings put global transition_animation_scale 0
timeout 10s adb shell settings put global animator_duration_scale 0
timeout 120s adb install -r "$APK_PATH"
timeout 10s adb shell am force-stop app.ghostftp.client
timeout 10s adb shell am start -n app.ghostftp.client/.MainActivity
sleep 2

dump_ui() {
  rm -f "$UI_XML_LOCAL"
  for _ in $(seq 1 20); do
    if ! kill -0 "$emulator_pid" 2>/dev/null; then
      cat "$EMULATOR_LOG" >&2 || true
      echo 'Android emulator terminated during UI capture.' >&2
      return 1
    fi
    # uiautomator dump is real runtime accessibility evidence, but on a busy
    # emulator it can occasionally stall. Bound each attempt and retry.
    if timeout 10s adb shell uiautomator dump "$UI_XML_DEVICE" >/dev/null 2>&1 &&
       timeout 10s adb pull "$UI_XML_DEVICE" "$UI_XML_LOCAL" >/dev/null 2>&1 &&
       [[ -s "$UI_XML_LOCAL" ]]; then
      return 0
    fi
    sleep 0.5
  done
  cat "$EMULATOR_LOG" >&2 || true
  echo 'Unable to obtain a bounded Android UI hierarchy dump.' >&2
  return 1
}

find_ui_coords() {
  local query="$1"
  python3 - "$UI_XML_LOCAL" "$query" <<'PY'
import re
import sys
import xml.etree.ElementTree as ET

path, query = sys.argv[1], sys.argv[2]
root = ET.parse(path).getroot()
for node in root.iter("node"):
    if node.attrib.get("text") == query or node.attrib.get("content-desc") == query:
        match = re.fullmatch(r"\[(\d+),(\d+)\]\[(\d+),(\d+)\]", node.attrib.get("bounds", ""))
        if match:
            x1, y1, x2, y2 = map(int, match.groups())
            print(f"{(x1 + x2) // 2} {(y1 + y2) // 2}")
            raise SystemExit(0)
raise SystemExit(1)
PY
}

tap_ui() {
  local query="$1"
  local coords=''
  for _ in $(seq 1 20); do
    dump_ui
    coords="$(find_ui_coords "$query" 2>/dev/null || true)"
    if [[ "$coords" =~ ^[0-9]+\ [0-9]+$ ]]; then
      read -r x y <<<"$coords"
      timeout 10s adb shell input tap "$x" "$y"
      sleep 0.7
      printf 'ANDROID_UI_TAP=%s X=%s Y=%s\n' "$query" "$x" "$y"
      return 0
    fi
    sleep 0.4
  done
  echo "UI node not found after bounded retries: $query" >&2
  return 1
}

capture() {
  local name="$1"
  if ! timeout 15s adb exec-out screencap -p >"$OUTPUT_DIR/$name"; then
    echo "Android screenshot command timed out: $name" >&2
    exit 1
  fi
  [[ -s "$OUTPUT_DIR/$name" ]] || {
    echo "Empty Android screenshot: $name" >&2
    exit 1
  }
}

capture 'ghost-ftp-android-files.png'
tap_ui 'Open navigation'
capture 'ghost-ftp-android-navigation.png'
timeout 10s adb shell input keyevent KEYCODE_BACK
sleep 0.4

for section in Sites Bookmarks Transfers Settings About; do
  tap_ui 'Open navigation'
  tap_ui "$section"
  lower="$(printf '%s' "$section" | tr '[:upper:]' '[:lower:]')"
  capture "ghost-ftp-android-${lower}.png"
done

for png in "$OUTPUT_DIR"/*.png; do
  dims="$(identify -format '%w %h %k' "$png")"
  read -r width height colors <<<"$dims"
  if (( width < 320 || height < 480 || colors < 8 )); then
    echo "Implausible Android screenshot: $png ($dims)" >&2
    exit 1
  fi
  printf 'ANDROID_UI=%s SHA256=%s\n' "$(basename "$png")" "$(sha256sum "$png" | awk '{print $1}')"
done

printf 'ANDROID_UI_SCREENSHOTS=PASS\n'
