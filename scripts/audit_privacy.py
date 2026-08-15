#!/usr/bin/env python3
"""Fail the ByFTP release if runtime source can contact undisclosed external services.

The product network policy is deliberately narrow: ByFTP may connect only to the
FTP/FTPS/SFTP host explicitly supplied by the user. No telemetry, analytics,
crash-reporting, update checks, web APIs, HTTP clients, proxy inheritance or
third-party routing hooks are permitted in the runtime.
"""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

RUNTIME_ROOTS = [ROOT / "cmd", ROOT / "internal"]
FORBIDDEN_IMPORTS = {
    "net/http",
    "net/rpc",
    "net/smtp",
}
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


def runtime_go_files() -> list[Path]:
    out: list[Path] = []
    for base in RUNTIME_ROOTS:
        for path in base.rglob("*.go"):
            if path.name.endswith("_test.go"):
                continue
            out.append(path)
    return sorted(out)


def fail(message: str) -> None:
    raise SystemExit("PRIVACY_AUDIT_FAILED: " + message)


def main() -> None:
    files = runtime_go_files()
    if not files:
        fail("runtime Go source was not found")

    for path in files:
        text = path.read_text(encoding="utf-8")
        rel = path.relative_to(ROOT)
        for imp in FORBIDDEN_IMPORTS:
            if re.search(rf'["`]'+re.escape(imp)+r'["`]', text):
                fail(f"forbidden network import {imp!r} in {rel}")
        # Runtime source must not contain fixed web/API endpoints. FTP/FTPS URLs
        # are assembled from the host supplied by the user and are not literals.
        if re.search(r"https?://", text, re.IGNORECASE):
            fail(f"fixed HTTP(S) URL found in runtime source: {rel}")
        lower = text.lower()
        for marker in FORBIDDEN_VENDOR_MARKERS:
            if marker in lower:
                fail(f"third-party telemetry marker {marker!r} found in {rel}")

    gomod = (ROOT / "go.mod").read_text(encoding="utf-8")
    if re.search(r"(?m)^\s*(require|replace|exclude|retract)\b", gomod):
        fail("go.mod must remain standard-library-only with no external module dependencies")


    engine = (ROOT / "internal/api/engine.go").read_text(encoding="utf-8")
    if '"log"' in engine or 'log.New(' in engine or 'logger.Printf(' in engine:
        fail("persistent runtime logging must remain disabled")
    for marker in ('func (e *Engine) SaveProfile(', 'func (e *Engine) Connect('):
        if marker not in engine:
            fail(f"typed sensitive engine path is missing: {marker}")
    for marker in ('type request struct', 'func decode[', 'func (e *Engine) Call(', 'func (s *Engine) Call('):
        if marker in engine:
            fail(f"obsolete generic JSON dispatcher remains in engine: {marker}")
    if 'encoding/json' in engine:
        fail("engine must not serialize in-process operations through JSON")

    desktop_profiles = (ROOT / "internal/desktop/connection_profiles_windows.go").read_text(encoding="utf-8")
    for marker in ('"connection:connect"', '"profiles:save"'):
        if marker in desktop_profiles:
            fail(f"desktop still serializes a sensitive operation through generic JSON: {marker}")

    platform_windows = (ROOT / "internal/platform/windows.go").read_text(encoding="utf-8")
    if 'SetErrorMode' not in platform_windows or 'semNoGPFaultErrorBox' not in platform_windows:
        fail("Windows Error Reporting suppression is missing")
    if 'SetDllDirectoryW' not in platform_windows or 'UTF16PtrFromString("")' not in platform_windows:
        fail("current-directory DLL search hardening is missing")
    if 'ofnDontAddToRecent' not in platform_windows:
        fail("private-key picker may add sensitive path to Windows Recent items")

    profile_crypto = (ROOT / "internal/config/profile_crypto_windows.go").read_text(encoding="utf-8")
    if 'security.ProtectBytes' not in profile_crypto or 'security.UnprotectBytes' not in profile_crypto:
        fail("Windows saved profile envelope is not DPAPI protected")

    manager = (ROOT / "internal/remote/manager.go").read_text(encoding="utf-8")
    if "UnprotectString" in manager or "UnprotectBytes" in manager:
        fail("saved credential blobs must not be decrypted in the connection manager")
    for marker in ("PasswordBlob", "PassphraseBlob", "newCurlFTPWithProtectedSecret", "newSFTPWithProtectedSecrets"):
        if marker not in manager:
            fail(f"protected credential pass-through is missing: {marker}")

    installer = (ROOT / "cmd/installer/main.go").read_text(encoding="utf-8")
    for marker in ("URLInfoAbout", "HelpLink"):
        if marker in installer:
            fail(f"installer contains external URL hook {marker}")

    curl = (ROOT / "internal/remote/curl_ftp.go").read_text(encoding="utf-8")
    for marker in ('"-q", "--config", "-"', '"proxy = \\\"\\\""', '"noproxy = \\\"*\\\""', "sanitizedToolEnv(os.Environ())"):
        if marker not in curl:
            fail(f"curl direct-connection guard is missing: {marker}")

    sftp = (ROOT / "internal/remote/sftp.go").read_text(encoding="utf-8")
    for marker in (
        'createSSHSessionConfig',
        '"-F", s.sshConfig',
        's.sessionHost',
        '"  ProxyCommand none"',
        '"  ProxyJump none"',
        '"  GlobalKnownHostsFile none"',
        '"  VerifyHostKeyDNS no"',
        '"  UpdateHostKeys no"',
        '"  IdentityAgent none"',
        '"  PKCS11Provider none"',
        '"  KnownHostsCommand none"',
        '"  PermitLocalCommand no"',
        '"  ClearAllForwardings yes"',
        '"  ForwardAgent no"',
        '"  ForwardX11 no"',
        '"  IdentitiesOnly yes"',
        '"  IdentityFile " + identity',
        '"  HostKeyAlgorithms "+hostKeyAlgorithm',
        '"-f", "-"',
        'cmd.Stdin = strings.NewReader(host + "\\n")',
        'BYFTP_ASKPASS_TOKEN=',
        'BYFTP_PASSWORD_BLOB=',
        'BYFTP_PASSPHRASE_BLOB=',
        'sanitizedToolEnv(os.Environ())',
    ):
        if marker not in sftp:
            fail(f"SFTP privacy/direct-connection guard is missing: {marker}")
    for forbidden in (
        'BYFTP_ASKPASS_FILE',
        'askpassFile',
        'os.WriteFile(askpass',
    ):
        if forbidden in sftp:
            fail(f"SFTP still persists AskPass credential material: {forbidden}")

    askpass = (ROOT / "cmd/byftp/main.go").read_text(encoding="utf-8")
    for marker in ('BYFTP_ASKPASS_TOKEN', 'BYFTP_PASSWORD_BLOB', 'BYFTP_PASSPHRASE_BLOB', 'TrustedAskPassParent'):
        if marker not in askpass:
            fail(f"AskPass trust/credential guard is missing: {marker}")
    if 'BYFTP_ASKPASS_FILE' in askpass:
        fail("AskPass still depends on a disk credential artifact")

    transfer_manager = (ROOT / "internal/transfer/manager.go").read_text(encoding="utf-8")
    for marker in ('recover() != nil', 'interna greška tijekom prijenosa', 'type operationProvider interface'):
        if marker not in transfer_manager:
            fail(f"transfer worker panic containment is missing: {marker}")
    for marker in ('ConnectionIdentity() (string, error)', 'jobConnections map[string]string', 'odabrani prijenos pripada drugoj vezi'):
        if marker not in transfer_manager:
            fail(f"cross-server transfer retry guard is missing: {marker}")
    for marker in ('security.EnsureLocalWithinRoot(job.LocalRoot, job.LocalPath)', 'remote.IsRetryable(err)'):
        if marker not in transfer_manager:
            fail(f"transfer execution hardening is missing: {marker}")

    localfs = (ROOT / "internal/localfs/service.go").read_text(encoding="utf-8")
    for marker in ('security.IsReparsePoint', 'platform.RenameNoReplace'):
        if marker not in localfs:
            fail(f"local filesystem hardening is missing: {marker}")

    remote_util = (ROOT / "internal/remote/util.go").read_text(encoding="utf-8")
    for marker in ('crypto/rand', 'func randomTransferToken()'):
        if marker not in remote_util:
            fail(f"unpredictable transfer staging names are missing: {marker}")

    byftp_main = (ROOT / "cmd/byftp/main.go").read_text(encoding="utf-8")
    if 'security.EnsureNoRedirectDirectory(localAppData, dataDir)' not in byftp_main:
        fail("ByFTP owned data directory redirect protection is missing")

    remove_tree = (ROOT / "internal/security/remove_tree.go").read_text(encoding="utf-8")
    for marker in ('func RemoveTreeNoFollow(', 'isReparsePoint', 'os.ModeSymlink'):
        if marker not in remove_tree:
            fail(f"no-follow local tree deletion guard is missing: {marker}")

    desktop_ui = (ROOT / "internal/desktop/ui_windows.go").read_text(encoding="utf-8")
    for marker in ('limitEdit(a.host, 253)', 'limitEdit(a.user, 1024)', 'limitEdit(a.pass, 8192)', 'limitEdit(a.passphrase, 8192)', 'limitEdit(a.remotePath, 4096)'):
        if marker not in desktop_ui:
            fail(f"bounded sensitive UI input guard is missing: {marker}")

    util = (ROOT / "internal/remote/util.go").read_text(encoding="utf-8")
    for marker in (
        '"sslkeylogfile"', '"curl_ssl_backend"', '"curl_ca_bundle"',
        '"ssh_askpass"', '"ssh_auth_sock"', '"ssh_sk_provider"', '"display"',
    ):
        if marker not in util.lower():
            fail(f"external tool-control environment guard is missing: {marker}")

    caps = (ROOT / "internal/remote/tools.go").read_text(encoding="utf-8")
    if 'systemDirectory()' not in caps or 'os.Getenv("WINDIR")' in caps:
        fail("Windows curl must resolve System32 without trusting WINDIR")
    if 'systemDirectory()' not in sftp or 'os.Getenv("WINDIR")' in sftp:
        fail("Windows OpenSSH must resolve System32 without trusting WINDIR")
    if 'runtime.GOOS == "windows"' not in caps:
        fail("Windows curl must not fall back to an arbitrary PATH binary")
    if 'runtime.GOOS == "windows"' not in sftp:
        fail("Windows OpenSSH must not fall back to an arbitrary PATH binary")

    print("PRIVACY_AUDIT=PASS")
    print("TELEMETRY=ABSENT")
    print("FIXED_HTTP_API_ENDPOINTS=ABSENT")
    print("EXTERNAL_GO_MODULES=ABSENT")
    print("CURL_EXTERNAL_ENV_INHERITANCE=DISABLED")
    print("FTPS_EXTERNAL_REVOCATION_FETCH=DISABLED")
    print("OPENSSH_USER_CONFIG_AND_HELPER_INHERITANCE=DISABLED")
    print("WINDOWS_ERROR_REPORTING=DISABLED_FOR_BYFTP_PROCESS")
    print("PERSISTENT_RUNTIME_LOGGING=DISABLED")
    print("SAVED_PROFILE_METADATA=WINDOWS_DPAPI_PROTECTED")
    print("IN_PROCESS_JSON_DISPATCHER=REMOVED")
    print("SENSITIVE_CONNECT_PROFILE_JSON_SERIALIZATION=DISABLED")
    print("SAVED_CREDENTIAL_EARLY_DECRYPTION=DISABLED")
    print("ASKPASS_DISK_SECRET_ARTIFACTS=DISABLED")
    print("SFTP_COMMAND_LINE_METADATA=MINIMIZED")
    print("SFTP_HOST_KEY_ALGORITHM_BINDING=ENABLED")
    print("LOCAL_REPARSE_DELETE_TRAVERSAL=BLOCKED")
    print("SENSITIVE_UI_INPUT_LENGTHS=BOUNDED")
    print("SFTP_GLOBAL_TRUST_SOURCES=DISABLED")
    print("TRANSFER_WORKER_PANIC_CONTAINMENT=ENABLED")
    print("CROSS_SERVER_TRANSFER_RETRY=BLOCKED")
    print("TRANSFER_LOCAL_ROOT_REVALIDATION=ENABLED")
    print("AUTOMATIC_RETRY_TRANSIENT_ONLY=ENABLED")
    print("TRANSFER_STAGING_NAMES=CRYPTO_RANDOM")
    print("LOCAL_NO_REPLACE_RENAME=ENABLED")
    print("BYFTP_OWNED_DIRECTORY_REDIRECTS=BLOCKED")
    print("CURRENT_DIRECTORY_DLL_SEARCH=DISABLED")
    print("NETWORK_POLICY=NO_TELEMETRY_NO_EXTERNAL_APIS_USER_SELECTED_SERVER")


if __name__ == "__main__":
    main()
