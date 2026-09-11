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
SDK_ROOT="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-}}"
EMULATOR_BIN="${SDK_ROOT:+$SDK_ROOT/emulator/emulator}"
AAPT_BIN="${SDK_ROOT:+$SDK_ROOT/build-tools/35.0.0/aapt}"
AVD_HOME="${GHOSTFTP_UI_AVD_HOME:-${RUNNER_TEMP:-/tmp}/ghostftp-avd}"

[[ -s "$APK_PATH" ]] || {
  echo "Missing Android APK: $APK_PATH" >&2
  exit 1
}
[[ -n "$SDK_ROOT" && -x "$EMULATOR_BIN" ]] || {
  echo "Installed Android emulator binary is unavailable: ${EMULATOR_BIN:-<unset SDK root>}" >&2
  exit 1
}
[[ -x "$AAPT_BIN" ]] || {
  echo "Installed Android aapt binary is unavailable: ${AAPT_BIN:-<unset SDK root>}" >&2
  exit 1
}
printf 'ANDROID_EMULATOR_BIN=%s\n' "$EMULATOR_BIN"

# Read the exact installed identity from the APK rather than duplicating Gradle's
# debug applicationId suffix or source namespace in the capture harness.
badging="$("$AAPT_BIN" dump badging "$APK_PATH")"
package_name="$(printf '%s\n' "$badging" | sed -n "s/^package: name='\([^']*\)'.*/\1/p" | head -n1)"
launcher_activity="$(printf '%s\n' "$badging" | sed -n "s/^launchable-activity: name='\([^']*\)'.*/\1/p" | head -n1)"
[[ -n "$package_name" && -n "$launcher_activity" ]] || {
  printf '%s\n' "$badging" >&2
  echo 'Unable to resolve the Android package and launcher activity from the exact APK.' >&2
  exit 1
}
case "$launcher_activity" in
  .*) launcher_component="${package_name}/${package_name}${launcher_activity}" ;;
  *) launcher_component="${package_name}/${launcher_activity}" ;;
esac
printf 'ANDROID_APK_IDENTITY=PACKAGE=%s ACTIVITY=%s COMPONENT=%s\n' \
  "$package_name" "$launcher_activity" "$launcher_component"

mkdir -p "$OUTPUT_DIR"
if [[ -e /dev/kvm ]]; then
  sudo chmod 666 /dev/kvm
fi

# avdmanager and emulator can otherwise resolve different HOME/SDK locations on
# hosted runners. Give both tools one disposable AVD registry and prove the AVD
# is visible to the exact emulator binary before attempting boot.
rm -rf "$AVD_HOME"
install -d -m 700 "$AVD_HOME"
export ANDROID_AVD_HOME="$AVD_HOME"
printf 'no\n' | avdmanager create avd --force --name "$AVD_NAME" --package "$SYSTEM_IMAGE" --device "$DEVICE"
if ! "$EMULATOR_BIN" -list-avds | grep -Fxq "$AVD_NAME"; then
  find "$AVD_HOME" -maxdepth 2 -type f -print >&2 || true
  echo "Android AVD was not registered in the shared AVD home: $AVD_HOME" >&2
  exit 1
fi
printf 'ANDROID_AVD_READY=%s HOME=%s\n' "$AVD_NAME" "$AVD_HOME"

"$EMULATOR_BIN" -avd "$AVD_NAME" -no-window -no-audio -no-boot-anim -no-snapshot -gpu swiftshader_indirect >"$EMULATOR_LOG" 2>&1 &
emulator_pid=$!
cleanup() {
  timeout 10s adb emu kill >/dev/null 2>&1 || true
  kill "$emulator_pid" 2>/dev/null || true
  rm -rf "$AVD_HOME"
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
timeout 10s adb shell am force-stop "$package_name"
timeout 10s adb shell am start -W -n "$launcher_component"
sleep 2

# Android 35 reports the foreground activity as `ResumedActivity:` (and also
# `topResumedActivity=`). Validate the exact launcher component instead of
# parsing the older `mResumedActivity:` key, which is absent on this runtime.
activity_dump="$(timeout 10s adb shell dumpsys activity activities 2>/dev/null || true)"
if ! printf '%s\n' "$activity_dump" | grep -F 'ResumedActivity:' | grep -Fq "$launcher_component"; then
  printf '%s\n' "$activity_dump" >&2
  echo "Android launcher did not become foreground: expected component $launcher_component." >&2
  exit 1
fi
printf 'ANDROID_FOREGROUND=%s\n' "$launcher_component"

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
