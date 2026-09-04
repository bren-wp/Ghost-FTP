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

    def test_unix_terminal_preserves_raw_identity_and_key_path(self) -> None:
        src = read("internal/desktop/other.go")
        prompt = between(src, "func prompt(", "func stty(")
        self.assertIn('strings.TrimRight(line, "\\r\\n")', prompt)
        self.assertNotIn("strings.TrimSpace(line)", prompt)
        self.assertIn("cfg.PrivateKeyPath = keyPath", src)
        self.assertNotIn("cfg.PrivateKeyPath = strings.TrimSpace(keyPath)", src)
        tests = read("internal/desktop/other_input_test.go")
        self.assertIn("TestPromptPreservesEdgeWhitespace", tests)
        self.assertIn("TestPromptUsesFallbackOnlyForEmptyLine", tests)

    def test_android_document_provider_name_is_nullable_safe(self) -> None:
        helper = read("android/app/src/main/java/com/ghostftp/client/model/DocumentName.java")
        activity = read("android/app/src/main/java/com/ghostftp/client/MainActivity.java")
        tests = read("android/app/src/test/java/com/ghostftp/client/model/DocumentNameTest.java")
        self.assertIn("providerName != null && !providerName.isBlank()", helper)
        self.assertIn('return "upload.bin";', helper)
        self.assertIn("DocumentName.resolve(providerName, uri.getLastPathSegment())", activity)
        self.assertIn("!cursor.isNull(index)", activity)
        self.assertIn("usesDeterministicFallbackWhenMetadataIsMissing", tests)

    def test_ios_preset_replacement_is_atomic(self) -> None:
        src = read("ios/GhostFTP/SessionStore.swift")
        save = between(src, "static func save(_ preset: ConnectionPreset) -> Bool", "static func clear()")
        self.assertIn("SecItemUpdate", save)
        self.assertIn("errSecItemNotFound", save)
        self.assertIn("SecItemAdd", save)
        self.assertNotIn("clear()", save)

    def test_release_workflow_refuses_stale_main_or_tag_rewrite(self) -> None:
        workflow = read(".github/workflows/release.yml")
        self.assertIn("RELEASE_TAG=ghostftp-v$version", workflow)
        self.assertIn("main moved from release commit", workflow)
        self.assertIn("refusing to rewrite it", workflow)
        self.assertIn("gh release create", workflow)
        guard = workflow.index("main moved from release commit")
        create = workflow.index("gh release create")
        self.assertLess(guard, create)

    def test_ghostftp_version_line_starts_at_1_0_0_and_tracks_current_version(self) -> None:
        version = read("VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        self.assertEqual(version, read("GhostFTP WEB/VERSION").strip())

        readme = read("README.md")
        changelog = read("CHANGELOG.md")
        self.assertIn(f"Current Ghost FTP version: **{version}**", readme)
        self.assertIn("Ghost FTP starts at **1.0.0**", readme)
        self.assertIn("ghostftp-v1.0.0", readme)
        self.assertIn(f"## {version}", changelog)
        self.assertIn("## 1.0.0", changelog)

        version_sections = [
            match.group(1)
            for match in re.finditer(r"^##\s+(\d+\.\d+\.\d+)(?:\s|$)", changelog, re.MULTILINE)
        ]
        self.assertIn("1.0.0", version_sections)
        self.assertIn(version, version_sections)

    def test_web_version_brand_and_fail_closed_boundaries(self) -> None:
        web = ROOT / "GhostFTP WEB"
        self.assertTrue(web.is_dir(), "GhostFTP web source directory is required")
        root_version = read("VERSION").strip()
        self.assertEqual(root_version, (web / "VERSION").read_text(encoding="utf-8").strip())

        composer = (web / "composer.json").read_text(encoding="utf-8")
        bootstrap = (web / "app/bootstrap.php").read_text(encoding="utf-8")
        paths = (web / "app/Remote/PathGuard.php").read_text(encoding="utf-8")
        profiles = (web / "app/Storage/ProfileStore.php").read_text(encoding="utf-8")
        host_guard = (web / "app/Security/HostGuard.php").read_text(encoding="utf-8")
        ftp = (web / "app/Remote/FtpClient.php").read_text(encoding="utf-8")
        sftp = (web / "app/Remote/SftpClient.php").read_text(encoding="utf-8")
        sw = (web / "service-worker.js").read_text(encoding="utf-8")
        tests = (web / "tests/unit.php").read_text(encoding="utf-8")

        self.assertIn('"name": "brendigo/ghost-ftp-web"', composer)
        self.assertIn(f'"version": "{root_version}"', composer)
        self.assertIn("GhostFTP_ROOT . '/VERSION'", bootstrap)
        self.assertNotRegex(bootstrap, r"const\s+GhostFTP_VERSION\s*=\s*['\"]\d")
        self.assertIn("str_contains($path, '\\\\')", paths)
        self.assertIn("str_contains($path, '//')", paths)
        self.assertIn("$part === '.' || $part === '..'", paths)
        self.assertIn("preg_match('/^[0-9]{1,5}$/', $rawPort)", profiles)
        self.assertIn("$rawHost !== trim($rawHost)", profiles)
        self.assertNotIn("trim((string)($input['host']", profiles)
        self.assertIn("$host !== trim($host)", host_guard)
        self.assertIn("FILTER_FLAG_NO_PRIV_RANGE", host_guard)
        self.assertIn("$this->profile['password'] = '';", ftp)
        self.assertIn("$this->profile['private_key'] = '';", sftp)
        self.assertIn(f"ghostftp-static-v{root_version}", sw)
        self.assertIn("key.startsWith('GhostFTP-static-')", sw)
        self.assertIn("request.mode === 'navigate'", sw)
        self.assertIn("preview", sw)
        self.assertIn("WEB_UNIT_TESTS=PASS", tests)


if __name__ == "__main__":
    unittest.main()
