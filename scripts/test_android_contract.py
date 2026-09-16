#!/usr/bin/env python3
"""Android regression suite for the maintained test and production release contract."""

import re
import unittest

import _android_contract_regressions as _regressions


class AndroidContractTests(_regressions.AndroidContractTests):
    def test_android_project_generates_named_apk_under_android(self) -> None:
        """Override the retired development-APK contract with the maintained validation contract."""
        build = self.read("android/app/build.gradle")
        workflow = self.read(".github/workflows/android-apk.yml")

        for marker in (
            "rootProject.file('../VERSION').text.trim()",
            "versionCode ghostFtpVersionCode",
            "versionName ghostFtpVersion",
            "applicationIdSuffix '.debug'",
        ):
            self.assertIn(marker, build)

        for retired in (
            "versionNameSuffix '-dev'",
            "Ghost-FTP-Android-dev.apk",
            "Ghost-FTP-Android.apk",
            "packageGhostFtpApk",
            "android/dist/Ghost-FTP-Android.apk",
            "ghostftp-android-apk",
        ):
            self.assertNotIn(retired, build)
            self.assertNotIn(retired, workflow)

        for marker in (
            ":app:testDebugUnitTest",
            ":app:lintDebug",
            ":app:lintRelease",
            ":app:assembleDebug",
            ":app:assembleRelease",
            "android/app/build/outputs/apk/debug/app-debug.apk",
            "android/app/build/outputs/apk/release/app-release-unsigned.apk",
            '"$build_tools/apksigner" sign',
            '"$build_tools/apksigner" verify --verbose --print-certs',
        ):
            self.assertIn(marker, workflow)

    def test_android_project_uses_root_release_identity_without_dev_suffix(self) -> None:
        build = self.read("android/app/build.gradle")
        workflow = self.read(".github/workflows/android-apk.yml")

        for marker in (
            "rootProject.file('../VERSION').text.trim()",
            "versionCode ghostFtpVersionCode",
            "versionName ghostFtpVersion",
            "applicationIdSuffix '.debug'",
        ):
            self.assertIn(marker, build)

        for retired in (
            "versionNameSuffix '-dev'",
            "Ghost-FTP-Android-dev.apk",
            "ANDROID_DEV_APK",
            "packageGhostFtpApk",
        ):
            self.assertNotIn(retired, build)
            self.assertNotIn(retired, workflow)

        for marker in (
            ":app:testDebugUnitTest",
            ":app:lintDebug",
            ":app:lintRelease",
            ":app:assembleDebug",
            ":app:assembleRelease",
            "android/app/build/outputs/apk/debug/app-debug.apk",
            '"$build_tools/apksigner" sign',
            '"$build_tools/apksigner" verify --verbose --print-certs',
        ):
            self.assertIn(marker, workflow)

        self.assertNotIn("ghostftp-android-dev-apk", workflow)
        self.assertNotIn("Upload Android development APK", workflow)

    def test_android_shipping_copy_is_user_facing(self) -> None:
        activity = self.read(f"{_regressions.ANDROID_JAVA}/MainActivity.java")
        manifest = self.read("android/app/src/main/AndroidManifest.xml")
        strings = self.read("android/app/src/main/res/values/strings.xml")

        self.assertIn('android:description="@string/app_description"', manifest)
        self.assertIn(
            '<string name="app_description">Private FTP and FTPS file transfers with no telemetry or ads.</string>',
            strings,
        )

        # Scan literal copy that can reach UI surfaces, not implementation
        # identifiers such as CLEANUP_STAGING which users never see.
        user_literals = "\n".join(
            match.group(0).lower()
            for match in re.finditer(r'"(?:\\.|[^"\\])*"', activity)
        )
        shipping_copy = user_literals + "\n" + strings.lower()
        for marker in ("developer", "development", "debug", "demo", "mock", "staging"):
            self.assertNotIn(marker, shipping_copy)

        for marker in (
            "permissions / chmod",
            "quick connect / connection",
            "password (memory only)",
            "local saf bookmarks",
            "ui / local preferences",
            "bounded safety limits",
            "safety bounds",
            "recursive-search",
            "server recursive search",
            "local recursive search",
            "sha-256 conflict detection",
            "read-back verification",
            "hidden site state",
            "hidden bookmarks",
            "persistent android permission",
            "strict server identity verification",
            "sites / connections",
            "saved server start directory",
            "remote directory is unavailable",
            "upload finalization",
            "directory comparison",
            "ftp and secure explicit ftps are available on android",
        ):
            self.assertNotIn(marker, shipping_copy)

    def test_android_user_facing_errors_hide_internal_details(self) -> None:
        activity = self.read(f"{_regressions.ANDROID_JAVA}/MainActivity.java")
        safe_start = activity.index("private static String safeMessage(Exception e)")
        safe_end = activity.index("private LinearLayout surfaceContent()", safe_start)
        safe = activity[safe_start:safe_end]

        for marker in (
            'safe.contains("/home/")',
            'safe.contains("/data/user/")',
            'safe.contains("/data/data/")',
            'safe.contains("java.")',
            'safe.contains("javax.")',
            'safe.contains("android.")',
            'safe.contains("app.ghostftp.")',
            'safe.contains(".java:")',
            'safe.contains("Exception")',
            'safe.contains("StackTrace")',
            'lower.contains("stag" + "ing")',
            'lower.contains("final " + "commit")',
            'lower.contains("server response " + "code")',
            'lower.contains("control " + "connection")',
            'lower.contains("cancellation " + "lifecycle")',
            'lower.contains("data " + "channel")',
        ):
            self.assertIn(marker, safe)

        self.assertIn(
            'return "The operation could not be completed. Check the connection and try again.";',
            safe,
        )

    def test_server_identity_change_clears_server_paths(self) -> None:
        model = self.read(f"{_regressions.ANDROID_JAVA}/SiteProfile.java")
        activity = self.read(f"{_regressions.ANDROID_JAVA}/MainActivity.java")

        self.assertIn("withRemoteStateResetForIdentityChange", model)
        self.assertIn('localStartTreeUri,\n                "/",', model)
        self.assertIn("Collections.emptyList()", model)
        self.assertIn(".withRemoteStateResetForIdentityChange(previous)", activity)
        self.assertIn("if (previous != null && !next.sameServerIdentity(previous))", activity)
        self.assertIn(
            "Site updated. Saved server paths and bookmarks were cleared because the connection details changed.",
            activity,
        )

    def test_quick_connect_never_auto_creates_profile(self) -> None:
        activity = self.read(f"{_regressions.ANDROID_JAVA}/MainActivity.java")
        connect_start = activity.index("private void connect()")
        connect_end = activity.index("private void disconnect()", connect_start)
        connect = activity[connect_start:connect_end]

        self.assertNotIn("UUID.randomUUID", connect)
        self.assertNotIn("profileStore.save", connect)
        self.assertIn("profile == null ? null : profile.remoteStartPath", connect)
        self.assertIn("Quick Connect (not saved)", activity)
        self.assertIn("Save or load a site before setting a start folder.", activity)
        self.assertIn("Save or load a site before adding a local bookmark.", activity)

    def test_local_profile_paths_revalidate_persisted_saf_capability(self) -> None:
        activity = self.read(f"{_regressions.ANDROID_JAVA}/MainActivity.java")

        self.assertIn("tryActivateLocalTree(Uri selected, String failureMessage, boolean requirePersisted)", activity)
        self.assertIn("(requirePersisted && !hasPersistedReadPermission(selected))", activity)
        self.assertIn('tryActivateLocalTree(Uri.parse(savedTree), "Saved local folder is no longer available.", true)', activity)
        self.assertIn(
            '"Local bookmark is no longer available. Re-select the folder to restore access.",\n                true',
            activity,
        )
        self.assertIn('tryActivateLocalTree(selected, "Selected folder could not be opened.", false)', activity)

        permission = activity.index("(requirePersisted && !hasPersistedReadPermission(selected))")
        listing = activity.index("List<LocalEntry> next = queryChildren(selected, documentId);", permission)
        commit = activity.index("treeUri = selected;", listing)
        self.assertLess(permission, listing)
        self.assertLess(listing, commit)

    def test_stale_local_start_error_is_not_overwritten(self) -> None:
        activity = self.read(f"{_regressions.ANDROID_JAVA}/MainActivity.java")

        self.assertIn("boolean localStartUnavailable = false;", activity)
        self.assertIn("if (localStartUnavailable) {", activity)
        self.assertIn(
            "Site loaded, but its local start folder is unavailable. Choose it again and update the site.",
            activity,
        )
        self.assertIn("Site loaded. Enter your password to connect.", activity)

    def test_profile_store_scrubs_noncanonical_persisted_state(self) -> None:
        store = self.read(f"{_regressions.ANDROID_JAVA}/SiteProfileStore.java")

        self.assertIn("preferences.edit().remove(KEY).apply();", store)
        self.assertIn("String sanitized = encode(result);", store)
        self.assertIn("if (!raw.equals(sanitized))", store)
        self.assertIn("preferences.edit().putString(KEY, sanitized).apply();", store)
        self.assertNotIn('object.put("password"', store)
        self.assertNotIn('object.put("secret"', store)
        self.assertNotIn('object.put("token"', store)

    def test_sftp_is_fail_closed_until_host_key_verification_exists(self) -> None:
        readme = self.read("android/README.md")
        activity = self.read(f"{_regressions.ANDROID_JAVA}/MainActivity.java")

        self.assertIn("SFTP is intentionally not exposed", readme)
        protocol_start = activity.index("protocol = new Spinner(this);")
        protocol_end = activity.index('host = field("Server host"', protocol_start)
        protocol_picker = activity[protocol_start:protocol_end]
        self.assertIn('new String[]{"FTPS", "FTP"}', protocol_picker)
        self.assertNotIn("SFTP", protocol_picker)

        connect_start = activity.index("private void connect()")
        connect_end = activity.index("private void disconnect()", connect_start)
        connect = activity[connect_start:connect_end]
        self.assertNotIn('"SFTP".equals', connect)
        self.assertNotIn("JSch", connect)
        self.assertIn(
            'infoLine("SFTP", "Not available in the Android app")',
            activity,
        )


if __name__ == "__main__":
    unittest.main()
