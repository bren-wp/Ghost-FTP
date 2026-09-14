#!/usr/bin/env python3
from pathlib import Path
import shutil
import subprocess
import sys
import unittest

ROOT = Path(__file__).resolve().parents[1]


class WebContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_web_site_is_self_contained_and_branded(self) -> None:
        index = self.read("web/index.html")
        self.assertIn("Ghost FTP 0.0.6", index)
        self.assertIn("assets/images/ghost-ftp-main-workspace.png", index)
        self.assertNotIn("../docs/", index)
        self.assertNotRegex(index.lower(), r"by[\s_-]?ftp")

    def test_web_ftp_is_truthful_about_transport_boundary(self) -> None:
        readme = self.read("web/ftp/README.md")
        self.assertIn("browser cannot open raw FTP/FTPS/SFTP TCP sockets directly", readme)
        self.assertIn("ephemeral server-assisted transport", readme)
        self.assertIn("does not persist them", readme)

    def test_web_client_has_no_persistent_credential_storage(self) -> None:
        js = self.read("web/ftp/assets/app.js")
        self.assertNotIn("localStorage", js)
        self.assertNotIn("indexedDB", js)
        self.assertNotIn("document.cookie", js)

    def test_ftps_and_sftp_identity_fail_closed(self) -> None:
        ftp = self.read("web/ftp/lib/CurlFtpTransport.php")
        sftp = self.read("web/ftp/lib/SftpTransport.php")
        self.assertIn("CURLOPT_SSL_VERIFYPEER,true", ftp)
        self.assertIn("CURLOPT_SSL_VERIFYHOST,2", ftp)
        self.assertLess(sftp.index("ssh2_fingerprint"), sftp.index("ssh2_auth_password"))

    def test_download_cleanup_and_json_encoding_are_fail_safe(self) -> None:
        api = self.read("web/ftp/api.php")
        self.assertIn("JSON_INVALID_UTF8_SUBSTITUTE", api)
        download = api[api.index("case 'download':"):api.index("default:", api.index("case 'download':"))]
        self.assertIn("readfile($tmp)", download)
        self.assertIn("@unlink($tmp)", download)
        self.assertIn("exit;", download)
        self.assertLess(download.index("readfile($tmp)"), download.index("@unlink($tmp)"))
        self.assertLess(download.index("@unlink($tmp)"), download.index("exit;"))

    def test_ftp_listing_falls_back_and_supports_dos_iis_format(self) -> None:
        ftp = self.read("web/ftp/lib/CurlFtpTransport.php")
        listing = ftp[ftp.index("public function list"):ftp.index("public function mkdir")]
        self.assertIn("'MLSD'", listing)
        self.assertIn("catch (RuntimeException)", listing)
        self.assertIn("'LIST'", listing)
        self.assertLess(listing.index("'MLSD'"), listing.index("'LIST'"))
        self.assertIn("<DIR>", ftp)
        self.assertIn("AM|PM", ftp)

    def test_mobile_navigation_control_exists_on_every_marketing_page(self) -> None:
        for rel in (
            "web/index.html",
            "web/download.html",
            "web/security.html",
            "web/privacy.html",
            "web/legal.html",
        ):
            text = self.read(rel)
            self.assertIn("data-menu", text, rel)
            self.assertIn('aria-expanded="false"', text, rel)

    def test_full_web_audit_when_php_is_available(self) -> None:
        if shutil.which("php") is None:
            self.skipTest("PHP CLI not available on this runner")
        result = subprocess.run(
            [sys.executable, str(ROOT / "scripts/check_web_contract.py")],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
