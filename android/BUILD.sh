#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

GRADLE_BIN="${GRADLE_BIN:-gradle}"
"$GRADLE_BIN" --no-daemon --console=plain :app:lintDebug :app:assembleDebug

mkdir -p dist
cp app/build/outputs/apk/debug/app-debug.apk dist/Ghost-FTP-Android-debug.apk
sha256sum dist/Ghost-FTP-Android-debug.apk > dist/Ghost-FTP-Android-debug.apk.sha256

test -s dist/Ghost-FTP-Android-debug.apk
printf 'ANDROID_APK=%s\n' "$ROOT/dist/Ghost-FTP-Android-debug.apk"
