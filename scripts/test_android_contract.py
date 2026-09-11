#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
ANDROID_JAVA = "android/app/src/main/java/app/ghostftp/client"
AUTHOR_IDENTITY = "bren" + "digo"


class AndroidContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_android_project_generates_named_apk_under_android(self) -> None:
        build = self.read("android/app/build.gradle")
        workflow = self.read(".github/workflows/android-apk.yml")
        for marker in (
            "rootProject.file('../VERSION').text.trim()",
            "versionName \"${ghostFtpVersion}-dev\"",
            "tasks.register('packageGhostFtpApk', Copy)",
            "'Ghost-FTP-Android.apk'",
            "dist/Ghost-FTP-Android.apk",
            "dependsOn 'assembleDebug'",
        ):
            self.assertIn(marker, build)
        self.assertIn("android/dist/Ghost-FTP-Android.apk", workflow)
        self.assertIn("unzip -t android/dist/Ghost-FTP-Android.apk", workflow)
        self.assertIn("name: ghostftp-android-apk", workflow)

    def test_android_identity_is_ghostftp_only(self) -> None:
        build = self.read("android/app/build.gradle")
        self.assertIn("namespace 'app.ghostftp.client'", build)
        self.assertIn("applicationId 'app.ghostftp.client'", build)
        for rel in (
            f"{ANDROID_JAVA}/MainActivity.java",
            f"{ANDROID_JAVA}/FtpSession.java",
            f"{ANDROID_JAVA}/RemoteEntry.java",
            f"{ANDROID_JAVA}/SiteProfile.java",
            f"{ANDROID_JAVA}/SiteProfileStore.java",
        ):
            text = self.read(rel)
            self.assertIn("package app.ghostftp.client;", text)
            self.assertNotIn(AUTHOR_IDENTITY, text.lower())

    def test_android_permissions_and_backup_stay_narrow(self) -> None:
        manifest = self.read("android/app/src/main/AndroidManifest.xml")
        extraction = self.read("android/app/src/main/res/xml/data_extraction_rules.xml")
        legacy = self.read("android/app/src/main/res/xml/backup_rules.xml")
        self.assertIn("android.permission.INTERNET", manifest)
        self.assertIn('android:allowBackup="false"', manifest)
        self.assertIn('android:dataExtractionRules="@xml/data_extraction_rules"', manifest)
        self.assertIn('android:fullBackupContent="@xml/backup_rules"', manifest)
        self.assertIn('android:icon="@drawable/ic_ghostftp"', manifest)
        self.assertIn('android:label="@string/app_name"', manifest)
        for domain in ("root", "file", "database", "sharedpref", "external"):
            self.assertIn(f'<exclude domain="{domain}" path="." />', extraction)
            self.assertIn(f'<exclude domain="{domain}" path="." />', legacy)
        for forbidden in (
            "MANAGE_EXTERNAL_STORAGE",
            "READ_EXTERNAL_STORAGE",
            "WRITE_EXTERNAL_STORAGE",
            "ACCESS_FINE_LOCATION",
            "ACCESS_COARSE_LOCATION",
            "screenOrientation",
        ):
            self.assertNotIn(forbidden, manifest)

    def test_ftps_is_strict_and_has_no_trust_all_fallback(self) -> None:
        ftp = self.read(f"{ANDROID_JAVA}/FtpSession.java")
        for marker in (
            'command("AUTH TLS")',
            'parameters.setEndpointIdentificationAlgorithm("HTTPS")',
            'command("PBSZ 0")',
            'command("PROT P")',
            "tls.startHandshake()",
        ):
            self.assertIn(marker, ftp)
        for forbidden in (
            "X509TrustManager",
            "HostnameVerifier",
            "TrustManager[]",
            "setDefaultHostnameVerifier",
        ):
            self.assertNotIn(forbidden, ftp)

    def test_upload_stages_before_final_remote_name_commit(self) -> None:
        ftp = self.read(f"{ANDROID_JAVA}/FtpSession.java")
        upload_start = ftp.index("synchronized void upload(")
        upload_end = ftp.index("synchronized void download(", upload_start)
        upload = ftp[upload_start:upload_end]
        for marker in (
            "String tempPath = uploadTempPath(path);",
            'command("STOR " + sanitizeArgument(tempPath))',
            "expect(terminal, 226, 250);",
            'command("RNFR " + sanitizeArgument(tempPath))',
            'command("RNTO " + sanitizeArgument(path))',
            "deleteRemoteBestEffort(tempPath);",
            "hardClose();",
        ):
            self.assertIn(marker, upload)
        self.assertNotIn('command("STOR " + sanitizeArgument(path))', upload)
        completion = upload.index("expect(terminal, 226, 250);")
        rename_from = upload.index('command("RNFR " + sanitizeArgument(tempPath))')
        rename_to = upload.index('command("RNTO " + sanitizeArgument(path))')
        self.assertLess(completion, rename_from)
        self.assertLess(rename_from, rename_to)
        self.assertIn('".ghostftp-upload-" + UUID.randomUUID() + ".part"', ftp)

    def test_download_stages_saf_document_before_final_name_commit(self) -> None:
        activity = self.read(f"{ANDROID_JAVA}/MainActivity.java")
        download_start = activity.index("private void downloadSelected()")
        helper_start = activity.index("private void ensureNoLocalNameConflict(", download_start)
        helper_end = activity.index("private void clearLocalRoot()", helper_start)
        download = activity[download_start:helper_start]
        helpers = activity[helper_start:helper_end]
        for marker in (
            '".ghostftp-download-" + UUID.randomUUID() + ".part"',
            "ensureNoLocalNameConflict(selectedTree, selectedDocumentId, entry.name,",
            "current.download(FtpSession.joinRemote(remoteBase, entry.name), out);",
            "DocumentsContract.renameDocument(getContentResolver(), staged, entry.name)",
            "queryDocumentDisplayName(committed)",
            "DocumentsContract.deleteDocument(getContentResolver(), staged)",
        ):
            self.assertIn(marker, download)
        self.assertGreaterEqual(download.count("ensureNoLocalNameConflict("), 2)
        self.assertNotIn(
            'DocumentsContract.createDocument(getContentResolver(), parent, "application/octet-stream", entry.name)',
            download,
        )
        create = download.index("DocumentsContract.createDocument")
        transfer = download.index("current.download(")
        second_conflict = download.rindex("ensureNoLocalNameConflict(")
        rename = download.index("DocumentsContract.renameDocument")
        verify = download.index("queryDocumentDisplayName(committed)")
        success = download.index('setBusy(false, "Download completed and committed: "')
        self.assertLess(create, transfer)
        self.assertLess(transfer, second_conflict)
        self.assertLess(second_conflict, rename)
        self.assertLess(rename, verify)
        self.assertLess(verify, success)
        self.assertIn("List<LocalEntry> fresh = queryChildren(rootTreeUri, documentId);", helpers)
        self.assertIn("if (name.equals(local.name)) throw new IOException(message);", helpers)
        self.assertIn("DocumentsContract.Document.COLUMN_DISPLAY_NAME", helpers)

    def test_active_transfer_cancel_closes_transport_and_blocks_stale_success(self) -> None:
        ftp = self.read(f"{ANDROID_JAVA}/FtpSession.java")
        activity = self.read(f"{ANDROID_JAVA}/MainActivity.java")
        self.assertIn("private volatile Socket activeDataSocket;", ftp)
        self.assertIn("void cancelActiveTransfer() {\n        hardClose();\n    }", ftp)
        self.assertNotIn("synchronized void cancelActiveTransfer()", ftp)
        hard_close_start = ftp.index("private void hardClose()")
        hard_close_end = ftp.index("private static String sanitizeArgument", hard_close_start)
        hard_close = ftp[hard_close_start:hard_close_end]
        self.assertIn("Socket data = activeDataSocket;", hard_close)
        self.assertIn("Socket control = controlSocket;", hard_close)
        self.assertIn("closeQuietly(data);", hard_close)
        self.assertIn("closeQuietly(control);", hard_close)
        self.assertIn("activeDataSocket = plain;", ftp)

        for marker in (
            "private boolean transferActive;",
            "private long transferGeneration;",
            'disconnect.setText(canCancelTransfer ? "Cancel transfer" : "Disconnect");',
            "disconnect.setEnabled(canCancelTransfer || (!busy && connected));",
            "current.cancelActiveTransfer();",
            "transferGeneration++;",
            "if (!transferIsCurrent(current, generation)) return;",
            "if (transferGeneration != generation) return;",
            "session = null;",
            "connectedIdentityKey = null;",
            'setBusy(false, "Transfer cancelled. Connection closed; reconnect before continuing.");',
        ):
            self.assertIn(marker, activity)

        cancel_start = activity.index("private void cancelActiveTransfer()")
        cancel_end = activity.index("private void refreshRemote(", cancel_start)
        cancel = activity[cancel_start:cancel_end]
        generation = cancel.index("transferGeneration++;")
        detach = cancel.index("session = null;")
        close = cancel.index("current.cancelActiveTransfer();")
        self.assertLess(generation, detach)
        self.assertLess(detach, close)

    def test_password_is_memory_only_and_storage_uses_saf(self) -> None:
        activity = self.read(f"{ANDROID_JAVA}/MainActivity.java")
        for marker in (
            "Intent.ACTION_OPEN_DOCUMENT_TREE",
            "takePersistableUriPermission",
            "getPersistedUriPermissions()",
            "password.setText(\"\")",
            'getSharedPreferences(PREFS, MODE_PRIVATE)',
            "localParents.push(currentDocumentId)",
            "localParents.pop();",
        ):
            self.assertIn(marker, activity)
        self.assertNotIn('putString("password"', activity)
        self.assertNotIn('putString("passphrase"', activity)
        self.assertNotIn("lastIndexOf('/')", activity)

    def test_site_profiles_persist_only_non_secret_metadata(self) -> None:
        model = self.read(f"{ANDROID_JAVA}/SiteProfile.java")
        store = self.read(f"{ANDROID_JAVA}/SiteProfileStore.java")
        combined = (model + store).lower()
        for marker in (
            'object.put("id"',
            'object.put("protocol"',
            'object.put("host"',
            'object.put("username"',
            'object.put("localstarttreeuri"',
            'object.put("remotestartpath"',
            'object.put("localbookmarks"',
            'object.put("remotebookmarks"',
        ):
            self.assertIn(marker, combined)
        for forbidden in (
            'object.put("password"',
            'object.put("passphrase"',
            'object.put("privatekey"',
            'object.put("secret"',
            "encryptedpassword",
        ):
            self.assertNotIn(forbidden, combined)
        self.assertIn("MAX_PROFILES = 50", store)
        self.assertIn("MAX_BOOKMARKS_PER_KIND = 50", store)

    def test_server_identity_change_clears_server_paths(self) -> None:
        model = self.read(f"{ANDROID_JAVA}/SiteProfile.java")
        activity = self.read(f"{ANDROID_JAVA}/MainActivity.java")
        self.assertIn("withRemoteStateResetForIdentityChange", model)
        self.assertIn('"/",\n                localBookmarks,\n                Collections.emptyList()', model)
        self.assertIn(".withRemoteStateResetForIdentityChange(previous)", activity)
        self.assertIn("Server start path and server bookmarks were cleared", activity)
        self.assertIn("saved server paths will not be reused across identities", activity)

    def test_quick_connect_never_auto_creates_profile(self) -> None:
        activity = self.read(f"{ANDROID_JAVA}/MainActivity.java")
        connect_start = activity.index("private void connect()")
        connect_end = activity.index("private void disconnect()", connect_start)
        connect = activity[connect_start:connect_end]
        self.assertNotIn("UUID.randomUUID", connect)
        self.assertNotIn("profileStore.save", connect)
        self.assertIn("profile == null ? null : profile.remoteStartPath", connect)
        self.assertIn("Quick Connect mode", activity)
        self.assertIn("Quick Connect does not create hidden", activity)

    def test_remote_start_and_bookmarks_commit_only_after_fresh_listing(self) -> None:
        activity = self.read(f"{ANDROID_JAVA}/MainActivity.java")
        refresh_list = activity.index("List<RemoteEntry> entries = current.list(requested);")
        refresh_commit = activity.index("currentRemotePath = requested;", refresh_list)
        self.assertGreater(refresh_commit, refresh_list)
        connect_list = activity.index("List<RemoteEntry> entries = next.list(start);")
        connect_commit = activity.index("currentRemotePath = start;", connect_list)
        self.assertGreater(connect_commit, connect_list)
        self.assertIn("openRemoteBookmark()", activity)
        self.assertIn("refreshRemote(profile.remoteBookmarks.get(index));", activity)
        self.assertIn("if (session != current) return;", activity)

    def test_local_profile_paths_revalidate_persisted_saf_capability(self) -> None:
        activity = self.read(f"{ANDROID_JAVA}/MainActivity.java")
        self.assertIn("tryActivateLocalTree(Uri selected, String failureMessage, boolean requirePersisted)", activity)
        self.assertIn("(requirePersisted && !hasPersistedReadPermission(selected))", activity)
        self.assertIn('"Saved local folder is no longer available.", true', activity)
        self.assertIn('"Local bookmark is stale or its persisted permission is unavailable. Re-select the folder to restore access.",\n                true', activity)
        self.assertIn('tryActivateLocalTree(selected, "Selected folder could not be opened.", false)', activity)
        permission = activity.index("(requirePersisted && !hasPersistedReadPermission(selected))")
        listing = activity.index("List<LocalEntry> next = queryChildren(selected, documentId);", permission)
        commit = activity.index("treeUri = selected;", listing)
        self.assertLess(permission, listing)
        self.assertLess(listing, commit)
        self.assertIn("Local site start folder saved as a SAF capability URI", activity)
        self.assertIn("Local folder opened for this session only", activity)

    def test_stale_local_start_error_is_not_overwritten(self) -> None:
        activity = self.read(f"{ANDROID_JAVA}/MainActivity.java")
        self.assertIn("boolean localStartUnavailable = false;", activity)
        self.assertIn("if (localStartUnavailable) {", activity)
        self.assertIn("Site loaded, but its local start folder is unavailable", activity)
        self.assertIn("} else {\n            setStatus(\"Site loaded. Password remains blank", activity)

    def test_sftp_is_fail_closed_until_host_key_verification_exists(self) -> None:
        readme = self.read("android/README.md")
        activity = self.read(f"{ANDROID_JAVA}/MainActivity.java")
        self.assertIn("SFTP is intentionally not exposed", readme)
        self.assertIn('new String[]{"FTPS", "FTP"}', activity)
        self.assertNotIn('"SFTP"', activity)

    def test_android_has_no_telemetry_or_ad_sdk_dependency(self) -> None:
        build = self.read("android/app/build.gradle")
        settings = self.read("android/settings.gradle")
        combined = (build + settings).lower()
        for forbidden in (
            "firebase",
            "analytics",
            "crashlytics",
            "appsflyer",
            "facebook",
            "admob",
            "com.google.android.gms:play-services-ads",
        ):
            self.assertNotIn(forbidden, combined)


if __name__ == "__main__":
    unittest.main()
