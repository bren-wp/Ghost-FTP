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
adb install -r "$DEBUG_APK" >/dev/null
adb shell pm path com.ghostftp.android | grep -Fq 'package:'

adb shell am force-stop com.ghostftp.android || true
LAUNCH_OUTPUT="$(
  adb shell am start -W \
    -a android.intent.action.MAIN \
    -c android.intent.category.LAUNCHER \
    -p com.ghostftp.android
)"
printf '%s\n' "$LAUNCH_OUTPUT"
echo "$LAUNCH_OUTPUT" | grep -Fq 'Status: ok'
echo "$LAUNCH_OUTPUT" | grep -Fq 'com.ghostftp.android'
sleep 2

FOCUSED_WINDOW="$(adb shell dumpsys window | grep -m1 -E 'mCurrentFocus|mFocusedApp' || true)"
echo "Focused window: $FOCUSED_WINDOW"
echo "$FOCUSED_WINDOW" | grep -Fq 'com.ghostftp.android'

adb shell dumpsys window windows > dist/android/diagnostics/post-launch-window.txt || true
adb exec-out screencap -p > dist/android/GhostFTP-Android-UI-Smoke.png
test -s dist/android/GhostFTP-Android-UI-Smoke.png
