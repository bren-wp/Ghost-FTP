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
        self.assertIn('"ftp-skip-pasv-ip"', ftp)
        self.assertIn('{Value: "ftp", Label: "FTP", Port: "21"}', protocols)
        self.assertIn('{Value: "ftps", Label: "FTPS (eksplicitni)", Port: "21"}', protocols)
        self.assertIn('{Value: "ftpsi", Label: "FTPS (implicitni)", Port: "990"}', protocols)


if __name__ == "__main__":
    unittest.main()
