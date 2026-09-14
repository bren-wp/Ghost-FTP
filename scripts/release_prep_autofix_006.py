#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def one(path: str, old: str, new: str) -> None:
    target = ROOT / path
    text = target.read_text(encoding="utf-8")
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected exactly one marker, found {count}: {old[:100]!r}")
    target.write_text(text.replace(old, new, 1), encoding="utf-8")


one(
    "android/app/src/main/java/app/ghostftp/client/MainActivity.java",
    'TextView platform = label("Android development client", 11, GhostTheme.MUTED);',
    'TextView platform = label("Android app", 11, GhostTheme.MUTED);',
)
one(
    "android/app/src/main/java/app/ghostftp/client/MainActivity.java",
    'content.addView(surfaceHeading("About", "Build identity and privacy/security status for this Android development client."));',
    'content.addView(surfaceHeading("About", "Build identity and privacy/security status for this Android app."));',
)
one(
    "android/app/src/main/java/app/ghostftp/client/MainActivity.java",
    'card.addView(infoLine("Release status", "Repository build " + BuildConfig.VERSION_NAME + "; Android APK remains development-only and is not a public Android release"), matchWrapSpaced());',
    'card.addView(infoLine("Release status", "Repository build " + BuildConfig.VERSION_NAME + (BuildConfig.DEBUG ? "; development package; not the production-signed public APK" : "; release package; official publication requires verified publisher-signature evidence")), matchWrapSpaced());',
)
one(
    "android/UI-UX.md",
    "The Android version shown in About comes from `BuildConfig.VERSION_NAME`, which is derived from the repository root `VERSION` plus the Android development suffix. The CI APK remains a development/debug-signed artifact and must not be represented as a public Android release merely because the repository release identity advances.",
    "The Android version shown in About comes from `BuildConfig.VERSION_NAME`. Production release builds use the repository root `VERSION` exactly; development builds append `-dev`. The ordinary CI development APK remains explicitly separate from the production-signed public Android release, and official publication is accepted only after the protected release workflow verifies the configured publisher certificate fingerprint.",
)
one(
    "docs/THIRD-PARTY-NOTICES.md",
    "`ekstenzije/` contains the official Manifest V3 Chrome, Edge and Firefox companion packages. They use browser-provided extension APIs and one local shared JavaScript runtime; they do not bundle an FTP/SFTP networking stack, connect to transfer servers, or provide a supported browser-to-desktop URI/native-messaging handoff today.",
    "`ekstenzije/` contains the official Manifest V3 Chrome, Edge and Firefox companion packages. They use browser-provided extension APIs and one local shared JavaScript runtime; they do not bundle an FTP/SFTP networking stack or connect to transfer servers. They do not provide a supported browser-to-desktop URI/native-messaging handoff today.",
)
one(
    "CHANGELOG.md",
    "### Android parity, privacy and lifecycle hardening\n\n",
    "### Android parity, privacy and lifecycle hardening\n\n- Preserved the non-destructive current-folder filter over already-loaded entries with no hidden filesystem/network scan.\n- Preserved bounded recursive local/server search as a separate cancellable, deadline-limited workflow rather than conflating it with current-folder filtering.\n- Navigation bookmarks and profile start directories remain bound to fresh validation, account identity and stale-session protection across maintained desktop surfaces.\n",
)
one(
    "docs/RELEASE-VERIFICATION.md",
    "# Ghost FTP release verification\n\n",
    "# Ghost FTP release verification\n\nThe canonical 0.0.6 publication contains **18 platform artifacts / 21 public files**.\n\n",
)
one(
    "docs/RELEASE-HISTORY.md",
    "# Ghost FTP release history\n\n",
    "# Ghost FTP release history\n\n## 0.0.6 — 2026-09-14\n\nGhost FTP 0.0.6 expands the verified public release to Windows, Linux, Android and official Chrome/Edge/Firefox companion packages while preserving fail-closed transport and signing boundaries.\n\n- Completed Linux modal input/focus hardening and retained shared Windows/Linux Engine behavior.\n- Added Android file management, filtering/sorting, bounded recursive search, directory comparison, synchronized navigation and hardened Remote Edit workflows.\n- Added the protected production-signed Android release path while keeping Android SFTP hidden until strict maintained host-key verification exists.\n- Consolidated official browser helpers onto one shared runtime with enforced Ghost FTP branding, minimal permissions and no supported desktop launch/handoff.\n- Expanded publication to **18 platform artifacts / 21 public files** with exact SHA-256 readback and latest-only retention.\n- Preserved macOS as a separately validated development/source frontend until real Developer ID signing and Apple notarization succeed.\n\n",
)
one(
    "docs/NAVIGATION-BOOKMARKS.md",
    "saving a bookmark never creates a hidden Site Manager profile.",
    "Saving a bookmark does **not** create a hidden persistent Site Manager profile.",
)
one(
    "docs/NAVIGATION-BOOKMARKS.md",
    "The 0.0.6 contract is protected by bookmark/config/profile-binding Go tests, Linux desktop modal/viewport tests, `scripts/test_navigation_bookmarks_contract.py`, and the macOS development parity contract.",
    "The 0.0.6 contract is protected by bookmark/config/profile-binding Go tests, Linux desktop modal/viewport tests, `scripts/test_navigation_bookmarks_contract.py`, and the macOS development parity contract. Publication additionally requires exact-head CI/native-build/authentic-runtime evidence.",
)
one(
    "docs/QUEUE-PRIORITY.md",
    "Directory-tree structural preparation occurs before file jobs become reorderable, so priority controls cannot move a file ahead of an unexecuted directory-creation queue dependency.",
    "Directory-tree structural preparation occurs before `reservation.Commit()` makes file jobs reorderable, so priority controls cannot move a file ahead of an unexecuted directory-creation queue dependency.",
)
one(
    "scripts/test_android_mobile_navigation_contract.py",
    '        self.assertIn("Android APK remains development-only", about)',
    '        self.assertIn("BuildConfig.DEBUG", about)\n        self.assertIn("development package; not the production-signed public APK", about)\n        self.assertIn("release package; official publication requires verified publisher-signature evidence", about)\n        self.assertNotIn("Android APK remains development-only", about)',
)
one(
    "scripts/test_android_mobile_navigation_contract.py",
    '        self.assertIn("not a public Android release", about)',
    '        self.assertIn("not the production-signed public APK", about)',
)
one(
    "scripts/test_android_release_identity_contract.py",
    '        self.assertIn("intentionally hidden", readme)',
    '        self.assertIn("intentionally not exposed", readme)',
)
one(
    "scripts/test_maintenance.py",
    '        self.assertIn("ekstenzije/", release)',
    '        self.assertIn("build_browser_extensions.py", release)\n        self.assertIn("for browser in chrome edge firefox", release)',
)
Path(__file__).unlink()
