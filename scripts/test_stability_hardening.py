#!/usr/bin/env python3
"""Regression tests for cross-platform stability and WEB hardening invariants."""

from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class StabilityHardeningTests(unittest.TestCase):
    def test_web_diagnostics_are_admin_only(self) -> None:
        diagnostics = (ROOT / "ByFTP WEB/diagnostics.php").read_text(encoding="utf-8")
        self.assertIn("Auth::requireAdmin();", diagnostics)
        self.assertNotIn("Auth::requireAuth();", diagnostics)

    def test_empty_locale_compilation_unit_is_removed(self) -> None:
        self.assertFalse((ROOT / "internal/i18n/action_locale_de_fr.go").exists())

    def test_zip_entries_are_materialized_before_remote_mutation(self) -> None:
        source = (ROOT / "ByFTP WEB/app/Operations/RemoteOperations.php").read_text(encoding="utf-8")
        start = source.index("public function extractZip(")
        end = source.index("private function buildZip(", start)
        extract = source[start:end]

        markers = (
            "\\byftp_assert_temp_capacity($bytes);",
            "$stagedEntries = [];",
            "$actualBytes = 0;",
            "// Materialize and validate every file entry before any remote mutation.",
            "$remainingBytes = self::MAX_ARCHIVE_BYTES - $actualBytes;",
            "stream_copy_to_stream($stream, $out, $remainingBytes + 1)",
            "$actualBytes += $copied;",
            "$plan[$index]['local'] = $entryTmp;",
            "// Only a fully validated and materialized archive is allowed to mutate remote state.",
            "$this->uploadAtomic($entryTmp, $remote);",
            "foreach ($stagedEntries as $stagedEntry) @unlink($stagedEntry);",
            "return ['files'=>$files,'bytes'=>$actualBytes];",
        )
        for marker in markers:
            self.assertIn(marker, extract, marker)

        materialize = extract.index("// Materialize and validate every file entry before any remote mutation.")
        stream_copy = extract.index("stream_copy_to_stream($stream, $out, $remainingBytes + 1)")
        execution = extract.index("// Only a fully validated and materialized archive is allowed to mutate remote state.")
        ensure_directory = extract.index("$this->ensureDirectory($remote);", execution)
        upload = extract.index("$this->uploadAtomic($entryTmp, $remote);", execution)
        cleanup = extract.index("foreach ($stagedEntries as $stagedEntry) @unlink($stagedEntry);")

        self.assertLess(materialize, stream_copy)
        self.assertLess(stream_copy, execution)
        self.assertLess(execution, ensure_directory)
        self.assertLess(execution, upload)
        self.assertLess(upload, cleanup)

        pre_execution = extract[:execution]
        self.assertNotIn("$this->ensureDirectory($remote);", pre_execution)
        self.assertNotIn("$this->uploadAtomic($entryTmp, $remote);", pre_execution)


if __name__ == "__main__":
    unittest.main()
