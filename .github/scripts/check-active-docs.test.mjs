import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { activeDocs, auditActiveDocs } from "./check-active-docs.mjs";

const fixture = () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "ghostftp-docs-"));
  fs.writeFileSync(path.join(root, "version.json"), JSON.stringify({
    version: "0.99.59", previousPublishedVersion: "0.99.58",
  }));
  for (const rel of activeDocs) {
    const file = path.join(root, rel);
    fs.mkdirSync(path.dirname(file), {recursive:true});
    fs.writeFileSync(file, "Ghost FTP 0.99.59 Preview\n**Previous canonical release:** v0.99.58\nWindows and Linux.\n");
  }
  return root;
};

test("active desktop docs require synchronized source and predecessor versions", () => {
  const root = fixture();
  try { assert.equal(auditActiveDocs(root).passed, true); }
  finally { fs.rmSync(root, {recursive:true,force:true}); }
});

test("discontinued-platform statements and missing files fail closed", () => {
  const root = fixture();
  try {
    const file = path.join(root, activeDocs[0]);
    fs.appendFileSync(file, "Install the Android APK now.\n");
    assert.match(auditActiveDocs(root).issues.join(" "), /discontinued platform/);
    fs.writeFileSync(file, "Ghost FTP 0.99.58 Preview\nWindows Linux\n");
    assert.equal(auditActiveDocs(root).passed, false);
    fs.rmSync(file);
    assert.match(auditActiveDocs(root).issues.join(" "), /missing active documentation/);
  } finally { fs.rmSync(root, {recursive:true,force:true}); }
});
