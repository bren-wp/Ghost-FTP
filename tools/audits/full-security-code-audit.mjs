#!/usr/bin/env node
// Ghost FTP by Brendigo
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
const SOURCE_EXT = new Set([".rs", ".ts", ".tsx", ".go", ".js", ".mjs", ".sh", ".yml", ".yaml", ".py"]);
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
    "ghostftp-desktop/src/", // Also cover App.tsx/main.tsx, not only subfolders.
    "ghostftp-desktop/packages/file-ui/src/",
    "ghostftp-desktop/src-tauri/src/",
    "ghostftp-desktop/src-tauri/ghostftp-agent-proto/src/",
    "ghostftp-desktop/src-tauri/ghostftp-agentd/src/",
    "ghostftp-desktop/src/lib/",
    "ghostftp-desktop/src/components/", "ghostftp-desktop/src/stores/",
    "ghostftp-desktop/src/hooks/",
    "tools/ghostftp-runtime/", "tools/ghostftp-installer/"
  ];
  return production.some(p => file.startsWith(p)) &&
    !/\/(?:test|tests|__tests__)\//i.test(file) &&
    !/\/tests\.rs$/i.test(file) &&
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
    "fn verify_committed_length(", 
    "verify_download_writer(&writer, record.transferred).await",
    ".promote(&working_path, &final_path, &id, allow_replace, &writer)",
    "fn verify_download_writer(",
    "sync original download writer",
    "Ok(tokio::fs::File::from_std(file))",
    "commit_verified_delta_output(&out_path, local_path, size)",
    "checked_agent_delta_chunk_length(range_len - got, bytes.len())",
    "fn safe_transfer_failure(", "transfer.error = Some(safe_transfer_failure(&error).to_string())",
    "old_diagnostic_rewritten", "INTERRUPTED_TRANSFER_MESSAGE",
    "verify_committed_length(\"SFTP download\"",
    "verify_committed_length(\"SFTP upload\"",
    "\"SFTP upload source\",",
    
    "FtpTransferControl::Continue => verify_committed_length(",
    "sync SFTP download staging file", "sync FTP download staging file",
    "completed_transfer_size(transfer.size, transfer.transferred).map(|_| ())",
    // Prevent equal-length remote replacement from silently corrupting
    // paused transfers. FTP/FTPS must restart until prefix verification exists.
    "async fn byte_equal_prefix<", "async fn agent_remote_prefix_matches(",
    "byte_equal_prefix(&mut prior, &mut remote_file, resume_offset).await?",
    "agent_remote_prefix_matches(session, remote_path, local_path, offset).await?",
    "verify_committed_length(\"Agent download\"",
    "verify_committed_length(\"Agent upload\"",
    "verify_committed_length(\"Agent upload source\"",
    "bytes == n as u64",
    "verify_agent_delta_patch_ack(n, bytes)?",
    "verify_agent_delta_upload_completion(",
    "cleanup_owned_regular_staging_file(&local_patch).await",
    "sync Agent download staging file",
    "Agent download returned an empty non-final chunk",
    "Agent did not acknowledge an empty upload",
    "inspect_directory_upload_entry(&p).await?",
    "ensure_upload_directory(fs.as_ref(), &remote_root).await?",
    "cleanup_failed_download_staging(&working_path, error).await",
    "Session::Ssh(_) | Session::Agent(_)",
    "source_snapshot.verify(&local_file.metadata().await?)?;",
    "source_snapshot.verify(&source_after)?;",
    "verify_local_upload_snapshot(local, source_snapshot).await?;",
    "OpenFlags::WRITE | OpenFlags::CREATE | OpenFlags::EXCLUDE",
    "is_safe_staged_upload_retry(&e)",
    "UploadPromotionUnconfirmed",
    "fn is_terminal_transfer_status(",
    "ledger_serial: Mutex<()>",
    "retry_serial: Mutex<()>",
    "async fn mark_canceled(",
    "self.mark_canceled(id).await",
    "flag.store(true, Ordering::Release);",
    "async fn finish_transfer(",
    "async fn reset_for_retry(",
    "let _ledger = self.ledger_serial.lock().await;",
    "let _retry = self.retry_serial.lock().await;",
    "if is_terminal_transfer_status(transfer.status)",
    "mgr.finish_transfer(id, result).await",
    
    
    
  ]],
  
  // Guards and their regression cases remain mandatory after the transfer.rs
  // split. Check each *real* destination file, never a consolidated text blob.
  ["ghostftp-desktop/src-tauri/src/transfer/integrity.rs", [
    "fn completed_transfer_size(",
    "fn checked_object_download_progress(",
    "fn declared_full_http_download_length(",
    "fn checked_http_download_progress(",
    "fn declared_plain_http_length(",
    "fn verify_plain_http_download_length(",
    "headers.get_all(CONTENT_LENGTH).iter()",
    "headers.get_all(CONTENT_ENCODING).iter()",
    "ambiguous Content-Length headers",
    "ambiguous Content-Encoding headers",
  ]],
  ["ghostftp-desktop/src-tauri/src/transfer/http_length_integrity_tests.rs", [
    "verify_committed_length(\"FTP download\"",
    "fn ftp_ftps_sftp_downloads_cannot_confirm_truncated_or_oversized_streams()",
    "fn remote_errors_never_persist_authorization_headers_or_signed_urls()",
    "fn ambiguous_duplicate_wire_metadata_fails_closed()",
  ]],
  ["ghostftp-desktop/src-tauri/src/transfer/tests.rs", [
    "terminal_race_test_row(",
    "canceled_transfer_is_never_resurrected_by_late_completion_or_progress()",
    "completed_row_and_sqlite_ledger_reject_stale_progress_after_success()",
    "resume_prefix_requires_full_byte_identity_including_same_size_replacement()",
  ]],
  ["ghostftp-desktop/src-tauri/src/transfer/remote_upload_staging.rs", [
    "fn fresh_remote_upload_staging_path(",
    "Uuid::new_v4().simple()",
    "validate_protocol_mutation_path(&staging_path)?",
    "fn capture_local_upload_snapshot(",
    "fn verify_local_upload_snapshot(",
    "symlink_metadata(path)",
    "self.device == metadata.dev() && self.inode == metadata.ino()",
    "self.created == metadata.creation_time()",
    "self.len == metadata.len()",
  ]],
  // Staging and delta commits must be byte-verified, durable and never
  // clobber symlinks, directories, or preexisting unowned temporary files.
  ["ghostftp-desktop/src-tauri/src/transfer/download_staging.rs", [
    "fn create_new_download_staging(", "create_new(true)",
    "fn verify_staged_download_bytes(", "file.sync_all()",
    "fn promote_staged_download(", "fn commit_verified_delta_output(",
    "tokio::fs::rename(staging, final_path)",
    "promote_staged_download_noreplace(staging, final_path)",
    "move_local_noreplace(", "symlink_metadata(",
    "fn cleanup_failed_download_staging(",
    "fn cleanup_owned_regular_staging_file(",
    "fn create_new_delta_upload_patch(",
    "downcast_ref::<DownloadStagingOpenFailure>()",
    "fn promote_staged_download_noreplace(",
  ]],
  ["ghostftp-desktop/src-tauri/ghostftp-agent-proto/src/delta.rs", [
    "let mut output_created = false", "result.is_err() && output_created",
    "create_new(true)", "*output_created = true",
    "fn apply_delta_inner(",
  ]],
  ["ghostftp-desktop/src-tauri/ghostftp-agentd/src/ops.rs", [
    "fn promote_verified_agent_delta_output(", "symlink_metadata(temp)",
    "existing.file_type().is_file()", "total.checked_add(bytes)",
    "apply_delta cleans up only its own output",
    "async fn agent_read_file_reports_truncation_only_when_bytes_are_missing()",
    ".take((cap as u64).saturating_add(1))",
    "let truncated = buf.len() > cap;",
  ]],
  // The transfer path-safety domain was extracted from transfer.rs. Treat
  // removal of its fail-closed checks as a security regression, not dead code.
  ["ghostftp-desktop/src-tauri/src/transfer/path_safety.rs", [
    "fn checked_remote_download_relative(", "fn safe_local_download_leaf(",
    "fn ensure_download_subdirectory(", "symlink_metadata(&current)",
    "std::io::ErrorKind::AlreadyExists", "raced.file_type().is_dir()",
    "fn validate_directory_download_budget(", "fn validate_directory_upload_budget(",
    "fn inspect_directory_upload_entry(", "tokio::fs::symlink_metadata(path)",
  ]],
  ["ghostftp-desktop/src-tauri/src/transfer/remote_directory.rs", [
    "fn ensure_upload_directory(", "fs.create_dir(path).await",
    "fs.list_dir(path).await", "cannot create or list remote upload directory",
  ]],
  ["ghostftp-desktop/src-tauri/src/profiles/mod.rs", [
    "fn interrupted_profile_predecessor(", "profiles.json.bak",
    "profile changes are disabled until storage is repaired",
    "refusing to write plaintext profile credentials",
    "fn write(&self, profiles:",
    "profile_store_refuses_dangling_storage_symlink",
    "profiles_and_rail_order_survive_close_and_restart",
  ]],
  // On app restart, durable profiles must be rehydrated into the
  // frontend rather than leaving the saved sites list empty until another UI
  // action happens to fetch it.
  ["ghostftp-desktop/src/App.tsx", [
    "useConnections.getState().loadProfiles()",
    "Couldn't load saved connections",
  ]],
  ["ghostftp-desktop/src/stores/connectionsStore.ts", [
    "loadProfiles: async () => {",
    "const persisted = await ipc.listProfiles()",
    "await ipc.saveProfile(p);",
    "await get().loadProfiles();",
  ]],
  // Profile credentials may only be saved via the OS vault, never in the
  // exported profiles.json metadata or a replaced/linked export target.
  ["ghostftp-desktop/src-tauri/src/commands/profile_security.rs", [
    "record_keychain(", "credentials::set_secret(", "password.clear()",
    "credentials::get_secret(", "delete_secret_recorded(",
    "create_new(true)", "symlink_metadata(destination)", "file.sync_all()",
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
    "MAX_BRIDGE_CONNECTIONS", "try_acquire_owned()",
    "if req.has_origin", "create_new(true)", ".mode(0o600)",
    "fn decode_agent_file_response(", "decode_agent_file_response(&data, bytes)",
    "Agent read returned invalid or oversized data",
  ]],
  ["ghostftp-desktop/src-tauri/src/bridge/tests.rs", [
    "agent_file_reply_rejects_corruption_oversize_and_false_byte_counts",
    "revoke_all_access_clears_runtime_profiles_and_approval_policy",
  ]],
  ["ghostftp-desktop/src-tauri/src/bridge/http_request.rs", [
    "duplicate Content-Length", "Transfer-Encoding is not supported",
    "POST requires Content-Length", "truncated request body",
    "invalid HTTP header encoding", "invalid HTTP request line",
    "folded HTTP headers are not supported", "invalid HTTP header name",
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

/// Windows shell security contracts: source-level safeguards against executable
/// search-path hijacking, GUI stalls and malformed protocol-handler input.
/// This is a regression scanner, not a substitute for native penetration tests.
export function inspectWindowsShellSecurity(source) {
  const files = {
    shell: "ghostftp-desktop/src-tauri/src/windows_process.rs",
    app: "ghostftp-desktop/src-tauri/src/lib.rs",
    editor: "ghostftp-desktop/src-tauri/src/editor.rs",
    links: "ghostftp-desktop/src-tauri/src/deeplink.rs",
    terminal: "ghostftp-desktop/src-tauri/src/terminal.rs",
  };
  const failures = [];
  const required = [
    [files.shell, "ShellExecuteW("],
    [files.shell, "CoInitializeEx("],
    [files.app, "tokio::task::spawn_blocking(move ||"],
    [files.app, "open_with_shell(std::ffi::OsStr::new(&url))"],
    [files.editor, "open_with_shell(path.as_os_str())"],
    [files.links, "if raw.len() > MAX_DEEP_LINK_BYTES"],
    [files.links, "index >= MAX_DEEP_LINK_PARAMETERS"],
    [files.links, "v.chars().any(char::is_control)"],
    [files.terminal, "struct Utf8OutputDecoder"],
    [files.terminal, "stdout_decoder.push(data)"],
    [files.terminal, "stderr_decoder.push(data)"],
  ];
  for (const [file, text] of required) {
    if (!source[file]?.includes(text)) failures.push({path:file, required:text});
  }
  for (const file of [files.shell, files.app, files.editor]) {
    if (/Command::new\s*\(\s*"rundll32(?:\.exe)?"\s*\)/i.test(source[file] ?? "")) {
      failures.push({path:file, required:"Do not search Windows PATH/CWD for rundll32.exe"});
    }
  }
  return failures;
}

/**
 * A new UUID staging path has no prior byte prefix to resume. A resumed
 * offset in this worker is a data-integrity bug even if it matches an old
 * remote file size. Inspect this scoped worker, not unrelated SFTP download
 * or Agent resume logic, which still verifies committed prefixes.
 */
export function inspectFreshStagedSftpUpload(transferSource) {
  const start = transferSource.indexOf("    async fn run_ssh_upload(");
  const end = transferSource.indexOf("\n}\n\nimpl TransferManager {", start);
  if (start < 0 || end <= start) return ["SFTP staged upload worker missing"];
  const worker = transferSource.slice(start, end);
  const required = [
    "t.transferred = 0;",
    "let mut transferred: u64 = 0;",
    "OpenFlags::WRITE | OpenFlags::CREATE | OpenFlags::EXCLUDE",
    "source_snapshot.verify(&local_file.metadata().await?)?;",
    "source_snapshot.verify(&source_after)?;",
    'verify_committed_length("SFTP upload", local_len, transferred, remote_size)?',
  ];
  const failures = required.filter(guard => !worker.includes(guard));
  if (/\b(?:resume_offset|requested_offset)\b/.test(worker)) {
    failures.push("fresh SFTP staging must not reuse any previous upload offset");
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
  const stagedSftp = "ghostftp-desktop/src-tauri/src/transfer.rs";
  if (sourceFiles.includes(stagedSftp)) {
    const transfer = fs.readFileSync(path.join(root, stagedSftp), "utf8");
    for (const required of inspectFreshStagedSftpUpload(transfer)) {
      missingGuards.push({path: stagedSftp, required});
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
  const windowsPaths = [
    "ghostftp-desktop/src-tauri/src/windows_process.rs",
    "ghostftp-desktop/src-tauri/src/lib.rs",
    "ghostftp-desktop/src-tauri/src/editor.rs",
    "ghostftp-desktop/src-tauri/src/deeplink.rs",
    "ghostftp-desktop/src-tauri/src/terminal.rs",
  ];
  const windowsSources = Object.fromEntries(windowsPaths.map(file => {
    const absolute = path.join(root, file);
    return [file, fs.existsSync(absolute) ? fs.readFileSync(absolute, "utf8") : ""];
  }));
  const windowsShellBlockers = inspectWindowsShellSecurity(windowsSources);
  const counts = Object.fromEntries(["rs","ts","tsx","kt","go","js","mjs","sh","yml","yaml","kts","py"]
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
    windowsShellBlockers,
    findings,
    passed:offenders.length===0 && missingGuards.length===0 && platformConfigBlockers.length===0 && windowsShellBlockers.length===0
  };
}
if (process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  const report=audit();
  console.log(JSON.stringify(report,null,2));
  if (!report.passed) process.exitCode=1;
}
