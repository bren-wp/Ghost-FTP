#!/usr/bin/env python3
"""Fail closed when ByFTP privacy or network-policy invariants regress."""

from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RUNTIME_ROOTS = [ROOT / "cmd", ROOT / "internal"]
FORBIDDEN_IMPORTS = {"net/http", "net/rpc", "net/smtp"}
FORBIDDEN_VENDOR_MARKERS = {
    "sentry.io", "google-analytics", "googletagmanager", "segment.io", "mixpanel",
    "amplitude", "posthog", "datadog", "newrelic", "bugsnag", "crashlytics",
    "appcenter", "telemetrydeck",
}


def fail(message: str) -> None:
    raise SystemExit("PRIVACY_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing {rel}")
    return path.read_text(encoding="utf-8")


def require(rel: str, markers: tuple[str, ...]) -> str:
    text = read(rel)
    for marker in markers:
        if marker not in text:
            fail(f"{rel} is missing required privacy invariant: {marker}")
    return text


def main() -> None:
    runtime_files: list[Path] = []
    for base in RUNTIME_ROOTS:
        runtime_files.extend(path for path in base.rglob("*.go") if not path.name.endswith("_test.go"))
    if not runtime_files:
        fail("runtime Go source was not found")

    for path in sorted(runtime_files):
        text = path.read_text(encoding="utf-8")
        rel = path.relative_to(ROOT)
        for imp in FORBIDDEN_IMPORTS:
            if re.search(rf'["`]'+re.escape(imp)+r'["`]', text):
                fail(f"forbidden network import {imp!r} in {rel}")
        if re.search(r"https?://", text, re.IGNORECASE):
            fail(f"fixed HTTP(S) URL found in runtime source: {rel}")
        lower = text.lower()
        for marker in FORBIDDEN_VENDOR_MARKERS:
            if marker in lower:
                fail(f"telemetry/vendor marker {marker!r} found in {rel}")

    gomod = read("go.mod")
    if re.search(r"(?m)^\s*(require|replace|exclude|retract)\b", gomod):
        fail("go.mod must remain free of external Go module dependencies")

    engine = require("internal/api/engine.go", ('func (e *Engine) SaveProfile(', 'func (e *Engine) Connect('))
    if "encoding/json" in engine or "func (e *Engine) Call(" in engine:
        fail("engine must not expose a generic JSON dispatcher")
    if '"log"' in engine or "log.New(" in engine:
        fail("persistent runtime logging must remain disabled")

    require("internal/platform/windows.go", ("SetErrorMode", "semNoGPFaultErrorBox", "SetDllDirectoryW", 'UTF16PtrFromString("")', "ofnDontAddToRecent"))
    require("internal/config/profile_crypto_windows.go", ("security.ProtectBytes", "security.UnprotectBytes"))

    manager = require("internal/remote/manager.go", ("PasswordBlob", "PassphraseBlob", "newCurlFTPWithProtectedSecret", "newSFTPWithProtectedSecrets"))
    if "UnprotectString" in manager or "UnprotectBytes" in manager:
        fail("connection manager must not decrypt stored credentials early")

    curl = require(
        "internal/remote/curl_ftp.go",
        (
            '"-q", "--config", "-"',
            '"proxy = \\\"\\\""',
            '"noproxy = \\\"*\\\""',
            "sanitizedToolEnv(os.Environ())",
            "security.ProtectRuntimeString(password)",
            "security.UnprotectRuntimeBytes(c.passwordBlob)",
            "security.ForgetRuntimeSecret(c.passwordBlob)",
            'runtime.GOOS == "windows" && c.revokeBestEffort',
            '"ssl-revoke-best-effort"',
            "curlSupportsRevokeBestEffort(p)",
        ),
    )
    if "HTTP_PROXY" in curl or "HTTPS_PROXY" in curl:
        fail("CurlFTP must not inherit proxy variables directly")
    if "ssl-no-revoke" in curl:
        fail("FTPS must not disable certificate revocation checking")

    require(
        "internal/remote/curl_capability.go",
        (
            "func curlVersionSupportsRevokeBestEffort(",
            "func curlSupportsRevokeBestEffort(",
            'runtime.GOOS != "windows"',
            'exec.CommandContext(ctx, curlPath, "--version")',
            "context.WithTimeout(context.Background(), 3*time.Second)",
            "newBoundedOutput(maxCurlVersionOutput)",
            "sanitizedToolEnv(os.Environ())",
        ),
    )

    sftp = require(
        "internal/remote/sftp.go",
        (
            "createSSHSessionConfig", '"-F", s.sshConfig', "s.sessionHost",
            '"  ProxyCommand none"', '"  ProxyJump none"', '"  GlobalKnownHostsFile none"',
            '"  VerifyHostKeyDNS no"', '"  UpdateHostKeys no"', '"  IdentityAgent none"',
            '"  PKCS11Provider none"', '"  KnownHostsCommand none"', '"  PermitLocalCommand no"',
            '"  ClearAllForwardings yes"', '"  ForwardAgent no"', '"  ForwardX11 no"',
            '"  IdentitiesOnly yes"', '"  IdentityFile " + identity', '"  HostKeyAlgorithms "+hostKeyAlgorithm',
            '"-f", "-"', 'cmd.Stdin = strings.NewReader(scanHost + "\\n")',
            "BYFTP_ASKPASS_TOKEN=", "BYFTP_PASSWORD_BLOB=", "BYFTP_PASSPHRASE_BLOB=",
            "sanitizedToolEnv(os.Environ())",
        ),
    )
    for forbidden in ("BYFTP_ASKPASS_FILE", "askpassFile", "os.WriteFile(askpass"):
        if forbidden in sftp:
            fail(f"SFTP must not write AskPass secrets to disk: {forbidden}")

    askpass = require(
        "cmd/byftp/main.go",
        ("BYFTP_ASKPASS_TOKEN", "BYFTP_PASSWORD_BLOB", "BYFTP_PASSPHRASE_BLOB", "TrustedAskPassParent", "selectAskpassSecret"),
    )
    if "BYFTP_ASKPASS_FILE" in askpass:
        fail("AskPass must not depend on a disk credential artifact")

    require("internal/security/runtime_secret_windows.go", ("ProtectString", "UnprotectBytes", "ProtectRuntimeString"))
    require("internal/security/runtime_secret_other.go", ("crypto/rand", "runtimeValues", "WipeBytes(value)", "ForgetRuntimeSecret"))
    terminal = require(
        "internal/desktop/other.go",
        (
            "promptSecret", "stty", "engine.Connect",
            'if protocol == "sftp" {',
            "cfg.PrivateKeyPath = strings.TrimSpace(keyPath)",
            'i18n.T(language, "terminal.sftp_key_required")',
            'i18n.T(language, "terminal.sftp_passphrase_unsupported")',
        ),
    )
    if 'promptSecret(reader, language, "Password")' in terminal:
        fail("terminal source must not hard-code a plaintext credential prompt")

    require("internal/remote/util.go", (
        '"http_proxy"', '"https_proxy"', '"ftp_proxy"', '"all_proxy"', '"no_proxy"',
        '"sslkeylogfile"', '"curl_ssl_backend"', '"curl_ca_bundle"',
        '"ssh_askpass"', '"ssh_auth_sock"', '"ssh_sk_provider"', '"display"',
        "crypto/rand", "func randomTransferToken()",
    ))

    tools = require("internal/remote/tools.go", ("systemDirectory()", 'runtime.GOOS == "windows"'))
    if 'os.Getenv("WINDIR")' in tools or 'os.Getenv("WINDIR")' in sftp:
        fail("Windows network tools must not trust WINDIR")

    require("internal/transfer/manager.go", (
        "recover() != nil", "type operationProvider interface",
        "ConnectionIdentity() (string, error)", "jobConnections map[string]string",
        "security.EnsureLocalWithinRoot(job.LocalRoot, job.LocalPath)", "remote.IsRetryable(err)",
    ))
    require("internal/localfs/service.go", ("security.IsReparsePoint", "platform.RenameNoReplace"))
    require("cmd/byftp/main.go", ("security.EnsureNoRedirectDirectory(localAppData, dataDir)",))
    require("internal/security/remove_tree.go", ("func RemoveTreeNoFollow(", "isReparsePoint", "os.ModeSymlink"))
    require("internal/desktop/ui_windows.go", ("limitEdit(a.host, 253)", "limitEdit(a.user, 1024)", "limitEdit(a.pass, 8192)", "limitEdit(a.passphrase, 8192)", "limitEdit(a.remotePath, 4096)"))

    installer = read("cmd/installer/main.go")
    for marker in ("URLInfoAbout", "HelpLink"):
        if marker in installer:
            fail(f"installer contains external URL hook {marker}")

    # GOTELEMETRY is not a valid privacy control by itself. CI and release jobs
    # must execute `go telemetry off`, while production build scripts must refuse
    # to continue unless the Go tool reports telemetry mode `off`.
    ci = require(".github/workflows/ci.yml", ("go telemetry off", "Go telemetry is not disabled."))
    release = require(".github/workflows/release.yml", ("go telemetry off", "test \"$(go telemetry)\" = 'off'"))
    for rel, text in ((".github/workflows/ci.yml", ci), (".github/workflows/release.yml", release)):
        if re.search(r"(?m)^\s*GOTELEMETRY:\s*off\s*$", text):
            fail(f"{rel} uses ineffective GOTELEMETRY env assignment instead of `go telemetry off`")

    windows_build = require("BUILD-WINDOWS.ps1", ("$telemetryMode = (go telemetry).Trim()", "go telemetry off", "Go telemetry must be disabled"))
    linux_build = require("scripts/BUILD-LINUX.sh", ('telemetry="$(go telemetry)"', "go telemetry off", "Go telemetry must be disabled"))
    macos_build = require("scripts/BUILD-MACOS.sh", ('telemetry="$(go telemetry)"', "go telemetry off", "Go telemetry must be disabled"))
    local_build = require("scripts/BUILD-LOCAL.sh", ('GO_TELEMETRY="$(go telemetry)"', "go telemetry off", "Go telemetry must be disabled"))
    for rel, text in (
        ("BUILD-WINDOWS.ps1", windows_build),
        ("scripts/BUILD-LINUX.sh", linux_build),
        ("scripts/BUILD-MACOS.sh", macos_build),
        ("scripts/BUILD-LOCAL.sh", local_build),
    ):
        if "GOTELEMETRY=off" in text or "$env:GOTELEMETRY = 'off'" in text:
            fail(f"{rel} relies on ineffective GOTELEMETRY environment assignment")

    print("PRIVACY_AUDIT=PASS")
    print("TELEMETRY=ABSENT")
    print("GO_BUILD_TELEMETRY=OFF_REQUIRED_AND_CI_VERIFIED")
    print("FIXED_HTTP_API_ENDPOINTS=ABSENT")
    print("EXTERNAL_GO_MODULES=ABSENT")
    print("CURL_EXTERNAL_ENV_INHERITANCE=DISABLED")
    print("FTPS_CERTIFICATE_REVOCATION=NOT_DISABLED")
    print("WINDOWS_FTPS_REVOCATION=CAPABILITY_GATED_BEST_EFFORT")
    print("OPENSSH_USER_CONFIG_AND_HELPER_INHERITANCE=DISABLED")
    print("WINDOWS_ERROR_REPORTING=DISABLED_FOR_BYFTP_PROCESS")
    print("PERSISTENT_RUNTIME_LOGGING=DISABLED")
    print("SAVED_PROFILE_METADATA=WINDOWS_DPAPI_PROTECTED")
    print("ACTIVE_RUNTIME_SECRETS=WINDOWS_DPAPI_OR_PROCESS_MEMORY")
    print("ASKPASS_DISK_SECRET_ARTIFACTS=DISABLED")
    print("NETWORK_POLICY=NO_TELEMETRY_NO_EXTERNAL_APIS_USER_SELECTED_SERVER_PLUS_OS_CERTIFICATE_VALIDATION")


if __name__ == "__main__":
    main()
