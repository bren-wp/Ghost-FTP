#!/usr/bin/env bash
set -euo pipefail

DIAG_DIR="dist/android/diagnostics"
DEBUG_APK="android/app/build/outputs/apk/debug/app-debug.apk"
PACKAGE_ID="com.ghostftp.android"

mkdir -p "$DIAG_DIR"

collect_diagnostics() {
  adb shell dumpsys window windows > "$DIAG_DIR/window.txt" 2>/dev/null || true
  adb shell dumpsys activity activities > "$DIAG_DIR/activities.txt" 2>/dev/null || true
  adb logcat -d -v threadtime > "$DIAG_DIR/logcat.txt" 2>/dev/null || true
  adb exec-out screencap -p > "$DIAG_DIR/post-test-screen.png" 2>/dev/null || true
}

on_exit() {
  local status=$?
  if [ "$status" -ne 0 ]; then
    collect_diagnostics
  fi
}
trap on_exit EXIT

adb wait-for-device
adb shell input keyevent KEYCODE_WAKEUP || true
adb shell wm dismiss-keyguard || true
adb shell svc power stayon true || true
adb shell settings put system screen_off_timeout 2147483647 || true
adb shell settings put global window_animation_scale 0.0 || true
adb shell settings put global transition_animation_scale 0.0 || true
adb shell settings put global animator_duration_scale 0.0 || true

set +e
gradle -p android connectedDebugAndroidTest --stacktrace
TEST_EXIT=$?
set -e

collect_diagnostics

if [ "$TEST_EXIT" -ne 0 ]; then
  echo "Android instrumentation failed with exit code $TEST_EXIT." >&2
  exit "$TEST_EXIT"
fi

test -s "$DEBUG_APK"

# connectedDebugAndroidTest can remove the target application as part of test
# cleanup. Reinstall the exact debug APK that passed instrumentation before
# validating a real launcher start and screenshot.
set +e
INSTALL_OUTPUT="$(adb install -r "$DEBUG_APK" 2>&1)"
INSTALL_EXIT=$?
set -e
printf '%s\n' "$INSTALL_OUTPUT"

if [ "$INSTALL_EXIT" -ne 0 ]; then
  echo "Ghost FTP debug APK reinstall failed after instrumentation cleanup." >&2
  exit "$INSTALL_EXIT"
fi
case "$INSTALL_OUTPUT" in
  *Success*) ;;
  *)
    echo "Ghost FTP debug APK reinstall did not report Success." >&2
    exit 1
    ;;
esac

set +e
PACKAGE_PATH="$(adb shell pm path "$PACKAGE_ID" 2>&1)"
PACKAGE_EXIT=$?
set -e
PACKAGE_PATH="${PACKAGE_PATH//$'\r'/}"
printf 'Installed package path: %s\n' "$PACKAGE_PATH"

if [ "$PACKAGE_EXIT" -ne 0 ]; then
  echo "Package Manager could not resolve Ghost FTP after reinstall." >&2
  exit "$PACKAGE_EXIT"
fi
case "$PACKAGE_PATH" in
  package:*) ;;
  *)
    echo "Ghost FTP package is not installed after instrumentation cleanup recovery." >&2
    exit 1
    ;;
esac

set +e
LAUNCH_COMPONENT="$(
  adb shell cmd package resolve-activity --brief \
    -a android.intent.action.MAIN \
    -c android.intent.category.LAUNCHER \
    -p "$PACKAGE_ID" 2>&1
)"
RESOLVE_EXIT=$?
set -e
LAUNCH_COMPONENT="${LAUNCH_COMPONENT//
FOCUSED_WINDOW=""
WINDOW_DUMP=""
for _ in {1..20}; do
  WINDOW_DUMP="$(adb shell dumpsys window 2>&1)"
  while IFS= read -r window_line; do
    case "$window_line" in
      *mCurrentFocus*|*mFocusedApp*)
        FOCUSED_WINDOW="$window_line"
        break
        ;;
    esac
  done <<< "$WINDOW_DUMP"

  case "$FOCUSED_WINDOW" in
    *"$PACKAGE_ID"*) break ;;
  esac

  FOCUSED_WINDOW=""
  sleep 0.25
done

printf 'Focused window: %s\n' "$FOCUSED_WINDOW"
case "$FOCUSED_WINDOW" in
  *"$PACKAGE_ID"*) ;;
  *)
    printf '%s\n' "$WINDOW_DUMP" > "$DIAG_DIR/post-launch-window.txt"
    echo "Ghost FTP did not own the focused window after launch." >&2
    exit 1
    ;;
esac

APP_PID="$(adb shell pidof "$PACKAGE_ID" 2>/dev/null || true)"
APP_PID="${APP_PID//$'\r'/}"
test -n "$APP_PID"
printf 'Ghost FTP PID: %s\n' "$APP_PID"

adb shell dumpsys window windows > "$DIAG_DIR/post-launch-window.txt" || true
adb exec-out screencap -p > dist/android/GhostFTP-Android-UI-Smoke.png
test -s dist/android/GhostFTP-Android-UI-Smoke.png

trap - EXIT
echo "Ghost FTP Android instrumentation and post-launch smoke OK"
\r'/}"
printf 'Resolved launcher component: %s\n' "$LAUNCH_COMPONENT"

if [ "$RESOLVE_EXIT" -ne 0 ]; then
  echo "Package Manager could not resolve the Ghost FTP launcher activity." >&2
  exit "$RESOLVE_EXIT"
fi
case "$LAUNCH_COMPONENT" in
  "$PACKAGE_ID"/*) ;;
  *)
    echo "Unexpected Ghost FTP launcher component: $LAUNCH_COMPONENT" >&2
    exit 1
    ;;
esac

adb shell am force-stop "$PACKAGE_ID" || true

set +e
LAUNCH_OUTPUT="$(adb shell am start -W -n "$LAUNCH_COMPONENT" 2>&1)"
LAUNCH_EXIT=$?
set -e
printf '%s\n' "$LAUNCH_OUTPUT"

if [ "$LAUNCH_EXIT" -ne 0 ]; then
  echo "Ghost FTP launcher command failed with exit code $LAUNCH_EXIT." >&2
  exit "$LAUNCH_EXIT"
fi
case "$LAUNCH_OUTPUT" in
  *"Status: ok"*"$PACKAGE_ID"*) ;;
  *)
    echo "Ghost FTP launcher activity did not report a successful app launch." >&2
    exit 1
    ;;
esac

FOCUSED_WINDOW=""
WINDOW_DUMP=""
for _ in {1..20}; do
  WINDOW_DUMP="$(adb shell dumpsys window 2>&1)"
  while IFS= read -r window_line; do
    case "$window_line" in
      *mCurrentFocus*|*mFocusedApp*)
        FOCUSED_WINDOW="$window_line"
        break
        ;;
    esac
  done <<< "$WINDOW_DUMP"

  case "$FOCUSED_WINDOW" in
    *"$PACKAGE_ID"*) break ;;
  esac

  FOCUSED_WINDOW=""
  sleep 0.25
done

printf 'Focused window: %s\n' "$FOCUSED_WINDOW"
case "$FOCUSED_WINDOW" in
  *"$PACKAGE_ID"*) ;;
  *)
    printf '%s\n' "$WINDOW_DUMP" > "$DIAG_DIR/post-launch-window.txt"
    echo "Ghost FTP did not own the focused window after launch." >&2
    exit 1
    ;;
esac

APP_PID="$(adb shell pidof "$PACKAGE_ID" 2>/dev/null || true)"
APP_PID="${APP_PID//$'\r'/}"
test -n "$APP_PID"
printf 'Ghost FTP PID: %s\n' "$APP_PID"

adb shell dumpsys window windows > "$DIAG_DIR/post-launch-window.txt" || true
adb exec-out screencap -p > dist/android/GhostFTP-Android-UI-Smoke.png
test -s dist/android/GhostFTP-Android-UI-Smoke.png

trap - EXIT
echo "Ghost FTP Android instrumentation and post-launch smoke OK"
