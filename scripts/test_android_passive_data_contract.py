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

    def test_epsv_parser_is_strict_and_checked(self) -> None:
        ftp = FTP.read_text(encoding="utf-8")
        start = ftp.index("static int parseEpsvPort(")
        end = ftp.index("static int parsePasvPort(", start)
        epsv = ftp[start:end]

        for marker in (
            "payload.length() < 5",
            "delimiter < 33 || delimiter > 126",
            "payload.charAt(1) != delimiter",
            "payload.charAt(2) != delimiter",
            "payload.charAt(payload.length() - 1) != delimiter",
            "String portText = payload.substring(3, payload.length() - 1);",
            "if (ch < '0' || ch > '9')",
            "return requirePort(Integer.parseInt(portText));",
            "catch (IllegalArgumentException e)",
            'throw new IOException("Invalid EPSV port.", e);',
        ):
            self.assertIn(marker, epsv)

    def test_pasv_parser_requires_six_byte_values(self) -> None:
        ftp = FTP.read_text(encoding="utf-8")
        start = ftp.index("static int parsePasvPort(")
        end = ftp.index("static String joinRemote(", start)
        pasv = ftp[start:end]

        for marker in (
            'split(",", -1)',
            "values.length != 6",
            "int[] octets = new int[6];",
            "octets[i] = Integer.parseInt(value);",
            "octets[i] < 0 || octets[i] > 255",
            "int dataPort = octets[4] * 256 + octets[5];",
            "return requirePort(dataPort);",
            'throw new IOException("Invalid PASV port.", e);',
        ):
            self.assertIn(marker, pasv)

    def test_ftps_data_channel_keeps_strict_hostname_verification(self) -> None:
        ftp = FTP.read_text(encoding="utf-8")
        self.assertIn('parameters.setEndpointIdentificationAlgorithm("HTTPS")', ftp)
        self.assertIn("tls.startHandshake();", ftp)
        for forbidden in ("X509TrustManager", "HostnameVerifier", "TrustManager[]"):
            self.assertNotIn(forbidden, ftp)


if __name__ == "__main__":
    unittest.main()
