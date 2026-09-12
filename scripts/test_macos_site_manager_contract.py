#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def text(path: str) -> str:
    p = ROOT / path
    if not p.exists():
        raise SystemExit(f"missing required file: {path}")
    return p.read_text(encoding="utf-8")


def require(source: str, *needles: str) -> None:
    for needle in needles:
        if needle not in source:
            raise SystemExit(f"missing contract marker: {needle}")


keychain = text("internal/security/profile_secret_darwin.go")
require(
    keychain,
    "#cgo LDFLAGS: -framework Security -framework CoreFoundation",
    "kSecClassGenericPassword",
    "kSecAttrAccessibleWhenUnlockedThisDeviceOnly",
    "darwin-keychain-aesgcm-v1:",
    "ProtectPersistentProfileBytes",
    "UnprotectPersistentProfileBytes",
    "PersistentProfileSecretToRuntime",
    "ProtectRuntimeBytes",
    'persistentProfileSecretAAD    = "profile-secret-v1"',
)
if "/usr/bin/security" in keychain or "exec.Command" in keychain:
    raise SystemExit("Keychain integration must use Security.framework, not shell commands")

runtime_darwin = text("internal/security/runtime_secret_darwin.go")
require(runtime_darwin, "ProtectRuntimeBytes", "return ProtectBytes(value)", "ForgetProtectedSecret")

capability = text("internal/security/dpapi_darwin.go")
require(capability, "func PersistentSecretStorageAvailable() bool { return true }")

profile_crypto = text("internal/config/profile_crypto_darwin.go")
require(
    profile_crypto,
    "ProtectPersistentProfileBytes",
    "UnprotectPersistentProfileBytes",
    '"profile-envelope-v1"',
)
if "errDarwinPersistentProfilesUnavailable" in profile_crypto:
    raise SystemExit("Darwin saved-profile fail-closed placeholder must be replaced by Keychain protection")

profiles = text("internal/config/profiles.go")
require(profiles, "protectStoredProfileSecret(in.Password)", "protectStoredProfileSecret(in.Passphrase)")

remote = text("internal/remote/manager.go")
require(
    remote,
    "PersistentProfileSecretToRuntime",
    "ownsPasswordBlob",
    "ownsPassphraseBlob",
    "forgetOwnedSecrets",
    "stashPendingTrustResolved",
    "transferResolvedSecretOwnershipToCurl",
    "func (m *Manager) stashPendingTrust(cfg model.ConnectionConfig, resolved resolvedConnection",
)

bridge = text("macos/Bridge/profiles.go")
require(
    bridge,
    "GhostFTPRefreshProfiles",
    "GhostFTPProfileCount",
    "GhostFTPProfileID",
    "GhostFTPProfileHasPassword",
    "GhostFTPProfileHasPassphrase",
    "GhostFTPSaveProfile",
    "GhostFTPRemoveProfile",
    "GhostFTPConnectProfile",
    "engine.Profiles()",
    "engine.SaveProfile",
    "engine.RemoveProfile",
    "engine.Connect",
)
for forbidden in ("json.Marshal", "json.Unmarshal", "PasswordBlob", "PassphraseBlob"):
    if forbidden in bridge:
        raise SystemExit(f"macOS typed profile bridge must not expose generic/secret payloads: {forbidden}")

swift = text("macos/Sources/GhostFTPApp/main.swift")
require(
    swift,
    "Site Manager",
    "SiteManagerWindowController",
    "Save Profile",
    "Remove Profile",
    "Save credentials on this computer?",
    "Keep saved credentials on this computer?",
    "GhostFTPRefreshProfiles",
    "GhostFTPSaveProfile",
    "GhostFTPRemoveProfile",
    "GhostFTPConnectProfile",
    "HasPassword",
    "HasPassphrase",
)
for forbidden in ("PasswordBlob", "PassphraseBlob"):
    if forbidden in swift:
        raise SystemExit(f"Swift must never receive saved credential blobs: {forbidden}")

parity = text("macos/PARITY.md")
for action in ("Site Manager", "Save Profile", "Remove Profile"):
    require(parity, f"- [x] {action}")
require(parity, "Security.framework", "WhenUnlockedThisDeviceOnly")

global_parity = text("scripts/test_macos_windows_parity_contract.py")
for action in ("Site Manager", "Save Profile", "Remove Profile"):
    require(global_parity, f'"{action}"')

print("macOS Site Manager / Keychain profile contract OK")
