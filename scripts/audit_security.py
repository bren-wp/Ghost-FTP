#!/usr/bin/env python3
"""Validate security invariants that must not regress."""

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
        fail("SFTP command args use -b; OpenSSH would force BatchMode=yes and disable AskPass")

    curl = require("internal/remote/curl_ftp.go", (
        "security.ProtectRuntimeString(password)", "security.UnprotectRuntimeBytes(c.passwordBlob)",
        "security.ForgetRuntimeSecret(c.passwordBlob)", "ctxErr := ctx.Err()",
    ))
    if not curl:
        fail("CurlFTP runtime credential contract is unavailable")

    require("cmd/ghostftp/main.go", (
        "func selectAskpassSecret(",
        "unknown or unsupported credential request",
        "invalid authentication request",
        "untrusted parent process",
        "clearAskpassEnvironment()",
        "platform.TrustedAskPassParent()",
        "security.WipeBytes(password)",
        "security.WipeBytes(passphrase)",
    ))
    require("cmd/ghostftp/askpass_test.go", ("TestSelectAskpassSecret", "Verification code", "One-time password token"))
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
    manager = require("internal/remote/manager.go", (
        "profilebinding.EndpointMatches", "profilebinding.AccountMatches", "profilebinding.PrivateKeyMatches",
        "func sanitizeProtocolState(cfg model.ConnectionConfig) model.ConnectionConfig", "cfg.PrivateKeyPath = \"\"", "cfg.Passphrase = \"\"", "cfg.Fingerprint = \"\"",
        "base.PrivateKeyPath = override.PrivateKeyPath", "if in.Password == \"\" && profileAccountMatches", "if in.Passphrase == \"\" && profilePrivateKeyMatches",
        "profileEndpoint && profile.Fingerprint != \"\"", "remember && profileID != \"\" && profileEndpoint",
        "ErrSessionClosing", "ErrDisconnectTimeout", "activeOps     sync.WaitGroup", "closing       *sessionCloseState",
        "m.activeOps.Add(1)", "m.activeOps.Wait()", "m.activeOps.Done()", "var once sync.Once", "go m.finishSessionClose(state, s)",
        "waitForSessionClose(ctx, state)", "errors.Is(ctx.Err(), context.Canceled)", "m.closing = nil",
        "Diagnostics   ConnectionDiagnostics", "initial, err := s.List(cctx, probePathForSession(s))",
        "diagnostics := diagnoseConnection(s.Protocol(), initial)",
        "return ConnectResult{Connected: true, Diagnostics: diagnostics}, nil",
    ))
    if manager.count("diagnoseConnection(s.Protocol(), initial)") != 1:
        fail("desktop shared-hosting diagnostics must derive exactly once from the existing initial listing")

    diagnostics = require("internal/remote/shared_hosting_diagnostics.go", (
        "type ConnectionDiagnostics struct", "Secure          bool", "RootMode        string", "WebRoot         string",
        "WebRootDetected bool", "RootEntryCount  int", "func diagnoseConnection(protocol string, items []model.Item)",
        '"public_html"', '"httpdocs"', '"htdocs"', '"www"', '"web"', '"html"',
        "if !item.IsDirectory || item.IsSymlink", "protocol != \"ftp\"", "protocol == \"sftp\"",
    ))
    for forbidden in (
        "Password", "Passphrase", "PrivateKey", "Username", "Fingerprint", "Certificate", "ServerBanner",
        "net.", "http.", "url.", ".List(", "Connect(", "Dial(", "os.Exec", "exec.Command",
    ):
        if forbidden in diagnostics:
            fail(f"shared-hosting diagnostics gained secret or network behavior: {forbidden}")
    require("internal/remote/shared_hosting_diagnostics_test.go", (
        "TestDiagnoseConnectionFindsPreferredWebRoot",
        "TestDiagnoseConnectionDoesNotTreatFilesOrSymlinksAsWebRoot",
        "TestDiagnoseConnectionUsesSFTPHomeRootWithoutInventingWebRoot",
    ))

    require("internal/remote/protocol_state_regression_test.go", (
        "TestResolveClearsSFTPOnlyStateForFTPFamily", "TestConnectionIdentityIgnoresSFTPOnlyStateForFTPFamily", "TestResolvePreservesSFTPStateForSFTP",
    ))
    profile_store = require("internal/config/profiles.go", (
        "sameProfileAccount(previous, x)", "sameProfilePrivateKey(previous, x)", "sameSFTPEndpoint(previous, x)",
        "requestedFingerprint := in.Fingerprint", "x.Protocol = in.Protocol", "x.Host = in.Host", "x.Username = in.Username",
        "security.ValidateSFTPFingerprint(requestedFingerprint)", "security.ValidateConnection(x.Protocol, x.Host, x.Username, x.Port)",
        "security.ValidateSFTPFingerprint(fp)", "x.PasswordBlob = \"\"", "x.PassphraseBlob = \"\"", "zaporka privatnog ključa zahtijeva odabran privatni ključ",
    ))
    for forbidden, label in (
        ("strings.ToLower(strings.TrimSpace(in.Protocol))", "protocol"),
        ("strings.TrimSpace(in.Host)", "host"),
        ("strings.TrimSpace(in.Username)", "username"),
        ("strings.TrimSpace(in.Fingerprint)", "fingerprint"),
        ("fp = strings.TrimSpace(fp)", "direct fingerprint update"),
    ):
        if forbidden in profile_store:
            fail(f"profile persistence normalizes raw {label} before fail-closed validation")
    require("internal/config/profile_raw_input_test.go", (
        "TestProfileSaveRejectsNonCanonicalRawProtocol",
        "TestProfileSaveRejectsRawHostBeforeNormalization",
        "TestProfileSaveRejectsUsernameControlsBeforeNormalization",
        "TestProfileSavePreservesUsernameVerbatim",
        "TestProfileSaveRejectsNonCanonicalFingerprintBeforeNormalization",
        "TestProfileUpdateFingerprintRejectsNonCanonicalRawInput",
        "TestProfileSaveAcceptsCanonicalConnectionAndFingerprint",
    ))
    require("internal/desktop/connection_input.go", (
        "func validateRawConnectionInput(", "strconv.Atoi(portText)", "security.ValidateConnection(protocol, host, username, port)",
    ))
    require("internal/desktop/connection_input_test.go", (
        "TestValidateRawConnectionInputRejectsNonCanonicalPortText",
        "TestValidateRawConnectionInputRejectsUsernameControlsBeforeNormalization",
        "TestValidateRawConnectionInputDoesNotTrimUsername",
        "TestValidateRawConnectionInputKeepsHostFailClosed",
    ))
    desktop_connection = require("internal/desktop/connection_profiles_windows.go", (
        "profilebinding.AccountMatches", "profilebinding.PrivateKeyMatches", "Stare vjerodajnice neće se prenijeti",
        "Zadržati spremljene vjerodajnice?", "currentEndpointMatchesProfile", "Unesene tajne ostaju u zaključanim edit kontrolama",
        "cfg.Password = getText(a.pass)", "cfg.Passphrase = getText(a.passphrase)",
        "validateRawConnectionInput(protocol, host, getText(a.port), user)",
        "validateRawConnectionInput(protocol, host, getText(a.port), username)",
        "a.onConnected(host, r.Diagnostics)", "a.onConnected(cfg.Host, r.Diagnostics)",
        "func connectionDiagnosticStatus(host string, diagnostics remote.ConnectionDiagnostics) string",
        "remoteStart := \"/\"", "remoteStart = p.RemotePath",
    ))
    for forbidden in ("remoteStart = diagnostics.WebRoot", "setText(a.remotePath, diagnostics.WebRoot)", "SaveProfile(diagnostics"):
        if forbidden in desktop_connection:
            fail(f"Windows shared-hosting diagnostics auto-navigate or persist derived state: {forbidden}")
    if "host := strings.TrimSpace(getText(a.host))" in desktop_connection:
        fail("Windows connection/profile UI trims raw host before fail-closed host validation")
    if desktop_connection.count("host := getText(a.host)") < 2:
        fail("Windows connection/profile UI must pass raw host input to validation")
    if "strings.TrimSpace(getText(a.user))" in desktop_connection:
        fail("Windows connection/profile UI trims raw username before fail-closed credential validation")
    if "strconv.Atoi(strings.TrimSpace(getText(a.port)))" in desktop_connection:
        fail("Windows connection/profile UI trims raw port text before strict parsing")
    require("internal/desktop/connection_diagnostics_windows_test.go", (
        "TestConnectionDiagnosticStatusShowsSecureWebRoot", "TestConnectionDiagnosticStatusShowsPlainFTPAccountRoot",
        "TestConnectionDiagnosticStatusShowsSFTPHome", "web root: public_html", "FTP bez enkripcije",
    ))
    require("internal/desktop/other.go", (
        'i18n.T(language, "terminal.sftp_key_required")', "promptSecret", "engine.Connect", "engine.RemoteList", "engine.AddTransfer",
    ))
    require("internal/api/engine.go", (
        "e.remote.Disconnect(ctx)", "context.WithTimeout(context.Background(), 4*time.Second)",
        "return e.profiles.Save(in)",
    ))

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
        "TestMessageSessionStillClosing", "TestMessageDisconnectCleanupStillRunning", "TestMessageDisconnectLifecycleWinsJoinedDeadline", "TestMessageSFTPHostKeyScanFailure",
        "TestMessageSharedHostingDataChannelFailure", "TestMessageSharedHostingTLSFailure", "TestMessageSharedHostingQuotaFailure",
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
    print("PROFILE_RAW_CONNECTION_INPUT=FAIL_CLOSED")
    print("PROFILE_RAW_FINGERPRINT_INPUT=FAIL_CLOSED")
    print("NON_SFTP_KEY_TRUST_STATE=STRIPPED")
    print("WINDOWS_RAW_HOST_VALIDATION=FAIL_CLOSED")
    print("WINDOWS_RAW_USERNAME_VALIDATION=FAIL_CLOSED")
    print("WINDOWS_RAW_PORT_VALIDATION=FAIL_CLOSED")
    print("SHARED_HOSTING_DIAGNOSTICS=EXISTING_INITIAL_LISTING_ONLY")
    print("SHARED_HOSTING_DIAGNOSTIC_SECRETS=BLOCKED")
    print("SHARED_HOSTING_AUTO_NAVIGATION=BLOCKED")
    print("DOWNLOAD_STAGING_REPARSE_VALIDATION=ENABLED")
    print("SFTP_PRIVATE_KEY_REPARSE=BLOCKED")
    print("REMOTE_SESSION_CLOSE_RACE=BLOCKED")
    print("REMOTE_SESSION_CLOSE_TIMEOUT=BOUNDED")
    print("FILESYSTEM_ROOT_DELETE=BLOCKED")
    print("STATE_SAFE_OPEN=ENABLED")
    return 0


if __name__ == "__main__":
    main()
