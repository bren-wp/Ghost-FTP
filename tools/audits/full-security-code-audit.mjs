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
const IGNORED = new Set([".git", "node_modules", "target", "build", ".build", ".gradle", "dist", "coverage", ".idea"]);
const SOURCE_EXT = new Set([".rs", ".ts", ".tsx", ".go", ".js", ".mjs", ".sh", ".yml", ".yaml"]);
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
  { id: "unsafe-local-cors", pattern: /Access-Control-Allow-Origin:\s*\*/i },
  { id: "private-key-literal", pattern: /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----\r?\n(?:[A-Za-z0-9+/=]{32,}\r?\n)+-----END (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/i },
  { id: "curl-pipe-shell", pattern: /\bcurl\b[^\n|]{0,300}\|\s*(?:ba)?sh\b/i },
];
export function isRuntimeCode(file) {
  const production = [
    "ghostftp-desktop/src-tauri/src/", "ghostftp-desktop/src/lib/",
    "ghostftp-desktop/src/components/", "ghostftp-desktop/src/stores/",
    "ghostftp-desktop/src/hooks/",
    "tools/ghostftp-runtime/", "tools/ghostftp-installer/"
  ];
  return production.some(p => file.startsWith(p)) &&
    !/\/(?:test|tests|__tests__)\//i.test(file) &&
    !/(?:\.test|_test)\.(?:rs|go|ts|tsx|kt)$/i.test(file);
}
export function scanDangerous(file, content) {
  if (!isRuntimeCode(file)) return [];
  return dangerousPatterns.flatMap(({id,pattern}) => {
    const global = new RegExp(pattern.source, pattern.flags.includes("g") ? pattern.flags : pattern.flags + "g");
    return [...content.matchAll(global)].map((hit) => ({
      id,
      path: file,
      line: content.slice(0, hit.index).split("\n").length,
    }));
  });
}
const REQUIRED = [
  // Bearer grant URLs must never be sent through a proxy or parsed from an
  // unlimited HTTP response. These are production Windows/Linux Rust guards.
  ["ghostftp-desktop/src-tauri/src/grant.rs", [
    ".no_proxy()", ".resolve(host, resolved)",
    ".redirect(reqwest::redirect::Policy::none())",
    "read_bounded_grant_json(resp, MAX_GRANT_MANIFEST_BYTES)",
    "read_bounded_grant_json(resp, MAX_GRANT_KEY_REPLY_BYTES)",
    "validate_host(&conn.host)?",
  ]],
  // All Windows/Linux download destinations must be byte-verified before
  // marking Done; uploaded SFTP bytes must match both local and server size.
  ["ghostftp-desktop/src-tauri/src/transfer.rs", [
    "fn verify_committed_length(", "fn completed_transfer_size(",
    "async fn verify_staged_download_bytes(", "verify_staged_download_bytes(&working_path, record.transferred)",
    "fn safe_transfer_failure(", "t.error = Some(safe_transfer_failure(&e).to_string())",
    "old_diagnostic_rewritten", "INTERRUPTED_TRANSFER_MESSAGE",
    "verify_committed_length(\"SFTP download\"",
    "verify_committed_length(\"SFTP upload\"",
    "verify_committed_length(\"SFTP upload source\"",
    "verify_committed_length(\"FTP download\"",
    "FtpTransferControl::Continue => verify_committed_length(",
    "sync SFTP download staging file", "sync FTP download staging file",
    "Some(t) => completed_transfer_size(t.size, t.transferred)",
    // Prevent equal-length remote replacement from silently corrupting
    // paused transfers. FTP/FTPS must restart until prefix verification exists.
    "async fn byte_equal_prefix<", "async fn agent_remote_prefix_matches(",
    "byte_equal_prefix(&mut prior, &mut remote_file, resume_offset).await?",
    "byte_equal_prefix(&mut local_prefix, &mut remote_prefix, resume_offset).await?",
    "agent_remote_prefix_matches(session, remote_path, local_path, offset).await?",
    "verify_committed_length(\"Agent download\"",
    "verify_committed_length(\"Agent upload\"",
    "verify_committed_length(\"Agent upload source\"",
    "bytes == n as u64",
    "sync Agent download staging file",
    "Agent download returned an empty non-final chunk",
    "Agent did not acknowledge an empty upload",
    "Session::Ssh(_) | Session::Agent(_)",
  ]],
  ["ghostftp-desktop/src-tauri/src/known_hosts.rs", ["inspect_known_hosts_for_replace", "ensure_unchanged_known_hosts", "O_NOFOLLOW"]],
  ["ghostftp-desktop/src-tauri/src/virtualfs/mod.rs", [
    "fn read_settings(", "std::io::ErrorKind::NotFound", "next_registration_journal(",
    "failed_unregistrations", "if !self.teardown(", "registration journal persistence failed",
  ]],
  [".github/scripts/publish-release.sh", ["--data-binary", "uploads.github.com", "sha256sum", "draft:false,prerelease:true"]],
  [".github/workflows/ghostftp-release.yml", ["workflow_run:", "SOURCE_SHA", "wait_for_gate", "ghostftp-protocol-e2e.yml"]],
  [".github/scripts/check-release-governance.test.mjs", ["released editions and tags must never be automatically pruned"]],
  [".github/scripts/enforce-preview-release-metadata.sh", ["0.99.0", "0.95.53", "make_latest", "enforce_preview", "sha256:"]],
  [".github/workflows/ghostftp-preview-metadata.yml", ["workflow_run:", "Ghost FTP release", "contents: write"]],
  ["ghostftp-desktop/src-tauri/src/bridge.rs", [
    "HTTP_READ_TIMEOUT", "tokio::time::timeout(HTTP_READ_TIMEOUT",
    "duplicate Content-Length", "Transfer-Encoding is not supported",
    "POST requires Content-Length", "truncated request body",
    "if req.has_origin", "create_new(true)", ".mode(0o600)",
  ]]
];
/** Structured/high-value native platform privacy contracts. These checks
 * intentionally reject unsafe release configurations rather than pretending
 * source-pattern scanning proves all code paths safe. */
export function inspectPlatformSecurityConfig(source) {
  const failures = [];
  const requireText = (file, value, label) => {
    if (!source[file]?.includes(value)) failures.push({ path: file, required: label });
  };
  const parse = (file) => {
    try { return JSON.parse(source[file] ?? ""); }
    catch { failures.push({ path: file, required: "valid JSON" }); return {}; }
  };
  const config = "ghostftp-desktop/src-tauri/tauri.conf.json";
  const tauri = parse(config);
  const csp = tauri.app?.security?.csp ?? "";
  for (const policy of ["script-src 'self'", "object-src 'none'", "frame-src 'none'", "form-action 'self'"]) {
    if (!csp.includes(policy)) failures.push({ path: config, required: "strict CSP: " + policy });
  }
  if (tauri.plugins?.updater?.allowDowngrades !== false ||
      tauri.plugins?.updater?.requireSignedVersion !== true) {
    failures.push({ path: config, required: "signed updater, downgrade prevention" });
  }
  if (!Array.isArray(tauri.plugins?.updater?.endpoints) ||
      tauri.plugins.updater.endpoints.some(endpoint => !/^https:\/\/[a-z0-9.-]+\//i.test(endpoint))) {
    failures.push({ path: config, required: "HTTPS-only updater endpoints" });
  }

  const capsPath = "ghostftp-desktop/src-tauri/capabilities/default.json";
  const capabilities = parse(capsPath);
  const permissions = capabilities.permissions ?? [];
  const broad = ["shell:allow-execute", "shell:allow-spawn", "fs:allow-write-all", "fs:allow-read-all"];
  if (!Array.isArray(permissions) || permissions.some(x => broad.includes(x)) ||
      JSON.stringify(capabilities.windows) !== JSON.stringify(["main"])) {
    failures.push({ path: capsPath, required: "single main window without broad shell/filesystem capabilities" });
  }

  const go = "tools/ghostftp-runtime/security.go";
  for (const check of ["dec.Decode(&struct{}{})", "pathContainsSymlink(", "chmodWithoutLinks("]) {
    requireText(go, check, "Go local runtime safety: " + check);
  }
  return failures;
}

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
  const configPaths = [
    "ghostftp-desktop/src-tauri/tauri.conf.json",
    "ghostftp-desktop/src-tauri/capabilities/default.json",
    "tools/ghostftp-runtime/security.go",
  ];
  const configSource = Object.fromEntries(configPaths.map(file => {
    const absolute = path.join(root, file);
    return [file, fs.existsSync(absolute) ? fs.readFileSync(absolute, "utf8") : ""];
  }));
  const platformConfigBlockers = inspectPlatformSecurityConfig(configSource);
  const counts = Object.fromEntries(["rs","ts","tsx","kt","go","js","mjs","sh","yml","yaml","kts"]
    .map(ext=>[ext, sourceFiles.filter(file=>file.endsWith("."+ext)).length]));
  const findings = [
    { severity:"blocking-production", surface:"desktop signing", state:"not-verified", reason:"Windows Authenticode and signed in-place upgrade proofs remain acceptance blockers." },
    { severity:"manual-verification", surface:"data-integrity", state:"incomplete", reason:"Large (>4 GiB) SHA-256 end-to-end transfers, remote hashes, crash/power-loss recovery and physical-device storage migration were not shown by the static audit." },
    { severity:"manual-verification", surface:"credential protection", state:"requires-device-validation", reason:"Native Windows Credential Manager and Linux Secret Service are used or referenced; verify secret lifetime, OS API errors and actual-user upgrade on target hardware." },
    { severity:"manual-verification", surface:"visual UX/performance", state:"unverified", reason:"The 59 Windows/Linux reference screens and long running memory/battery benchmarks lack complete certified baseline evidence." }
  ];
  return {
    kind:"full repository static security code review",
    scope:"All checked-in active source and release scripts. Dynamic security proof requires independently executed tests.",
    inventory:{sourceFileCount:sourceFiles.length,byExtension:counts},
    blockingPatterns:offenders,
    missingSecurityGuards:missingGuards,
    platformConfigBlockers,
    findings,
    passed:offenders.length===0 && missingGuards.length===0 && platformConfigBlockers.length===0
  };
}
if (process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  const report=audit();
  console.log(JSON.stringify(report,null,2));
  if (!report.passed) process.exitCode=1;
}
