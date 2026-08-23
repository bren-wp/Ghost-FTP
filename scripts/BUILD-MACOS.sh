#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

[[ "$(uname -s)" == "Darwin" ]] || { echo 'BUILD-MACOS.sh mora se pokrenuti na macOS-u.' >&2; exit 1; }
VERSION="$(tr -d '\r\n' < VERSION)"
MIN_GO_MAJOR=1
MIN_GO_MINOR=26
MIN_GO_PATCH=5
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Neispravan VERSION.' >&2; exit 1; }
for tool in go lipo pkgbuild sips iconutil; do command -v "$tool" >/dev/null || { echo "Nedostaje $tool." >&2; exit 1; }; done

raw_go="$(go env GOVERSION)"
if [[ ! "$raw_go" =~ ^go([0-9]+)\.([0-9]+)(\.([0-9]+))?$ ]]; then
  echo "Nije moguće provjeriti Go verziju: $raw_go" >&2
  exit 1
fi
go_major="${BASH_REMATCH[1]}"
go_minor="${BASH_REMATCH[2]}"
go_patch="${BASH_REMATCH[4]:-0}"
if (( go_major < MIN_GO_MAJOR ||
      (go_major == MIN_GO_MAJOR && go_minor < MIN_GO_MINOR) ||
      (go_major == MIN_GO_MAJOR && go_minor == MIN_GO_MINOR && go_patch < MIN_GO_PATCH) )); then
  echo "Za produkcijski ByFTP build potreban je Go 1.26.5+; trenutačno: $raw_go" >&2
  exit 1
fi

telemetry="$(go telemetry)"
[[ "$telemetry" == "off" ]] || {
  echo "Go telemetrija mora biti isključena prije produkcijskog builda. Pokrenite: go telemetry off (trenutačno: $telemetry)" >&2
  exit 1
}

export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=darwin
mkdir -p dist
work="dist/macos-work"
root="$work/root"
rm -rf "$work" "dist/ByFTP-${VERSION}-macOS-Universal.pkg"
mkdir -p "$work/bin" "$root/usr/local/bin" "$root/Applications/ByFTP.app/Contents/MacOS" "$root/Applications/ByFTP.app/Contents/Resources"

for arch in amd64 arm64; do
  echo "[macOS ${arch}] Izgradnja"
  GOARCH="$arch" go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o "$work/bin/byftp-${arch}" ./cmd/byftp
done
lipo -create "$work/bin/byftp-amd64" "$work/bin/byftp-arm64" -output "$root/usr/local/bin/byftp"
chmod 0755 "$root/usr/local/bin/byftp"
cp "$root/usr/local/bin/byftp" "$root/Applications/ByFTP.app/Contents/Resources/byftp"
chmod 0755 "$root/Applications/ByFTP.app/Contents/Resources/byftp"

iconset="$work/ByFTP.iconset"
mkdir -p "$iconset"
sips -z 16 16 build/icon.png --out "$iconset/icon_16x16.png" >/dev/null
sips -z 32 32 build/icon.png --out "$iconset/icon_16x16@2x.png" >/dev/null
sips -z 32 32 build/icon.png --out "$iconset/icon_32x32.png" >/dev/null
sips -z 64 64 build/icon.png --out "$iconset/icon_32x32@2x.png" >/dev/null
sips -z 128 128 build/icon.png --out "$iconset/icon_128x128.png" >/dev/null
sips -z 256 256 build/icon.png --out "$iconset/icon_128x128@2x.png" >/dev/null
sips -z 256 256 build/icon.png --out "$iconset/icon_256x256.png" >/dev/null
sips -z 512 512 build/icon.png --out "$iconset/icon_256x256@2x.png" >/dev/null
sips -z 512 512 build/icon.png --out "$iconset/icon_512x512.png" >/dev/null
sips -z 1024 1024 build/icon.png --out "$iconset/icon_512x512@2x.png" >/dev/null
iconutil -c icns "$iconset" -o "$root/Applications/ByFTP.app/Contents/Resources/ByFTP.icns"

cat > "$root/Applications/ByFTP.app/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleName</key><string>ByFTP</string>
<key>CFBundleDisplayName</key><string>ByFTP</string>
<key>CFBundleIdentifier</key><string>io.github.bren-wp.byftp</string>
<key>CFBundleVersion</key><string>${VERSION}</string>
<key>CFBundleShortVersionString</key><string>${VERSION}</string>
<key>CFBundleExecutable</key><string>ByFTP</string>
<key>CFBundleIconFile</key><string>ByFTP</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>NSHighResolutionCapable</key><true/>
</dict></plist>
EOF

cat > "$root/Applications/ByFTP.app/Contents/MacOS/ByFTP" <<'EOF'
#!/bin/zsh
set -eu
APP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$APP_DIR/Resources/byftp"
/usr/bin/osascript <<APPLESCRIPT
set cmd to quoted form of "$BIN"
tell application "Terminal"
  activate
  do script cmd
end tell
APPLESCRIPT
EOF
chmod 0755 "$root/Applications/ByFTP.app/Contents/MacOS/ByFTP"

pkg="dist/ByFTP-${VERSION}-macOS-Universal.pkg"
pkgbuild --root "$root" --identifier io.github.bren-wp.byftp --version "$VERSION" --install-location / "$pkg" >/dev/null
test -s "$pkg"
rm -rf "$work"
echo "MACOS_PACKAGE_OK=$pkg"
echo "ByFTP ${VERSION} macOS paket izgrađen je s ${raw_go} i telemetry=${telemetry}."
echo 'Napomena: paket nije Developer ID potpisan dok nije dostupan pravi Apple certifikat.'
