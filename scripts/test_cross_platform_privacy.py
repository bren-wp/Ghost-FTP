#!/usr/bin/env python3
"""Regression tests for cross-platform Ghost FTP privacy invariants."""

from __future__ import annotations

import unittest

import audit_cross_platform_privacy as privacy


class CrossPlatformPrivacyAuditTest(unittest.TestCase):
    def test_android_privacy_invariants(self) -> None:
        privacy.audit_android()

    def test_browser_privacy_invariants(self) -> None:
        privacy.audit_browser_extensions()


if __name__ == "__main__":
    unittest.main()
