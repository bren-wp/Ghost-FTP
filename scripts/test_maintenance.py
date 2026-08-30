#!/usr/bin/env python3
from __future__ import annotations

import re
import subprocess
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
        helper = read("android/app/src/main/java/com/byftp/client/model/DocumentName.java")
        activity = read("android/app/src/main/java/com/byftp/client/MainActivity.java")
        tests = read("android/app/src/test/java/com/byftp/client/model/DocumentNameTest.java")
        self.assertIn("providerName != null && !providerName.isBlank()", helper)
        self.assertIn('return "upload.bin";', helper)
        self.assertIn("DocumentName.resolve(providerName, uri.getLastPathSegment())", activity)
        self.assertIn("!cursor.isNull(index)", activity)
        self.assertIn("usesDeterministicFallbackWhenMetadataIsMissing", tests)

    def test_ios_preset_replacement_is_atomic(self) -> None:
        src = read("ios/ByFTP/SessionStore.swift")
        save = between(src, "static func save(_ preset: ConnectionPreset) -> Bool", "static func clear()")
        self.assertIn("SecItemUpdate", save)
        self.assertIn("errSecItemNotFound", save)
        self.assertIn("SecItemAdd", save)
        self.assertNotIn("clear()", save)

    def test_active_product_history_has_no_pre_1_3_byftp_line(self) -> None:
        tracked = subprocess.check_output(["git", "ls-files", "-z"], cwd=ROOT).split(b"\0")
        patterns = (
            re.compile(r"\bByFTP(?:\s+WEB)?\s+v?1\.[0-2]\.\d+\b", re.IGNORECASE),
            re.compile(r"(?m)^##\s+1\.[0-2]\.\d+\b"),
            re.compile(r"\bv1\.[0-2]\.\d+\b", re.IGNORECASE),
            re.compile(r"\bVersion\s+1\.[0-2]\.\d+\b", re.IGNORECASE),
        )
        offenders: list[str] = []
        for raw in tracked:
            if not raw:
                continue
            path = ROOT / raw.decode("utf-8", "strict")
            try:
                text = path.read_text(encoding="utf-8")
            except (UnicodeDecodeError, OSError):
                continue
            if any(pattern.search(text) for pattern in patterns):
                offenders.append(str(path.relative_to(ROOT)))
        self.assertEqual([], offenders, "pre-1.3 active ByFTP version references: " + ", ".join(offenders))

    def test_web_version_and_fail_closed_boundaries(self) -> None:
        web = ROOT / "ByFTP WEB"
        self.assertTrue(web.is_dir(), "ByFTP WEB source directory is required")
        root_version = read("VERSION").strip()
        self.assertEqual(root_version, (web / "VERSION").read_text(encoding="utf-8").strip())

        bootstrap = (web / "app/bootstrap.php").read_text(encoding="utf-8")
        paths = (web / "app/Remote/PathGuard.php").read_text(encoding="utf-8")
        profiles = (web / "app/Storage/ProfileStore.php").read_text(encoding="utf-8")
        host_guard = (web / "app/Security/HostGuard.php").read_text(encoding="utf-8")
        ftp = (web / "app/Remote/FtpClient.php").read_text(encoding="utf-8")
        sftp = (web / "app/Remote/SftpClient.php").read_text(encoding="utf-8")
        sw = (web / "service-worker.js").read_text(encoding="utf-8")
        tests = (web / "tests/unit.php").read_text(encoding="utf-8")

        self.assertIn("BYFTP_ROOT . '/VERSION'", bootstrap)
        self.assertNotRegex(bootstrap, r"const\s+BYFTP_VERSION\s*=\s*['\"]\d")
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
        self.assertIn(f"byftp-static-v{root_version}", sw)
        self.assertIn("request.mode === 'navigate'", sw)
        self.assertIn("preview", sw)
        self.assertIn("WEB_UNIT_TESTS=PASS", tests)


if __name__ == "__main__":
    unittest.main()
