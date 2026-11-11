import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { auditRetiredPlatform } from "./check-retired-platform.mjs";

test("current active release tree has no retired app or package requirements", () => {
  const result = auditRetiredPlatform();
  assert.equal(result.passed, true, JSON.stringify(result.findings));
  assert.deepEqual(result.activePlatforms, ["windows", "linux"]);
});

test("retired source and release gate restoration are detected", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(),"ghostftp-retired-platform-"));
  try {
    fs.mkdirSync(path.join(root,"android","app"),{recursive:true});
    fs.mkdirSync(path.join(root,"macos","Sources"),{recursive:true});
    fs.mkdirSync(path.join(root,".github","workflows"),{recursive:true});
    fs.writeFileSync(path.join(root,".github/workflows/ghostftp-release.yml"),"release: requires ghostftp-android.yml");
    const report = auditRetiredPlatform(root);
    assert.equal(report.passed,false);
    assert.ok(report.findings.some(x=>x.path==="android"));
    assert.ok(report.findings.some(x=>x.path==="macos"));
    assert.ok(report.findings.some(x=>x.path===".github/workflows/ghostftp-release.yml"));
  } finally {
    fs.rmSync(root,{recursive:true,force:true});
  }
});
