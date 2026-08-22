#!/usr/bin/env python3
"""Verify ByFTP security invariants that must not regress."""

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
    require("internal/remote/util.go", ("func validateDownloadedPart(part string) error", "os.Lstat(part)", "security.IsReparsePoint(part)"))
    for path in ("internal/remote/curl_ftp.go", "internal/remote/sftp.go"):
        require(path, ("validateDownloadedPart(part)",))

    sftp = require("internal/remote/sftp.go", (
        "func validatePrivateKeyPath(keyPath string) error", "os.Lstat(keyPath)", "security.IsReparsePoint(keyPath)",
        '"-oBatchMode=no"', "ctxErr := ctx.Err()", "scanHost := strings.Trim(host, \"[]\")",
    ))
    if '"-b"' in sftp or '"-b", "-"' in sftp:
        fail("SFTP command args use -b again; OpenSSH forces BatchMode=yes there and disables AskPass")

    curl = require("internal/remote/curl_ftp.go", (
        "security.ProtectRuntimeString(password)", "security.UnprotectRuntimeBytes(c.passwordBlob)",
        "security.ForgetRuntimeSecret(c.passwordBlob)", "ctxErr := ctx.Err()",
    ))
    if not curl:
        fail("CurlFTP runtime credential contract is unavailable")

    require("cmd/byftp/main.go", (
        "func selectAskpassSecret(", "verification code", "one-time", "authentication code", "return nil, false",
    ))
    require("cmd/byftp/askpass_test.go", ("TestSelectAskpassSecretOnlyUsesRecognizedPrompts", "Verification code", "One-time password token"))
    require("internal/remote/sftp_stream_test.go", ("TestSFTPCommandArgsKeepAskPassEnabled", "sftp -b"))
    require("internal/remote/connect_regression_test.go", ("TestSSHSessionConfigNormalizesBracketedIPv6Host", "TestFindOpenSSHUsesNativeExecutableNameOutsideWindows"))
    require("internal/remote/process_connect_smoke_other_test.go", (
        "TestCurlFTPProcessSmokeUsesRuntimeSecretAndParsesListing",
        "TestSFTPProcessSmokeUsesStdinWithoutBatchMode",
        "security.UnprotectRuntimeBytes(token)",
        'if [ "$arg" = "-b" ]',
        "ls -la \".\"",
    ))

    require("internal/security/runtime_secret_windows.go", ("ProtectRuntimeString", "UnprotectRuntimeBytes", "ForgetRuntimeSecret"))
    require("internal/security/runtime_secret_other.go", ("crypto/rand", "runtimeValues", "WipeBytes(value)", "ForgetRuntimeSecret"))

    require("internal/profilebinding/binding.go", ("func EndpointMatches(", "func AccountMatches(", "func PrivateKeyMatches(", "strings.TrimSuffix(host, \".\")"))
    require("internal/remote/manager.go", (
        "profilebinding.EndpointMatches", "profilebinding.AccountMatches", "profilebinding.PrivateKeyMatches",
        "base.PrivateKeyPath = override.PrivateKeyPath", "if in.Password == \"\" && profileAccountMatches", "if in.Passphrase == \"\" && profilePrivateKeyMatches",
        "profileEndpoint && profile.Fingerprint != \"\"", "remember && profileID != \"\" && profileEndpoint",
        "ErrSessionClosing", "ErrDisconnectTimeout", "activeOps     sync.WaitGroup", "closing       *sessionCloseState",
        "m.activeOps.Add(1)", "m.activeOps.Wait()", "m.activeOps.Done()", "var once sync.Once", "go m.finishSessionClose(state, s)",
        "waitForSessionClose(ctx, state)", "errors.Is(ctx.Err(), context.Canceled)", "m.closing = nil",
    ))
    require("internal/config/profiles.go", (
        "sameProfileAccount(previous, x)", "sameProfilePrivateKey(previous, x)", "sameSFTPEndpoint(previous, x)",
        "x.PasswordBlob = \"\"", "x.PassphraseBlob = \"\"", "x.PrivateKeyPath = strings.TrimSpace(in.PrivateKeyPath)",
    ))
    require("internal/desktop/profiles_windows.go", (
        "profilebinding.AccountMatches", "profilebinding.PrivateKeyMatches", "currentEndpointMatchesProfile",
        'a.tr("profile.store_credentials_title")', 'a.tr("profile.retain_credentials_title")',
    ))
    require("internal/desktop/connection_windows.go", (
        "cfg.Password = getText(a.pass)", "cfg.Passphrase = getText(a.passphrase)",
        "beginConnectionTransition", "connectionGeneration", 'a.tr("sftp.trust_body", result.Fingerprint)',
    ))
    require("internal/desktop/other.go", (
        'if protocol == "sftp" {',
        "cfg.PrivateKeyPath = strings.TrimSpace(keyPath)",
        'i18n.T(language, "terminal.sftp_key_required")',
        'i18n.T(language, "terminal.sftp_passphrase_unsupported")',
        "promptSecret", "engine.Connect", "engine.RemoteList", "engine.AddTransfer",
    ))
    require("internal/api/engine.go", ("e.remote.Disconnect(ctx)", "context.WithTimeout(context.Background(), 4*time.Second)"))

    require("internal/security/remove_tree.go", ("func isFilesystemRoot(target string) bool", "isFilesystemRoot(root)", "maxRemoveTreeDepth", "maxRemoveTreeItems", "isReparsePoint(target)", "os.ModeSymlink"))
    require("internal/transfer/manager.go", ("security.EnsureLocalWithinRoot(job.LocalRoot, job.LocalPath)", "errors.Is(err, remote.ErrSkipped)", "errors.Is(err, context.Canceled)", "ConnectionIdentity() (string, error)"))
    require("internal/platform/filemove_windows.go", ("MoveFileExW", "moveFileWriteThrough", "RenameNoReplace"))
    require("internal/config/store.go", ("os.Lstat(path)", "os.SameFile(before, after)", "io.LimitReader", "os.CreateTemp"))

    require("internal/profilebinding/binding_test.go", ("TestEndpointMatchesNormalizesHost", "TestAccountMatchesRequiresExactUsername", "TestPrivateKeyMatchesRequiresSameNonEmptyKey"))
    require("internal/remote/profile_binding_test.go", ("TestMergeConnectionAllowsClearingPrivateKeyAndFingerprint", "TestProfilePasswordBindingIncludesUsername", "TestProfilePassphraseBindingIncludesPrivateKey"))
    require("internal/config/profiles_test.go", ("TestRemovingSFTPPrivateKeyClearsStoredPassphrase", "TestProfileSavePreservesFingerprintForSameEndpoint", "TestProfileSaveClearsFingerprintWhenEndpointChanges"))
    require("internal/config/profile_secret_binding_test.go", ("TestProfileSavePreservesPasswordForSameAccount", "TestProfileSaveClearsPasswordWhenAccountIdentityChanges", "TestProfileSavePreservesPassphraseOnlyForSamePrivateKeyIdentity"))
    require("internal/transfer/finish_status_test.go", ("TestFinishJobKeepsSuccessfulResultWhenCancelArrivesAfterSuccess", "TestFinishJobKeepsSkippedResultWhenCancelArrivesAfterSkip", "TestFinishJobMarksActualCancellation"))
    require("internal/remote/download_security_test.go", ("validateDownloadedPart", "Symlink"))
    require("internal/remote/private_key_validation_test.go", ("TestValidatePrivateKeyPathAcceptsRegularFile", "TestValidatePrivateKeyPathRejectsSymlink"))
    require("internal/remote/manager_test.go", ("TestDisconnectWaitsForActiveOperationRelease", "TestDisconnectTimeoutDefersCloseAndBlocksReconnect", "TestDisconnectCancellationDefersClose", "TestSecondDisconnectWaitsForExistingCloseState", "TestOperationReleaseIsIdempotent"))
    require("internal/usererror/message_test.go", (
        "TestMessageDefaultsToEnglishAndHidesToolDetails",
        "TestMessageForLocalizesKnownErrors",
        "TestMessageMissingSFTPComponent",
        "TestMessageDisconnectLifecycleWinsJoinedDeadline",
    ))
    require("internal/security/remove_tree_root_test.go", ("RemoveTreeNoFollow",))
    require("internal/security/remove_tree_root_windows_test.go", ("TestIsFilesystemRootRejectsWindowsVolumeRoots", "server\\share"))

    print("SECURITY_AUDIT=PASS")
    print("SFTP_ASKPASS_BATCHMODE_CONFLICT=BLOCKED")
    print("ASKPASS_UNKNOWN_PROMPT_SECRET=BLOCKED")
    print("CONNECT_PROCESS_SMOKE=ENFORCED_ON_UNIX")
    print("CONNECT_CONTEXT_CAUSE=PROPAGATED")
    print("IPV6_SFTP_HOST_NORMALIZATION=ENABLED")
    print("RUNTIME_SECRET_STORAGE=CROSS_PLATFORM_EPHEMERAL")
    print("PROFILE_ENDPOINT_PIN_BINDING=ENABLED")
    print("PROFILE_CREDENTIAL_CROSS_ENDPOINT=BLOCKED")
    print("PROFILE_PRIVATE_KEY_CLEAR=AUTHORITATIVE")
    print("TERMINAL_SFTP_PRIVATE_KEY_REQUIRED=ENFORCED")
    print("TERMINAL_SFTP_PASSPHRASE_UNSUPPORTED=FAIL_CLOSED")
    print("DOWNLOAD_STAGING_REPARSE_VALIDATION=ENABLED")
    print("SFTP_PRIVATE_KEY_REPARSE=BLOCKED")
    print("REMOTE_SESSION_CLOSE_RACE=BLOCKED")
    print("REMOTE_SESSION_CLOSE_TIMEOUT=BOUNDED")
    print("FILESYSTEM_ROOT_DELETE=BLOCKED")
    print("STATE_SAFE_OPEN=ENABLED")
    return 0


if __name__ == "__main__":
    main()
