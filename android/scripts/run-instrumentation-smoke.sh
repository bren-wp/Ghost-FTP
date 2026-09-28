#!/usr/bin/env bash
set -uo pipefail

mkdir -p dist/android/diagnostics

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
test_exit=$?
set -e

adb shell dumpsys window windows > dist/android/diagnostics/window.txt || true
adb shell dumpsys activity activities > dist/android/diagnostics/activities.txt || true
adb logcat -d -v threadtime > dist/android/diagnostics/logcat.txt || true
adb exec-out screencap -p > dist/android/diagnostics/post-test-screen.png || true

if [ "$test_exit" -ne 0 ]; then
  exit "$test_exit"
fi

DEBUG_APK="android/app/build/outputs/apk/debug/app-debug.apk"
test -s "$DEBUG_APK"

# connectedDebugAndroidTest may uninstall the target package during cleanup.
# Reinstall the exact debug APK that passed instrumentation before the
# post-test launch/screenshot smoke so the final UI check validates a real app.
INSTALL_OUTPUT=""
if ! INSTALL_OUTPUT="$(adb install -r "$DEBUG_APK" 2>&1)"; then
  printf '%s\n' "$INSTALL_OUTPUT" >&2
  echo "Ghost FTP debug APK reinstall failed after instrumentation cleanup." >&2
  exit 1
fi
printf '%s\n' "$INSTALL_OUTPUT"
case "$INSTALL_OUTPUT" in
  *Success*) ;;
  *)
    echo "Ghost FTP debug APK reinstall did not report Success." >&2
    exit 1
    ;;
esac

PACKAGE_PATH="$(adb shell pm path com.ghostftp.android 2>&1)"
PACKAGE_PATH="${PACKAGE_PATH//case "$PACKAGE_PATH" in
  package:*) ;;
  *)
    echo "Ghost FTP package is not installed after instrumentation cleanup recovery." >&2
    exit 1
    ;;
esac

adb shell am force-stop com.ghostftp.android || true
LAUNCH_OUTPUT="$(
  adb shell am start -W \
    -a android.intent.action.MAIN \
    -c android.intent.category.LAUNCHER \
    -p com.ghostftp.android 2>&1
)"
printf '%s\n' "$LAUNCH_OUTPUT"
case "$LAUNCH_OUTPUT" in
  *"Status: ok"*"com.ghostftp.android"*) ;;
  *)
    echo "Ghost FTP launcher intent did not report a successful app launch." >&2
    exit 1
    ;;
esac
sleep 2

WINDOW_DUMP="$(adb shell dumpsys window 2>&1)"
FOCUSED_WINDOW=""
while IFS= read -r window_line; do
  case "$window_line" in
    *mCurrentFocus*|*mFocusedApp*)
      FOCUSED_WINDOW="$window_line"
      break
      ;;
  esac
done <<< "$WINDOW_DUMP"
printf 'Focused window: %s\n' "$FOCUSED_WINDOW"
case "$FOCUSED_WINDOW" in
  *com.ghostftp.android*) ;;
  *)
    printf '%s\n' "$WINDOW_DUMP" > dist/android/diagnostics/post-launch-window.txt
    echo "Ghost FTP did not own the focused window after launch." >&2
    exit 1
    ;;
esac

adb shell dumpsys window windows > dist/android/diagnostics/post-launch-window.txt || true
adb exec-out screencap -p > dist/android/GhostFTP-Android-UI-Smoke.png
test -s dist/android/GhostFTP-Android-UI-Smoke.png
\r'/}"
printf 'Installed package path: %s\n' "$PACKAGE_PATH"
case "$PACKAGE_PATH" in
  package:*) ;;
  *)
    echo "Ghost FTP package is not installed after instrumentation cleanup recovery." >&2
    exit 1
    ;;
esac

adb shell am force-stop com.ghostftp.android || true
LAUNCH_OUTPUT="$(
  adb shell am start -W \
    -a android.intent.action.MAIN \
    -c android.intent.category.LAUNCHER \
    -p com.ghostftp.android 2>&1
)"
printf '%s\n' "$LAUNCH_OUTPUT"
case "$LAUNCH_OUTPUT" in
  *"Status: ok"*"com.ghostftp.android"*) ;;
  *)
    echo "Ghost FTP launcher intent did not report a successful app launch." >&2
    exit 1
    ;;
esac
sleep 2

WINDOW_DUMP="$(adb shell dumpsys window 2>&1)"
FOCUSED_WINDOW="$(printf '%s\n' "$WINDOW_DUMP" | sed -n -E '/mCurrentFocus|mFocusedApp/{p;q;}')"
printf 'Focused window: %s\n' "$FOCUSED_WINDOW"
case "$FOCUSED_WINDOW" in
  *com.ghostftp.android*) ;;
  *)
    printf '%s\n' "$WINDOW_DUMP" > dist/android/diagnostics/post-launch-window.txt
    echo "Ghost FTP did not own the focused window after launch." >&2
    exit 1
    ;;
esac

adb shell dumpsys window windows > dist/android/diagnostics/post-launch-window.txt || true
adb exec-out screencap -p > dist/android/GhostFTP-Android-UI-Smoke.png
test -s dist/android/GhostFTP-Android-UI-Smoke.png
