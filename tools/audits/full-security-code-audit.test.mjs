import assert from "node:assert/strict";
import test from "node:test";
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
