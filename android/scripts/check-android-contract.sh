#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ANDROID_DIR="$ROOT/android"
APP_DIR="$ANDROID_DIR/app/src/main"
VERSION="$(node -p 'require("./version.json").version')"
BUILD="$(node -p 'require("./version.json").build')"

require_text() {
  local label="$1"
  local file="$2"
  local text="$3"
  if ! grep -Fq "$text" "$file"; then
    echo "Android contract failed: $label missing in $file"
    exit 1
  fi
}

require_exact_count() {
  local label="$1"
  local file="$2"
  local text="$3"
  local expected="$4"
  local actual
  actual="$(grep -Fc "$text" "$file" || true)"
  if [ "$actual" -ne "$expected" ]; then
    echo "Android contract failed: $label expected $expected occurrence(s), found $actual in $file"
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

audit_android_locale_parity() {
  local res_dir="$1"
  shift
  python3 - "$res_dir" "$@" <<'PY'
import pathlib
import re
import sys

res_dir = pathlib.Path(sys.argv[1])
locales = sys.argv[2:]
pattern = re.compile(r'<string\s+name="([^"]+)"')

def read_keys(path: pathlib.Path):
    text = path.read_text(encoding="utf-8")
    names = pattern.findall(text)
    seen = set()
    duplicates = []
    for name in names:
        if name in seen and name not in duplicates:
            duplicates.append(name)
        seen.add(name)
    if duplicates:
        print(
            f"Android contract failed: duplicate string resource(s) in {path}: "
            + ", ".join(sorted(duplicates)),
            file=sys.stderr,
        )
        sys.exit(1)
    return set(names)

base_file = res_dir / "values" / "strings.xml"
base_keys = read_keys(base_file)
if not base_keys:
    print(f"Android contract failed: no canonical strings found in {base_file}", file=sys.stderr)
    sys.exit(1)

for locale in locales:
    locale_file = res_dir / f"values-{locale}" / "strings.xml"
    if not locale_file.is_file():
        print(f"Android contract failed: missing localization resource {locale_file}", file=sys.stderr)
        sys.exit(1)
    locale_keys = read_keys(locale_file)
    missing = sorted(base_keys - locale_keys)
    if missing:
        print(
            f"Android contract failed: {locale} locale missing canonical key(s): "
            + ", ".join(missing),
            file=sys.stderr,
        )
        sys.exit(1)

print(f"Android locale parity OK for {len(locales)} locales and {len(base_keys)} canonical keys")
PY
}

audit_private_kotlin_symbols() {
  local root="$1"
  python3 - "$root" <<'PY'
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
dead = []
declaration = re.compile(
    r"(?m)^\s*private\s+(?:(?:lateinit|const|suspend|inline|tailrec|operator|infix)\s+)*(?:fun|val|var)\s+([A-Za-z_][A-Za-z0-9_]*)\b"
)

for path in sorted(root.rglob("*.kt")):
    source = path.read_text(encoding="utf-8")
    code = re.sub(r"/\*[\s\S]*?\*/", " ", source)
    code = re.sub(r"//[^\n]*", " ", code)
    code = re.sub(r'"(?:\\.|[^"\\])*"', '""', code)
    for match in declaration.finditer(code):
        name = match.group(1)
        if len(re.findall(rf"\b{re.escape(name)}\b", code)) < 2:
            dead.append(f"{path.relative_to(root)}: {name}")

if dead:
    print("Android contract failed: declaration-only private Kotlin symbols detected:", file=sys.stderr)
    for item in dead:
        print(f" - {item}", file=sys.stderr)
    sys.exit(1)

print("Android private Kotlin symbol audit OK")
PY
}

audit_private_kotlin_symbols "$APP_DIR/java"

MAIN_ACTIVITY="$ANDROID_DIR/app/src/main/java/com/ghostftp/android/MainActivity.kt"
CONNECTION_MODEL="$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ConnectionModel.kt"
RELEASE_INFO="$ANDROID_DIR/app/src/main/java/com/ghostftp/android/ReleaseInfo.kt"
DESKTOP_ADAPTER="$ROOT/ghostftp-desktop/src/lib/fileUiAdapter.ts"
DESKTOP_BROWSER="$ROOT/ghostftp-desktop/src/components/FileBrowser.tsx"

require_text "product name" "$RELEASE_INFO" 'PRODUCT_NAME = "Ghost FTP"'
require_text "brand" "$RELEASE_INFO" 'BRAND = "Brendigo"'
require_text "version" "$RELEASE_INFO" "VERSION = \"$VERSION\""
require_text "display version" "$RELEASE_INFO" "VERSION_DISPLAY = \"$VERSION\""
require_text "badge" "$RELEASE_INFO" "VERSION_BADGE = \"$VERSION\""
require_text "build" "$RELEASE_INFO" "BUILD = \"$BUILD\""
ANDROID_STRINGS="$ANDROID_DIR/app/src/main/res/values/strings.xml"
require_text "app label" "$ANDROID_STRINGS" '<string name="app_name">Ghost FTP</string>'
localization_keys=(
  workspace_files workspace_sites workspace_transfers workspace_settings workspace_about
  action_refresh action_upload action_download action_new_folder action_rename action_delete
  action_cancel action_confirm action_connect action_disconnect action_pick_file
  label_support label_documentation label_privacy_policy label_official_website
  field_host field_port field_username field_password field_protocol
  label_security label_connection
)
for key in "${localization_keys[@]}"; do
  require_text "default Android localization key $key" "$ANDROID_STRINGS" "<string name=\"$key\">"
done

android_locales=(hr cs sk hu ro bg el tr uk da sv no de fr es it pt nl pl sl sr bs mk)
audit_android_locale_parity "$ANDROID_DIR/app/src/main/res" "${android_locales[@]}"
for locale in "${android_locales[@]}"; do
  locale_file="$ANDROID_DIR/app/src/main/res/values-$locale/strings.xml"
  test -s "$locale_file" || {
    echo "Android contract failed: missing localization resource $locale_file"
    exit 1
  }
  for key in "${localization_keys[@]}"; do
    require_text "Android $locale localization key $key" "$locale_file" "<string name=\"$key\">"
  done
done

require_text "ftp protocol" "$CONNECTION_MODEL" 'FTP("FTP", 21)'
require_text "ftps protocol" "$CONNECTION_MODEL" 'EXPLICIT_FTPS("Explicit FTPS", 21)'
require_text "sftp protocol" "$CONNECTION_MODEL" 'SFTP("SFTP", 22)'
require_text "desktop shared rename action" "$DESKTOP_ADAPTER" 'rename: (sessionId, from, to) => ipc.renamePath(sessionId, from, to)'
require_text "desktop shared delete action" "$DESKTOP_ADAPTER" 'ipc.deletePath(sessionId, path, recursive)'
require_text "desktop shared new-folder action" "$DESKTOP_ADAPTER" 'mkdir: (sessionId, path) => ipc.createDirectory(sessionId, path)'
require_text "desktop shared upload action" "$DESKTOP_BROWSER" 'await enqueueUploads(serverSid, items, serverRemotePath)'
require_text "desktop shared download action" "$DESKTOP_BROWSER" 'await enqueueDownloads(serverSid, entries.map(toTransferItem), dest)'

require_text "desktop parity ghost mark" "$MAIN_ACTIVITY" 'GhostMarkView'
require_text "left workspace navigation rail" "$MAIN_ACTIVITY" 'buildNavigationRail()'
require_text "scrollable left workspace navigation" "$MAIN_ACTIVITY" 'private fun buildNavigationRail(): View = ScrollView(this).apply'
require_text "left rail accessibility remains child-focused" "$MAIN_ACTIVITY" 'importantForAccessibility = View.IMPORTANT_FOR_ACCESSIBILITY_NO'
require_text "left workspace rail fill viewport" "$MAIN_ACTIVITY" 'isFillViewport = true'
require_text "Files workspace localization" "$MAIN_ACTIVITY" 'FILES(R.string.workspace_files)'
require_text "Sites workspace localization" "$MAIN_ACTIVITY" 'SITES(R.string.workspace_sites)'
require_text "Transfers workspace localization" "$MAIN_ACTIVITY" 'TRANSFERS(R.string.workspace_transfers)'
require_text "Settings workspace localization" "$MAIN_ACTIVITY" 'SETTINGS(R.string.workspace_settings)'
require_text "Help workspace localization" "$MAIN_ACTIVITY" 'ABOUT(R.string.workspace_about)'
require_text "localized private session label" "$MAIN_ACTIVITY" 'getString(R.string.label_private_session)'
require_text "localized session label" "$MAIN_ACTIVITY" 'label(getString(R.string.label_session))'
require_text "localized navigation open label" "$MAIN_ACTIVITY" 'getString(R.string.nav_open)'
require_text "localized navigation close label" "$MAIN_ACTIVITY" 'getString(R.string.nav_close)'
require_text "localized SFTP fingerprint hint" "$MAIN_ACTIVITY" 'getString(R.string.hint_sftp_fingerprint)'
require_text "localized remote path field" "$MAIN_ACTIVITY" 'formLabel(getString(R.string.field_remote_path))'
require_text "localized Sites description" "$MAIN_ACTIVITY" 'sectionDescription(getString(R.string.desc_sites))'
require_text "localized Files description" "$MAIN_ACTIVITY" 'sectionDescription(getString(R.string.desc_files))'
require_text "localized Transfers description" "$MAIN_ACTIVITY" 'sectionDescription(getString(R.string.desc_transfers))'
require_text "localized Settings description" "$MAIN_ACTIVITY" 'sectionDescription(getString(R.string.desc_settings))'
require_text "localized empty upload state" "$MAIN_ACTIVITY" 'getString(R.string.state_no_local_file)'
require_text "localized idle transfer state" "$MAIN_ACTIVITY" 'getString(R.string.state_no_transfer)'
require_text "localized EULA label" "$MAIN_ACTIVITY" 'getString(R.string.label_eula)'
require_text "localized external link accessibility" "$MAIN_ACTIVITY" 'getString(R.string.link_open, title)'
require_text "localized protocol field" "$MAIN_ACTIVITY" 'formLabel(getString(R.string.field_protocol))'
require_text "localized host field" "$MAIN_ACTIVITY" 'formLabel(getString(R.string.field_host))'
require_text "localized username field" "$MAIN_ACTIVITY" 'formLabel(getString(R.string.field_username))'
require_text "localized password field" "$MAIN_ACTIVITY" 'formLabel(getString(R.string.field_password))'
require_text "localized support link" "$MAIN_ACTIVITY" 'officialLinkRow(getString(R.string.label_support)'
require_text "localized documentation link" "$MAIN_ACTIVITY" 'officialLinkRow(getString(R.string.label_documentation)'
require_text "localized privacy link" "$MAIN_ACTIVITY" 'officialLinkRow(getString(R.string.label_privacy_policy)'
require_text "localized website link" "$MAIN_ACTIVITY" 'officialLinkRow(getString(R.string.label_official_website)'
require_text "exclusive workspace visibility" "$MAIN_ACTIVITY" 'view.visibility = if (key == workspace) View.VISIBLE else View.GONE'
require_text "workspace navigation control" "$MAIN_ACTIVITY" 'workspaceNavItem(workspace)'
require_text "workspace state persistence" "$MAIN_ACTIVITY" 'STATE_WORKSPACE'
require_text "selected workspace scroll target" "$MAIN_ACTIVITY" 'workspaceContainer.top'
require_text "selected workspace brought into view" "$MAIN_ACTIVITY" 'contentScroll.scrollTo(0, workspaceTop)'
require_absent "legacy horizontal workspace scroller" "$MAIN_ACTIVITY" 'HorizontalScrollView'
require_text "mobile Settings surface" "$MAIN_ACTIVITY" 'buildSettingsCard()'
require_text "mobile Help surface" "$MAIN_ACTIVITY" 'buildAboutCard()'
require_text "accessible protocol selector" "$MAIN_ACTIVITY" 'contentDescription = getString(R.string.field_protocol)'
require_text "protocol default-port behavior" "$MAIN_ACTIVITY" 'currentPort == selectedProtocol.defaultPort.toString()'
require_text "protocol-specific SFTP security field" "$MAIN_ACTIVITY" 'if (protocol == ConnectionProtocol.SFTP) View.VISIBLE else View.GONE'
require_absent "obsolete Android queue-row alias" "$MAIN_ACTIVITY" 'MAX_QUEUE_ROWS'
require_text "session-aware connect availability" "$MAIN_ACTIVITY" 'connectButton.isEnabled = !busy && !hasActiveSession'
require_text "session-aware refresh availability" "$MAIN_ACTIVITY" 'refreshButton.isEnabled = !busy && hasActiveSession'
require_text "session-aware disconnect availability" "$MAIN_ACTIVITY" 'disconnectButton.isEnabled = hasActiveSession'
require_text "session-aware remote actions" "$MAIN_ACTIVITY" 'remoteActionButtons.forEach { it.isEnabled = !busy && hasActiveSession }'
require_text "busy-sensitive Android local actions" "$MAIN_ACTIVITY" 'busySensitiveLocalButtons.forEach { it.isEnabled = !busy }'
require_text "session-aware Android settings disconnect" "$MAIN_ACTIVITY" 'sessionDisconnectButtons.forEach { it.isEnabled = hasActiveSession }'
require_text "busy-sensitive Android action tracker" "$MAIN_ACTIVITY" 'private fun trackBusySensitiveLocalAction(button: Button): Button'
require_text "session disconnect Android action tracker" "$MAIN_ACTIVITY" 'private fun trackSessionDisconnect(button: Button): Button'
require_text "desktop parity refresh toolbar" "$MAIN_ACTIVITY" 'trackRemoteAction(toolbarButton(getString(R.string.action_refresh))'
require_text "desktop parity upload toolbar" "$MAIN_ACTIVITY" 'toolbarButton(getString(R.string.action_upload))'
require_text "desktop parity download toolbar" "$MAIN_ACTIVITY" 'toolbarButton(getString(R.string.action_download))'
require_text "desktop parity new folder toolbar" "$MAIN_ACTIVITY" 'toolbarButton(getString(R.string.action_new_folder))'
require_text "desktop parity rename toolbar" "$MAIN_ACTIVITY" 'toolbarButton(getString(R.string.action_rename))'
require_text "desktop parity delete toolbar" "$MAIN_ACTIVITY" 'toolbarButton(getString(R.string.action_delete), destructive = true)'
require_text "desktop parity ghost midnight background" "$MAIN_ACTIVITY" 'Color.rgb(13, 17, 23)'
require_text "desktop parity ghost midnight accent" "$MAIN_ACTIVITY" 'Color.rgb(47, 129, 247)'

require_text "connect action" "$MAIN_ACTIVITY" 'primaryButton(getString(R.string.action_connect))'
require_text "disconnect action" "$MAIN_ACTIVITY" 'secondaryButton(getString(R.string.action_disconnect))'
require_text "refresh action" "$MAIN_ACTIVITY" 'secondaryButton(getString(R.string.action_refresh))'
require_text "upload pick action" "$MAIN_ACTIVITY" 'secondaryButton(getString(R.string.action_pick_file))'
require_text "upload action" "$MAIN_ACTIVITY" 'secondaryButton(getString(R.string.action_upload))'
require_text "working Android Settings clear activity action" "$MAIN_ACTIVITY" 'contentDescription = getString(R.string.action_clear_activity)'
require_text "working Android Settings reset transfers action" "$MAIN_ACTIVITY" 'contentDescription = getString(R.string.action_reset_transfers)'
require_text "working Android Settings reset connection action" "$MAIN_ACTIVITY" 'contentDescription = getString(R.string.action_reset_connection)'
require_text "working Android Settings disconnect action" "$MAIN_ACTIVITY" 'contentDescription = "${getString(R.string.workspace_settings)} ${getString(R.string.action_disconnect)}"'
require_text "Android document picker" "$MAIN_ACTIVITY" 'Intent.ACTION_OPEN_DOCUMENT'
require_text "transfer state text" "$MAIN_ACTIVITY" 'transferStateText'
require_text "destructive action confirmation" "$MAIN_ACTIVITY" 'confirmDestructiveRemoteAction'
require_text "upload confirmation" "$MAIN_ACTIVITY" 'confirmUploadTarget'
require_text "bounded activity log" "$MAIN_ACTIVITY" 'MAX_ACTIVITY_ROWS'
require_text "remote path safety validation" "$MAIN_ACTIVITY" 'normalizeRemoteInput'
require_text "post-transfer refresh" "$MAIN_ACTIVITY" 'refreshAfter'
require_text "lifecycle close cleanup" "$MAIN_ACTIVITY" 'override fun onDestroy()'
require_text "disconnect generation invalidation" "$MAIN_ACTIVITY" 'operationGeneration += 1'
require_text "stale async result guard" "$MAIN_ACTIVITY" 'generation != operationGeneration'
require_text "refresh explicit profile" "$MAIN_ACTIVITY" 'openConnection(refreshedProfile)'
require_text "destroy operation invalidation" "$MAIN_ACTIVITY" 'operationGeneration += 1'
require_text "parallel operation guard" "$MAIN_ACTIVITY" 'if (operationInFlight) return'
require_text "password view-state disabled" "$MAIN_ACTIVITY" 'isSaveEnabled = false'
require_text "password destroy cleanup" "$MAIN_ACTIVITY" 'if (::passwordInput.isInitialized) passwordInput.text.clear()'
require_text "Activity non-secret state persistence" "$MAIN_ACTIVITY" 'override fun onSaveInstanceState(outState: Bundle)'
require_text "Activity active operation cancellation" "$MAIN_ACTIVITY" 'activeCancellation?.cancel()'
require_text "SAF persistable read grant" "$MAIN_ACTIVITY" 'Intent.FLAG_GRANT_READ_URI_PERMISSION'
require_text "SAF persistable picker flag" "$MAIN_ACTIVITY" 'Intent.FLAG_GRANT_PERSISTABLE_URI_PERMISSION'
require_text "restored session requires reauthentication" "$MAIN_ACTIVITY" 'Reconnect to authenticate before remote actions.'
require_text "remote root delete guard" "$CONNECTION_MODEL" 'Refusing to delete the remote root path.'
require_text "lifecycle safe ui wrapper" "$MAIN_ACTIVITY" 'private fun safeUi'
require_text "lifecycle destroyed guard" "$MAIN_ACTIVITY" 'closingOrDestroyed()'
require_text "lateinit ui readiness guard" "$MAIN_ACTIVITY" 'private fun uiReady()'
require_text "picker metadata fallback" "$MAIN_ACTIVITY" 'displayNameFor(uri: Uri): String'
require_text "download folder fallback" "$MAIN_ACTIVITY" 'File(filesDir, "downloads")'

require_text "download operation" "$CONNECTION_MODEL" 'fun downloadRemote'
require_text "cooperative Android cancellation" "$CONNECTION_MODEL" 'class OperationCancellation'
require_text "cancellable transfer copy loop" "$CONNECTION_MODEL" 'private fun copyCancelable'
require_text "FTP streaming download" "$CONNECTION_MODEL" 'retrieveFileStream(remoteFilePath)'
require_text "FTP staged streaming upload" "$CONNECTION_MODEL" 'client.storeFileStream(temporaryPath)'
require_absent "direct FTP target overwrite" "$CONNECTION_MODEL" 'storeFileStream(remoteFilePath)'
require_text "FTP pending command completion" "$CONNECTION_MODEL" 'client.completePendingCommand()'
require_text "FTP staged upload" "$CONNECTION_MODEL" 'client.storeFileStream(temporaryPath)'
require_text "FTP staged upload promotion" "$CONNECTION_MODEL" 'client.rename(temporaryPath, remoteFilePath)'
require_text "FTP existing-target backup" "$CONNECTION_MODEL" 'client.rename(remoteFilePath, backupPath)'
require_text "FTP target existence proof" "$CONNECTION_MODEL" 'ftpRemoteTargetExists(client, remoteFilePath)'
require_text "SFTP target existence proof" "$CONNECTION_MODEL" 'sftpRemoteTargetExists(channel, remoteFilePath)'
require_exact_count "shared staged replacement definition and integrations" "$CONNECTION_MODEL" 'commitStagedRemoteReplacement(' 3
require_text "SFTP no-such-file classifier" "$CONNECTION_MODEL" 'sftpTargetExistsFromLookupFailure(error)'
require_absent "ambiguous FTP backup rename fallback" "$CONNECTION_MODEL" 'runCatching { client.rename(remoteFilePath, backupPath) }'
require_text "staged upload recovery failure visibility" "$CONNECTION_MODEL" 'Original remote file preserved at $backupPath for manual recovery.'
require_text "SFTP streaming download" "$CONNECTION_MODEL" 'channel.get(remoteFilePath).use'
require_text "SFTP staged upload" "$CONNECTION_MODEL" 'channel.put(temporaryPath).use'
require_text "SFTP staged upload promotion" "$CONNECTION_MODEL" 'channel.rename(temporaryPath, remoteFilePath)'
require_text "SFTP existing-target backup" "$CONNECTION_MODEL" 'channel.rename(remoteFilePath, backupPath)'
require_text "local download staging" "$CONNECTION_MODEL" 'commitLocalReplacement(temporaryFile, outputFile)'
require_text "failed staged download cleanup" "$CONNECTION_MODEL" 'if (temporaryFile.exists()) temporaryFile.delete()'
require_absent "failed download target deletion" "$CONNECTION_MODEL" 'if (outputFile.exists()) outputFile.delete()'
require_text "download directory validation" "$CONNECTION_MODEL" 'parent.exists() || parent.mkdirs()'
require_text "picker URI readability guard" "$MAIN_ACTIVITY" 'private fun canReadUploadUri(uri: Uri): Boolean'
require_text "picker URI stream probe" "$MAIN_ACTIVITY" 'contentResolver.openInputStream(uri)?.use { true } ?: false'
require_text "picker URI immediate rejection" "$MAIN_ACTIVITY" 'if (!canReadUploadUri(uri))'
require_text "restored picker URI readability guard" "$MAIN_ACTIVITY" 'val restoredUploadReadable = restoredUploadUri?.let(::canReadUploadUri) == true'
require_text "upload operation" "$CONNECTION_MODEL" 'fun uploadRemote'
require_text "upload stream ownership" "$CONNECTION_MODEL" '): TransferResult = input.use { source ->'
require_text "delete operation" "$CONNECTION_MODEL" 'fun deleteRemoteFile'
require_text "FTP empty-folder delete parity" "$CONNECTION_MODEL" 'client.removeDirectory(remoteFilePath)'
require_text "SFTP empty-folder delete parity" "$CONNECTION_MODEL" 'channel.rmdir(remoteFilePath)'
require_text "delete lifecycle cancellation" "$MAIN_ACTIVITY" 'controller.deleteRemoteFile(profile, remotePath, cancellation)'
require_text "rename operation" "$CONNECTION_MODEL" 'fun renameRemote'
require_text "FTP rename parity" "$CONNECTION_MODEL" 'client.rename(sourcePath, destinationPath)'
require_text "SFTP rename parity" "$CONNECTION_MODEL" 'channel.rename(sourcePath, destinationPath)'
require_text "rename lifecycle cancellation" "$MAIN_ACTIVITY" 'controller.renameRemote(profile, sourcePath, destinationPath, cancellation)'
require_text "rename source root guard" "$CONNECTION_MODEL" 'Refusing to rename the remote root path.'
require_text "rename destination root guard" "$CONNECTION_MODEL" 'Refusing to rename to the remote root path.'
require_text "folder operation" "$CONNECTION_MODEL" 'fun createRemoteDirectory'
require_text "folder lifecycle cancellation" "$MAIN_ACTIVITY" 'controller.createRemoteDirectory(profile, remoteTarget, cancellation)'
require_text "FTP connect timeout" "$CONNECTION_MODEL" 'connectTimeout = CONNECT_TIMEOUT_MS'
require_text "FTP default timeout" "$CONNECTION_MODEL" 'defaultTimeout = CONNECT_TIMEOUT_MS'
require_text "FTP data timeout" "$CONNECTION_MODEL" 'dataTimeout = Duration.ofMillis(CONNECT_TIMEOUT_MS.toLong())'
require_text "FTP login requirement" "$CONNECTION_MODEL" 'require(client.login(profile.username, profile.password))'
require_text "FTP passive mode" "$CONNECTION_MODEL" 'enterLocalPassiveMode()'
require_text "FTP binary transfer mode" "$CONNECTION_MODEL" 'setFileType(FTP.BINARY_FILE_TYPE)'
require_text "FTP logout cleanup" "$CONNECTION_MODEL" 'client.logout()'
require_text "FTP disconnect cleanup" "$CONNECTION_MODEL" 'client.disconnect()'
require_text "Explicit FTPS client mode" "$CONNECTION_MODEL" 'FTPSClient(false)'
require_text "FTPS hostname verification" "$CONNECTION_MODEL" 'setEndpointCheckingEnabled(true)'
require_text "Explicit FTPS PBSZ" "$CONNECTION_MODEL" 'execPBSZ(0)'
require_text "Explicit FTPS protected data channel" "$CONNECTION_MODEL" 'execPROT("P")'
require_text "SFTP host key verification" "$CONNECTION_MODEL" 'StrictHostKeyChecking", "yes"'
require_text "SFTP password-only authentication hardening" "$CONNECTION_MODEL" 'PreferredAuthentications", "password"'
require_absent "SFTP keyboard-interactive authentication" "$CONNECTION_MODEL" 'PreferredAuthentications", "keyboard-interactive'
require_text "SFTP fingerprint input" "$MAIN_ACTIVITY" 'formLabel(getString(R.string.field_sftp_fingerprint))'
require_text "SFTP fingerprint repository" "$CONNECTION_MODEL" 'FingerprintHostKeyRepository(profile.hostKeyFingerprint)'
require_text "SFTP byte-array password API" "$CONNECTION_MODEL" 'session.setPassword(passwordBytes)'
require_text "SFTP temporary password buffer cleanup" "$CONNECTION_MODEL" 'passwordBytes.fill(0)'
require_absent "deprecated SFTP string password API" "$CONNECTION_MODEL" 'session.setPassword(profile.password)'
require_text "collapsible Android navigation" "$MAIN_ACTIVITY" 'private fun setNavigationOpen(open: Boolean, announce: Boolean = true)'
require_text "phone navigation auto-close" "$MAIN_ACTIVITY" 'resources.configuration.screenWidthDp < 600'
require_text "compact navigation overlay container" "$MAIN_ACTIVITY" 'FrameLayout(this).apply'
require_text "compact navigation scrim" "$MAIN_ACTIVITY" 'contentDescription = getString(R.string.nav_close_overlay)'
require_text "compact navigation scrim close action" "$MAIN_ACTIVITY" 'setOnClickListener { setNavigationOpen(false) }'
require_text "inline action confirmation" "$MAIN_ACTIVITY" 'private fun requestInlineConfirmation('
require_absent "popup AlertDialog confirmation" "$MAIN_ACTIVITY" 'AlertDialog.Builder'
require_text "global cleartext disabled" "$ANDROID_DIR/app/src/main/AndroidManifest.xml" 'android:usesCleartextTraffic="false"'
require_absent "global cleartext enabled" "$ANDROID_DIR/app/src/main/AndroidManifest.xml" 'android:usesCleartextTraffic="true"'
require_text "plain FTP risk disclosure" "$MAIN_ACTIVITY" 'FTP sends credentials and file data without transport encryption.'
require_text "FTPS hostname validation disclosure" "$MAIN_ACTIVITY" 'validates the server hostname.'
require_text "SFTP host-key verification disclosure" "$MAIN_ACTIVITY" 'requires strict SSH host-key verification.'
require_text "official support link" "$MAIN_ACTIVITY" 'https://ghostftp.com/support/'
require_text "official privacy link" "$MAIN_ACTIVITY" 'https://ghostftp.com/privacy/'
require_text "official documentation link" "$MAIN_ACTIVITY" 'https://ghostftp.com/docs/'
require_text "canonical EULA link" "$MAIN_ACTIVITY" 'https://github.com/bren-wp/Ghost-FTP/blob/main/EULA.txt'
require_text "Android 15 edge-to-edge API gate" "$MAIN_ACTIVITY" 'if (Build.VERSION.SDK_INT >= 35)'
require_text "Android 15 system bar insets" "$MAIN_ACTIVITY" 'WindowInsets.Type.systemBars()'
require_text "Android 15 root inset padding" "$MAIN_ACTIVITY" 'view.setPadding(bars.left, bars.top, bars.right, bars.bottom)'
require_absent "deprecated status bar color setter" "$MAIN_ACTIVITY" 'window.statusBarColor'
require_absent "deprecated navigation bar color setter" "$MAIN_ACTIVITY" 'window.navigationBarColor'
require_text "dark status bar icons contract" "$ANDROID_DIR/app/src/main/res/values-v35/styles.xml" '<item name="android:windowLightStatusBar">false</item>'
require_text "dark navigation bar icons contract" "$ANDROID_DIR/app/src/main/res/values-v35/styles.xml" '<item name="android:windowLightNavigationBar">false</item>'
require_absent "Android 15 deprecated status bar color attribute" "$ANDROID_DIR/app/src/main/res/values-v35/styles.xml" 'android:statusBarColor'
require_absent "Android 15 deprecated navigation bar color attribute" "$ANDROID_DIR/app/src/main/res/values-v35/styles.xml" 'android:navigationBarColor'
require_text "SFTP session timeout" "$CONNECTION_MODEL" 'session.timeout = CONNECT_TIMEOUT_MS'
require_text "SFTP connect timeout" "$CONNECTION_MODEL" 'session.connect(CONNECT_TIMEOUT_MS)'
require_text "SFTP channel timeout" "$CONNECTION_MODEL" 'channel.connect(CONNECT_TIMEOUT_MS)'
require_text "SFTP channel cleanup" "$CONNECTION_MODEL" 'channel?.disconnect()'
require_text "SFTP session cleanup" "$CONNECTION_MODEL" 'session.disconnect()'
require_text "host normalization" "$CONNECTION_MODEL" 'IDN.toASCII(host)'
require_text "IPv6 bracket parsing" "$CONNECTION_MODEL" "value.startsWith('[')"
require_text "embedded credential rejection" "$CONNECTION_MODEL" "'@' !in value"
require_text "host scheme validation" "$CONNECTION_MODEL" 'scheme in setOf("ftp", "ftps", "sftp")'
require_text "separate host and port inputs" "$CONNECTION_MODEL" 'Enter the port in the Port field.'
require_text "installable preview build type" "$ANDROID_DIR/app/build.gradle.kts" 'create("preview")'
require_text "installable preview package isolation" "$ANDROID_DIR/app/build.gradle.kts" 'applicationIdSuffix = ".preview"'
require_text "installable preview signing" "$ANDROID_DIR/app/build.gradle.kts" 'signingConfig = signingConfigs.getByName("debug")'
require_text "preview remains non-debuggable" "$ANDROID_DIR/app/build.gradle.kts" 'isDebuggable = false'
require_text "current Apache Commons Net dependency" "$ANDROID_DIR/app/build.gradle.kts" 'commons-net:commons-net:3.13.0'
require_text "maintained JSch dependency" "$ANDROID_DIR/app/build.gradle.kts" 'com.github.mwiede:jsch:2.28.7'
require_text "Android instrumentation runner" "$ANDROID_DIR/app/build.gradle.kts" 'androidx.test.runner.AndroidJUnitRunner'
require_text "AndroidX test core dependency" "$ANDROID_DIR/app/build.gradle.kts" 'androidx.test:core:1.7.0'
require_text "AndroidX test runner dependency" "$ANDROID_DIR/app/build.gradle.kts" 'androidx.test:runner:1.7.0'
require_text "AndroidX JUnit dependency" "$ANDROID_DIR/app/build.gradle.kts" 'androidx.test.ext:junit:1.3.0'
require_text "Android Espresso UI test dependency" "$ANDROID_DIR/app/build.gradle.kts" 'androidx.test.espresso:espresso-core:3.7.0'
require_text "Android JVM JUnit dependency" "$ANDROID_DIR/app/build.gradle.kts" 'testImplementation("junit:junit:4.13.2")'
UNIT_TEST="$ANDROID_DIR/app/src/test/java/com/ghostftp/android/ConnectionModelTest.kt"
test -s "$UNIT_TEST"
require_text "Android cancellation unit coverage" "$UNIT_TEST" 'cancellationBecomesStickyAndThrows'
require_text "Android path traversal unit coverage" "$UNIT_TEST" 'remoteTargetsRejectDotSegmentsAndRootDestruction'
require_text "Android host validation unit coverage" "$UNIT_TEST" 'hostNormalizationRejectsEmbeddedCredentialsPortsAndSchemes'
require_text "Android staged absent-target coverage" "$UNIT_TEST" 'stagedReplacementPromotesWhenTargetIsAbsent'
require_text "Android staged existing-target coverage" "$UNIT_TEST" 'stagedReplacementBacksUpExistingTargetBeforePromotion'
require_text "Android staged existence-failure coverage" "$UNIT_TEST" 'stagedReplacementFailsClosedWhenExistenceCheckFails'
require_text "Android staged backup-failure coverage" "$UNIT_TEST" 'stagedReplacementCleansTemporaryWhenBackupRenameFails'
require_text "Android staged promotion rollback coverage" "$UNIT_TEST" 'stagedReplacementRestoresOriginalWhenPromotionFails'
require_text "Android staged rollback-failure coverage" "$UNIT_TEST" 'stagedReplacementPreservesBackupWhenRollbackFails'
require_text "Android staged late-target coverage" "$UNIT_TEST" 'stagedReplacementRejectsTargetThatAppearsDuringTransfer'
require_text "Android SFTP no-such-file coverage" "$UNIT_TEST" 'sftpNoSuchFileMeansTargetAbsent'
require_text "Android SFTP error fail-closed coverage" "$UNIT_TEST" 'sftpPermissionAndProtocolErrorsFailClosed'
test -s "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt"
SMOKE_SCRIPT="$ANDROID_DIR/scripts/run-instrumentation-smoke.sh"
bash -n "$SMOKE_SCRIPT"
require_exact_count "single instrumentation execution" "$SMOKE_SCRIPT" 'gradle -p android connectedDebugAndroidTest --stacktrace' 1
require_exact_count "single debug APK reinstall" "$SMOKE_SCRIPT" 'INSTALL_OUTPUT="$(adb install -r "$DEBUG_APK" 2>&1)"' 1
require_exact_count "single debug launcher execution" "$SMOKE_SCRIPT" 'LAUNCH_OUTPUT="$(adb shell am start -W -n "$LAUNCH_COMPONENT" 2>&1)"' 1
require_exact_count "single release-candidate clean install" "$SMOKE_SCRIPT" 'PREVIEW_INSTALL_OUTPUT="$(adb install "$PREVIEW_APK" 2>&1)"' 1
require_exact_count "single release-candidate reinstall" "$SMOKE_SCRIPT" 'PREVIEW_REINSTALL_OUTPUT="$(adb install -r "$PREVIEW_APK" 2>&1)"' 1
require_exact_count "single release-candidate launcher execution" "$SMOKE_SCRIPT" 'PREVIEW_LAUNCH_OUTPUT="$(adb shell am start -W -n "$PREVIEW_COMPONENT" 2>&1)"' 1
require_exact_count "single debug UI smoke screenshot" "$SMOKE_SCRIPT" 'adb exec-out screencap -p > dist/android/GhostFTP-Android-UI-Smoke.png' 1
require_exact_count "single release-candidate UI smoke screenshot" "$SMOKE_SCRIPT" 'adb exec-out screencap -p > dist/android/GhostFTP-Android-Release-Candidate-Smoke.png' 1
require_text "release-candidate APK path" "$SMOKE_SCRIPT" 'PREVIEW_APK="android/app/build/outputs/apk/preview/app-preview.apk"'
require_text "release-candidate package id" "$SMOKE_SCRIPT" 'PREVIEW_PACKAGE_ID="com.ghostftp.android.preview"'
require_text "release-candidate launcher class" "$SMOKE_SCRIPT" 'PREVIEW_ACTIVITY_CLASS="com.ghostftp.android.MainActivity"'
require_text "release-candidate non-debuggable gate" "$SMOKE_SCRIPT" 'Release-candidate APK must remain non-debuggable.'
require_text "post-launch success marker" "$SMOKE_SCRIPT" 'Ghost FTP Android instrumentation, release-candidate install and launch smoke OK'
require_text "post-launch resumed activity gate" "$SMOKE_SCRIPT" 'Ghost FTP MainActivity did not reach RESUMED state after launch.'
require_text "release-candidate resumed activity gate" "$SMOKE_SCRIPT" 'Ghost FTP release-candidate MainActivity did not reach RESUMED state.'
require_absent "brittle debug am-start output parsing" "$SMOKE_SCRIPT" '"Status: ok"*"$PACKAGE_ID"'
require_absent "brittle preview am-start output parsing" "$SMOKE_SCRIPT" '"Status: ok"*"$PREVIEW_PACKAGE_ID"'
require_text "debug process liveness gate" "$SMOKE_SCRIPT" 'APP_PID="$(adb shell pidof "$PACKAGE_ID" 2>/dev/null || true)"'
require_text "preview process liveness gate" "$SMOKE_SCRIPT" 'PREVIEW_PID="$(adb shell pidof "$PREVIEW_PACKAGE_ID" 2>/dev/null || true)"'
require_text "ATD Bluetooth overlay handling" "$SMOKE_SCRIPT" 'Application Error: com.android.bluetooth'
require_text "Activity recreation credential regression" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'Password must never survive Activity recreation.'
require_text "protocol-aware form smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'protocolSelectionUpdatesDefaultsWithoutClobberingCustomPort'
require_text "exclusive workspace smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'workspacesAreExclusiveInsteadOfOneLongScreen'
require_text "collapsible navigation smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'navigationRailCanOpenAndCloseWithoutPopupNavigation'
require_text "compact overlay navigation smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'compactNavigationUsesOverlayScrimInsteadOfShrinkingContent'
require_text "official product link smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'helpWorkspaceExposesCanonicalProductLinks'
require_text "working action smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'transferAndSettingsActionsAreWiredClickByClick'
require_text "idle action-state smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'guardedFileActionsStayDisabledWithoutActiveSession'
require_text "resource-aware Android smoke strings" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'private fun appString(resId: Int, vararg formatArgs: Any): String'
require_text "idle settings disconnect smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'R.string.action_disconnect'
require_text "unreadable picker URI smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'unreadablePickerUriIsRejectedFailClosed'
require_text "rename smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'R.string.action_rename'
require_text "Settings reset smoke coverage" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'R.string.action_reset_connection'
for stale_smoke_copy in \
  'Connect to server' \
  'Disconnect from server' \
  'Refresh current session' \
  'Refresh action' \
  'Pick upload file' \
  'Upload selected file' \
  'Connection protocol' \
  'Ghost FTP by Brendigo'; do
  require_absent "stale Android smoke copy" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" "$stale_smoke_copy"
done
require_absent "window-focus instrumentation dependency" "$ANDROID_DIR/app/src/androidTest/java/com/ghostftp/android/MainActivitySmokeTest.kt" 'hasWindowFocus()'
require_absent "window-focus smoke gate" "$SMOKE_SCRIPT" 'Ghost FTP did not own the focused window after launch.'

ANDROID_WORKFLOW="$ROOT/.github/workflows/ghostftp-android.yml"
ANDROID_SMOKE_SCRIPT="$ANDROID_DIR/scripts/run-instrumentation-smoke.sh"
require_text "Android click-through workflow" "$ANDROID_WORKFLOW" 'bash android/scripts/run-instrumentation-smoke.sh'
require_text "Android JVM unit-test workflow" "$ANDROID_WORKFLOW" 'testDebugUnitTest'
require_text "Android connected instrumentation" "$ANDROID_SMOKE_SCRIPT" 'connectedDebugAndroidTest'
require_text "Android instrumentation diagnostics" "$ANDROID_SMOKE_SCRIPT" 'dist/android/diagnostics'
CANONICAL_RELEASE_WORKFLOW="$ROOT/.github/workflows/ghostftp-release.yml"
PUBLISH_SCRIPT="$ROOT/.github/scripts/publish-release.sh"

require_text "Android workflow builds preview" "$ANDROID_WORKFLOW" 'assemblePreview'
require_text "Android workflow retains unsigned release-check" "$ANDROID_WORKFLOW" 'app-release-unsigned.apk'
require_text "Android workflow verifies installable signature" "$ANDROID_WORKFLOW" 'apksigner'
require_text "Android workflow verifies preview version code" "$ANDROID_WORKFLOW" 'test "$PREVIEW_VERSION_CODE" = "$ANDROID_VERSION_CODE"'
require_text "Android workflow verifies preview version name" "$ANDROID_WORKFLOW" 'test "$PREVIEW_VERSION_NAME" = "${VERSION}-preview"'
require_text "Android workflow verifies preview minSdk" "$ANDROID_WORKFLOW" 'test "$PREVIEW_MIN_SDK" = "26"'
require_text "Android workflow verifies preview targetSdk" "$ANDROID_WORKFLOW" 'test "$PREVIEW_TARGET_SDK" = "35"'
require_text "Android workflow verifies preview launcher" "$ANDROID_WORKFLOW" 'test "$PREVIEW_LAUNCH_ACTIVITY" = "com.ghostftp.android.MainActivity"'
require_text "Android workflow rejects debuggable preview" "$ANDROID_WORKFLOW" 'Installable preview APK unexpectedly remains debuggable.'
require_text "unsigned APK is clearly non-installable" "$ANDROID_WORKFLOW" 'Release-Unsigned.apk.unsigned'
require_text "unsigned release package verification" "$ANDROID_WORKFLOW" 'test "$RELEASE_PACKAGE_ID" = "com.ghostftp.android"'
require_text "unsigned release version verification" "$ANDROID_WORKFLOW" 'test "$RELEASE_VERSION_NAME" = "$VERSION"'
require_text "canonical release publishes unsigned Android artifact" "$CANONICAL_RELEASE_WORKFLOW" 'GhostFTP-Android-v$VERSION.apk.unsigned'
require_text "canonical release retains installable Android preview" "$CANONICAL_RELEASE_WORKFLOW" 'GhostFTP-Android-v$VERSION-Installable-Preview.apk'
NAMING_DOC="$ROOT/docs/architecture/NAMING.md"
STRUCTURE_DOC="$ROOT/docs/architecture/STRUCTURE.md"
require_text "canonical naming documents unsigned Android artifact" "$NAMING_DOC" 'GhostFTP-Android-v<version>.apk.unsigned'
require_text "canonical naming documents installable Android preview" "$NAMING_DOC" 'GhostFTP-Android-v<version>-Installable-Preview.apk'
require_text "architecture structure documents unsigned Android artifact" "$STRUCTURE_DOC" 'GhostFTP-Android-v<version>.apk.unsigned'
require_text "architecture structure documents installable Android preview" "$STRUCTURE_DOC" 'GhostFTP-Android-v<version>-Installable-Preview.apk'
require_absent "private Android release-signing secrets" "$ANDROID_WORKFLOW" 'ANDROID_RELEASE_'
require_absent "Android signing properties file" "$ANDROID_DIR/app/build.gradle.kts" 'signing.properties'
require_text "generic verified Android artifact bundle" "$ANDROID_WORKFLOW" 'name: GhostFTP-Android-APK'
require_text "canonical release consumes verified Android artifact" "$CANONICAL_RELEASE_WORKFLOW" 'GhostFTP-Android-APK'
require_text "canonical release delegates digest verification" "$CANONICAL_RELEASE_WORKFLOW" 'publish-release.sh'
require_text "release upload digest verification" "$PUBLISH_SCRIPT" 'REMOTE_DIGEST'

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
  'RC22'
  'Win32'
  'Win 32'
  'Developer:'
  'Brendigo LTD'
  'Brendigo Ltd'
)

for pattern in "${blocked_patterns[@]}"; do
  require_absent "product copy" "$APP_DIR" "$pattern"
done

echo "Ghost FTP Android $VERSION production contract OK"
