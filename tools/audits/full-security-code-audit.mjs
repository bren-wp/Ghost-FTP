#!/usr/bin/env node
/**
 * Exact-source security code audit across all maintained platforms and release
 * automation. Static checks are regression gates, not a penetration test.
 * Does not print file contents, usernames, passwords or tokens.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const IGNORED = new Set([".git", "node_modules", "target", "build", ".build", ".gradle", ".swiftpm", "dist", "coverage", ".idea"]);
const SOURCE_EXT = new Set([".rs", ".ts", ".tsx", ".kt", ".swift", ".go", ".js", ".mjs", ".sh", ".yml", ".yaml", ".kts"]);
const isSource = (p) => SOURCE_EXT.has(path.extname(p).toLowerCase());
const toPosix = (p) => p.split(path.sep).join("/");

export function trackedSourceFiles(root) {
  const entries = [];
  function walk(dir) {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      if (entry.isSymbolicLink()) continue;
      if (entry.isDirectory()) {
        if (!IGNORED.has(entry.name)) walk(path.join(dir, entry.name));
      } else if (entry.isFile() && isSource(entry.name)) {
        entries.push(toPosix(path.relative(root, path.join(dir, entry.name))));
      }
    }
  }
  walk(root);
  return entries.sort();
}

export const dangerousPatterns = [
  { id: "tls-certificate-bypass", pattern: /\bdanger_accept_invalid_certs\s*\(\s*true\s*\)|\bInsecureSkipVerify\s*:\s*true\b|\bsetAllowsAnyHTTPSCertificate\s*\(\s*true\s*\)/i },
  { id: "ssh-host-key-bypass", pattern: /\bStrictHostKeyChecking\s*=\s*no\b|setHostKeyVerifier\s*\(\s*\{\s*true\s*\}\s*\)/i },
  { id: "unsafe-curl-tls", pattern: /\bcurl\b[^\n]*(?:\s-k\b|\s--insecure\b)/i },
  { id: "password-logging", pattern: /(?:console\.log|println!|print|Log\.[diwe]|NSLog)\s*\(\s*(?:password|secret|accessToken|privateKey)(?:\b|[)\s,])/i },
];
export function isRuntimeCode(file) {
  const production = [
    "ghostftp-desktop/src-tauri/src/", "ghostftp-desktop/src/lib/",
    "ghostftp-desktop/src/components/", "android/app/src/main/java/",
    "macos/Sources/", "tools/ghostftp-runtime/", "tools/ghostftp-installer/"
  ];
  return production.some(p => file.startsWith(p)) &&
    !/\/(?:test|tests|__tests__)\//i.test(file) &&
    !/(?:\.test|_test)\.(?:rs|go|ts|tsx|kt|swift)$/i.test(file);
}
export function scanDangerous(file, content) {
  if (!isRuntimeCode(file)) return [];
  return dangerousPatterns.flatMap(({id,pattern}) => {
    const hit = pattern.exec(content);
    if (!hit) return [];
    return [{ id, path: file, line: content.slice(0, hit.index).split("\n").length }];
  });
}
const REQUIRED = [
  ["macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift", ["O_NOFOLLOW", "O_EXCL", "PrivateUploadSource.verifyUnchanged", "FTPControlCodec.appendListingChunk"]],
  ["ghostftp-desktop/src-tauri/src/known_hosts.rs", ["inspect_known_hosts_for_replace", "ensure_unchanged_known_hosts", "O_NOFOLLOW"]],
  [".github/scripts/publish-release.sh", ["--data-binary", "uploads.github.com", "sha256sum", "draft:false,prerelease:true"]],
  [".github/workflows/ghostftp-release.yml", ["workflow_run:", "SOURCE_SHA", "wait_for_gate", "ghostftp-android.yml"]],
  [".github/scripts/prune-release-history-keep10.sh", ["KEEP_10", "jq -r '.prerelease'", "--cleanup-tag"]],
  ["android/app/src/main/java/com/ghostftp/android/CloudNetworkSafety.kt", [".dns(PublicCloudDns())", ".proxy(Proxy.NO_PROXY)", ".followRedirects(false)"]]
];
export function audit(root = ROOT) {
  const sourceFiles = trackedSourceFiles(root);
  const offenders = [];
  for (const file of sourceFiles) {
    const content = fs.readFileSync(path.join(root, file), "utf8");
    offenders.push(...scanDangerous(file, content));
  }
  const missingGuards = [];
  for (const [file, guards] of REQUIRED) {
    const dest = path.join(root, file);
    if (!fs.existsSync(dest)) {
      missingGuards.push({path:file, required:"source file exists"});
      continue;
    }
    const content = fs.readFileSync(dest, "utf8");
    for (const guard of guards) if (!content.includes(guard)) {
      missingGuards.push({path:file, required:guard});
    }
  }
  const counts = Object.fromEntries(["rs","ts","tsx","kt","swift","go","js","mjs","sh","yml","yaml","kts"]
    .map(ext=>[ext, sourceFiles.filter(file=>file.endsWith("."+ext)).length]));
  const findings = [
    { severity:"blocking-production", surface:"macos FTPS/SFTP", state:"not-implemented", reason:"Native macOS session engine still rejects FTPS/SFTP. No validated TLS/SSH transfer implementation or live macOS protocol acceptance." },
    { severity:"blocking-production", surface:"mobile and desktop signing", state:"not-verified", reason:"Unsigned Play Android packages, Windows Authenticode, Apple Developer ID notarization and in-place signed upgrade proofs remain acceptance blockers." },
    { severity:"manual-verification", surface:"data-integrity", state:"incomplete", reason:"Large (>4 GiB) SHA-256 end-to-end transfers, remote hashes, crash/power-loss recovery and physical-device storage migration were not shown by the static audit." },
    { severity:"manual-verification", surface:"credential protection", state:"requires-device-validation", reason:"Native Windows Credential Manager, Linux Secret Service, Android Keystore and Apple Keychain are used or referenced; verify secret lifetime, OS API errors and actual-user upgrade on target hardware." },
    { severity:"manual-verification", surface:"visual UX/performance", state:"unverified", reason:"The 75 premium reference screens and long running memory/battery benchmarks lack complete certified baseline evidence." }
  ];
  return {
    kind:"full repository static security code review",
    scope:"All checked-in active source and release scripts. Dynamic security proof requires independently executed tests.",
    inventory:{sourceFileCount:sourceFiles.length,byExtension:counts},
    blockingPatterns:offenders,
    missingSecurityGuards:missingGuards,
    findings,
    passed:offenders.length===0 && missingGuards.length===0
  };
}
if (process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  const report=audit();
  console.log(JSON.stringify(report,null,2));
  if (!report.passed) process.exitCode=1;
}
