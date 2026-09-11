#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class RemoteDesktopContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_shared_target_validation_is_bounded_and_defaults_to_3389(self) -> None:
        source = self.read("internal/desktop/remote_desktop_target.go")
        for marker in (
            "defaultRemoteDesktopPort = 3389",
            "remoteDesktopTarget(value string)",
            "net.JoinHostPort",
            "port < 1 || port > 65535",
            'strings.ContainsAny(input, "\\x00\\r\\n\\t /\\\\@?#")',
        ):
            self.assertIn(marker, source)

    def test_windows_uses_native_mstsc_without_shell_or_credentials(self) -> None:
        source = self.read("internal/desktop/remote_desktop_windows.go")
        self.assertIn('exec.Command("mstsc.exe", "/v:"+target)', source)
        self.assertIn("PromptDialogWithLabels", source)
        self.assertIn("never stores or forwards the RDP password", source)
        for forbidden in ("cmd.exe", "powershell", "/p:", "password=", "credential"):
            self.assertNotIn(forbidden, source.lower())

    def test_linux_uses_freerdp_without_password_or_certificate_bypass(self) -> None:
        source = self.read("internal/desktop/remote_desktop_linux.go")
        self.assertIn('exec.LookPath("xfreerdp3")', source)
        self.assertIn('exec.LookPath("xfreerdp")', source)
        self.assertIn('exec.Command(client, "/v:"+target)', source)
        for forbidden in ("/p:", "/cert:ignore", "/cert:", "sh -c", "bash -c", "password"):
            self.assertNotIn(forbidden, source.lower())

    def test_android_delegates_to_registered_rdp_app_without_credentials(self) -> None:
        target = self.read("android/app/src/main/java/app/ghostftp/client/RemoteDesktopTarget.java")
        panel = self.read("android/app/src/main/java/app/ghostftp/client/RemoteDesktopPanel.java")
        activity = self.read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        self.assertIn('Uri.parse("rdp://" + authority(raw))', target)
        self.assertIn("DEFAULT_PORT = 3389", target)
        self.assertIn("new Intent(Intent.ACTION_VIEW, uri)", panel)
        self.assertIn("resolveActivity(activity.getPackageManager())", panel)
        self.assertIn("RemoteDesktopPanel.create(this, host)", activity)
        self.assertIn('new String[]{"FTPS", "FTP"}', activity)
        self.assertNotIn('"RDP"', activity.split('new String[]{"FTPS", "FTP"}')[0])
        for forbidden in ("putextra(\"password", "putextra(\"credential", "rdp://user:", "password="):
            self.assertNotIn(forbidden, (target + panel).lower())

    def test_documented_boundary_does_not_claim_built_in_rdp_transport(self) -> None:
        docs = self.read("docs/REMOTE-DESKTOP.md")
        readme = self.read("README.md")
        for marker in (
            "Ghost FTP does not implement the RDP protocol",
            "RDP credentials are not stored by Ghost FTP",
            "mstsc.exe",
            "FreeRDP",
            "rdp://",
        ):
            self.assertIn(marker, docs)
        self.assertIn("Remote Desktop", readme)
        self.assertIn("RDP", readme)


if __name__ == "__main__":
    unittest.main()
