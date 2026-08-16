#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="$(tr -d '\r\n' < VERSION)"
MIN_GO="1.26.5"
DIST="$PWD/dist"
PAYLOAD="$PWD/cmd/installer/payload"
ICON="$PWD/build/icon.ico"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Neispravna ByFTP verzija u VERSION datoteci: $VERSION" >&2
  exit 1
fi

# Produkcijski build namjerno je offline i koristi samo standardnu Go biblioteku.
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOTELEMETRY=off

command -v go >/dev/null
command -v python3 >/dev/null
GO_VERSION="$(go env GOVERSION)"
python3 - "$GO_VERSION" "$MIN_GO" <<'PY'
import re, sys
current, minimum = sys.argv[1:3]
m = re.fullmatch(r'go(\d+)\.(\d+)(?:\.(\d+))?', current)
if not m:
    raise SystemExit(f'Nije moguće provjeriti Go verziju: {current}')
cur = tuple(map(int, (m.group(1), m.group(2), m.group(3) or 0)))
req = tuple(map(int, minimum.split('.')))
if cur < req:
    raise SystemExit(f'Za produkcijski ByFTP build potreban je Go {minimum}+ sigurnosni patch; trenutačno: {current}')
PY

echo "[1/12] Provjera slikovnih resursa"
python3 scripts/generate_brand_assets.py --check

echo "[2/12] Provjera hrvatskog korisničkog sadržaja"
python3 scripts/audit_croatian.py

python3 scripts/audit_version.py

echo "[3/12] Provjera privatnosti i mrežne politike"
python3 scripts/audit_privacy.py

echo "[4/12] Testovi i statička provjera ($GO_VERSION)"
go test ./...
go vet ./...

echo "[5/12] Čišćenje izlaznih datoteka"
rm -rf "$DIST"
mkdir -p "$DIST" "$PAYLOAD"
rm -f "$PAYLOAD/payload.zip"

export GOOS=windows GOARCH=amd64 CGO_ENABLED=0
LDFLAGS="-s -w -H=windowsgui -X main.version=$VERSION"

echo "[6/12] Portable"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$DIST/ByFTP-$VERSION-Portable-x64.exe" ./cmd/byftp
python3 scripts/pe_resources.py "$DIST/ByFTP-$VERSION-Portable-x64.exe" --ico "$ICON" --version "$VERSION" --role portable --original-filename "ByFTP-$VERSION-Portable-x64.exe"

echo "[7/12] Program za uklanjanje"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$DIST/ByFTP-Uninstall.exe" ./cmd/uninstaller
python3 scripts/pe_resources.py "$DIST/ByFTP-Uninstall.exe" --ico "$ICON" --version "$VERSION" --role uninstaller --original-filename "ByFTP-Uninstall.exe"

echo "[8/12] Komprimirani instalacijski paket"
python3 scripts/make_payload.py --app "$DIST/ByFTP-$VERSION-Portable-x64.exe" --uninstaller "$DIST/ByFTP-Uninstall.exe" --output "$PAYLOAD/payload.zip"
trap 'rm -f "$PAYLOAD/payload.zip"' EXIT

echo "[9/12] Instalacijski program"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$DIST/ByFTP-$VERSION-Setup-x64.exe" ./cmd/installer
rm -f "$PAYLOAD/payload.zip"
trap - EXIT
python3 scripts/pe_resources.py "$DIST/ByFTP-$VERSION-Setup-x64.exe" --ico "$ICON" --version "$VERSION" --role setup --original-filename "ByFTP-$VERSION-Setup-x64.exe"

echo "[10/12] PE i sigurnosna provjera"
python3 scripts/verify_release.py "$DIST/ByFTP-$VERSION-Setup-x64.exe" "$DIST/ByFTP-$VERSION-Portable-x64.exe" "$DIST/ByFTP-Uninstall.exe" | tee "$DIST/verification.txt"

echo "[11/12] SHA-256"
sha256sum "$DIST/ByFTP-$VERSION-Setup-x64.exe" "$DIST/ByFTP-$VERSION-Portable-x64.exe" "$DIST/ByFTP-Uninstall.exe" > "$DIST/SHA256.txt"

echo "[12/12] Status digitalnog potpisa"
if grep -q 'AUTHENTICODE_SIGNED=NO' "$DIST/verification.txt"; then
  echo 'UPOZORENJE: binariji nisu Authenticode potpisani; potpišite release binarije prije šire javne distribucije.' >&2
fi

echo "ByFTP $VERSION build dovršen: $DIST"
