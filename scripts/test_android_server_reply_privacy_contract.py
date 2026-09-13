#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
FTP = ROOT / "android/app/src/main/java/app/ghostftp/client/FtpSession.java"


class AndroidServerReplyPrivacyContractTests(unittest.TestCase):
    def test_server_reply_bodies_are_not_embedded_in_user_visible_errors(self) -> None:
        source = FTP.read_text(encoding="utf-8")

        forbidden = (
            '"MLSD failed: " + start.message',
            '"Upload rejected: " + start.message',
            '"Server does not support safe staged upload commit: " + renameFrom.message',
            '"Server rejected final staged upload commit: " + renameTo.message',
            '"Download rejected: " + start.message',
            '"FTP server rejected operation: " + reply.message',
            '"Invalid FTP response: " + line',
        )
        for marker in forbidden:
            self.assertNotIn(marker, source)

    def test_rejections_keep_numeric_response_code_without_raw_reply_text(self) -> None:
        source = FTP.read_text(encoding="utf-8")

        for marker in (
            '"Directory listing failed (server response code " + start.code + ")."',
            '"Upload rejected (server response code " + start.code + ")."',
            '"Server does not support safe staged upload commit (server response code " + renameFrom.code + ")."',
            '"Server rejected final staged upload commit (server response code " + renameTo.code + ")."',
            '"Download rejected (server response code " + start.code + ")."',
            '"FTP server rejected operation (server response code " + reply.code + ")."',
            'throw new IOException("Invalid FTP response.", e);',
        ):
            self.assertIn(marker, source)


if __name__ == "__main__":
    unittest.main()
