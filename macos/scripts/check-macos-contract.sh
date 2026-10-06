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
  "macos/Sources/GhostFTPMacApp/Security/KeychainStore.swift" \
  "macos/Sources/GhostFTPMacApp/Views/ContentView.swift" \
  "macos/Tests/GhostFTPMacAppTests/ConnectionValidatorTests.swift" \
  "macos/Tests/GhostFTPMacAppTests/FTPControlSessionTests.swift"; do
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
grep -Fq 'kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly' macos/Sources/GhostFTPMacApp/Security/KeychainStore.swift
grep -Fq 'NWConnection' macos/Sources/GhostFTPMacApp/Services/EndpointProbe.swift
grep -Fq 'actor FTPControlSession' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func currentDirectory' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func changeDirectory' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func noop' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func listDirectory' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'extendedPassivePort' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'parseMLSD' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'func disconnect' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'unsafeCommandArgument' macos/Sources/GhostFTPMacApp/Services/FTPControlSession.swift
grep -Fq 'Open FTP session' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'Refresh listing' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'Remember password in macOS Keychain' macos/Sources/GhostFTPMacApp/Views/ContentView.swift
grep -Fq 'verifies TCP reachability only' macos/Sources/GhostFTPMacApp/Views/ContentView.swift

if grep -RniE '(TODO|FIXME|placeholder|demo)' macos/Sources macos/Tests; then
  echo "macOS source contains development markers." >&2
  exit 1
fi

if grep -Fqi 'password' macos/Sources/GhostFTPMacApp/Models/ConnectionProfile.swift; then
  echo "ConnectionProfile must never persist a password field." >&2
  exit 1
fi

echo "Ghost FTP macOS contract OK"
