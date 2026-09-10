#!/usr/bin/env python3
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[1]


def source(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


class FileFilterUIContractTests(unittest.TestCase):
    def test_shared_filter_is_non_destructive_io_free_and_unicode_aware(self) -> None:
        text = source("internal/itemlist/filter.go")
        tests = source("internal/itemlist/filter_test.go")
        self.assertIn("func Filter(items []model.Item, query string) []model.Item", text)
        self.assertIn("out := make([]model.Item, 0, len(items))", text)
        self.assertIn("return append(out, items...)", text)
        self.assertIn("MatchesName(item.Name, query)", text)
        self.assertIn("func MatchesName(name, query string) bool", text)
        self.assertIn("containsFold(name, token)", text)
        self.assertIn("unicode.SimpleFold", text)
        self.assertNotIn("strings.ToLower", text)
        self.assertIn("TestFilterHandlesGreekFinalSigmaCaseFold", tests)
        self.assertIn('Name: "ΟΣ.txt"', tests)
        self.assertIn('Filter(source, "ος")', tests)
        for forbidden in ("os.", "net.", "http.", "LocalList", "RemoteList"):
            self.assertNotIn(forbidden, text)

    def test_windows_rows_use_visible_snapshot_not_source_snapshot(self) -> None:
        filter_source = source("internal/desktop/file_filter_windows.go")
        helpers = source("internal/desktop/helpers_windows.go")
        commands = source("internal/desktop/commands_windows.go")
        layout = source("internal/desktop/workspace_layout_windows.go")

        for marker in (
            "localAll",
            "remoteAll",
            "remoteGeneration",
            "itemlist.Filter(state.localAll, state.localQuery)",
            "itemlist.Filter(state.remoteAll, state.remoteQuery)",
            "a.localItems = visible",
            "a.remoteItems = visible",
        ):
            self.assertIn(marker, filter_source)
        self.assertIn("visible := owner.acceptFileFilterSnapshot(list, items)", helpers)
        self.assertIn("owner.sortFileItems(list, visible)", helpers)
        self.assertIn("case idLocalFilter:", commands)
        self.assertIn("case idRemoteFilter:", commands)
        self.assertIn("a.ensureFileFilterControls()", layout)
        self.assertIn("a.layoutFileFilterControls()", layout)

    def test_windows_remote_filter_is_connection_generation_bound(self) -> None:
        text = source("internal/desktop/file_filter_windows.go")
        self.assertIn("state.remoteGeneration = a.connectionGeneration", text)
        self.assertIn("if state.remoteGeneration == a.connectionGeneration", text)
        self.assertIn("a.connected && !a.connectionBusy", text)

    def test_windows_workspace_command_ids_are_unique(self) -> None:
        paths = (
            "internal/desktop/win32_defs_windows.go",
            "internal/desktop/remote_edit_windows.go",
            "internal/desktop/queue_priority_windows.go",
            "internal/desktop/file_filter_windows.go",
            "internal/desktop/recursive_search_windows.go",
        )
        pattern = re.compile(
            r"(?m)^\s*(?:const\s+)?(id[A-Z][A-Za-z0-9_]*)\s*=\s*(\d+)\s*$"
        )
        by_value: dict[int, str] = {}
        by_name: dict[str, int] = {}

        for path in paths:
            for name, raw_value in pattern.findall(source(path)):
                value = int(raw_value)
                previous = by_value.get(value)
                if previous is not None:
                    self.fail(
                        f"duplicate Windows workspace command id {value}: "
                        f"{previous} and {name}"
                    )
                by_value[value] = name
                by_name[name] = value

        self.assertGreaterEqual(len(by_value), 41)
        self.assertEqual(by_name.get("idRemoteEdit"), 309)
        self.assertEqual(by_name.get("idLocalFilter"), 209)
        self.assertEqual(by_name.get("idRemoteFilter"), 310)
        self.assertEqual(by_name.get("idLocalRecursiveSearch"), 210)
        self.assertEqual(by_name.get("idLocalRecursiveSearchNavigate"), 211)
        self.assertEqual(by_name.get("idLocalRecursiveSearchList"), 212)
        self.assertEqual(by_name.get("idRemoteRecursiveSearch"), 311)
        self.assertEqual(by_name.get("idRemoteRecursiveSearchNavigate"), 312)
        self.assertEqual(by_name.get("idRemoteRecursiveSearchList"), 313)

    def test_linux_render_selection_and_refresh_use_visible_slice(self) -> None:
        ui = source("internal/desktop/gui_linux.go")
        filter_source = source("internal/desktop/file_filter_linux.go")
        actions = source("internal/desktop/gui_linux_actions.go")

        self.assertIn("u.renderFileFilterControls()", ui)
        self.assertIn("u.renderItemRows(u.fileFilterListRect(false), u.localItems", ui)
        self.assertIn("u.renderItemRows(u.fileFilterListRect(true), u.remoteItems", ui)
        self.assertIn("u.handleFileFilterMouse(x, y)", ui)
        self.assertIn("u.selectRow(u.fileFilterListRect(false), y, len(u.localItems))", ui)
        self.assertIn("u.selectRow(u.fileFilterListRect(true), y, len(u.remoteItems))", ui)
        self.assertIn("u.acceptLinuxFileFilterSnapshot(false, result.localItems)", ui)
        self.assertIn("u.acceptLinuxFileFilterSnapshot(true, result.remoteItems)", ui)
        self.assertIn("u.clearLinuxRemoteFilterSource()", ui)
        self.assertIn("u.openFileFilterPrompt(false)", ui)

        self.assertIn("state.localAll = source", filter_source)
        self.assertIn("state.remoteAll = source", filter_source)
        self.assertIn("u.localItems = itemlist.Filter(state.localAll, query)", filter_source)
        self.assertIn("u.remoteItems = itemlist.Filter(state.remoteAll, query)", filter_source)
        self.assertIn("linuxPromptLocalFilter", actions)
        self.assertIn("linuxPromptRemoteFilter", actions)
        self.assertIn("isFilter := kind == linuxPromptLocalFilter || kind == linuxPromptRemoteFilter", actions)

    def test_current_folder_ui_does_not_implement_recursive_search(self) -> None:
        combined = "\n".join(
            source(path).lower()
            for path in (
                "internal/desktop/file_filter_windows.go",
                "internal/desktop/filter_words.go",
            )
        )
        self.assertNotIn("recursive search", combined)
        self.assertNotIn("search server recursively", combined)
        shared = source("internal/itemlist/filter.go")
        self.assertIn("Recursive search reuses this helper", shared)
        self.assertNotIn("os.", shared)
        self.assertNotIn("net.", shared)

    def test_active_docs_distinguish_filter_from_bounded_recursive_search(self) -> None:
        roadmap = source("docs/ROADMAP.md").lower()
        testing = source("docs/TESTING.md").lower()
        changelog_lines = [line.strip().lower() for line in source("CHANGELOG.md").splitlines()]

        self.assertIn("non-destructive current-folder filter", roadmap)
        self.assertIn("p0 — bounded recursive local/server search", roadmap)
        self.assertIn("status: implemented in the maintained unreleased source line", roadmap)
        self.assertIn("current-folder filter regression contract", testing)
        self.assertIn("deliberately separate from bounded recursive search", testing)
        self.assertIn("bounded recursive search regression contract", testing)

        current_filter_lines = [line for line in changelog_lines if "current-folder filter" in line]
        recursive_search_lines = [line for line in changelog_lines if "bounded recursive local/server search" in line]
        self.assertTrue(current_filter_lines, "changelog must describe the implemented current-folder filter")
        self.assertTrue(recursive_search_lines, "changelog must describe the implemented bounded recursive search")
        self.assertTrue(
            any("no hidden filesystem/network scan" in line for line in current_filter_lines),
            "implemented filter must remain documented as local to already-loaded entries",
        )


if __name__ == "__main__":
    result = unittest.main(exit=False)
    if not result.result.wasSuccessful():
        raise SystemExit(1)
    print("FILE_FILTER_UI_CONTRACT=PASS")
