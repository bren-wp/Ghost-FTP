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

    def test_phone_drawer_and_tablet_sidebar_are_real_runtime_navigation(self) -> None:
        activity = self.read(ACTIVITY)
        for marker in (
            "private static final int TABLET_SIDEBAR_MIN_DP = 700;",
            "FILES,",
            "SITES,",
            "BOOKMARKS,",
            "TRANSFERS,",
            "SETTINGS,",
            "ABOUT",
            'navButton("Files", R.drawable.ic_files, Section.FILES)',
            'navButton("Sites", R.drawable.ic_sites, Section.SITES)',
            'navButton("Bookmarks", R.drawable.ic_bookmarks, Section.BOOKMARKS)',
            'navButton("Transfers", R.drawable.ic_transfers, Section.TRANSFERS)',
            'navButton("Settings", R.drawable.ic_settings, Section.SETTINGS)',
            'navButton("About", R.drawable.ic_about, Section.ABOUT)',
            "tabletLayout = getResources().getConfiguration().screenWidthDp >= TABLET_SIDEBAR_MIN_DP;",
            "menuToggle.setOnClickListener(v -> openNavigationDrawer());",
            "navigationPanel.setVisibility(View.GONE);",
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
            "aboutSurface.setVisibility(section == Section.ABOUT ? View.VISIBLE : View.GONE);",
        ):
            self.assertIn(marker, show)

    def test_navigation_uses_local_vector_assets_and_no_emoji_controls(self) -> None:
        for name in (
            "ic_menu.xml",
            "ic_files.xml",
            "ic_sites.xml",
            "ic_bookmarks.xml",
            "ic_transfers.xml",
            "ic_settings.xml",
            "ic_about.xml",
        ):
            content = self.read(DRAWABLES / name)
            self.assertIn("<vector", content)
            self.assertIn("<path", content)

        activity = self.read(ACTIVITY)
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
        self.assertIn("No decorative queue or fake history is displayed.", surface)

        buttons_start = activity.index("private void refreshButtons()")
        buttons_end = activity.index("private void updateConnectionBadge(", buttons_start)
        buttons = activity[buttons_start:buttons_end]
        self.assertIn("transferCancel.setEnabled(transferActive && !transferFinalizing);", buttons)
        self.assertIn('transferCancel.setText(transferFinalizing ? "Finalizing…" : "Cancel active transfer");', buttons)

    def test_settings_controls_have_runtime_owners(self) -> None:
        activity = self.read(ACTIVITY)
        settings_start = activity.index("private View buildSettingsSurface()")
        settings_end = activity.index("private View buildAboutSurface()", settings_start)
        settings = activity[settings_start:settings_end]
        self.assertIn("rememberEndpointToggle.setOnClickListener", settings)
        self.assertIn("showFileSizesToggle.setOnClickListener", settings)
        self.assertIn("savePreferences();", settings)
        self.assertIn("renderLocal();", settings)
        self.assertIn("renderRemote();", settings)
        self.assertIn("Runtime security policy is informational here and cannot be weakened from the UI.", settings)

    def test_about_uses_generated_build_identity_not_hardcoded_version(self) -> None:
        activity = self.read(ACTIVITY)
        build = self.read(ANDROID_BUILD)
        self.assertIn("buildFeatures {", build)
        self.assertIn("buildConfig true", build)
        self.assertIn("versionName \"${ghostFtpVersion}-dev\"", build)
        self.assertIn('infoLine("Version", BuildConfig.VERSION_NAME)', activity)
        self.assertIn('infoLine("Package", BuildConfig.APPLICATION_ID)', activity)
        about_start = activity.index("private View buildAboutSurface()")
        about_end = activity.index("private void addSurface(", about_start)
        about = activity[about_start:about_end]
        self.assertNotIn('infoLine("Version", "0.0.3', about)
        self.assertIn('infoLine("Release status", "Repository build " + BuildConfig.VERSION_NAME', about)
        self.assertIn("Android APK remains development-only", about)
        self.assertIn("not a public Android release", about)

    def test_android_docs_describe_the_same_surface_contract(self) -> None:
        readme = self.read(ANDROID_README)
        ui_doc = self.read(UI_DOC)
        for marker in ("Files", "Sites", "Bookmarks", "Transfers", "Settings", "About"):
            self.assertIn(marker, readme)
            self.assertIn(marker, ui_doc)
        self.assertIn("navigation drawer", readme.lower())
        self.assertIn("persistent sidebar", readme.lower())
        self.assertIn("Remote Desktop", ui_doc)
        self.assertIn("not shown", ui_doc.lower())
        self.assertIn("development", readme.lower())


if __name__ == "__main__":
    unittest.main()
