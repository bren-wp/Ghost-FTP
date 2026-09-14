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

    def test_active_public_and_development_surfaces(self) -> None:
        self.assertTrue((ROOT / "android").is_dir())
        self.assertTrue((ROOT / ".github/workflows/android-apk.yml").is_file())
        self.assertTrue((ROOT / "macos").is_dir())
        self.assertTrue((ROOT / ".github/workflows/macos-app.yml").is_file())
        self.assertTrue((ROOT / "ekstenzije").is_dir())
        self.assertTrue((ROOT / ".github/workflows/browser-extensions.yml").is_file())
        for rel in (
            "ios",
            "GhostFTP WEB",
            "scripts/package_web.py",
            "scripts/test_package_web.py",
            "scripts/package_nuget.py",
            "scripts/audit_web.py",
        ):
            self.assertFalse((ROOT / rel).exists(), f"retired application/release surface exists: {rel}")

        release = read(".github/workflows/release.yml").lower()
        self.assertNotIn("macos/", release)
        self.assertNotIn("runs-on: macos", release)
        self.assertIn("android/", release)
        self.assertIn("build_browser_extensions.py", release)
        self.assertIn("for browser in chrome edge firefox", release)

    def test_platform_contract_rejects_only_retired_target_reintroduction(self) -> None:
        audit = read("scripts/audit_platform_contract.py")
        self.assertIn('RETIRED_ROOTS = ("ios/", "GhostFTP WEB/")', audit)
        self.assertIn("ANDROID_REQUIRED", audit)
        self.assertIn("MACOS_REQUIRED", audit)
        self.assertIn("BROWSER_REQUIRED", audit)
        self.assertIn('fail(f"active {label} source contract is incomplete:', audit)
        self.assertIn("retired application platform/surface is tracked", audit)
        self.assertIn("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER", audit)
        self.assertIn("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS", audit)
        self.assertIn("ANDROID_PUBLIC_RELEASE_ARTIFACT=YES_PRODUCTION_SIGNED", audit)
        self.assertIn("BROWSER_PUBLIC_RELEASE_PACKAGES=CHROME,EDGE,FIREFOX", audit)
        self.assertIn("MACOS_PUBLIC_RELEASE_ARTIFACT=NO", audit)
        self.assertIn("RETIRED_APPLICATION_PLATFORMS=IOS", audit)

    def test_release_workflow_refuses_stale_main_or_tag_rewrite(self) -> None:
        workflow = read(".github/workflows/release.yml")
        self.assertIn("RELEASE_TAG=ghostftp-v$version", workflow)
        self.assertIn("main moved from release commit", workflow)
        self.assertIn("refusing to rewrite an existing release tag", workflow)
        self.assertIn("release already exists; refusing to rewrite published assets", workflow)
        self.assertNotIn("gh release upload", workflow)
        self.assertNotIn("--clobber", workflow)
        self.assertIn("gh release create", workflow)
        self.assertLess(workflow.index("main moved from release commit"), workflow.index("gh release create"))

    def test_current_release_requires_protected_windows_and_android_signing(self) -> None:
        workflow = read(".github/workflows/release.yml")
        verifier = read("scripts/verify_release.py")
        self.assertNotIn("state=unsigned", workflow)
        self.assertIn("state=signed", workflow)
        self.assertIn("Official Ghost FTP publication requires GHOSTFTP_SIGNING_PFX_BASE64.", workflow)
        self.assertIn("Official Ghost FTP publication requires GHOSTFTP_SIGNING_PASSWORD.", workflow)
        self.assertIn("Get-AuthenticodeSignature -FilePath $path", workflow)
        self.assertIn("test \"$WINDOWS_SIGNING_STATE\" = 'signed'", workflow)
        self.assertNotIn("New-DevCodeSigningCertificate.ps1", workflow)
        self.assertIn("public Windows release artifacts must be Authenticode signed", verifier)
        self.assertIn("GHOSTFTP_ANDROID_KEYSTORE_BASE64", workflow)
        self.assertIn("GHOSTFTP_ANDROID_CERT_SHA256", workflow)
        self.assertNotIn("GHOSTFTP_ANDROID_SIGNER_SHA256", workflow)
        self.assertIn("apksigner", workflow)

    def test_version_history_and_current_release_contract(self) -> None:
        version = read("VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        parts = tuple(int(part) for part in version.split("."))
        self.assertGreaterEqual(parts, (0, 0, 1))
        self.assertNotEqual(parts, (0, 0, 0))

        readme = read("README.md")
        docs_index = read("docs/README.md")
        changelog = read("CHANGELOG.md")
        history = read("docs/RELEASE-HISTORY.md")
        releases = read("docs/GITHUB-RELEASES.md")

        self.assertIn(f"Current Ghost FTP version: **{version}**", readme)
        self.assertIn("Development status: **Active**", readme)
        self.assertIn("Release channel: **Current**", readme)
        self.assertIn(f"**Current Ghost FTP release: {version}**", docs_index)
        self.assertIn("PRERELEASE=false", docs_index)
        self.assertIn(f"Tag: ghostftp-v{version}", releases)
        self.assertIn("Exact-head and immutable-current transaction", releases)
        self.assertIn(f"## {version}", changelog)
        self.assertIn(f"## {version}", history)
        self.assertIn("latest public Ghost FTP version", history)
        self.assertIn("release-retention.yml", history)
        self.assertNotRegex(changelog, r"(?m)^##\s+1\.\d+\.\d+")
        self.assertNotRegex(history, r"(?m)^##\s+1\.\d+\.\d+")

        sections = [m.group(1) for m in re.finditer(r"^##\s+(\d+\.\d+\.\d+)(?:\s|$)", changelog, re.MULTILINE)]
        self.assertTrue(sections)
        self.assertEqual(sections[0], version)
        self.assertEqual(len(sections), len(set(sections)))
        section_parts = [tuple(int(part) for part in value.split(".")) for value in sections]
        self.assertEqual(section_parts, sorted(section_parts, reverse=True))
        self.assertTrue(all(value <= parts for value in section_parts))


if __name__ == "__main__":
    unittest.main()
