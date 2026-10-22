import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
const workflow=readFileSync(".github/workflows/ghostftp-macos-developer-id.yml","utf8");
const guide=readFileSync("docs/qa/MACOS_PRODUCTION_RELEASE.md","utf8");
for(const required of ["workflow_dispatch:","macos-production-signing","Developer ID Application:",
"macos-15-intel","arch: arm64","arch: x86_64","GHOSTFTP_MACOS_DEVELOPER_ID_P12_BASE64",
"xcrun notarytool submit","xcrun stapler staple","xcrun stapler validate",
"codesign --verify --deep --strict","spctl --assess --type execute",
"shasum -a 256 -c","physicalMacAcceptance","publicProductionRelease:false",
"security delete-keychain","if: always()"])
  assert.ok(workflow.includes(required), "Production gate missing "+required);
for(const forbidden of ["gh release create","gh release upload","continue-on-error: true","|| true"])
  assert.ok(!workflow.includes(forbidden), "Production workflow has unsafe override: "+forbidden);
assert.ok(guide.includes("Physical-device acceptance"),"Physical Mac acceptance must be documented");
const quality=readFileSync(".github/workflows/ghostftp-quality.yml","utf8");
assert.ok(quality.includes("node .github/scripts/check-macos-production-gate.mjs"),
  "Production security gate must run in mandatory quality workflow");
const native=readFileSync("ghostftp-desktop/src-tauri/src/lib.rs","utf8");
const titlebar=readFileSync("ghostftp-desktop/src/components/TitleBar.tsx","utf8");
const css=readFileSync("ghostftp-desktop/src/styles.css","utf8");
assert.ok(native.includes('.decorations(cfg!(target_os = "macos"))'),"macOS must expose native window controls");
assert.ok(native.includes("tauri::TitleBarStyle::Overlay"),"macOS title bar must be native");
assert.ok(titlebar.includes("{!nativeMac && ("),"macOS must not render duplicate fake window buttons");
assert.ok(css.includes(".ghost-app-header.ghost-macos-native-titlebar .ghost-title-row"),
  "Reserve layout space for macOS native window controls");
const jobHeader=workflow.split("    steps:")[0];
assert.ok(!jobHeader.includes("P12_BASE64:") && !jobHeader.includes("NOTARY_P8_BASE64:"),
  "Do not expose Apple private keys to every dependency/build job step");
console.log("macOS notarization candidate pipeline gates verified.");
