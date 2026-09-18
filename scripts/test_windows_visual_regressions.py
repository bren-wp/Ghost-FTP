import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class WindowsVisualRegressionTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_workspace_refinement_does_not_erase_entire_parent(self):
        source = self.read("internal/desktop/workspace_layout_windows.go")
        self.assertNotIn("invalidateRect.Call(a.hwnd, 0, 1)", source)
        self.assertIn("a.stabilizeWorkspaceChrome()", source)
        connection = self.read("internal/desktop/connection_profiles_windows.go")
        self.assertNotIn("invalidateRect.Call(a.hwnd, 0, 1)", connection)

    def test_workspace_uses_real_pe_brand_icon_and_full_title_gutter(self):
        chrome = self.read("internal/desktop/chrome_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        self.assertIn('loadImageW.Call(hinst, 1, imageIcon', chrome)
        self.assertIn("stmSetImage", chrome)
        self.assertNotIn("a.move(a.brandTitle, 54, 10, 106, 35)", chrome)
        self.assertIn("a.move(a.brandTitle, 54, headerY, 126, 35)", ui)
        self.assertIn("subtitleX := 188", ui)

    def test_sidebar_brand_and_connection_status_cannot_overlap_profile_actions(self):
        sidebar = self.read("internal/desktop/sidebar_windows.go")
        self.assertIn("sendMessageW.Call(a.brandTitle, wmSetFont, a.font, 1)", sidebar)
        self.assertIn("badgeW, buttonW, gap := 118, 116, 8", sidebar)
        self.assertIn("a.move(a.connectionBadge, badgeX, 18, badgeW, 21)", sidebar)
        self.assertIn("saveX := badgeX + badgeW + gap", sidebar)
        self.assertIn("a.move(a.saveProfile, saveX, 13, buttonW, 31)", sidebar)
        self.assertIn("a.move(a.removeProfile, saveX+buttonW+gap, 13, buttonW, 31)", sidebar)

    def test_native_dark_chrome_and_site_manager_use_premium_controls(self):
        dark = self.read("internal/desktop/dark_mode_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        site = self.read("internal/desktop/site_manager_windows.go")
        self.assertIn("enableImmersiveDarkMode(hwnd)", ui)
        self.assertIn("SetPreferredAppMode", dark)
        self.assertIn('case "COMBOBOX", "EDIT":', ui)
        self.assertIn("case wmDrawItem:", site)
        self.assertIn("bsOwnerDraw", site)
        self.assertIn("IconSm:     icon", site)

    def test_resize_is_batched_without_background_erase(self):
        source = self.read("internal/desktop/layout_batch_windows.go")
        windows = self.read("internal/desktop/windows.go")
        self.assertIn("wmSetRedraw", source)
        self.assertIn("rdwAllChildren", source)
        self.assertNotIn("rdwErase", source)
        self.assertIn("a.reflowWorkspace(w, h)", windows)

    def test_full_wordmark_dark_headers_and_sidebar_are_guarded(self):
        chrome = self.read("internal/desktop/chrome_windows.go")
        header = self.read("internal/desktop/header_draw_windows.go")
        sidebar = self.read("internal/desktop/sidebar_windows.go")
        navigation = self.read("internal/desktop/navigation_labels.go")
        wnd = self.read("internal/desktop/windows.go")
        self.assertIn("titleWidth, subtitleX = 54, 168, 230", chrome)
        self.assertIn("installWorkspaceHeaderDraw(a, list)", chrome)
        self.assertIn("SetWindowSubclass", header)
        self.assertIn("workspaceListSubclass", header)
        self.assertIn("nmCustomDraw", header)
        self.assertIn("fillRectHeader.Call", header)
        self.assertIn("setTextColor.Call(d.HDC, textColor())", header)
        self.assertIn("cdrfSkipDefault", header)
        self.assertIn("applyApplicationSidebar", sidebar)
        self.assertIn("applicationSidebarWidth", sidebar)
        self.assertIn("applicationContentLeft", sidebar)
        self.assertIn("setSidebarButtonVisual", sidebar)
        self.assertIn("buttonNavActive", sidebar)
        self.assertIn("updateSidebarTransferBadge", sidebar)
        self.assertIn("navigationLabelsForLanguage", navigation)
        self.assertIn('"en": {"Files", "Connections", "Transfer Queue", "Site Manager", "Connection info"}', navigation)
        self.assertIn('"ko": {"파일", "연결", "전송", "서버 관리자", "연결 정보"}', navigation)
        self.assertFalse((ROOT / "internal/desktop/menu_draw_windows.go").exists())
        self.assertFalse((ROOT / "internal/desktop/menu_windows.go").exists())
        self.assertNotIn("a.measureMenuItem(lParam)", wnd)
        self.assertNotIn("a.drawMenuItem(&d)", wnd)

    def test_master_workspace_uses_reference_titles_and_queue_alignment(self):
        source = self.read("internal/desktop/master_workspace_windows.go")
        self.assertIn('setText(a.sectionLocal, "Local Files")', source)
        self.assertIn('setText(a.sectionRemote, "Remote Files")', source)
        self.assertIn('setText(a.sectionTransfers, "Transfer Queue")', source)
        self.assertIn("sendMessageW.Call(a.sectionLocal, wmSetFont, a.font, 1)", source)
        self.assertIn("listY := pathY + 29 + 10", source)
        self.assertIn("a.move(a.clearQueue, contentRight-clearW, queueButtonsY, clearW, 31)", source)
        self.assertNotIn("listY := actionY + 29 + 44", source)

    def test_queue_priority_controls_never_clip_past_master_row(self):
        source = self.read("internal/desktop/queue_priority_windows.go")
        self.assertIn("left := int(retryBottomRight.X) + gap", source)
        self.assertIn("right := int(clearTopLeft.X) - gap", source)
        self.assertIn("showControls(false, controls...)", source)
        self.assertIn("showControls(true, controls...)", source)
        self.assertIn("minButtonWidth := a.scale(44)", source)

    def test_master_more_menu_routes_only_to_real_engine_backed_actions(self):
        source = self.read("internal/desktop/master_workspace_windows.go")
        for marker in (
            'appendItem(masterMoreLocalChoose, "Local: Choose folder")',
            'appendItem(masterMoreLocalFilter, "Local: Filter")',
            'appendItem(masterMoreLocalSearch, "Local: Recursive search")',
            'appendItem(masterMoreRemotePermissions, "Remote: Permissions")',
            'appendItem(masterMoreRemoteEdit, "Remote Edit")',
            'appendItem(masterMoreCompare, "Compare local and remote folders")',
            'appendItem(masterMoreConnectionInfo, "Connection info")',
            'appendItem(masterMoreAbout, "About Ghost FTP")',
            "a.localFilterAction()",
            "a.recursiveSearchCommand(false)",
            "a.remoteFilterAction()",
            "a.recursiveSearchCommand(true)",
            "a.remoteChmodAction()",
            "a.remoteEditAction()",
            "a.directoryComparisonCommand()",
            "a.showDiagnostics()",
            "a.openAbout()",
        ):
            self.assertIn(marker, source)
        self.assertNotIn("Coming Soon", source)

    def test_compact_master_toolbar_preserves_all_action_labels(self):
        master = self.read("internal/desktop/master_workspace_windows.go")
        buttons = self.read("internal/desktop/button_draw_windows.go")
        sidebar = self.read("internal/desktop/sidebar_windows.go")
        self.assertIn("compactToolbar := contentWidth < 860", master)
        self.assertIn("buttonsPerRow = 4", master)
        self.assertIn("toolbarRows = 2", master)
        self.assertIn("row := index / buttonsPerRow", master)
        self.assertIn("case contentWidth >= a.scale(54):", buttons)
        self.assertIn('setText(a.brandSubtitle, "One client.\\r\\nFive platforms.\\r\\nZero friction.")', sidebar)
        self.assertIn("showControls(false, bookmarks, diagnostics, a.aboutBtn)", sidebar)
        self.assertIn("showControls(true, a.brandSubtitle)", sidebar)

    def test_master_files_surface_hides_duplicate_inline_action_chrome(self):
        master = self.read("internal/desktop/master_workspace_windows.go")
        filters = self.read("internal/desktop/file_filter_windows.go")
        search = self.read("internal/desktop/recursive_search_windows.go")
        compare = self.read("internal/desktop/directory_compare_windows.go")
        self.assertIn("showControls(false,", master)
        self.assertIn("a.localUp, a.localChoose, a.localRefresh", master)
        self.assertIn("a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod", master)
        self.assertIn("a.move(a.localPath, leftX, pathY, paneW, 29)", master)
        self.assertIn("a.move(a.remotePath, rightX, pathY, paneW, 29)", master)
        self.assertIn("showControls(false, button)", filters)
        self.assertNotIn("a.move(list, left, listTop, width, bottom-listTop)", filters)
        self.assertIn("showControls(false, filterButton, pane.searchButton, pane.navigateButton, pane.list)", search)
        self.assertIn("showControls(false, state.compareButton, state.openBothButton, state.localList, state.remoteList)", compare)

    def test_disconnected_remote_list_keeps_dark_enabled_surface(self):
        source = self.read("internal/desktop/chrome_windows.go")
        self.assertIn("setControlEnabled(a.remoteList, true)", source)
        self.assertIn("styleWorkspaceList(list)", source)

    def test_startup_settings_skip_redundant_language_rebuild(self):
        source = self.read("internal/desktop/settings_windows.go")
        self.assertIn("previousLanguage := a.languageCode()", source)
        self.assertIn("if a.languageCode() != previousLanguage", source)

    def test_connect_timeout_and_cancel_are_real_ui_behaviors(self):
        source = self.read("internal/desktop/connection_profiles_windows.go")
        self.assertIn("connectionTimeoutDuration(a.settings)", source)
        self.assertIn("a.connectionBusy && !a.connected", source)
        self.assertIn("a.cancelConnectionAttempt()", source)

    def test_x86_ftp_and_sftp_have_secure_sysnative_fallbacks(self):
        ftp = self.read("internal/remote/tools.go")
        sftp = self.read("internal/remote/sftp.go")
        self.assertIn('arch == "386"', ftp)
        self.assertIn('"Sysnative", "curl.exe"', ftp)
        self.assertNotIn('exec.LookPath("curl.exe")', ftp)
        self.assertIn("windowsOpenSSHCandidates", sftp)
        self.assertIn('"Sysnative", "OpenSSH", name', sftp)

    def test_release_prep_version_and_localization_changes_capture_authentic_ui(self):
        workflow = self.read(".github/workflows/ui-screenshots.yml")
        self.assertIn("- 'release-prep/**'", workflow)
        self.assertIn("- 'VERSION'", workflow)
        self.assertIn("- 'internal/i18n/**'", workflow)
        self.assertIn("Capture authentic main, Site Manager, Bookmarks, Settings and About windows", workflow)
        self.assertIn("Ghost-FTP-$version-Portable-x64.exe", workflow)
        evidence = workflow.split("  evidence:", 1)[1]
        self.assertIn("SOURCE_SHA:", evidence)
        self.assertIn("github.event_name == 'pull_request'", evidence)
        self.assertIn("ref: ${{ env.SOURCE_SHA }}", evidence)
        self.assertIn("ghostftp-authentic-ui-verified-bundle", evidence)
        self.assertNotIn("git push", workflow)
        self.assertNotIn("git commit", workflow)
        self.assertNotIn("github-actions[bot]", workflow)
        self.assertNotIn("[skip ci]", workflow)


if __name__ == "__main__":
    unittest.main()
