#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
JAVA = ROOT / "android/app/src/main/java/app/ghostftp/client"


class AndroidTransferProgressContractTests(unittest.TestCase):
    def read(self, name: str) -> str:
        return (JAVA / name).read_text(encoding="utf-8")

    def test_progress_streams_count_real_io_bytes(self) -> None:
        streams = self.read("ProgressStreams.java")
        self.assertIn("extends FilterInputStream", streams)
        self.assertIn("extends FilterOutputStream", streams)
        self.assertIn("int read = super.read(buffer, offset, length);", streams)
        self.assertIn("if (read > 0) publish(read);", streams)
        self.assertIn("out.write(buffer, offset, length);", streams)
        self.assertIn("if (length > 0) publish(length);", streams)
        self.assertIn("transferred += bytes;", streams)
        self.assertNotIn("Thread.sleep", streams)
        self.assertNotIn("Timer", streams)

    def test_runtime_uses_known_entry_size_without_inventing_totals(self) -> None:
        activity = self.read("MainActivity.java")
        progress = self.read("TransferProgress.java")
        self.assertIn('transferProgress(entry.size, "Uploading", current, transferToken)', activity)
        self.assertIn('transferProgress(entry.size, "Downloading", current, transferToken)', activity)
        self.assertIn("this.totalBytes = totalBytes > 0L ? totalBytes : -1L;", progress)
        self.assertIn("boolean trustworthyTotal = totalBytes > 0L && transferredBytes <= totalBytes;", progress)
        self.assertIn("if (trustworthyTotal)", progress)
        self.assertIn("if (trustworthyTotal && transferredBytes < totalBytes)", progress)

    def test_progress_wraps_actual_upload_and_download_streams(self) -> None:
        activity = self.read("MainActivity.java")
        upload_start = activity.index("private void uploadSelected()")
        download_start = activity.index("private void downloadSelected()", upload_start)
        helper_start = activity.index("private TransferProgress transferProgress", download_start)
        upload = activity[upload_start:download_start]
        download = activity[download_start:helper_start]
        self.assertIn("ProgressStreams.input(source, progress::onTransferred)", upload)
        self.assertIn("current.upload(FtpSession.joinRemote(remoteBase, entry.name), in);", upload)
        self.assertLess(upload.index("ProgressStreams.input"), upload.index("current.upload("))
        self.assertIn("ProgressStreams.output(destination, progress::onTransferred)", download)
        self.assertIn("current.download(FtpSession.joinRemote(remoteBase, entry.name), out);", download)
        self.assertLess(download.index("ProgressStreams.output"), download.index("current.download("))

    def test_progress_ui_is_guarded_by_transfer_generation_and_session(self) -> None:
        activity = self.read("MainActivity.java")
        helper_start = activity.index("private TransferProgress transferProgress")
        helper_end = activity.index("private long beginTransfer", helper_start)
        helper = activity[helper_start:helper_end]
        for marker in (
            "!transferActive",
            "transferCancelled(token)",
            "session != current",
            "!current.isConnected()",
            "setStatus(text);",
        ):
            self.assertIn(marker, helper)

    def test_progress_never_replaces_final_commit_semantics(self) -> None:
        activity = self.read("MainActivity.java")
        upload_start = activity.index("private void uploadSelected()")
        download_start = activity.index("private void downloadSelected()", upload_start)
        helper_start = activity.index("private TransferProgress transferProgress", download_start)
        upload = activity[upload_start:download_start]
        download = activity[download_start:helper_start]
        self.assertLess(upload.index("current.upload("), upload.index('"Upload completed: "'))
        self.assertLess(download.index("current.download("), download.index("DocumentsContract.renameDocument"))
        self.assertLess(download.index("queryDocumentDisplayName(committed)"), download.index('"Download completed and committed: "'))

    def test_progress_has_no_network_or_telemetry_side_channel(self) -> None:
        combined = (self.read("TransferProgress.java") + self.read("ProgressStreams.java")).lower()
        for forbidden in (
            "http://",
            "https://",
            "socket(",
            "analytics",
            "telemetry",
            "firebase",
            "crashlytics",
        ):
            self.assertNotIn(forbidden, combined)


if __name__ == "__main__":
    unittest.main()
