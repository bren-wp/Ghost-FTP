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

MAIN_ACTIVITY="$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt"
CONNECTION_MODEL="$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt"
RELEASE_INFO="$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt"

require_text "product name" "$RELEASE_INFO" 'PRODUCT_NAME = "Ghost FTP"'
require_text "brand" "$RELEASE_INFO" 'BRAND = "Brendigo"'
require_text "version" "$RELEASE_INFO" 'VERSION = "2.1.1-rc.22"'
require_text "display version" "$RELEASE_INFO" 'VERSION_DISPLAY = "2.1.1 RC22"'
require_text "badge" "$RELEASE_INFO" 'VERSION_BADGE = "RC22"'
require_text "build" "$RELEASE_INFO" 'BUILD = "2026.09.24.22"'
require_text "app label" "$ANDROID_DIR/app/src/main/res/values/strings.xml" '<string name="app_name">Ghost FTP</string>'

require_text "ftp protocol" "$CONNECTION_MODEL" 'FTP("FTP", 21)'
require_text "ftps protocol" "$CONNECTION_MODEL" 'EXPLICIT_FTPS("Explicit FTPS", 21)'
require_text "sftp protocol" "$CONNECTION_MODEL" 'SFTP("SFTP", 22)'

require_text "desktop parity ghost mark" "$MAIN_ACTIVITY" 'GhostMarkView'
require_text "desktop parity workspace" "$MAIN_ACTIVITY" 'workspaceChip("Files")'
require_text "desktop parity refresh toolbar" "$MAIN_ACTIVITY" 'toolbarButton("Refresh")'
require_text "desktop parity upload toolbar" "$MAIN_ACTIVITY" 'toolbarButton("Upload")'
require_text "desktop parity download toolbar" "$MAIN_ACTIVITY" 'toolbarButton("Download")'
require_text "desktop parity new folder toolbar" "$MAIN_ACTIVITY" 'toolbarButton("New Folder")'
require_text "desktop parity delete toolbar" "$MAIN_ACTIVITY" 'toolbarButton("Delete", destructive = true)'
require_text "desktop parity ghost midnight background" "$MAIN_ACTIVITY" 'Color.rgb(13, 17, 23)'
require_text "desktop parity ghost midnight accent" "$MAIN_ACTIVITY" 'Color.rgb(47, 129, 247)'

require_text "connect action" "$MAIN_ACTIVITY" 'primaryButton("Connect")'
require_text "disconnect action" "$MAIN_ACTIVITY" 'secondaryButton("Disconnect")'
require_text "refresh action" "$MAIN_ACTIVITY" 'secondaryButton("Refresh")'
require_text "upload pick action" "$MAIN_ACTIVITY" 'secondaryButton("Pick file")'
require_text "upload action" "$MAIN_ACTIVITY" 'secondaryButton("Upload")'
require_text "Android document picker" "$MAIN_ACTIVITY" 'Intent.ACTION_OPEN_DOCUMENT'
require_text "transfer state text" "$MAIN_ACTIVITY" 'transferStateText'
require_text "destructive action confirmation" "$MAIN_ACTIVITY" 'confirmDestructiveRemoteAction'
require_text "upload confirmation" "$MAIN_ACTIVITY" 'confirmUploadTarget'
require_text "bounded activity log" "$MAIN_ACTIVITY" 'MAX_ACTIVITY_ROWS'
require_text "remote path safety validation" "$MAIN_ACTIVITY" 'normalizeRemoteInput'
require_text "post-transfer refresh" "$MAIN_ACTIVITY" 'refreshAfter'

require_text "download operation" "$CONNECTION_MODEL" 'fun downloadRemote'
require_text "upload operation" "$CONNECTION_MODEL" 'fun uploadRemote'
require_text "delete operation" "$CONNECTION_MODEL" 'fun deleteRemoteFile'
require_text "folder operation" "$CONNECTION_MODEL" 'fun createRemoteDirectory'
require_text "SFTP host key verification" "$CONNECTION_MODEL" 'StrictHostKeyChecking", "yes"'
require_text "SFTP fingerprint input" "$MAIN_ACTIVITY" 'SFTP host key fingerprint'
require_text "release signing configuration" "$ANDROID_DIR/app/build.gradle.kts" 'signingConfigs'

blocked_patterns=(
  'lorem'
  'placeholder'
  'demo'
  'example.com'
  'server.example'
  'ftp.company.com'
  'debug build'
  'dev text'
  'developer text'
  'Native Android workspace'
  'Remote workspace'
  'Transfer actions'
  'Create folder'
  'Delete file'
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
