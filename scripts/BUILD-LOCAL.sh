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
  echo "Neispravna ByFTP verzija u VERSION datoteci: $VERSION" >&2
  exit 1
fi

# Produkcijski build namjerno je offline i koristi samo standardnu Go biblioteku.
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off

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

GO_TELEMETRY="$(go telemetry)"
if [[ "$GO_TELEMETRY" != "off" ]]; then
  echo "Go telemetrija mora biti isključena prije produkcijskog builda. Pokrenite: go telemetry off (trenutačno: $GO_TELEMETRY)" >&2
  exit 1
fi

echo "[1/16] Provjera slikovnih resursa"
python3 scripts/generate_brand_assets.py --check

echo "[2/16] Provjera hrvatskog korisničkog sadržaja i verzije"
python3 scripts/audit_croatian.py
python3 scripts/audit_version.py

echo "[3/16] Provjera dokumentacije"
python3 scripts/audit_docs.py

echo "[4/16] Provjera sigurnosnih invarijanti"
python3 scripts/audit_security.py

echo "[5/16] Provjera privatnosti i mrežne politike"
python3 scripts/audit_privacy.py

echo "[6/16] Provjera release pipelinea"
python3 scripts/audit_release.py

echo "[7/16] Python regresije release alata"
python3 -m unittest discover -s scripts -p 'test_*.py'

echo "[8/16] Go testovi i statička provjera ($GO_VERSION, telemetry=$GO_TELEMETRY)"
go test ./...
go vet ./...

echo "[9/16] Čišćenje izlaznih datoteka"
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

echo "[11/16] Interna komponenta uklanjanja"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$INTERNAL_REMOVE" ./cmd/uninstaller
python3 scripts/pe_resources.py "$INTERNAL_REMOVE" --ico "$ICON" --version "$VERSION" --role uninstaller --original-filename "ByFTP-$VERSION-Remove-x64.exe"

echo "[12/16] Komprimirani instalacijski paket"
python3 scripts/make_payload.py --app "$PORTABLE" --uninstaller "$INTERNAL_REMOVE" --output "$PAYLOAD/payload.zip"

echo "[13/16] Instalacijski program"
go build -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o "$SETUP" ./cmd/installer
rm -f "$PAYLOAD/payload.zip"

echo "[14/16] PE i sigurnosna provjera"
python3 scripts/pe_resources.py "$SETUP" --ico "$ICON" --version "$VERSION" --role setup --original-filename "ByFTP-$VERSION-Setup-x64.exe"
python3 scripts/verify_release.py "$SETUP" "$PORTABLE" "$INTERNAL_REMOVE" --arch x64 | tee "$INTERNAL_VERIFY"

echo "[15/16] SHA-256 javnih datoteka"
sha256sum "$SETUP" "$PORTABLE" > "$DIST/SHA256.txt"

echo "[16/16] Status digitalnog potpisa"
if grep -q 'AUTHENTICODE_SIGNED=NO' "$INTERNAL_VERIFY"; then
  echo 'UPOZORENJE: javni Windows binariji nisu Authenticode potpisani; za Verified Publisher potreban je stvarni Brendigo certifikat.' >&2
fi

for public in "$SETUP" "$PORTABLE" "$DIST/SHA256.txt"; do
  [[ -s "$public" ]] || { echo "Nedostaje javni izlaz: $public" >&2; exit 1; }
done

echo "ByFTP $VERSION lokalni x64 build dovršen: javni izlazi su u $DIST, tehnički dokazi u $INTERNAL"
