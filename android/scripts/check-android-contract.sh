#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ANDROID_DIR="$ROOT/android"
APP_DIR="$ANDROID_DIR/app/src/main"

require_text() {
  local label="$1"
  local file="$2"
  local text="$3"
  if ! grep -Fq "$text" "$file"; then
    echo "Android contract failed: $label missing in $file"
    exit 1
  fi
}

require_text "product name" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'PRODUCT_NAME = "Ghost FTP"'
require_text "brand" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'BRAND = "Brendigo"'
require_text "version" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'VERSION = "2.1.1-rc.21"'
require_text "display version" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'VERSION_DISPLAY = "2.1.1 RC21"'
require_text "badge" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'VERSION_BADGE = "RC21"'
require_text "build" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'BUILD = "2026.09.24.21"'
require_text "app label" "$ANDROID_DIR/app/src/main/res/values/strings.xml" '<string name="app_name">Ghost FTP</string>'
require_text "ftp protocol" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'FTP("FTP", 21)'
require_text "ftps protocol" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'EXPLICIT_FTPS("Explicit FTPS", 21)'
require_text "sftp protocol" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'SFTP("SFTP", 22)'
require_text "connect action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'primaryButton("Connect")'
require_text "disconnect action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'secondaryButton("Disconnect")'
require_text "refresh action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'secondaryButton("Refresh")'

blocked_patterns=(
  'lorem'
  'placeholder'
  'demo'
  'RC20'
  'Win32'
  'Win 32'
  'Developer:'
  'Brendigo LTD'
  'Brendigo Ltd'
)

for pattern in "${blocked_patterns[@]}"; do
  if grep -RInF "$pattern" "$APP_DIR"; then
    echo "Android contract failed: blocked product copy found: $pattern"
    exit 1
  fi
done

echo "Ghost FTP Android contract OK"
