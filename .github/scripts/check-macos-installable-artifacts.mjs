import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
const read = (p) => readFileSync(p, "utf8");
const mac = read(".github/workflows/ghostftp-macos-shared-desktop.yml");
const release = read(".github/workflows/ghostftp-release.yml");
const publish = read(".github/scripts/publish-release.sh");
const quality = read(".github/workflows/ghostftp-quality.yml");
const guide = read("docs/guides/MACOS_INSTALLATION.md");
for (const token of ["hdiutil create", "hdiutil verify", "hdiutil attach",
  'ln -s /Applications "$ROOT/Applications"', 'test -L "$MOUNT/Applications"',
  "Installable-Preview.dmg", "codesign --verify --deep --strict",
  "shasum -a 256 -c", "lipo -archs", 'codesign --force --deep --sign - "$APP"'])
  assert.ok(mac.includes(token), "Missing native macOS installable artifact gate: " + token);
for (const arch of ["arm64", "x86_64"]) {
  assert.ok(release.includes("GhostFTP-macOS-Shared-" + arch + "-v$VERSION-Installable-Preview.dmg"),
    "Release must ship DMG for " + arch);
  assert.ok(publish.includes("GhostFTP-macOS-Shared-" + arch + '-v' + "$" + '{VERSION}-Installable-Preview.dmg'),
    "Published release inventory must require DMG for " + arch);
}
assert.ok(release.includes('sha256sum -c "$MANIFEST"'));
assert.ok(quality.includes("node .github/scripts/check-macos-installable-artifacts.mjs"));
assert.ok(guide.includes("Open Anyway") && guide.includes("Applications"));
assert.ok(guide.includes("sudo spctl --master-disable") && guide.includes("com.apple.quarantine"));
for(const dangerous of ["spctl --master-disable", "xattr -dr com.apple.quarantine", "gh release create"])
  assert.ok(!mac.includes(dangerous), "Unsafe Mac installer workflow behavior: " + dangerous);
console.log("Non-Developer-ID installable macOS DMG distribution checks passed.");
