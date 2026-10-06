#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

for file in \
  "macos/Package.swift" \
  "macos/Sources/GhostFTPMacApp/GhostFTPMacApp.swift" \
  "macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift" \
  "macos/Sources/GhostFTPMacApp/Services/ConnectionValidator.swift" \
  "macos/Sources/GhostFTPMacApp/Services/ProfileStore.swift" \
  "macos/Sources/GhostFTPMacApp/Services/EndpointProbe.swift" \
  "macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift" \
  "macos/Sources/GhostFTPMacApp/Services/FTPConnectionController.swift" \
  "macos/Sources/GhostFTPMacApp/Services/TransferHistoryStore.swift" \
  "macos/Sources/GhostFTPMacApp/Security/KeychainStore.swift" \
  "macos/Sources/GhostFTPMacApp/Views/ContentView.swift" \
  "macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift" \
  "macos/Tests/GhostFTPMacAppTests/ConnectionValidatorTests.swift" \
  "macos/Tests/GhostFTPMacAppTests/FTPControlSessionTests.swift" \
  "macos/Tests/GhostFTPMacAppTests/WorkspaceStoreTests.swift"; do
  test -s "$file" || { echo "Required macOS source missing: $file" >&2; exit 1; }
done

grep -Fq '.macOS(.v13)' macos/Package.swift
grep -Fq 'case ftp' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift
grep -Fq 'case ftps' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift
grep -Fq 'case sftp' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift
grep -Fq 'var keepAliveSeconds: UInt16' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift
grep -Fq 'var reconnectAttempts: UInt8' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift
grep -Fq 'invalidKeepAlive' macos/Sources/GhostFTPMacApp/Services/ConnectionValidator.swift
grep -Fq 'Automatic reconnect attempts' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'FTP traffic is not encrypted.' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift
grep -Fq 'certificate and hostname' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift
grep -Fq 'verify the server host key' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift
grep -Fq 'kSecClassGenericPassword' macos/Sources/GhostFTPMacApp/Security/KeychainStore.swift
grep -Fq 'kSecAttrAccessibleWhenUnlockedThisDeviceOnly' macos/Sources/GhostFTPMacApp/Security/KeychainStore.swift
grep -Fq 'kSecAttrSynchronizable as String: kCFBooleanFalse as Any' macos/Sources/GhostFTPMacApp/Security/KeychainStore.swift
grep -Fq 'NWConnection' macos/Sources/GhostFTPMacApp/Services/EndpointProbe.swift
grep -Fq 'actor FTPControlSession' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func currentDirectory' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func changeDirectory' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func noop' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func listDirectory' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func uploadFile' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func downloadFile' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'STOR ' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'RETR ' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq '64 * 1024' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq '.ghostftp-' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'extendedPassivePort' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'parseMLSD' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func disconnect' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'unsafeCommandArgument' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'Open FTP session' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'Refresh listing' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'Upload file' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'chooseDownload' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'Remember password in macOS Keychain' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'verifies TCP reachability only' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'case files' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'case sites' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'case transfers' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'case sync' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'case settings' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'case about' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'New connection' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'Sync & Backup' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'TransferHistoryStore.shared' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'exportProfiles()' macos/Sources/GhostFTPMacApp/Services/ProfileStore.swift
grep -Fq 'importProfiles(from data: Data)' macos/Sources/GhostFTPMacApp/Services/ProfileStore.swift
grep -Fq 'maximumRecords = 200' macos/Sources/GhostFTPMacApp/Services/TransferHistoryStore.swift
grep -Fq 'status == .running' macos/Sources/GhostFTPMacApp/Services/TransferHistoryStore.swift
grep -Fq 'readBoundedBackup(' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'maximumBytes: Int = 256 * 1024' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'maximumImportedProfiles = 512' macos/Sources/GhostFTPMacApp/Services/ProfileStore.swift
grep -Fq 'maximumBackupBytes = 256 * 1024' macos/Sources/GhostFTPMacApp/Services/ProfileStore.swift
grep -Fq '.posixPermissions: 0o600' macos/Sources/GhostFTPMacApp/Views/WorkspaceShell.swift
grep -Fq 'history.begin(direction: .upload' macos/Sources/GhostFTPMacApp/Services/FTPConnectionController.swift
grep -Fq 'history.begin(direction: .download' macos/Sources/GhostFTPMacApp/Services/FTPConnectionController.swift

if grep -Fq 'Settings {' macos/Sources/GhostFTPMacApp/GhostFTPMacApp.swift; then
  echo "macOS app must keep Settings inside the single persistent workspace shell." >&2
  exit 1
fi


audit_private_swift_symbols() {
  python3 - "macos/Sources/GhostFTPMacApp" <<'PY'
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
dead = []
declaration = re.compile(r"\bprivate\s+(?:(?:static|class)\s+)?(?:func|var|let)\s+([A-Za-z_][A-Za-z0-9_]*)\b")

for path in sorted(root.rglob("*.swift")):
    source = path.read_text(encoding="utf-8")
    code = re.sub(r"/\*[\s\S]*?\*/", " ", source)
    code = re.sub(r"//[^\n]*", " ", code)
    code = re.sub(r'"(?:\\.|[^"\\])*"', '""', code)

    for match in declaration.finditer(code):
        name = match.group(1)
        if len(re.findall(rf"\b{re.escape(name)}\b", code)) < 2:
            dead.append(f"{path.relative_to(root)}: {name}")

if dead:
    print("macOS contract failed: declaration-only private Swift symbols detected:", file=sys.stderr)
    for item in dead:
        print(f" - {item}", file=sys.stderr)
    sys.exit(1)

print("macOS private Swift symbol audit OK")
PY
}

audit_private_swift_symbols

if grep -RniE '(TODO|FIXME|placeholder|demo)' macos/Sources macos/Tests; then
  echo "macOS source contains development markers." >&2
  exit 1
fi

if grep -Fqi 'password' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift; then
  echo "ConnectionProfile must never persist a password field." >&2
  exit 1
fi

echo "Ghost FTP macOS contract OK"
