#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
FTP = ROOT / "android/app/src/main/java/app/ghostftp/client/FtpSession.java"


class AndroidPassiveDataContractTests(unittest.TestCase):
    def test_passive_data_setup_failure_closes_session(self) -> None:
        ftp = FTP.read_text(encoding="utf-8")
        start = ftp.index("private Socket openPassiveDataSocket()")
        end = ftp.index("private SSLSocket wrapTls(", start)
        passive = ftp[start:end]

        for marker in (
            'command("EPSV")',
            'command("PASV")',
            "parseEpsvPort(epsv.message)",
            "parsePasvPort(pasv.message)",
            "plain.connect(new InetSocketAddress(host, dataPort), CONNECT_TIMEOUT_MS);",
            "SSLSocket tls = wrapTls(plain);",
            "catch (IOException e)",
            "hardClose();",
            'throw new IOException("Passive data connection setup failed; the FTP session was closed.", e);',
        ):
            self.assertIn(marker, passive)

        catch_pos = passive.index("catch (IOException e)")
        hard_close_pos = passive.index("hardClose();", catch_pos)
        throw_pos = passive.index("Passive data connection setup failed", hard_close_pos)
        self.assertLess(catch_pos, hard_close_pos)
        self.assertLess(hard_close_pos, throw_pos)

    def test_invalid_epsv_port_is_checked_io_failure(self) -> None:
        ftp = FTP.read_text(encoding="utf-8")
        start = ftp.index("private static int parseEpsvPort(")
        end = ftp.index("private static int parsePasvPort(", start)
        epsv = ftp[start:end]

        self.assertIn("if (payload.isEmpty())", epsv)
        self.assertIn("catch (NumberFormatException e)", epsv)
        self.assertIn('throw new IOException("Invalid EPSV port.", e);', epsv)
        self.assertNotIn("return requirePort(Integer.parseInt(parts[3]));\n    }", epsv)

    def test_ftps_data_channel_keeps_strict_hostname_verification(self) -> None:
        ftp = FTP.read_text(encoding="utf-8")
        self.assertIn('parameters.setEndpointIdentificationAlgorithm("HTTPS")', ftp)
        self.assertIn("tls.startHandshake();", ftp)
        for forbidden in ("X509TrustManager", "HostnameVerifier", "TrustManager[]"):
            self.assertNotIn(forbidden, ftp)


if __name__ == "__main__":
    unittest.main()
