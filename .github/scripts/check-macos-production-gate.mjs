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
console.log("macOS notarization candidate pipeline gates verified.");
