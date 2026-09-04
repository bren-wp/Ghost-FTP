#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

[[ "$(uname -s)" == "Darwin" ]] || { echo 'macos/BUILD.sh must run on macOS.' >&2; exit 1; }
VERSION="$(tr -d '\r\n' < VERSION)"
MIN_GO_MAJOR=1
MIN_GO_MINOR=26
MIN_GO_PATCH=5
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid VERSION.' >&2; exit 1; }
for tool in go lipo pkgbuild sips iconutil sed; do command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }; done

raw_go="$(go env GOVERSION)"
if [[ ! "$raw_go" =~ ^go([0-9]+)\.([0-9]+)(\.([0-9]+))?$ ]]; then
  echo "Unable to verify Go version: $raw_go" >&2
  exit 1
fi
go_major="${BASH_REMATCH[1]}"
go_minor="${BASH_REMATCH[2]}"
go_patch="${BASH_REMATCH[4]:-0}"
if (( go_major < MIN_GO_MAJOR ||
      (go_major == MIN_GO_MAJOR && go_minor < MIN_GO_MINOR) ||
      (go_major == MIN_GO_MAJOR && go_minor == MIN_GO_MINOR && go_patch < MIN_GO_PATCH) )); then
  echo "GhostFTP production builds require Go 1.26.5 or newer; current: $raw_go" >&2
  exit 1
fi

telemetry="$(go telemetry)"
[[ "$telemetry" == "off" ]] || {
  echo "Go telemetry must be disabled before a production build. Run: go telemetry off (current: $telemetry)" >&2
  exit 1
}

export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=darwin
mkdir -p dist
work="dist/macos-work"
root="$work/root"
rm -rf "$work" "dist/GhostFTP-${VERSION}-macOS-Universal.pkg"
mkdir -p "$work/bin" "$root/usr/local/bin" "$root/Applications/GhostFTP.app/Contents/MacOS" "$root/Applications/GhostFTP.app/Contents/Resources"

for arch in amd64 arm64; do
  echo "[macOS ${arch}] Building GhostFTP"
  GOARCH="$arch" go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o "$work/bin/GhostFTP-${arch}" ./cmd/GhostFTP
done
lipo -create "$work/bin/GhostFTP-amd64" "$work/bin/GhostFTP-arm64" -output "$root/usr/local/bin/GhostFTP"
chmod 0755 "$root/usr/local/bin/GhostFTP"
cp "$root/usr/local/bin/GhostFTP" "$root/Applications/GhostFTP.app/Contents/Resources/GhostFTP"
chmod 0755 "$root/Applications/GhostFTP.app/Contents/Resources/GhostFTP"

iconset="$work/GhostFTP.iconset"
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
iconutil -c icns "$iconset" -o "$root/Applications/GhostFTP.app/Contents/Resources/GhostFTP.icns"

sed "s/@VERSION@/${VERSION}/g" macos/Info.plist.in > "$root/Applications/GhostFTP.app/Contents/Info.plist"
cp macos/launcher.zsh "$root/Applications/GhostFTP.app/Contents/MacOS/GhostFTP"
chmod 0755 "$root/Applications/GhostFTP.app/Contents/MacOS/GhostFTP"

pkg="dist/GhostFTP-${VERSION}-macOS-Universal.pkg"
pkgbuild --root "$root" --identifier io.github.bren-wp.GhostFTP --version "$VERSION" --install-location / "$pkg" >/dev/null
test -s "$pkg"
rm -rf "$work"
echo "MACOS_PACKAGE_OK=$pkg"
echo "GhostFTP ${VERSION} macOS package built with ${raw_go} and telemetry=${telemetry}."
echo 'Note: the package is not Developer ID signed until a valid Apple signing certificate is configured.'
