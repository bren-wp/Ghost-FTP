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

adb shell am force-stop com.ghostftp.android || true
adb shell am start -W -n com.ghostftp.android/.MainActivity
sleep 2
adb exec-out screencap -p > dist/android/GhostFTP-Android-UI-Smoke.png
test -s dist/android/GhostFTP-Android-UI-Smoke.png
