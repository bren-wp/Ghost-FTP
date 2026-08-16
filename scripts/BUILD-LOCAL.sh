#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="2.12.0"
MIN_GO="1.26.5"
DIST="$PWD/dist"
PAYLOAD="$PWD/cmd/installer/payload"
ICON="$PWD/build/icon.ico"

# Production builds are intentionally offline and standard-library-only.
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off

command -v go >/dev/null
command -v python3 >/dev/null
GO_VERSION="$(go env GOVERSION)"
python3 - "$GO_VERSION" "$MIN_GO" <<'PY'
import re, sys
current, minimum = sys.argv[1:3]
m = re.fullmatch(r'go(\d+)\.(\d+)(?:\.(\d+))?', current)
if not m:
    raise SystemExit(f'Cannot verify Go version: {current}')
cur = tuple(map(int, (m.group(1), m.group(2), m.group(3) or 0)))
req = tuple(map(int, minimum.split('.')))
if cur < req:
    raise SystemExit(f'Production ByFTP build requires Go {minimum}+ security patch release; current: {current}')
PY

echo "[1/10] Privacy/network audit"
python3 scripts/audit_privacy.py

echo "[2/10] Tests and static checks ($GO_VERSION)"
go test ./...
go vet ./...

echo "[3/10] Clean output"
rm -rf "$DIST"
mkdir -p "$DIST" "$PAYLOAD"
rm -f "$PAYLOAD/payload.zip"

export GOOS=windows GOARCH=amd64 CGO_ENABLED=0
LDFLAGS="-s -w -H=windowsgui -X main.version=$VERSION"

echo "[4/10] Portable"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$DIST/ByFTP-$VERSION-Portable-x64.exe" ./cmd/byftp
python3 scripts/pe_resources.py "$DIST/ByFTP-$VERSION-Portable-x64.exe" --ico "$ICON" --version "$VERSION" --role portable --original-filename "ByFTP-$VERSION-Portable-x64.exe"

echo "[5/10] Uninstaller"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$DIST/ByFTP-Uninstall.exe" ./cmd/uninstaller
python3 scripts/pe_resources.py "$DIST/ByFTP-Uninstall.exe" --ico "$ICON" --version "$VERSION" --role uninstaller --original-filename "ByFTP-Uninstall.exe"

echo "[6/10] Compressed setup payload"
python3 scripts/make_payload.py --app "$DIST/ByFTP-$VERSION-Portable-x64.exe" --uninstaller "$DIST/ByFTP-Uninstall.exe" --output "$PAYLOAD/payload.zip"
trap 'rm -f "$PAYLOAD/payload.zip"' EXIT

echo "[7/10] Setup"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$DIST/ByFTP-$VERSION-Setup-x64.exe" ./cmd/installer
rm -f "$PAYLOAD/payload.zip"
trap - EXIT
python3 scripts/pe_resources.py "$DIST/ByFTP-$VERSION-Setup-x64.exe" --ico "$ICON" --version "$VERSION" --role setup --original-filename "ByFTP-$VERSION-Setup-x64.exe"

echo "[8/10] PE/security verification"
python3 scripts/verify_release.py "$DIST/ByFTP-$VERSION-Setup-x64.exe" "$DIST/ByFTP-$VERSION-Portable-x64.exe" "$DIST/ByFTP-Uninstall.exe" | tee "$DIST/verification.txt"

echo "[9/10] SHA-256"
sha256sum "$DIST/ByFTP-$VERSION-Setup-x64.exe" "$DIST/ByFTP-$VERSION-Portable-x64.exe" "$DIST/ByFTP-Uninstall.exe" > "$DIST/SHA256.txt"

echo "[10/10] Signature status"
if grep -q 'AUTHENTICODE_SIGNED=NO' "$DIST/verification.txt"; then
  echo 'WARNING: binaries are not Authenticode signed; sign release binaries before public distribution.' >&2
fi

echo "Build complete: $DIST"
