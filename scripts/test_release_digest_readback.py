from __future__ import annotations

import hashlib
import json
import tempfile
import unittest
from pathlib import Path

from verify_release_digest_readback import expected_release_names, verify_release


COMMIT = "0123456789abcdef0123456789abcdef01234567"
VERSION = "9.8.7"
TAG = f"ghostftp-v{VERSION}"


class ReleaseDigestReadbackTests(unittest.TestCase):
    def make_fixture(self) -> tuple[tempfile.TemporaryDirectory[str], Path, Path]:
        temp = tempfile.TemporaryDirectory()
        root = Path(temp.name)
        bundle = root / "release"
        bundle.mkdir()

        metadata = "\n".join(
            [
                "BRAND=Ghost FTP",
                f"VERSION={VERSION}",
                f"RELEASE_TAG={TAG}",
                f"COMMIT={COMMIT}",
                "PUBLIC_RELEASE_FILES=17",
                "TELEMETRY=disabled",
                "",
            ]
        )
        (bundle / "BUILD-METADATA.txt").write_text(metadata, encoding="utf-8")
        (bundle / "RELEASE-NOTES.txt").write_text("Synthetic release notes\n", encoding="utf-8")

        payload_names = expected_release_names(VERSION) - {
            "BUILD-METADATA.txt",
            "RELEASE-NOTES.txt",
            "SHA256.txt",
        }
        for index, name in enumerate(sorted(payload_names), start=1):
            (bundle / name).write_bytes(f"synthetic-release-payload-{index}\n".encode())

        manifest_lines = []
        for path in sorted(bundle.iterdir()):
            digest = hashlib.sha256(path.read_bytes()).hexdigest()
            manifest_lines.append(f"{digest}  {path.name}")
        (bundle / "SHA256.txt").write_text("\n".join(manifest_lines) + "\n", encoding="utf-8")

        assets = []
        for path in sorted(bundle.iterdir()):
            assets.append(
                {
                    "name": path.name,
                    "digest": "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest(),
                }
            )
        release = {
            "tag_name": TAG,
            "target_commitish": COMMIT,
            "draft": False,
            "prerelease": False,
            "immutable": False,
            "assets": assets,
        }
        release_json = root / "release.json"
        release_json.write_text(json.dumps(release), encoding="utf-8")
        return temp, bundle, release_json

    def load_release(self, release_json: Path) -> dict:
        return json.loads(release_json.read_text(encoding="utf-8"))

    def save_release(self, release_json: Path, value: dict) -> None:
        release_json.write_text(json.dumps(value), encoding="utf-8")

    def test_valid_exact_bundle_passes(self) -> None:
        temp, bundle, release_json = self.make_fixture()
        self.addCleanup(temp.cleanup)
        result = verify_release(bundle, release_json, COMMIT)
        self.assertEqual(result["tag"], TAG)
        self.assertEqual(result["commit"], COMMIT)
        self.assertEqual(result["files"], "17")
        self.assertEqual(result["immutable"], "NO")

    def test_remote_digest_mismatch_fails(self) -> None:
        temp, bundle, release_json = self.make_fixture()
        self.addCleanup(temp.cleanup)
        release = self.load_release(release_json)
        release["assets"][0]["digest"] = "sha256:" + "0" * 64
        self.save_release(release_json, release)
        with self.assertRaisesRegex(ValueError, "digest mismatch"):
            verify_release(bundle, release_json, COMMIT)

    def test_missing_remote_digest_fails_closed(self) -> None:
        temp, bundle, release_json = self.make_fixture()
        self.addCleanup(temp.cleanup)
        release = self.load_release(release_json)
        release["assets"][0]["digest"] = None
        self.save_release(release_json, release)
        with self.assertRaisesRegex(ValueError, "has no SHA-256 digest"):
            verify_release(bundle, release_json, COMMIT)

    def test_remote_asset_set_mismatch_fails(self) -> None:
        temp, bundle, release_json = self.make_fixture()
        self.addCleanup(temp.cleanup)
        release = self.load_release(release_json)
        release["assets"].pop()
        self.save_release(release_json, release)
        with self.assertRaisesRegex(ValueError, "asset set"):
            verify_release(bundle, release_json, COMMIT)

    def test_source_commit_mismatch_fails(self) -> None:
        temp, bundle, release_json = self.make_fixture()
        self.addCleanup(temp.cleanup)
        with self.assertRaisesRegex(ValueError, "does not match source run"):
            verify_release(bundle, release_json, "f" * 40)

    def test_internal_sha256_manifest_mismatch_fails(self) -> None:
        temp, bundle, release_json = self.make_fixture()
        self.addCleanup(temp.cleanup)
        artifact = bundle / f"Ghost-FTP-{VERSION}-Setup.exe"
        artifact.write_bytes(b"changed after manifest\n")
        with self.assertRaisesRegex(ValueError, "SHA256.txt digest mismatch"):
            verify_release(bundle, release_json, COMMIT)

    def test_noncanonical_local_asset_name_fails_even_with_same_count(self) -> None:
        temp, bundle, release_json = self.make_fixture()
        self.addCleanup(temp.cleanup)
        source = bundle / f"Ghost-FTP-{VERSION}-Portable.exe"
        source.rename(bundle / "unexpected.bin")
        with self.assertRaisesRegex(ValueError, "canonical 17-file release set"):
            verify_release(bundle, release_json, COMMIT)

    def test_non_regular_bundle_entry_fails_closed(self) -> None:
        temp, bundle, release_json = self.make_fixture()
        self.addCleanup(temp.cleanup)
        (bundle / "unexpected-directory").mkdir()
        with self.assertRaisesRegex(ValueError, "non-regular"):
            verify_release(bundle, release_json, COMMIT)


if __name__ == "__main__":
    unittest.main()
