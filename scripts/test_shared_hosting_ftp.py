#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class SharedHostingFTPTests(unittest.TestCase):
    def test_ftp_adapter_keeps_home_relative_control_commands(self) -> None:
        text = (ROOT / "internal/remote/curl_ftp.go").read_text(encoding="utf-8")
        for marker in (
            '"MKD "+ftpCommandPath(',
            '"RNFR "+ftpCommandPath(',
            '"RNTO "+ftpCommandPath(',
            '"DELE "+ftpCommandPath(',
            '"RMD "+ftpCommandPath(',
            '"SITE CHMOD "+mode+" "+ftpCommandPath(',
        ):
            self.assertIn(marker, text)
        self.assertIn('strings.TrimPrefix(strings.ReplaceAll(p, "\\\\", "/"), "/")', text)

    def test_ftp_urls_cannot_switch_to_server_absolute_namespace(self) -> None:
        text = (ROOT / "internal/remote/curl_ftp.go").read_text(encoding="utf-8")
        block = text[text.index("func ftpURLPath"):text.index("func (c *CurlFTP) baseURL")]
        self.assertIn('strings.ReplaceAll(p, "\\\\", "/")', block)
        self.assertIn('"/" + strings.TrimLeft(p, "/")', block)
        base = text[text.index("func (c *CurlFTP) baseURL"):text.index("func (c *CurlFTP) configFor")]
        self.assertIn("escapeURLPath(ftpURLPath(p))", base)

    def test_quote_is_control_channel_only(self) -> None:
        text = (ROOT / "internal/remote/curl_ftp.go").read_text(encoding="utf-8")
        quote = text[text.index("func (c *CurlFTP) quote"):text.index("func (c *CurlFTP) Mkdir")]
        self.assertIn('"no-body"', quote)
        self.assertIn('c.baseURL("/")', quote)

    def test_mlsd_fallback_is_cached_for_session(self) -> None:
        text = (ROOT / "internal/remote/curl_ftp.go").read_text(encoding="utf-8")
        block = text[text.index("func (c *CurlFTP) List"):text.index("func ftpCommandPath")]
        self.assertIn("mlsdFallback := false", block)
        self.assertIn("if mlsdFallback {", block)
        self.assertIn("c.mlsdState.Store(-1)", block)

    def test_passive_nat_and_shared_hosting_ports_remain_supported(self) -> None:
        ftp = (ROOT / "internal/remote/curl_ftp.go").read_text(encoding="utf-8")
        protocols = (ROOT / "internal/desktop/protocols_windows.go").read_text(encoding="utf-8")
        ui = (ROOT / "internal/desktop/ui_windows.go").read_text(encoding="utf-8")
        self.assertIn('"ftp-skip-pasv-ip"', ftp)
        for marker in (
            '{Value: "ftp", Port: "21"}',
            '{Value: "ftps", Port: "21"}',
            '{Value: "sftp", Port: "22"}',
            '{Value: "ftpsi", Port: "990"}',
        ):
            self.assertIn(marker, protocols)
        # Protocol display labels are intentionally localized at render time;
        # the protocol model must not drift back to hard-coded language text.
        self.assertNotIn("Label string", protocols)
        self.assertIn("protocolLabel(a.languageCode(), spec.Value)", ui)

    def test_windows_ui_explains_shared_hosting_credentials(self) -> None:
        ui = (ROOT / "internal/desktop/ui_windows.go").read_text(encoding="utf-8")
        catalogs = (ROOT / "internal/i18n/catalogs.go").read_text(encoding="utf-8")
        for marker in (
            'a.tr("app.subtitle")',
            'cue(a.host, a.tr("cue.host"))',
            'cue(a.user, a.tr("cue.user"))',
            'cue(a.pass, a.tr("cue.password"))',
        ):
            self.assertIn(marker, ui)
        for marker in (
            '"app.subtitle":    "FTP • FTPS • SFTP  ·  Fast hosting management"',
            '"cue.host":           "FTP/SFTP server, e.g. ftp.example.com"',
            '"cue.user": "Username, may be user@example.com"',
        ):
            self.assertIn(marker, catalogs)
        # Preserve usable horizontal space for full shared-hosting usernames.
        self.assertIn("a.move(a.user, x, y, 220, rowH)", ui)
        self.assertIn("limitEdit(a.user, 1024)", ui)

    def test_marketing_docs_include_shared_hosting_path(self) -> None:
        readme = (ROOT / "README.md").read_text(encoding="utf-8")
        guide = (ROOT / "docs/SHARED-HOSTING.md").read_text(encoding="utf-8")
        docs = (ROOT / "docs/README.md").read_text(encoding="utf-8")
        for marker in (
            "## Shared-hosting workflow",
            "public_html",
            "[Shared hosting](docs/SHARED-HOSTING.md)",
        ):
            self.assertIn(marker, readme)
        for marker in (
            "# Shared-hosting compatibility",
            "account@domain",
            "public_html",
            "MLSD to LIST",
            "Passive connections",
        ):
            self.assertIn(marker, guide)
        self.assertIn("SHARED-HOSTING.md", docs)


if __name__ == "__main__":
    unittest.main()
