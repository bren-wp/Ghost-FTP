#!/usr/bin/env python3
from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(rel: str) -> str:
    return (ROOT / rel).read_text(encoding="utf-8")


def between(text: str, start: str, end: str) -> str:
    a = text.index(start)
    b = text.index(end, a)
    return text[a:b]


class MaintenanceRegressionTests(unittest.TestCase):
    def test_windows_remote_names_are_validated_before_use(self) -> None:
        src = read("internal/desktop/files_actions_windows.go")
        mkdir = between(src, "func (a *app) remoteMkdirAction()", "func (a *app) remoteRenameAction()")
        rename = between(src, "func (a *app) remoteRenameAction()", "func (a *app) remoteDeleteAction()")
        self.assertIn("security.ValidateRemoteName(name)", mkdir)
        self.assertIn("security.ValidateRemoteName(name)", rename)
        self.assertNotIn("strings.TrimSpace(name)", mkdir)
        self.assertNotIn("strings.TrimSpace(name)", rename)
        self.assertIn("RemoteMkdir(ctx, base, name)", mkdir)
        self.assertIn("RemoteRename(ctx, base, item.Name, name)", rename)

    def test_linux_terminal_preserves_identity_and_normalizes_optional_key_path(self) -> None:
        src = read("internal/desktop/other.go")
        prompt = between(src, "func prompt(", "func stty(")
        self.assertIn(r'strings.TrimRight(line, "\r\n")', prompt)
        self.assertNotIn("strings.TrimSpace(line)", prompt)
        self.assertIn("cfg.PrivateKeyPath = strings.TrimSpace(keyPath)", src)
        self.assertIn('if cfg.PrivateKeyPath == "" {', src)
        self.assertIn("cfg.Password = password", src)
        self.assertIn("cfg.Passphrase = passphrase", src)

    def test_retired_application_targets_and_release_surfaces_remain_absent(self) -> None:
        for rel in (
            "android",
            "ios",
            "macos",
            "GhostFTP WEB",
            "scripts/package_web.py",
            "scripts/test_package_web.py",
            "scripts/package_nuget.py",
            "scripts/audit_web.py",
        ):
            self.assertFalse((ROOT / rel).exists(), f"retired application/release surface exists: {rel}")

    def test_platform_contract_rejects_retired_target_reintroduction(self) -> None:
        audit = read("scripts/audit_platform_contract.py")
        self.assertIn('RETIRED_ROOTS = ("android/", "ios/", "macos/", "GhostFTP WEB/")', audit)
        self.assertIn("retired application platform/surface is tracked", audit)
        self.assertIn("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX", audit)
        self.assertIn("RETIRED_APPLICATION_PLATFORMS=ANDROID,IOS,MACOS", audit)

    def test_release_workflow_refuses_stale_main_or_tag_rewrite(self) -> None:
        workflow = read(".github/workflows/release.yml")
        self.assertIn("RELEASE_TAG=ghostftp-v$version", workflow)
        self.assertIn("main moved from release commit", workflow)
        self.assertIn("refusing to rewrite it", workflow)
        self.assertIn("gh release create", workflow)
        self.assertLess(workflow.index("main moved from release commit"), workflow.index("gh release create"))

    def test_version_history_and_current_desktop_contract(self) -> None:
        version = read("VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        major = int(version.split(".", 1)[0])
        self.assertGreaterEqual(major, 1, "current production line must be stable")

        readme = read("README.md")
        docs_index = read("docs/README.md")
        changelog = read("CHANGELOG.md")
        history = read("docs/RELEASE-HISTORY.md")
        releases = read("docs/GITHUB-RELEASES.md")

        self.assertIn(f"Current Ghost FTP version: **{version}**", readme)
        self.assertIn("Development status: **Stable**", readme)
        self.assertIn("Release channel: **Stable**", readme)
        self.assertIn("First stable release: **Ghost FTP 1.0.0**", readme)
        self.assertIn(f"**Current Ghost FTP release: {version}**", docs_index)
        self.assertIn("prerelease=false", docs_index)
        self.assertIn(f"Tag: ghostftp-v{version}", releases)
        self.assertIn("Immutable tag rule", releases)
        self.assertIn(f"## {version}", changelog)

        # Current public changelog stays concise, while detailed older engineering
        # provenance remains available in RELEASE-HISTORY.md and Git history.
        self.assertIn("## 0.2.1", changelog)
        self.assertIn("## 0.2.0", changelog)
        self.assertIn("## Historical engineering history", changelog)
        self.assertIn("docs/RELEASE-HISTORY.md", changelog)
        self.assertIn("## 2.0.0", history)
        self.assertIn("## 1.0.0", history)

        sections = [
            match.group(1)
            for match in re.finditer(r"^##\s+(\d+\.\d+\.\d+)(?:\s|$)", changelog, re.MULTILINE)
        ]
        for expected in (version, "0.2.1", "0.2.0"):
            self.assertIn(expected, sections)


if __name__ == "__main__":
    unittest.main()
