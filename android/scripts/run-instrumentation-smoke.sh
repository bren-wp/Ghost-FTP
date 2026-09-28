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

# ATD can surface unrelated background-system crash dialogs (observed from
# com.android.bluetooth) that steal Espresso's window focus. Suppress those
# dialogs and stop the unused Bluetooth stack before launching instrumentation.
adb shell settings put global hide_error_dialogs 1
HIDE_ERROR_DIALOGS="$(adb shell settings get global hide_error_dialogs 2>/dev/null || true)"
case "$HIDE_ERROR_DIALOGS" in
  1*) ;;
  *)
    echo "Unable to enable Android global hide_error_dialogs before UI tests." >&2
    exit 1
    ;;
esac
adb shell settings put global show_first_crash_dialog 0 || true
adb shell settings put global show_restart_in_crash_dialog 0 || true
adb shell cmd bluetooth_manager disable || true
adb shell am force-stop com.android.bluetooth || true
adb shell am broadcast -a android.intent.action.CLOSE_SYSTEM_DIALOGS >/dev/null 2>&1 || true

PRETEST_WINDOW_DUMP=""
for _ in {1..4}; do
  PRETEST_WINDOW_DUMP="$(adb shell dumpsys window windows 2>&1)"
  case "$PRETEST_WINDOW_DUMP" in
    *"Application Error:"*)
      adb shell input keyevent KEYCODE_BACK || true
      adb shell am force-stop com.android.bluetooth || true
      sleep 0.5
      ;;
    *) break ;;
  esac
done
case "$PRETEST_WINDOW_DUMP" in
  *"Application Error:"*)
    printf '%s\n' "$PRETEST_WINDOW_DUMP" > "$DIAG_DIR/pre-test-window.txt"
    echo "A system crash dialog still owns the emulator before instrumentation." >&2
    exit 1
    ;;
esac

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

# connectedDebugAndroidTest can remove the target package during cleanup.
# Reinstall the exact APK that passed instrumentation before the final launch smoke.
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

AAPT="$(find "$ANDROID_HOME/build-tools" -maxdepth 2 -type f -name aapt | sort -V | tail -n1)"
test -x "$AAPT"

APK_BADGING="$("$AAPT" dump badging "$DEBUG_APK")"
LAUNCH_ACTIVITY=""
while IFS= read -r badging_line; do
  case "$badging_line" in
    launchable-activity:*)
      launch_value="${badging_line#*name=\'}"
      LAUNCH_ACTIVITY="${launch_value%%\'*}"
      break
      ;;
  esac
done <<< "$APK_BADGING"

case "$LAUNCH_ACTIVITY" in
  "$PACKAGE_ID".*) ;;
  *)
    echo "Debug APK does not expose the expected Ghost FTP launchable activity: $LAUNCH_ACTIVITY" >&2
    exit 1
    ;;
esac

LAUNCH_COMPONENT="$PACKAGE_ID/$LAUNCH_ACTIVITY"
printf 'APK launcher component: %s\n' "$LAUNCH_COMPONENT"

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

RESUMED_ACTIVITY=""
ACTIVITY_DUMP=""
for _ in {1..20}; do
  ACTIVITY_DUMP="$(adb shell dumpsys activity activities 2>&1)"
  while IFS= read -r activity_line; do
    case "$activity_line" in
      *ResumedActivity*"$PACKAGE_ID/"*|*topResumedActivity*"$PACKAGE_ID/"*)
        RESUMED_ACTIVITY="$activity_line"
        break
        ;;
    esac
  done <<< "$ACTIVITY_DUMP"

  if [ -n "$RESUMED_ACTIVITY" ]; then
    break
  fi

  sleep 0.25
done

printf 'Resumed activity: %s\n' "$RESUMED_ACTIVITY"
if [ -z "$RESUMED_ACTIVITY" ]; then
  printf '%s\n' "$ACTIVITY_DUMP" > "$DIAG_DIR/post-launch-activities.txt"
  echo "Ghost FTP MainActivity did not reach RESUMED state after launch." >&2
  exit 1
fi

# Android ATD images can surface an unrelated com.android.bluetooth crash dialog
# above the app. It must not invalidate Ghost FTP instrumentation, but dismiss
# that known emulator-only overlay before collecting visual evidence.
for _ in {1..3}; do
  WINDOW_DUMP="$(adb shell dumpsys window windows 2>&1)"
  CURRENT_FOCUS=""
  while IFS= read -r window_line; do
    case "$window_line" in
      *mCurrentFocus*)
        CURRENT_FOCUS="$window_line"
        break
        ;;
    esac
  done <<< "$WINDOW_DUMP"

  case "$CURRENT_FOCUS" in
    *"Application Error: com.android.bluetooth"*)
      echo "Dismissing Android ATD Bluetooth crash overlay before screenshot."
      adb shell input keyevent KEYCODE_BACK || true
      sleep 0.25
      ;;
    *)
      break
      ;;
  esac
done

APP_PID="$(adb shell pidof "$PACKAGE_ID" 2>/dev/null || true)"
test -n "$APP_PID"
printf 'Ghost FTP PID: %s\n' "$APP_PID"

adb shell dumpsys activity activities > "$DIAG_DIR/post-launch-activities.txt" || true
adb shell dumpsys window windows > "$DIAG_DIR/post-launch-window.txt" || true
adb exec-out screencap -p > dist/android/GhostFTP-Android-UI-Smoke.png
test -s dist/android/GhostFTP-Android-UI-Smoke.png

trap - EXIT
echo "Ghost FTP Android instrumentation and post-launch smoke OK"
