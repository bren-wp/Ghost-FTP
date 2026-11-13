#!/usr/bin/env node
/**
 * Check ONLY active Windows/Linux user instructions. Keep historical audit
 * snapshots intact; this is not a compiler, threat model or pen-test.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
export const activeDocs = [
  "docs/cloud/PROVIDERS.md",
  "docs/qa/CLOUD_E2E.md",
  "docs/qa/INSTALLER.md",
  "docs/status/PRODUCTION_READINESS.md",
];
const obsolete = /\b(?:android|macos|swiftui|apk|aab|xcode|play console)\b/i;

export function auditActiveDocs(root = ROOT) {
  const issues = [];
  let version = "";
  let previous = "";
  try {
    const meta = JSON.parse(fs.readFileSync(path.join(root, "version.json"), "utf8"));
    version = meta.version;
    previous = meta.previousPublishedVersion;
    if (!/^\d+\.\d+\.\d+$/.test(version) || !/^\d+\.\d+\.\d+$/.test(previous)) {
      throw new Error("missing current or preceding SemVer");
    }
  } catch (error) {
    return { passed: false, issues: ["version.json: " + error.message] };
  }
  for (const rel of activeDocs) {
    let source;
    try { source = fs.readFileSync(path.join(root, rel), "utf8"); }
    catch { issues.push(rel + ": missing active documentation"); continue; }
    if (!source.includes("Ghost FTP " + version + " Preview")) {
      issues.push(rel + ": active Preview version does not match version.json");
    }
    if (!source.includes("**Previous canonical release:** v" + previous)) {
      issues.push(rel + ": predecessor release does not match version.json");
    }
    if (!/Windows/i.test(source) || !/Linux/i.test(source)) {
      issues.push(rel + ": supported Windows/Linux desktop platforms are missing");
    }
    if (obsolete.test(source)) {
      issues.push(rel + ": discontinued platform advertised in active documentation");
    }
  }
  return { passed: issues.length === 0, checked: activeDocs.length, version, issues };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const result = auditActiveDocs();
  if (result.passed) console.log("Current Windows/Linux documentation aligned: " + result.version);
  else { for (const issue of result.issues) console.error("::error::" + issue); process.exitCode = 1; }
}
