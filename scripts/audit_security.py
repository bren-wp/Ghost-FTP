#!/usr/bin/env python3
"""Validate Ghost FTP security invariants that must not regress."""

from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("SECURITY_AUDIT_FAILED: " + message)


def read(path: str) -> str:
    target = ROOT / path
    if not target.is_file():
        fail(f"missing {path}")
    return target.read_text(encoding="utf-8")


def require(path: str, markers: tuple[str, ...]) -> str:
    text = read(path)
    for marker in markers:
        if marker not in text:
            fail(f"{path} is missing required security invariant: {marker}")
    return text


def main() -> int:
    # Download staging and filesystem traversal protection.
    require("internal/remote/util.go", (
        "func validateDownloadedPart(part string) error",
        "os.Lstat(part)",
        "security.IsReparsePoint(part)",
        "func randomTransferToken()",
        "crypto/rand",
    ))
    for path in ("internal/remote/curl_ftp.go", "internal/remote/sftp.go"):
        require(path, ("validateDownloadedPart(part)",))

    require("internal/security/remove_tree.go", (
        "func RemoveTreeNoFollow(",
        "func isFilesystemRoot(target string) bool",
        "isFilesystemRoot(root)",
        "maxRemoveTreeDepth",
        "maxRemoveTreeItems",
        "isReparsePoint(target)",
        "os.ModeSymlink",
    ))
    require("internal/localfs/service.go", ("security.IsReparsePoint", "platform.RenameNoReplace"))
    require("internal/platform/filemove_windows.go", ("MoveFileExW", "moveFileWriteThrough", "RenameNoReplace"))

    # SFTP trust, AskPass and private-key handling.
    sftp = require("internal/remote/sftp.go", (
        "func validatePrivateKeyPath(keyPath string) error",
        "os.Lstat(keyPath)",
        "security.IsReparsePoint(keyPath)",
        '"-oBatchMode=no"',
        "createSSHSessionConfig",
        '"  ProxyCommand none"',
        '"  ProxyJump none"',
        '"  GlobalKnownHostsFile none"',
        '"  VerifyHostKeyDNS no"',
        '"  UpdateHostKeys no"',
        '"  IdentityAgent none"',
        '"  ClearAllForwardings yes"',
        '"  ForwardAgent no"',
        "GhostFTP_ASKPASS_TOKEN=",
        "GhostFTP_PASSWORD_BLOB=",
        "GhostFTP_PASSPHRASE_BLOB=",
        "sanitizedToolEnv(os.Environ())",
    ))
    if '"-b"' in sftp or '"-b", "-"' in sftp:
        fail("SFTP command args use -b; OpenSSH would force BatchMode=yes and disable AskPass")
    for forbidden in ("GhostFTP_ASKPASS_FILE", "askpassFile", "os.WriteFile(askpass"):
        if forbidden in sftp:
            fail(f"SFTP must not write AskPass secrets to disk: {forbidden}")

    askpass = require("cmd/ghostftp/main.go", (
        "GhostFTP_ASKPASS_TOKEN",
        "GhostFTP_PASSWORD_BLOB",
        "GhostFTP_PASSPHRASE_BLOB",
        "TrustedAskPassParent",
        "selectAskpassSecret",
        "clearAskpassEnvironment()",
    ))
    if "GhostFTP_ASKPASS_FILE" in askpass:
        fail("AskPass must not depend on a disk credential artifact")

    # FTP/FTPS credentials and proxy isolation.
    curl = require("internal/remote/curl_ftp.go", (
        "security.ProtectRuntimeString(password)",
        "security.UnprotectRuntimeBytes(c.passwordBlob)",
        "security.ForgetRuntimeSecret(c.passwordBlob)",
        '"-q", "--config", "-"',
        '"proxy = \\\"\\\""',
        '"noproxy = \\\"*\\\""',
        "sanitizedToolEnv(os.Environ())",
    ))
    if "HTTP_PROXY" in curl or "HTTPS_PROXY" in curl:
        fail("CurlFTP must not directly inherit proxy variables")
    if "ssl-no-revoke" in curl:
        fail("FTPS must not fully disable certificate-revocation checking")

    # Secret lifetime and profile binding.
    manager = require("internal/remote/manager.go", (
        "PasswordBlob",
        "PassphraseBlob",
        "newCurlFTPWithProtectedSecret",
        "newSFTPWithProtectedSecrets",
        "profilebinding.EndpointMatches",
        "profilebinding.AccountMatches",
        "profilebinding.PrivateKeyMatches",
        "ErrSessionClosing",
        "ErrDisconnectTimeout",
        "activeOps     sync.WaitGroup",
        "m.activeOps.Wait()",
        "m.activeOps.Done()",
    ))
    if "UnprotectString" in manager or "UnprotectBytes" in manager:
        fail("connection manager must not decrypt saved credentials early")

    require("internal/security/runtime_secret_windows.go", ("ProtectRuntimeString", "UnprotectRuntimeBytes", "ForgetRuntimeSecret"))
    require("internal/security/runtime_secret_other.go", ("crypto/rand", "runtimeValues", "WipeBytes(value)", "ForgetRuntimeSecret"))
    require("internal/config/profile_crypto_windows.go", ("security.ProtectBytes", "security.UnprotectBytes"))
    require("internal/profilebinding/binding.go", ("func EndpointMatches(", "func AccountMatches(", "func PrivateKeyMatches("))

    # Raw-input validation and conservative transfer behavior.
    require("internal/desktop/connection_input.go", (
        "func validateRawConnectionInput(",
        "strconv.Atoi(portText)",
        "security.ValidateConnection(protocol, host, username, port)",
    ))
    require("internal/transfer/manager.go", (
        "security.EnsureLocalWithinRoot(job.LocalRoot, job.LocalPath)",
        "remote.IsRetryable(err)",
        "errors.Is(err, remote.ErrSkipped)",
        "ConnectionIdentity() (string, error)",
    ))
    require("internal/config/store.go", ("os.Lstat(path)", "os.SameFile(before, after)", "io.LimitReader", "os.CreateTemp"))

    # Windows process hardening and Linux authentication/queue parity.
    require("internal/platform/windows.go", (
        "SetErrorMode",
        "semNoGPFaultErrorBox",
        "SetDllDirectoryW",
        'UTF16PtrFromString("")',
        "ofnDontAddToRecent",
    ))
    linux_ui = require("internal/desktop/other.go", (
        "//go:build linux",
        "cfg.Password = password",
        "cfg.Passphrase = passphrase",
        "engine.PauseTransfers()",
        "engine.ResumeTransfers()",
        "engine.CancelTransfer(fields[1])",
        "engine.RetryTransfer(fields[1])",
        "engine.ClearFinishedTransfers()",
        "engine.SetSettings(next)",
        "engine.Profiles()",
        "engine.Connect",
        "engine.RemoteList",
        "engine.AddTransfer",
    ))
    for obsolete in ("terminal.sftp_key_required", "terminal.sftp_passphrase_unsupported"):
        if obsolete in linux_ui:
            fail(f"Linux frontend still enforces obsolete SFTP restriction: {obsolete}")

    # Regression-test anchors for high-risk behavior.
    require("cmd/ghostftp/askpass_test.go", ("TestSelectAskpassSecret", "Verification code", "One-time password token"))
    require("internal/remote/sftp_stream_test.go", ("TestSFTPCommandArgsKeepAskPassEnabled", "sftp -b"))
    require("internal/remote/private_key_validation_test.go", ("TestValidatePrivateKeyPathAcceptsRegularFile", "TestValidatePrivateKeyPathRejectsSymlink"))
    require("internal/remote/manager_test.go", ("TestDisconnectWaitsForActiveOperationRelease", "TestDisconnectTimeoutDefersCloseAndBlocksReconnect"))
    require("internal/transfer/finish_status_test.go", ("TestFinishJobKeepsSuccessfulResultWhenCancelArrivesAfterSuccess", "TestFinishJobMarksActualCancellation"))
    require("internal/security/remove_tree_root_test.go", ("RemoveTreeNoFollow",))
    require("internal/security/remove_tree_root_windows_test.go", ("TestIsFilesystemRootRejectsWindowsVolumeRoots",))

    print("SECURITY_AUDIT=PASS")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("SFTP_PASSWORD_AUTH_LINUX=ENABLED")
    print("SFTP_KEY_PASSPHRASE_LINUX=ENABLED")
    print("SFTP_ASKPASS_BATCHMODE_CONFLICT=BLOCKED")
    print("RUNTIME_CREDENTIAL_FILES=BLOCKED")
    print("PROFILE_CREDENTIAL_CROSS_ENDPOINT=BLOCKED")
    print("DOWNLOAD_STAGING_REPARSE_VALIDATION=ENABLED")
    print("SFTP_PRIVATE_KEY_REPARSE=BLOCKED")
    print("REMOTE_SESSION_CLOSE_RACE=BLOCKED")
    print("FILESYSTEM_ROOT_DELETE=BLOCKED")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
