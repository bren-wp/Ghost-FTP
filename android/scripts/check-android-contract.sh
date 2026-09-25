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

require_absent() {
  local label="$1"
  local path="$2"
  local text="$3"
  if grep -RInF "$text" "$path"; then
    echo "Android contract failed: blocked $label found: $text"
    exit 1
  fi
}

require_text "product name" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'PRODUCT_NAME = "Ghost FTP"'
require_text "brand" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'BRAND = "Brendigo"'
require_text "version" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'VERSION = "2.1.1-rc.22"'
require_text "display version" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'VERSION_DISPLAY = "2.1.1 RC22"'
require_text "badge" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'VERSION_BADGE = "RC22"'
require_text "build" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt" 'BUILD = "2026.09.24.22"'
require_text "app label" "$ANDROID_DIR/app/src/main/res/values/strings.xml" '<string name="app_name">Ghost FTP</string>'
require_text "ftp protocol" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'FTP("FTP", 21)'
require_text "ftps protocol" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'EXPLICIT_FTPS("Explicit FTPS", 21)'
require_text "sftp protocol" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'SFTP("SFTP", 22)'
require_text "connect action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'primaryButton("Connect")'
require_text "disconnect action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'secondaryButton("Disconnect")'
require_text "refresh action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'secondaryButton("Refresh")'
require_text "download action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'primaryButton("Download")'
require_text "upload action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'secondaryButton("Upload")'
require_text "delete action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'secondaryButton("Delete file")'
require_text "folder action" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'secondaryButton("Create folder")'
require_text "Android document picker" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'Intent.ACTION_OPEN_DOCUMENT'
require_text "transfer state text" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'transferStateText'
require_text "destructive action confirmation" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'confirmDestructiveRemoteAction'
require_text "upload confirmation" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'confirmUploadTarget'
require_text "bounded transfer queue" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'MAX_QUEUE_ROWS'
require_text "remote path safety validation" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'validateRemoteTarget'
require_text "post-transfer refresh" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'refreshAfter'
require_text "download operation" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'fun downloadRemote'
require_text "upload operation" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'fun uploadRemote'
require_text "delete operation" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'fun deleteRemoteFile'
require_text "folder operation" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'fun createRemoteDirectory'
require_text "SFTP host key verification" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt" 'StrictHostKeyChecking", "yes"'
require_text "SFTP fingerprint input" "$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt" 'SFTP host key fingerprint'
require_text "release signing configuration" "$ANDROID_DIR/app/build.gradle.kts" 'signingConfigs'

blocked_patterns=(
  'lorem'
  'placeholder'
  'demo'
  'example.com'
  'server.example'
  'ftp.company.com'
  'debug build'
  'RC20'
  'RC21'
  'Win32'
  'Win 32'
  'Developer:'
  'Brendigo LTD'
  'Brendigo Ltd'
)

for pattern in "${blocked_patterns[@]}"; do
  require_absent "product copy" "$APP_DIR" "$pattern"
done

echo "Ghost FTP Android RC22 production contract OK"
