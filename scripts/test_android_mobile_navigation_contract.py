#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
ACTIVITY = ROOT / "android/app/src/main/java/app/ghostftp/client/MainActivity.java"
ANDROID_BUILD = ROOT / "android/app/build.gradle"
ANDROID_README = ROOT / "android/README.md"
UI_DOC = ROOT / "android/UI-UX.md"
DRAWABLES = ROOT / "android/app/src/main/res/drawable"


class AndroidMobileNavigationContractTests(unittest.TestCase):
    def read(self, path: Path) -> str:
        return path.read_text(encoding="utf-8")

    def test_phone_bottom_navigation_and_tablet_sidebar_are_real_runtime_navigation(self) -> None:
        activity = self.read(ACTIVITY)
        for marker in (
            "private static final int TABLET_SIDEBAR_MIN_DP = 700;",
            "FILES,",
            "SITES,",
            "BOOKMARKS,",
            "TRANSFERS,",
            "SETTINGS,",
            "CONNECTION_INFO,",
            "ABOUT",
            'navButton("Files", R.drawable.ic_files, Section.FILES)',
            'navButton("Connections", R.drawable.ic_sites, Section.SITES)',
            'navButton("Transfer Queue", R.drawable.ic_transfers, Section.TRANSFERS)',
            'navButton("Settings", R.drawable.ic_settings, Section.SETTINGS)',
            'navButton("Bookmarks", R.drawable.ic_bookmarks, Section.BOOKMARKS)',
            'navButton("Connection info", R.drawable.ic_connection_info, Section.CONNECTION_INFO)',
            'navButton("About", R.drawable.ic_about, Section.ABOUT)',
            "tabletLayout = getResources().getConfiguration().screenWidthDp >= TABLET_SIDEBAR_MIN_DP;",
            "menuToggle.setOnClickListener(v -> openNavigationDrawer());",
            "navigationPanel.setVisibility(View.GONE);",
            "drawerParams.gravity = Gravity.END;",
            "private LinearLayout buildBottomNavigation()",
            'bottomNavButton("Files", R.drawable.ic_files, Section.FILES)',
            'bottomNavButton("Connections", R.drawable.ic_sites, Section.SITES)',
            'bottomNavButton("Bookmarks", R.drawable.ic_bookmarks, Section.BOOKMARKS)',
            'bottomNavButton("Transfer Queue", R.drawable.ic_transfers, Section.TRANSFERS)',
            'bottomNavButton("Settings", R.drawable.ic_settings, Section.SETTINGS)',
            "private void showSection(Section section)",
        ):
            self.assertIn(marker, activity)

        show = activity[activity.index("private void showSection(Section section)") : activity.index("private String sectionTitle(")]
        for marker in (
            "filesSurface.setVisibility(section == Section.FILES ? View.VISIBLE : View.GONE);",
            "sitesSurface.setVisibility(section == Section.SITES ? View.VISIBLE : View.GONE);",
            "bookmarksSurface.setVisibility(section == Section.BOOKMARKS ? View.VISIBLE : View.GONE);",
            "transfersSurface.setVisibility(section == Section.TRANSFERS ? View.VISIBLE : View.GONE);",
            "settingsSurface.setVisibility(section == Section.SETTINGS ? View.VISIBLE : View.GONE);",
            "connectionInfoSurface.setVisibility(section == Section.CONNECTION_INFO ? View.VISIBLE : View.GONE);",
            "aboutSurface.setVisibility(section == Section.ABOUT ? View.VISIBLE : View.GONE);",
        ):
            self.assertIn(marker, show)

    def test_phone_files_matches_master_information_hierarchy_without_fake_state(self) -> None:
        activity = self.read(ACTIVITY)
        start = activity.index("private View buildFilesSurface()")
        end = activity.index("private LinearLayout buildLocalFilesCard()", start)
        files = activity[start:end]

        for marker in (
            'connectionCard.setBackground(GhostTheme.rounded(this, GhostTheme.PANEL, GhostTheme.BORDER, 14))',
            'currentConnectionTitle = label("No active connection", 15, GhostTheme.TEXT)',
            'currentConnectionSummary = label("Open Connections", 12, GhostTheme.MUTED)',
            'connectionCard.setOnClickListener(v -> showSection(Section.SITES));',
            'quickActionsCard.setOrientation(LinearLayout.VERTICAL)',
            'filesBack = iconButton("Back", R.drawable.ic_back)',
            'filesForward = iconButton("Forward", R.drawable.ic_forward)',
            'Button refreshAll = iconButton("Refresh", R.drawable.ic_refresh)',
            'Button newFolder = iconButton("New Folder", R.drawable.ic_new_folder)',
            'Button bookmarks = iconButton("Bookmarks", R.drawable.ic_bookmarks)',
            'Button uploadQuick = iconButton("Upload", R.drawable.ic_upload)',
            'Button downloadQuick = iconButton("Download", R.drawable.ic_download)',
            'Button more = iconButton("More", R.drawable.ic_more)',
            'filesBack.setOnClickListener(v -> navigateFilesHistory(true));',
            'filesForward.setOnClickListener(v -> navigateFilesHistory(false));',
            'more.setOnClickListener(v -> showFilesMoreActions());',
            'workspaceCard("Transfer Queue", R.drawable.ic_transfers',
            'filesTransferStatus = label("No active transfer."',
            'transferCard.setOnClickListener(v -> showSection(Section.TRANSFERS));',
            "upload = uploadQuick;",
            "download = downloadQuick;",
        ):
            self.assertIn(marker, files)

        self.assertNotIn("fake", files.lower())
        self.assertNotIn("demo", files.lower())
        self.assertIn('workspaceCard("Local Files", R.drawable.ic_local_files', activity)
        self.assertIn('workspaceCard("Remote Files", R.drawable.ic_remote_files', activity)
        self.assertIn("screenWidthDp >= 400", files)
        self.assertIn("brandIcon.setImageResource(R.drawable.ic_ghost_brand);", activity)
        self.assertIn("menuToggle.setImageResource(R.drawable.ic_overflow_vertical);", activity)
        self.assertIn("connectionBadge.setMinHeight(dp(40));", activity)
        self.assertIn("connectionBadge.setCompoundDrawablesWithIntrinsicBounds(0, 0, R.drawable.ic_expand_more, 0);", activity)
        self.assertIn("connectionBadge.setOnClickListener(v -> showSection(Section.SITES));", activity)
        self.assertIn('button.setTextSize("Connections".equals(text) ? 9f : 10f);', activity)
        self.assertIn("button.setSingleLine(false);", activity)
        self.assertIn("button.setMaxLines(2);", activity)
        self.assertIn("button.setMinWidth(dp(48));", activity)
        self.assertIn("button.setMinHeight(dp(56));", activity)
        self.assertIn("appBar.setPadding(dp(12), dp(8), dp(10), dp(8));", activity)
        self.assertIn('TextView brand = label("Ghost FTP", 18, GhostTheme.TEXT);', activity)
        self.assertIn("ViewGroup.LayoutParams.MATCH_PARENT, dp(72)));", activity)
        self.assertIn("overflow.setBackgroundColor(Color.TRANSPARENT);", activity)
        self.assertIn("header.setBackgroundColor(Color.TRANSPARENT);", activity)
        bottom_start = activity.index("private Button bottomNavButton(")
        bottom_end = activity.index("private LinearLayout.LayoutParams bottomNavParams()", bottom_start)
        bottom = activity[bottom_start:bottom_end]
        self.assertLess(bottom.index("styleNavigationButton(button, false);"), bottom.index("button.setMinHeight(dp(56));"))
        self.assertNotIn("button.setTextSize(8);", activity)
        self.assertIn("if (tabletLayout) card.addView(navigationActions, matchWrap());", activity)
        self.assertIn("private void showFilesMoreActions()", activity)
        self.assertIn('filesBack = iconButton("Back", R.drawable.ic_back);', activity)
        self.assertIn('filesForward = iconButton("Forward", R.drawable.ic_forward);', activity)
        self.assertIn('Button refreshAll = iconButton("Refresh", R.drawable.ic_refresh);', activity)
        self.assertIn('Button newFolder = iconButton("New Folder", R.drawable.ic_new_folder);', activity)
        self.assertIn('Button uploadQuick = iconButton("Upload", R.drawable.ic_upload);', activity)
        self.assertIn('Button downloadQuick = iconButton("Download", R.drawable.ic_download);', activity)
        self.assertIn('workspaceCard("Local Files", R.drawable.ic_local_files', activity)
        self.assertIn('workspaceCard("Remote Files", R.drawable.ic_remote_files', activity)
        self.assertIn('workspaceTableHeader(false)', activity)
        self.assertIn('workspaceTableHeader(true)', activity)
        self.assertIn('connectionIcon.setImageResource(R.drawable.ic_link);', activity)
        self.assertIn("private void navigateFilesHistory(boolean back)", activity)
        self.assertIn("private void refreshRemoteInternal(String target, boolean recordHistory, Runnable onSuccess)", activity)
        self.assertIn("localBackHistory", activity)
        self.assertIn("remoteBackHistory", activity)
        self.assertIn("private void showNewFolderTarget()", activity)
        self.assertIn('labels.add("Choose local folder")', activity)
        self.assertIn('labels.add("Connection info")', activity)
        self.assertIn("actions.add(this::renameLocalSelected)", activity)
        self.assertIn("actions.add(this::renameRemoteSelected)", activity)
        self.assertIn("actions.add(this::chmodRemoteSelected)", activity)
        self.assertIn("actions.add(this::openRemoteEditorSelected)", activity)
        self.assertNotIn('more.setOnClickListener(v -> openNavigationDrawer())', files)

        refresh_start = activity.index("private void refreshFilesMasterSummary()")
        refresh_end = activity.index("private void styleNavigationButton(", refresh_start)
        refresh = activity[refresh_start:refresh_end]
        self.assertIn("session != null && session.isConnected()", refresh)
        self.assertIn("connectedProtocol", refresh)
        self.assertIn("transferFinalizing", refresh)
        self.assertIn("transferActive", refresh)
        self.assertNotIn("password", refresh.lower())
        self.assertNotIn("username", refresh.lower())

    def test_connection_info_is_runtime_owned_and_privacy_safe(self) -> None:
        activity = self.read(ACTIVITY)
        start = activity.index("private View buildConnectionInfoSurface()")
        end = activity.index("private View buildAboutSurface()", start)
        surface = activity[start:end]

        for marker in (
            'surfaceHeading(',
            '"Connection info"',
            'connectionInfoState = infoLine("State", "Disconnected")',
            'connectionInfoProtocol = infoLine("Protocol", "—")',
            'connectionInfoSecurity = infoLine("Security", "No active connection")',
            'connectionInfoTransfer = infoLine("Transfer Queue", "No active transfer.")',
            "refreshConnectionInfoSurface()",
            'security = "Certificate and hostname verification enabled"',
            'security = "Unencrypted compatibility connection"',
        ):
            self.assertIn(marker, surface)

        for forbidden in (
            'connectionInfoHost',
            'connectionInfoUsername',
            'connectionInfoPassword',
            'privateKeyPath',
            'password.getText()',
            'host.getText()',
            'username.getText()',
        ):
            self.assertNotIn(forbidden, surface)

    def test_live_connection_protocol_identity_cannot_drift_with_form_edits(self) -> None:
        activity = self.read(ACTIVITY)

        self.assertIn("private String connectedProtocol;", activity)
        connect_start = activity.index("private void connect()")
        connect_end = activity.index("private void disconnect()", connect_start)
        connect = activity[connect_start:connect_end]
        self.assertIn("String protocolValue = protocol.getSelectedItem().toString();", connect)
        self.assertIn('boolean secure = "FTPS".equals(protocolValue);', connect)
        self.assertIn("String identity = identityKey(protocolValue, hostValue, portValue, userValue);", connect)
        self.assertIn("connectedProtocol = protocolValue;", connect)

        info_start = activity.index("private void refreshConnectionInfoSurface()")
        info_end = activity.index("private View buildAboutSurface()", info_start)
        info = activity[info_start:info_end]
        self.assertIn(
            'String selectedProtocol = connected && connectedProtocol != null ? connectedProtocol : "—";',
            info,
        )
        self.assertNotIn("protocol.getSelectedItem()", info)

        badge_start = activity.index("private void updateConnectionBadge(boolean connected)")
        badge_end = activity.index("private void updateTransferSurface()", badge_start)
        badge = activity[badge_start:badge_end]
        self.assertIn('boolean secure = "FTPS".equals(connectedProtocol);', badge)
        self.assertNotIn("protocol.getSelectedItem()", badge)

        self.assertEqual(
            activity.count("connectedIdentityKey = null;"),
            activity.count("connectedProtocol = null;"),
        )

    def test_authentic_capture_follows_bottom_nav_and_utility_drawer(self) -> None:
        capture = self.read(ROOT / "scripts/capture_android_screenshots.sh")
        self.assertIn("open_utility_navigation()", capture)
        self.assertIn("tap_ui 'Open utility menu'", capture)
        self.assertIn("wait_ui 'Navigate to Connection info'", capture)
        self.assertIn("wait_ui 'Navigate to About'", capture)
        self.assertIn("wait_ui 'Navigate to Files'", capture)
        self.assertIn("wait_ui 'Navigate to Connections'", capture)
        self.assertIn("wait_ui 'Navigate to Bookmarks'", capture)
        self.assertIn("wait_ui 'Navigate to Transfer Queue'", capture)
        self.assertIn("wait_ui 'Navigate to Settings'", capture)
        self.assertIn("capture_primary_surface 'Connections'", capture)
        self.assertIn("capture_primary_surface 'Transfer Queue'", capture)
        self.assertNotIn("tap_ui 'Open navigation'", capture)
        self.assertNotIn("ANDROID_NAV_DRAWER=VISIBLE", capture)

    def test_navigation_uses_local_vector_assets_and_no_emoji_controls(self) -> None:
        for name in (
            "ic_ghost_brand.xml",
            "ic_files.xml",
            "ic_sites.xml",
            "ic_bookmarks.xml",
            "ic_transfers.xml",
            "ic_settings.xml",
            "ic_connection_info.xml",
            "ic_about.xml",
            "ic_back.xml",
            "ic_forward.xml",
            "ic_refresh.xml",
            "ic_new_folder.xml",
            "ic_upload.xml",
            "ic_download.xml",
            "ic_more.xml",
            "ic_link.xml",
            "ic_local_files.xml",
            "ic_remote_files.xml",
            "ic_overflow_vertical.xml",
            "ic_expand_more.xml",
        ):
            content = self.read(DRAWABLES / name)
            self.assertIn("<vector", content)
            self.assertIn("<path", content)

        activity = self.read(ACTIVITY)
        self.assertNotIn("R.drawable.ic_menu", activity)
        self.assertFalse((DRAWABLES / "ic_menu.xml").exists())
        for emoji in ("📁", "🔖", "⚙", "ℹ", "💻", "⬆", "⬇"):
            self.assertNotIn(emoji, activity)

    def test_android_sftp_and_rdp_are_fail_closed_in_navigation(self) -> None:
        activity = self.read(ACTIVITY)
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
        self.assertNotIn("ssh", connect.lower())

        navigation_start = activity.index("private LinearLayout buildNavigationPanel()")
        navigation_end = activity.index("private Button navButton(", navigation_start)
        navigation = activity[navigation_start:navigation_end]
        self.assertNotIn("Remote Desktop", navigation)
        self.assertNotIn("RDP", navigation)
        self.assertNotIn("Coming Soon", navigation)

    def test_transfers_surface_is_owned_by_real_transfer_lifecycle(self) -> None:
        activity = self.read(ACTIVITY)
        surface_start = activity.index("private View buildTransfersSurface()")
        surface_end = activity.index("private View buildSettingsSurface()", surface_start)
        surface = activity[surface_start:surface_end]
        self.assertIn('transferCancel = dangerButton("Cancel active transfer")', surface)
        self.assertIn("transferCancel.setOnClickListener(v -> cancelTransfer());", surface)
        self.assertIn("No active transfer.", surface)
        self.assertNotIn("Retry", surface)
        self.assertNotIn("Resume", surface)
        self.assertNotIn("Clear history", surface)
        self.assertNotIn("fake history", surface.lower())
        self.assertIn('Button openFiles = button("Open Files")', surface)
        self.assertIn("openFiles.setOnClickListener(v -> showSection(Section.FILES));", surface)

        buttons_start = activity.index("private void refreshButtons()")
        buttons_end = activity.index("private void updateConnectionBadge(", buttons_start)
        buttons = activity[buttons_start:buttons_end]
        self.assertIn("transferCancel.setEnabled(transferActive && !transferFinalizing);", buttons)
        self.assertIn('transferCancel.setText(transferFinalizing ? "Finalizing…" : "Cancel active transfer");', buttons)

    def test_settings_controls_have_runtime_owners(self) -> None:
        activity = self.read(ACTIVITY)
        settings_start = activity.index("private View buildSettingsSurface()")
        settings_end = activity.index("private View buildConnectionInfoSurface()", settings_start)
        settings = activity[settings_start:settings_end]
        self.assertIn("rememberEndpointToggle.setOnClickListener", settings)
        self.assertIn("showFileSizesToggle.setOnClickListener", settings)
        self.assertIn("confirmDeleteToggle.setOnClickListener", settings)
        self.assertIn('Button restoreDefaults = button("Restore app defaults")', settings)
        self.assertIn("restoreDefaults.setOnClickListener(v -> restoreDefaultPreferences());", settings)
        self.assertIn("savePreferences();", settings)
        self.assertIn("renderLocal();", settings)
        self.assertIn("renderRemote();", settings)
        for marker in (
            'LinearLayout securityCard = card("SECURITY", "Security protections stay enforced automatically.")',
            'infoLine("FTPS", "Secure certificate checks are enabled")',
            'infoLine("Passwords", "Never saved")',
            'infoLine("Local storage", "Access limited to folders you select")',
            'infoLine("SFTP", "Not available in the Android app")',
            'infoLine("Privacy", "No telemetry, analytics, ads or Ghost FTP cloud")',
        ):
            self.assertIn(marker, settings)

    def test_delete_confirmation_and_restore_defaults_have_real_runtime_owners(self) -> None:
        activity = self.read(ACTIVITY)
        self.assertIn('prefs.getBoolean("confirmDelete", true)', activity)
        self.assertIn('.putBoolean("confirmDelete", confirmDelete)', activity)
        self.assertIn("if (!confirmDelete) {", activity)
        self.assertIn("action.run();", activity)
        self.assertIn("private void restoreDefaultPreferences()", activity)
        self.assertIn("rememberEndpoint = false;", activity)
        self.assertIn("showFileSizes = true;", activity)
        self.assertIn("confirmDelete = true;", activity)
        self.assertIn("Saved connections and your selected local folder are kept.", activity)

    def test_about_uses_canonical_release_identity_without_dev_surface(self) -> None:
        activity = self.read(ACTIVITY)
        build = self.read(ANDROID_BUILD)
        self.assertIn("buildFeatures {", build)
        self.assertIn("buildConfig true", build)
        self.assertIn("versionName ghostFtpVersion", build)
        self.assertIn("applicationIdSuffix '.debug'", build)
        self.assertNotIn("versionNameSuffix '-dev'", build)
        self.assertNotIn('versionName "${ghostFtpVersion}-dev"', build)

        about_start = activity.index("private View buildAboutSurface()")
        about_end = activity.index("private void addSurface(", about_start)
        about = activity[about_start:about_end]
        self.assertIn('infoLine("Version", BuildConfig.VERSION_NAME)', about)
        self.assertIn('infoLine("Protocols", "FTP and explicit FTPS")', about)
        self.assertIn('infoLine("Data collection", "No telemetry, analytics or ads")', about)
        self.assertNotIn('infoLine("Version", "0.0.3', about)
        self.assertNotIn("BuildConfig.DEBUG", about)
        self.assertNotIn("development package", about.lower())
        self.assertNotIn("development-only", about.lower())
        self.assertNotIn("not the production-signed public APK", about)

    def test_master_button_typography_uses_regular_weight(self) -> None:
        theme = self.read(ROOT / "android/app/src/main/java/app/ghostftp/client/GhostTheme.java")
        self.assertIn("button.setTypeface(Typeface.DEFAULT, Typeface.NORMAL);", theme)
        self.assertNotIn("button.setTypeface(Typeface.DEFAULT, Typeface.BOLD);", theme)

    def test_android_docs_describe_the_same_surface_contract(self) -> None:
        readme = self.read(ANDROID_README)
        ui_doc = self.read(UI_DOC)
        for marker in ("Files", "Connections", "Transfer Queue", "Settings", "Bookmarks", "Connection info", "About"):
            self.assertIn(marker, readme)
            self.assertIn(marker, ui_doc)
        self.assertIn("bottom navigation", readme.lower())
        self.assertIn("utility drawer", readme.lower())
        self.assertIn("persistent sidebar", readme.lower())
        self.assertIn("bottom navigation", ui_doc.lower())
        self.assertIn("Remote Desktop", ui_doc)
        self.assertIn("not shown", ui_doc.lower())
        self.assertIn("temporary compatibility-signed", readme)
        self.assertIn("protected production-signing workflow", readme)
        self.assertIn("validation outputs keep the canonical visible version", ui_doc)
        self.assertNotIn("development builds append `-dev`", ui_doc)


if __name__ == "__main__":
    unittest.main()
