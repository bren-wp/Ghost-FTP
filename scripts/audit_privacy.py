#!/usr/bin/env python3
"""Fail-closed Ghost FTP privacy and runtime network-policy audit."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RUNTIME_ROOTS = (ROOT / "cmd", ROOT / "internal")
FORBIDDEN_IMPORTS = {"net/http", "net/rpc", "net/smtp"}
FORBIDDEN_VENDOR_MARKERS = {
    "sentry.io",
    "google-analytics",
    "googletagmanager",
    "segment.io",
    "mixpanel",
    "amplitude",
    "posthog",
    "datadog",
    "newrelic",
    "bugsnag",
    "crashlytics",
    "appcenter",
    "telemetrydeck",
}
URL_RE = re.compile(r"https?://[^\s\"'`]+", re.IGNORECASE)


def fail(message: str) -> None:
    raise SystemExit("PRIVACY_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing {rel}")
    try:
        return path.read_text(encoding="utf-8")
    except UnicodeError as exc:
        fail(f"{rel} is not valid UTF-8: {exc}")


def require(rel: str, markers: tuple[str, ...]) -> str:
    text = read(rel)
    for marker in markers:
        if marker not in text:
            fail(f"{rel} is missing privacy guard: {marker}")
    return text


def require_pattern(rel: str, pattern: str, invariant: str) -> str:
    text = read(rel)
    if re.search(pattern, text) is None:
        fail(f"{rel} is missing privacy guard: {invariant}")
    return text


def audit_runtime_sources() -> None:
    runtime_files: list[Path] = []
    for base in RUNTIME_ROOTS:
        runtime_files.extend(path for path in base.rglob("*.go") if not path.name.endswith("_test.go"))
    if not runtime_files:
        fail("runtime Go source was not found")

    for path in sorted(runtime_files):
        text = path.read_text(encoding="utf-8")
        rel = path.relative_to(ROOT)
        for imp in FORBIDDEN_IMPORTS:
            if re.search(rf'["`]{re.escape(imp)}["`]', text):
                fail(f"forbidden network import {imp!r} in {rel}")
        urls = sorted(set(URL_RE.findall(text)))
        if urls:
            fail(f"fixed HTTP(S) URL found in runtime source {rel}: {urls[0]}")
        lower = text.lower()
        for marker in FORBIDDEN_VENDOR_MARKERS:
            if marker in lower:
                fail(f"telemetry/vendor marker {marker!r} found in {rel}")
        if rel.as_posix() != "internal/remote/util.go" and "toolError{" in text:
            fail(f"runtime toolError must be constructed through newToolError: {rel}")


def audit_credentials_and_network_tools() -> None:
    gomod = read("go.mod")
    if re.search(r"(?m)^\s*(require|replace|exclude|retract)\b", gomod):
        fail("go.mod must remain free of external Go dependencies")

    engine = require("internal/api/engine.go", ("func (e *Engine) SaveProfile(", "func (e *Engine) Connect("))
    if "encoding/json" in engine or "func (e *Engine) Call(" in engine:
        fail("engine must not expose a generic JSON dispatcher")
    if '"log"' in engine or "log.New(" in engine:
        fail("persistent runtime logging must remain disabled")

    require("internal/platform/windows.go", ("SetErrorMode", "semNoGPFaultErrorBox", "SetDllDirectoryW", 'UTF16PtrFromString("")', "ofnDontAddToRecent"))
    require("internal/config/profile_crypto_windows.go", ("security.ProtectBytes", "security.UnprotectBytes"))

    manager = require("internal/remote/manager.go", ("PasswordBlob", "PassphraseBlob", "newCurlFTPWithProtectedSecret", "newSFTPWithProtectedSecrets"))
    if "UnprotectString" in manager or "UnprotectBytes" in manager:
        fail("connection manager must not decrypt saved credentials early")

    curl = require("internal/remote/curl_ftp.go", (
        '"-q", "--config", "-"', '"proxy = \\\"\\\""', '"noproxy = \\\"*\\\""',
        "sanitizedToolEnv(os.Environ())", "security.ProtectRuntimeString(password)",
        "security.UnprotectRuntimeBytes(c.passwordBlob)", "security.ForgetRuntimeSecret(c.passwordBlob)",
        "prepareLocalDownloadTarget(local, options.LocalRoot, options.SkipExisting)",
    ))
    if "HTTP_PROXY" in curl or "HTTPS_PROXY" in curl:
        fail("CurlFTP must not directly inherit proxy variables")
    if "ssl-no-revoke" in curl:
        fail("FTPS must not fully disable certificate-revocation checking")

    sftp = require("internal/remote/sftp.go", (
        "createSSHSessionConfig", '"  ProxyCommand none"', '"  ProxyJump none"',
        '"  GlobalKnownHostsFile none"', '"  VerifyHostKeyDNS no"', '"  UpdateHostKeys no"',
        '"  IdentityAgent none"', '"  ClearAllForwardings yes"', '"  ForwardAgent no"',
        "GhostFTP_ASKPASS_TOKEN=", "GhostFTP_PASSWORD_BLOB=", "GhostFTP_PASSPHRASE_BLOB=",
        "sanitizedToolEnv(os.Environ())", "sshKeyscanFailure(err, er.String())",
        "prepareLocalDownloadTarget(local, options.LocalRoot, options.SkipExisting)",
    ))
    for forbidden in ("GhostFTP_ASKPASS_FILE", "askpassFile", "os.WriteFile(askpass"):
        if forbidden in sftp:
            fail(f"SFTP must not write AskPass secrets to disk: {forbidden}")
    if 'fmt.Errorf("nije moguće dohvatiti SFTP host ključ: %s"' in sftp:
        fail("ssh-keyscan stderr must not be copied into a user-facing error")

    keyscan_error = require("internal/remote/ssh_keyscan_error.go", (
        "func sshKeyscanFailure(runErr error, diagnostic string) error",
        'newToolError("sftp", runErr, diagnostic)',
    ))
    if 'fmt.Errorf("nije moguće dohvatiti SFTP host ključ: %s"' in keyscan_error:
        fail("ssh-keyscan helper must preserve the redacted tool-error boundary")

    require("cmd/ghostftp/main.go", ("GhostFTP_ASKPASS_TOKEN", "GhostFTP_PASSWORD_BLOB", "GhostFTP_PASSPHRASE_BLOB", "TrustedAskPassParent", "selectAskpassSecret"))
    util = require("internal/remote/util.go", (
        '"http_proxy"', '"https_proxy"', '"ftp_proxy"', '"all_proxy"', '"no_proxy"', '"sslkeylogfile"',
        '"ssh_askpass"', '"ssh_auth_sock"', "crypto/rand", "func randomTransferToken()",
        "func (e *toolError) Error() string", "func toolErrorPublicLabel(tool string) string",
        "func toolErrorPublicDetail(kind string) string", "e.UserErrorKind()",
        'return "network tool"', "kind      string", "retryable bool",
        "classifyToolDiagnostic(tool, code, diagnostic)", "classifyToolRetryable(tool, code, diagnostic)",
        "return te.retryable",
    ))
    tool_diagnostics = require("internal/remote/tool_diagnostics.go", (
        "func classifyToolDiagnostic(tool string, code int, diagnostic string) string",
        "func classifyToolRetryable(tool string, code int, diagnostic string) bool",
        "return e.kind",
    ))
    tool_error_match = re.search(r"type toolError struct \{(?P<body>.*?)\n\}", util, re.DOTALL)
    if tool_error_match is None:
        fail("toolError structure was not found")
    if "message" in tool_error_match.group("body") or "diagnostic" in tool_error_match.group("body"):
        fail("toolError must not retain raw child-process diagnostic text")
    for forbidden in ("return e.message", "base = label + e.message", 'fmt.Sprintf("%s %s", label, e.message)', "te.message", "e.message"):
        if forbidden in util or forbidden in tool_diagnostics:
            fail(f"raw child-process diagnostic retention/exposure must remain blocked: {forbidden}")

    require("internal/transfer/manager.go", (
        "recover() != nil",
        "ConnectionIdentity() (string, error)",
        "localRoot := job.LocalRoot",
        'job.Direction == "download" && localRoot == ""',
        "security.EnsureLocalWithinRoot(localRoot, job.LocalPath)",
        "remote.IsRetryable(err)",
    ))
    require_pattern(
        "internal/transfer/manager.go",
        r"\bLocalRoot:\s+localRoot,",
        "TransferOptions.LocalRoot propagates the validated localRoot",
    )
    require("internal/remote/local_download_root.go", (
        "os.OpenRoot(rootPath)",
        "security.EnsureLocalWithinRoot(d.rootPath, d.targetPath)",
        "d.root.Rename(d.partName, d.targetRel)",
    ))
    require("internal/localfs/service.go", ("security.IsReparsePoint", "platform.RenameNoReplace"))
    require("internal/security/remove_tree.go", ("func RemoveTreeNoFollow(", "isReparsePoint", "os.ModeSymlink"))

    installer = read("cmd/installer/main.go")
    for marker in ("URLInfoAbout", "HelpLink"):
        if marker in installer:
            fail(f"installer contains external URL hook {marker}")


def audit_build_privacy() -> None:
    ci = require(".github/workflows/ci.yml", ("go telemetry off", "GOPROXY: off", "GOSUMDB: off"))
    release = require(".github/workflows/release.yml", ("go telemetry off", "GOPROXY: off", "GOSUMDB: off", "test \"$(go telemetry)\" = 'off'"))
    for rel, text in ((".github/workflows/ci.yml", ci), (".github/workflows/release.yml", release)):
        if re.search(r"(?m)^\s*GOTELEMETRY:\s*off\s*$", text):
            fail(f"{rel} uses ineffective GOTELEMETRY env-only guard instead of `go telemetry off`")

    build_contracts = {
        "BUILD-WINDOWS.ps1": ("$telemetryMode = Invoke-NativeCapture", "Go telemetry must be disabled before a production build"),
        "linux/BUILD.sh": ('telemetry="$(go telemetry)"', "Go telemetry must be disabled before a production build"),
        "scripts/BUILD-LOCAL.sh": ('GO_TELEMETRY="$(go telemetry)"', "Go telemetry must be disabled before a production build"),
    }
    for rel, markers in build_contracts.items():
        text = require(rel, markers)
        if "GOTELEMETRY=off" in text or "$env:GOTELEMETRY = 'off'" in text:
            fail(f"{rel} relies on an ineffective GOTELEMETRY environment variable")

    for legacy in ("scripts/BUILD-LINUX.sh", "scripts/BUILD-MACOS.sh", "scripts/BUILD-IOS.sh"):
        if (ROOT / legacy).exists():
            fail(f"obsolete platform build wrapper still exists: {legacy}")


def main() -> None:
    audit_runtime_sources()
    audit_credentials_and_network_tools()
    audit_build_privacy()
    print("PRIVACY_AUDIT=PASS")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("FIXED_RUNTIME_HTTP_URLS=BLOCKED")
    print("TELEMETRY_VENDOR_MARKERS=BLOCKED")
    print("RUNTIME_CREDENTIAL_FILES=BLOCKED")
    print("RAW_TOOL_DIAGNOSTICS_USER_SURFACE=BLOCKED")
    print("RAW_TOOL_DIAGNOSTICS_ERROR_RETENTION=BLOCKED")
    print("SAFE_TOOL_ERROR_CLASSIFICATION=PRESERVED")
    print("SSH_KEYSCAN_DIAGNOSTICS_USER_SURFACE=REDACTED")
    print("DOWNLOAD_LOCAL_ROOT_PROPAGATION=ENFORCED")
    print("DOWNLOAD_ROOT_RELATIVE_COMMIT=ENFORCED")


if __name__ == "__main__":
    main()
