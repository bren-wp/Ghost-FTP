#!/usr/bin/env python3

from __future__ import annotations

import importlib.util
import subprocess
import tempfile
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("audit_release_version_guard.py")
SPEC = importlib.util.spec_from_file_location("release_version_guard", MODULE_PATH)
assert SPEC and SPEC.loader
GUARD = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(GUARD)


class ReleaseVersionGuardTests(unittest.TestCase):
    def make_repo(self) -> tuple[Path, str]:
        root = Path(tempfile.mkdtemp(prefix="byftp-version-guard-"))
        subprocess.run(["git", "init", "-q"], cwd=root, check=True)
        subprocess.run(["git", "config", "user.email", "test@example.invalid"], cwd=root, check=True)
        subprocess.run(["git", "config", "user.name", "ByFTP Test"], cwd=root, check=True)
        (root / "VERSION").write_text("1.0.0\n", encoding="utf-8")
        (root / "internal").mkdir()
        (root / "internal" / "core.go").write_text("package internal\n", encoding="utf-8")
        (root / "scripts").mkdir()
        (root / "scripts" / "audit_only.py").write_text("print('ok')\n", encoding="utf-8")
        subprocess.run(["git", "add", "."], cwd=root, check=True)
        subprocess.run(["git", "commit", "-qm", "base"], cwd=root, check=True)
        base = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
        subprocess.run(["git", "tag", "v1.0.0"], cwd=root, check=True)
        return root, base

    def commit(self, root: Path, message: str) -> str:
        subprocess.run(["git", "add", "."], cwd=root, check=True)
        subprocess.run(["git", "commit", "-qm", message], cwd=root, check=True)
        return subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()

    def test_production_change_after_existing_tag_requires_version_bump(self) -> None:
        root, base = self.make_repo()
        (root / "internal" / "core.go").write_text("package internal\n// hardened\n", encoding="utf-8")
        head = self.commit(root, "runtime change")
        ok, message = GUARD.validate_release_version(base, head, root)
        self.assertFalse(ok)
        self.assertIn("bump VERSION", message)

    def test_version_bump_allows_production_change(self) -> None:
        root, base = self.make_repo()
        (root / "internal" / "core.go").write_text("package internal\n// hardened\n", encoding="utf-8")
        (root / "VERSION").write_text("1.0.1\n", encoding="utf-8")
        head = self.commit(root, "new version")
        ok, message = GUARD.validate_release_version(base, head, root)
        self.assertTrue(ok, message)

    def test_audit_only_change_does_not_force_product_version(self) -> None:
        root, base = self.make_repo()
        (root / "scripts" / "audit_only.py").write_text("print('stronger audit')\n", encoding="utf-8")
        head = self.commit(root, "audit change")
        ok, message = GUARD.validate_release_version(base, head, root)
        self.assertTrue(ok, message)

    def test_unreleased_version_allows_production_work(self) -> None:
        root, base = self.make_repo()
        (root / "VERSION").write_text("2.0.0\n", encoding="utf-8")
        head = self.commit(root, "start unreleased line")
        ok, message = GUARD.validate_release_version(base, head, root)
        self.assertTrue(ok, message)
        self.assertIn("has not been tagged yet", message)

    def test_public_docs_are_release_content(self) -> None:
        self.assertTrue(GUARD.is_production_path("docs/SIGURNOST.md"))
        self.assertTrue(GUARD.is_production_path("README.md"))
        self.assertFalse(GUARD.is_production_path("scripts/audit_privacy.py"))
        self.assertFalse(GUARD.is_production_path(".github/workflows/ci.yml"))


if __name__ == "__main__":
    unittest.main()
