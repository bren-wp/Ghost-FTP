#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def replace_once(path: str, old: str, new: str) -> None:
    p = ROOT / path
    source = p.read_text(encoding="utf-8")
    count = source.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected one anchor, found {count}: {old[:120]!r}")
    p.write_text(source.replace(old, new, 1), encoding="utf-8")


replace_once(
    "internal/remote/curl_ftp.go",
    '''type CurlFTP struct {\n\tprotocol         string\n\thost             string\n\tusername         string\n\tpasswordBlob     string\n\tport             int\n''',
    '''type CurlFTP struct {\n\tprotocol         string\n\thost             string\n\tusername         string\n\tpasswordBlob     string\n\townsPasswordBlob bool\n\tport             int\n''',
)

replace_once(
    "internal/remote/curl_ftp.go",
    '''\tif password != "" {\n\t\tpasswordBlob, err = security.ProtectRuntimeString(password)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t}\n\treturn &CurlFTP{\n''',
    '''\townsPasswordBlob := false\n\tif password != "" {\n\t\tpasswordBlob, err = security.ProtectRuntimeString(password)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\townsPasswordBlob = passwordBlob != ""\n\t}\n\treturn &CurlFTP{\n''',
)

replace_once(
    "internal/remote/curl_ftp.go",
    '''\t\tusername:         username,\n\t\tpasswordBlob:     passwordBlob,\n\t\tconnectTimeout:   connectTimeout,\n''',
    '''\t\tusername:         username,\n\t\tpasswordBlob:     passwordBlob,\n\t\townsPasswordBlob: ownsPasswordBlob,\n\t\tconnectTimeout:   connectTimeout,\n''',
)

replace_once(
    "internal/remote/curl_ftp.go",
    '''func (c *CurlFTP) Close() error {\n\tsecurity.ForgetRuntimeSecret(c.passwordBlob)\n\tc.passwordBlob = ""\n''',
    '''func (c *CurlFTP) Close() error {\n\tif c.ownsPasswordBlob {\n\t\tsecurity.ForgetRuntimeSecret(c.passwordBlob)\n\t}\n\tc.passwordBlob = ""\n\tc.ownsPasswordBlob = false\n''',
)

replace_once(
    "internal/remote/manager.go",
    '''func transferResolvedSecretOwnershipToSFTP(resolved *resolvedConnection, s *SFTP) {\n\tif resolved == nil || s == nil {\n\t\treturn\n\t}\n\tif resolved.ownsPasswordBlob && resolved.PasswordBlob != "" && s.passwordBlob == resolved.PasswordBlob {\n\t\ts.ownsPasswordBlob = true\n\t\tresolved.ownsPasswordBlob = false\n\t}\n\tif resolved.ownsPassphraseBlob && resolved.PassphraseBlob != "" && s.passphraseBlob == resolved.PassphraseBlob {\n\t\ts.ownsPassphraseBlob = true\n\t\tresolved.ownsPassphraseBlob = false\n\t}\n}\n''',
    '''func transferResolvedSecretOwnershipToSFTP(resolved *resolvedConnection, s *SFTP) {\n\tif resolved == nil || s == nil {\n\t\treturn\n\t}\n\tif resolved.ownsPasswordBlob && resolved.PasswordBlob != "" && s.passwordBlob == resolved.PasswordBlob {\n\t\ts.ownsPasswordBlob = true\n\t\tresolved.ownsPasswordBlob = false\n\t}\n\tif resolved.ownsPassphraseBlob && resolved.PassphraseBlob != "" && s.passphraseBlob == resolved.PassphraseBlob {\n\t\ts.ownsPassphraseBlob = true\n\t\tresolved.ownsPassphraseBlob = false\n\t}\n}\n\nfunc transferResolvedSecretOwnershipToCurl(resolved *resolvedConnection, s *CurlFTP) {\n\tif resolved == nil || s == nil {\n\t\treturn\n\t}\n\tif resolved.ownsPasswordBlob && resolved.PasswordBlob != "" && s.passwordBlob == resolved.PasswordBlob {\n\t\ts.ownsPasswordBlob = true\n\t\tresolved.ownsPasswordBlob = false\n\t}\n}\n''',
)

replace_once(
    "internal/remote/manager.go",
    '''\t} else {\n\t\ts, err = newCurlFTPWithProtectedSecret(cfg.Protocol, cfg.Host, cfg.Port, cfg.Username, cfg.Password, resolved.PasswordBlob, connectTimeout)\n\t\tif err != nil {\n\t\t\treturn ConnectResult{}, err\n\t\t}\n\t\t// Curl consumes the protected secret synchronously per operation. Keep the\n\t\t// owned runtime capability with this connection result until Connect exits;\n\t\t// it will be forgotten by the deferred cleanup above after the probe.\n\t}\n''',
    '''\t} else {\n\t\tcurlSession, curlErr := newCurlFTPWithProtectedSecret(cfg.Protocol, cfg.Host, cfg.Port, cfg.Username, cfg.Password, resolved.PasswordBlob, connectTimeout)\n\t\tif curlErr != nil {\n\t\t\treturn ConnectResult{}, curlErr\n\t\t}\n\t\t// A converted macOS saved credential must survive beyond the initial probe:\n\t\t// every later FTP/FTPS operation asks CurlFTP to unlock the same runtime\n\t\t// capability. Transfer ownership to the live session and release it on Close.\n\t\ttransferResolvedSecretOwnershipToCurl(&resolved, curlSession)\n\t\ts = curlSession\n\t}\n''',
)

for action in ("Site Manager", "Save Profile", "Remove Profile"):
    replace_once("macos/PARITY.md", f"- [ ] {action}", f"- [x] {action}")

intro = "The Windows desktop is the canonical visual and behavior reference for the macOS client. macOS may use native AppKit windowing, accessibility, file panels and menu conventions, but capability/state/security parity is mandatory. The Mac frontend must use the same typed `internal/api.Engine`; no decorative or dead controls are allowed.\n"
paragraph = "\nSaved profiles are protected by a native `Security.framework` Keychain-held AES-256 wrapping key with `WhenUnlockedThisDeviceOnly`. The profile envelope and credential payloads are authenticated and type-marked before use. Saved credential values never cross into Swift: Site Manager receives only public profile metadata and `HasPassword` / `HasPassphrase` state. After exact shared profile binding succeeds, macOS converts a durable credential into the existing same-user ephemeral runtime broker; ownership follows the pending SFTP trust flow or the live SFTP/Curl session and is released when that owner closes.\n"
p = ROOT / "macos/PARITY.md"
source = p.read_text(encoding="utf-8")
if source.count(intro) != 1:
    raise SystemExit("macos/PARITY.md: introduction anchor is not unique")
p.write_text(source.replace(intro, intro + paragraph, 1), encoding="utf-8")

# Extend the existing integration contract with the exact saved-FTP lifetime
# boundary that was missing from PR #299 review coverage.
p = ROOT / "scripts/test_macos_site_manager_integration.py"
source = p.read_text(encoding="utf-8")
anchor = '''    def test_preparer_rejects_missing_anchor(self) -> None:\n'''
method = '''    def test_saved_ftp_runtime_secret_uses_darwin_broker_and_session_ownership(self) -> None:\n        runtime_other = (ROOT / "internal" / "security" / "runtime_secret_other.go").read_text(encoding="utf-8")\n        runtime_darwin = (ROOT / "internal" / "security" / "runtime_secret_darwin.go").read_text(encoding="utf-8")\n        curl = (ROOT / "internal" / "remote" / "curl_ftp.go").read_text(encoding="utf-8")\n        manager = (ROOT / "internal" / "remote" / "manager.go").read_text(encoding="utf-8")\n        self.assertIn("//go:build !windows && !darwin", runtime_other)\n        for marker in ("return ProtectString(value)", "return UnprotectBytes(encoded)", "ForgetProtectedSecret(encoded)"):\n            self.assertIn(marker, runtime_darwin)\n        for marker in ("ownsPasswordBlob bool", "if c.ownsPasswordBlob", "c.ownsPasswordBlob = false"):\n            self.assertIn(marker, curl)\n        for marker in ("transferResolvedSecretOwnershipToCurl", "s.ownsPasswordBlob = true", "transferResolvedSecretOwnershipToCurl(&resolved, curlSession)"):\n            self.assertIn(marker, manager)\n\n'''
if source.count(anchor) != 1:
    raise SystemExit("site manager integration test anchor is not unique")
p.write_text(source.replace(anchor, method + anchor, 1), encoding="utf-8")

print("macOS Site Manager credential lifetime hardening applied")
