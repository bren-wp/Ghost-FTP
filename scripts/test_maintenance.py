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
        self.assertEqual(version, "0.2.1")
        readme = read("README.md")
        changelog = read("CHANGELOG.md")
        history = read("docs/RELEASE-HISTORY.md")
        self.assertIn(f"Current Ghost FTP version: **{version}**", readme)
        self.assertIn("Development status: **Beta**", readme)
        self.assertIn("0.1.0 Beta → 0.1.1 Beta → 0.2.0 Beta", readme)
        self.assertIn("first stable", readme.lower())
        self.assertIn("**1.0.0**", readme)
        self.assertIn(f"## {version}", changelog)
        self.assertIn("## 2.0.0", changelog)
        self.assertIn("## 1.0.0", changelog)
        self.assertIn("## 2.0.0", history)
        self.assertIn("## 1.0.0", history)
        self.assertIn("ghostftp-vX.Y.Z", readme)
        sections = [
            match.group(1)
            for match in re.finditer(r"^##\s+(\d+\.\d+\.\d+)(?:\s|$)", changelog, re.MULTILINE)
        ]
        for expected in (version, "2.0.0", "1.0.0"):
            self.assertIn(expected, sections)


if __name__ == "__main__":
    unittest.main()
