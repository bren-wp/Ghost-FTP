#!/usr/bin/env python3

from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class CurlCapabilityCacheSourceTests(unittest.TestCase):
    def test_constructor_skips_probe_for_plain_ftp(self) -> None:
        text = (ROOT / "internal/remote/curl_ftp.go").read_text(encoding="utf-8")
        self.assertIn(
            "protocolNeedsRevokeCapability(protocol) && curlSupportsRevokeBestEffort(p)",
            text,
        )

    def test_capability_is_cached_concurrently_per_curl_path(self) -> None:
        text = (ROOT / "internal/remote/curl_capability.go").read_text(encoding="utf-8")
        for marker in (
            "var curlRevokeCapabilityCache sync.Map",
            "once      sync.Once",
            "curlRevokeCapabilityCache.LoadOrStore(key, &curlCapabilityResult{})",
            "entry.once.Do(func()",
            "entry.supported = probeCurlRevokeBestEffort(key)",
        ):
            self.assertIn(marker, text)

    def test_only_ftps_protocols_need_revocation_capability(self) -> None:
        text = (ROOT / "internal/remote/curl_capability.go").read_text(encoding="utf-8")
        self.assertIn('return protocol == "ftps" || protocol == "ftpsi"', text)


if __name__ == "__main__":
    unittest.main()
