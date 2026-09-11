#!/usr/bin/env bash
set -euo pipefail

APK_PATH="${1:-android/dist/Ghost-FTP-Android.apk}"
OUTPUT_DIR="${2:-ui-screenshots/android}"
AVD_NAME="${GHOSTFTP_UI_AVD_NAME:-ghostftp-ui}"
SYSTEM_IMAGE="${GHOSTFTP_UI_SYSTEM_IMAGE:-system-images;android-35;google_apis;x86_64}"
DEVICE="${GHOSTFTP_UI_DEVICE:-pixel_6}"

[[ -s "$APK_PATH" ]] || {
  echo "Missing Android APK: $APK_PATH" >&2
  exit 1
}

mkdir -p "$OUTPUT_DIR"
if [[ -e /dev/kvm ]]; then
  sudo chmod 666 /dev/kvm
fi

printf 'no\n' | avdmanager create avd --force --name "$AVD_NAME" --package "$SYSTEM_IMAGE" --device "$DEVICE"
emulator -avd "$AVD_NAME" -no-window -no-audio -no-boot-anim -gpu swiftshader_indirect >"$RUNNER_TEMP/android-emulator.log" 2>&1 &
emulator_pid=$!
cleanup() {
  adb emu kill >/dev/null 2>&1 || true
  kill "$emulator_pid" 2>/dev/null || true
}
trap cleanup EXIT

adb wait-for-device
booted=''
for _ in $(seq 1 180); do
  booted="$(adb shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')"
  [[ "$booted" == '1' ]] && break
  sleep 1
done
if [[ "$booted" != '1' ]]; then
  cat "$RUNNER_TEMP/android-emulator.log" >&2 || true
  exit 1
fi

adb shell settings put global window_animation_scale 0
adb shell settings put global transition_animation_scale 0
adb shell settings put global animator_duration_scale 0
adb install -r "$APK_PATH"
adb shell am force-stop app.ghostftp.client
adb shell am start -n app.ghostftp.client/.MainActivity
sleep 2

tap_ui() {
  local query="$1"
  adb shell uiautomator dump /sdcard/ghostftp-window.xml >/dev/null
  adb pull /sdcard/ghostftp-window.xml "$RUNNER_TEMP/ghostftp-window.xml" >/dev/null
  local coords
  coords="$(python3 - "$RUNNER_TEMP/ghostftp-window.xml" "$query" <<'PY'
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
raise SystemExit(f"UI node not found: {query}")
PY
)"
  read -r x y <<<"$coords"
  adb shell input tap "$x" "$y"
  sleep 0.7
}

capture() {
  local name="$1"
  adb exec-out screencap -p >"$OUTPUT_DIR/$name"
  [[ -s "$OUTPUT_DIR/$name" ]] || {
    echo "Empty Android screenshot: $name" >&2
    exit 1
  }
}

capture 'ghost-ftp-android-files.png'
tap_ui 'Open navigation'
capture 'ghost-ftp-android-navigation.png'
adb shell input keyevent KEYCODE_BACK
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
