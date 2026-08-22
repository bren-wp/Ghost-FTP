#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="$(tr -d '\r\n' < VERSION)"
MIN_GO="1.26.5"
DIST="$PWD/dist"
INTERNAL="$DIST/internal"
PAYLOAD="$PWD/cmd/installer/payload"
ICON="$PWD/build/icon.ico"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid ByFTP version in VERSION: $VERSION" >&2
  exit 1
fi

# Production builds are intentionally offline and use only the Go standard
# library. They must not fetch a toolchain or modules behind the user's back.
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off

command -v go >/dev/null
command -v python3 >/dev/null
GO_VERSION="$(go env GOVERSION)"
python3 - "$GO_VERSION" "$MIN_GO" <<'PY'
import re, sys
current, minimum = sys.argv[1:3]
m = re.fullmatch(r'go(\d+)\.(\d+)(?:\.(\d+))?', current)
if not m:
    raise SystemExit(f'Unable to verify Go version: {current}')
cur = tuple(map(int, (m.group(1), m.group(2), m.group(3) or 0)))
req = tuple(map(int, minimum.split('.')))
if cur < req:
    raise SystemExit(f'ByFTP production builds require Go {minimum}+ security patch; current: {current}')
PY

GO_TELEMETRY="$(go telemetry)"
if [[ "$GO_TELEMETRY" != "off" ]]; then
  echo "Go telemetry must be disabled before a production build. Run: go telemetry off (current: $GO_TELEMETRY)" >&2
  exit 1
fi

echo "[1/16] Verify generated brand assets"
python3 scripts/generate_brand_assets.py --check

echo "[2/16] Verify localization and version"
python3 scripts/audit_localization.py
python3 scripts/audit_version.py

echo "[3/16] Verify documentation"
python3 scripts/audit_docs.py

echo "[4/16] Verify security invariants"
python3 scripts/audit_security.py

echo "[5/16] Verify privacy and network policy"
python3 scripts/audit_privacy.py

echo "[6/16] Verify release pipeline"
python3 scripts/audit_release.py

echo "[7/16] Python release-tool regressions"
python3 -m unittest discover -s scripts -p 'test_*.py'

echo "[8/16] Go tests and static analysis ($GO_VERSION, telemetry=$GO_TELEMETRY)"
go test ./...
go vet ./...

echo "[9/16] Clean output directories"
rm -rf "$DIST"
mkdir -p "$DIST" "$INTERNAL" "$PAYLOAD"
rm -f "$PAYLOAD/payload.zip"

export GOOS=windows GOARCH=amd64 CGO_ENABLED=0
LDFLAGS="-s -w -H=windowsgui -X main.version=$VERSION"
PORTABLE="$DIST/ByFTP-$VERSION-Portable-x64.exe"
SETUP="$DIST/ByFTP-$VERSION-Setup-x64.exe"
INTERNAL_REMOVE="$INTERNAL/ByFTP-$VERSION-Remove-x64.exe"
INTERNAL_VERIFY="$INTERNAL/windows-x64-verification.txt"

cleanup() {
  rm -f "$PAYLOAD/payload.zip"
}
trap cleanup EXIT

echo "[10/16] Portable"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$PORTABLE" ./cmd/byftp
python3 scripts/pe_resources.py "$PORTABLE" --ico "$ICON" --version "$VERSION" --role portable --original-filename "ByFTP-$VERSION-Portable-x64.exe"

echo "[11/16] Internal uninstaller"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$INTERNAL_REMOVE" ./cmd/uninstaller
python3 scripts/pe_resources.py "$INTERNAL_REMOVE" --ico "$ICON" --version "$VERSION" --role uninstaller --original-filename "ByFTP-$VERSION-Remove-x64.exe"

echo "[12/16] Installer payload"
python3 scripts/make_payload.py --app "$PORTABLE" --uninstaller "$INTERNAL_REMOVE" --output "$PAYLOAD/payload.zip"

echo "[13/16] Installer"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$SETUP" ./cmd/installer
rm -f "$PAYLOAD/payload.zip"

echo "[14/16] PE and security verification"
python3 scripts/pe_resources.py "$SETUP" --ico "$ICON" --version "$VERSION" --role setup --original-filename "ByFTP-$VERSION-Setup-x64.exe"
python3 scripts/verify_release.py "$SETUP" "$PORTABLE" "$INTERNAL_REMOVE" --arch x64 | tee "$INTERNAL_VERIFY"

echo "[15/16] SHA-256 of public files"
sha256sum "$SETUP" "$PORTABLE" > "$DIST/SHA256.txt"

echo "[16/16] Digital-signature status"
if grep -q 'AUTHENTICODE_SIGNED=NO' "$INTERNAL_VERIFY"; then
  echo 'WARNING: public Windows binaries are not Authenticode-signed; Verified Publisher requires a real Brendigo certificate.' >&2
fi

for public in "$SETUP" "$PORTABLE" "$DIST/SHA256.txt"; do
  [[ -s "$public" ]] || { echo "Missing public output: $public" >&2; exit 1; }
done

echo "ByFTP $VERSION local x64 build completed: public outputs in $DIST, technical evidence in $INTERNAL"
