from __future__ import annotations

import unittest

import audit_repository

CURRENT_RELEASE_PREFIX = b"Current " + b"release: "


class RepositoryAuditTests(unittest.TestCase):
    def test_rejects_case_insensitive_path_collision(self) -> None:
        seen: dict[str, str] = {}
        self.assertEqual([], audit_repository.validate_path("docs/Readme.md", "100644", seen))
        errors = audit_repository.validate_path("docs/README.md", "100644", seen)
        self.assertTrue(any("case-insensitive path collision" in item for item in errors))

    def test_rejects_generated_and_reserved_paths(self) -> None:
        errors = audit_repository.validate_path("dist/ByFTP.exe", "100644", {})
        self.assertTrue(any("generated/cache path" in item for item in errors))
        errors = audit_repository.validate_path("docs/CON.txt", "100644", {})
        self.assertTrue(any("Windows-reserved" in item for item in errors))

    def test_rejects_symlink_mode(self) -> None:
        errors = audit_repository.validate_path("docs/link", "120000", {})
        self.assertTrue(any("tracked symlink" in item for item in errors))

    def test_text_rejects_bom_trailing_whitespace_and_missing_newline(self) -> None:
        data = b"\xef\xbb\xbf" + CURRENT_RELEASE_PREFIX + b"9.9.9  "
        errors = audit_repository.validate_text("README.md", data, "9.9.9")
        self.assertTrue(any("UTF-8 BOM" in item for item in errors))
        self.assertTrue(any("trailing whitespace" in item for item in errors))
        self.assertTrue(any("missing final newline" in item for item in errors))

    def test_text_rejects_stale_current_release(self) -> None:
        errors = audit_repository.validate_text(
            "README.md", CURRENT_RELEASE_PREFIX + b"9.9.8\n", "9.9.9"
        )
        self.assertTrue(any("stale current-release" in item for item in errors))

    def test_text_accepts_current_release(self) -> None:
        self.assertEqual(
            [],
            audit_repository.validate_text(
                "README.md", CURRENT_RELEASE_PREFIX + b"9.9.9\n", "9.9.9"
            ),
        )


if __name__ == "__main__":
    unittest.main()
