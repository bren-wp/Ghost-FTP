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
NAV_ANCHOR_X=''
NAV_FILES_Y=''
NAV_ROW_PITCH=''

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

# Fail closed if the exact APK package was installed but its launcher did not
# actually become the foreground activity. Android dumpsys field names differ
# between platform releases, so parse both ActivityTaskManager and WindowManager
# evidence instead of depending on one historical mResumedActivity label.
activity_dump="$(timeout 10s adb shell dumpsys activity activities 2>/dev/null | tr -d '\r' || true)"
resolved_component="$(printf '%s\n' "$activity_dump" | awk '
/topResumedActivity=ActivityRecord|ResumedActivity: ActivityRecord/ {
  for (i = 1; i <= NF; i++) {
    candidate = $i
    gsub(/[{}]/, "", candidate)
    if (candidate ~ /^[[:alnum:]_.]+\/[[:alnum:]_.$]+$/) {
      print candidate
      exit
    }
  }
}')"
window_dump=''
if [[ -z "$resolved_component" ]]; then
  window_dump="$(timeout 10s adb shell dumpsys window windows 2>/dev/null | tr -d '\r' || true)"
  resolved_component="$(printf '%s\n' "$window_dump" | awk '
/mCurrentFocus=Window|mFocusedApp=ActivityRecord/ {
  for (i = 1; i <= NF; i++) {
    candidate = $i
    gsub(/[{}]/, "", candidate)
    if (candidate ~ /^[[:alnum:]_.]+\/[[:alnum:]_.$]+$/) {
      print candidate
      exit
    }
  }
}')"
fi
[[ "$resolved_component" == "$package_name/"* ]] || {
  printf '%s\n' "$activity_dump" >&2
  [[ -z "$window_dump" ]] || printf '%s\n' "$window_dump" >&2
  echo "Android launcher did not become foreground: expected package $package_name, got ${resolved_component:-<none>}." >&2
  exit 1
}
printf 'ANDROID_FOREGROUND=%s\n' "$resolved_component"

dump_ui() {
  rm -f "$UI_XML_LOCAL"
  for _ in $(seq 1 20); do
    if ! kill -0 "$emulator_pid" 2>/dev/null; then
      cat "$EMULATOR_LOG" >&2 || true
      echo 'Android emulator terminated during UI capture.' >&2
      return 1
    fi
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

find_ui_bounds() {
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
            print(" ".join(match.groups()))
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

wait_ui() {
  local query="$1"
  local coords=''
  for _ in $(seq 1 20); do
    dump_ui
    coords="$(find_ui_coords "$query" 2>/dev/null || true)"
    if [[ "$coords" =~ ^[0-9]+\ [0-9]+$ ]]; then
      read -r x y <<<"$coords"
      printf 'ANDROID_UI_VISIBLE=%s X=%s Y=%s\n' "$query" "$x" "$y"
      return 0
    fi
    sleep 0.4
  done
  echo "Expected UI node did not become visible: $query" >&2
  [[ -s "$UI_XML_LOCAL" ]] && cat "$UI_XML_LOCAL" >&2 || true
  return 1
}

# Android 35 can omit the programmatically created drawer Button nodes from the
# uiautomator hierarchy even though the drawer's stable text labels remain
# exposed. Anchor the coordinate fallback to the bottom edge of the actual
# "Android development client" TextView. MainActivity adds Files immediately
# after that TextView, then six navigation rows with a 48dp minimum height and
# 5dp bottom margin. This keeps font metrics and top padding runtime-derived
# instead of guessed. Every coordinate tap is still followed by the requested
# section-title assertion, so a wrong fallback cannot produce PASS.
calibrate_navigation_geometry() {
  local screenshot="$OUTPUT_DIR/ghost-ftp-android-navigation.png"
  local dims=''
  local screen_width=''
  local screen_height=''
  local platform_bounds=''
  local platform_x1=''
  local platform_y1=''
  local platform_x2=''
  local platform_y2=''
  local density=''
  local button_height=''
  local row_margin=''
  local drawer_width=''

  [[ -s "$screenshot" ]] || {
    echo "Navigation screenshot is unavailable for geometry validation: $screenshot" >&2
    return 1
  }
  dims="$(identify -format '%w %h' "$screenshot" 2>/dev/null || true)"
  read -r screen_width screen_height <<<"$dims"
  [[ "$screen_width" =~ ^[0-9]+$ && "$screen_height" =~ ^[0-9]+$ ]] || {
    echo "Unable to read navigation screenshot dimensions: ${dims:-<none>}" >&2
    return 1
  }

  dump_ui
  platform_bounds="$(find_ui_bounds 'Android development client' 2>/dev/null || true)"
  read -r platform_x1 platform_y1 platform_x2 platform_y2 <<<"$platform_bounds"
  [[ "$platform_x1" =~ ^[0-9]+$ && "$platform_y1" =~ ^[0-9]+$ \
      && "$platform_x2" =~ ^[0-9]+$ && "$platform_y2" =~ ^[0-9]+$ ]] || {
    echo "Unable to resolve runtime drawer header bounds: ${platform_bounds:-<none>}" >&2
    [[ -s "$UI_XML_LOCAL" ]] && cat "$UI_XML_LOCAL" >&2 || true
    return 1
  }

  density="$(timeout 10s adb shell wm density | tr -d '\r' | awk -F': ' '/Physical density/{v=$2} /Override density/{v=$2} END{print v}')"
  [[ "$density" =~ ^[0-9]+$ ]] || {
    echo "Unable to resolve emulator density for navigation calibration: ${density:-<none>}" >&2
    return 1
  }
  button_height=$(((48 * density + 80) / 160))
  row_margin=$(((5 * density + 80) / 160))
  drawer_width=$(((286 * density + 80) / 160))
  (( drawer_width > screen_width )) && drawer_width="$screen_width"

  if (( platform_x1 < 0 || platform_x2 <= platform_x1 || platform_x2 > drawer_width \
        || platform_y1 < 0 || platform_y2 <= platform_y1 || platform_y2 >= screen_height / 2 )); then
    echo "Implausible runtime drawer header bounds: $platform_bounds drawer_width=$drawer_width screen=${screen_width}x${screen_height}" >&2
    return 1
  fi

  NAV_ANCHOR_X=$((drawer_width / 2))
  NAV_FILES_Y=$((platform_y2 + button_height / 2))
  NAV_ROW_PITCH=$((button_height + row_margin))
  if (( NAV_ANCHOR_X <= 0 || NAV_FILES_Y <= platform_y2 || NAV_ROW_PITCH <= 0 \
        || NAV_FILES_Y + 5 * NAV_ROW_PITCH >= screen_height )); then
    echo "Implausible navigation calibration: X=$NAV_ANCHOR_X FILES_Y=$NAV_FILES_Y PITCH=$NAV_ROW_PITCH" >&2
    return 1
  fi

  printf 'ANDROID_NAV_CALIBRATION=PASS X=%s FILES_Y=%s ROW_PITCH=%s PLATFORM_BOUNDS=%s DRAWER_WIDTH=%s DENSITY=%s\n' \
    "$NAV_ANCHOR_X" "$NAV_FILES_Y" "$NAV_ROW_PITCH" "$platform_bounds" "$drawer_width" "$density"
}

tap_nav_section() {
  local section="$1"
  local ordinal="$2"
  local expected_title="$3"
  local coords=''
  local x=''
  local y=''

  dump_ui
  coords="$(find_ui_coords "$section" 2>/dev/null || true)"
  if [[ "$coords" =~ ^[0-9]+\ [0-9]+$ ]]; then
    read -r x y <<<"$coords"
    timeout 10s adb shell input tap "$x" "$y"
    printf 'ANDROID_NAV_TAP=%s MODE=semantic X=%s Y=%s\n' "$section" "$x" "$y"
  else
    if [[ ! "$NAV_ANCHOR_X" =~ ^[0-9]+$ || ! "$NAV_FILES_Y" =~ ^[0-9]+$ || ! "$NAV_ROW_PITCH" =~ ^[0-9]+$ ]]; then
      calibrate_navigation_geometry
    fi
    x="$NAV_ANCHOR_X"
    y=$((NAV_FILES_Y + NAV_ROW_PITCH * ordinal))
    timeout 10s adb shell input tap "$x" "$y"
    printf 'ANDROID_NAV_TAP=%s MODE=runtime-header-anchor X=%s Y=%s FILES_Y=%s ROW_PITCH=%s\n' \
      "$section" "$x" "$y" "$NAV_FILES_Y" "$NAV_ROW_PITCH"
  fi
  sleep 0.7
  wait_ui "$expected_title"
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

first_section=1
ordinal=1
for section in Sites Bookmarks Transfers Settings About; do
  if (( first_section == 0 )); then
    tap_ui 'Open navigation'
  fi
  expected_title="$section"
  [[ "$section" == 'Sites' ]] && expected_title='Sites / Connections'
  tap_nav_section "$section" "$ordinal" "$expected_title"
  first_section=0
  ordinal=$((ordinal + 1))
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
