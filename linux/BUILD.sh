#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="$(tr -d '\r\n' < VERSION)"
MIN_GO_MAJOR=1
MIN_GO_MINOR=26
MIN_GO_PATCH=5
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid VERSION.' >&2; exit 1; }
for tool in go dpkg-deb sed; do command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }; done

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
  echo "Ghost FTP production builds require Go 1.26.5 or newer; current: $raw_go" >&2
  exit 1
fi

telemetry="$(go telemetry)"
[[ "$telemetry" == "off" ]] || {
  echo "Go telemetry must be disabled before a production build. Run: go telemetry off (current: $telemetry)" >&2
  exit 1
}

export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=linux
mkdir -p dist

build_arch() {
  local goarch="$1" debarch="$2"
  local root="dist/linux-${debarch}-root"
  local out="dist/Ghost-FTP-${VERSION}-Linux-${debarch}.deb"
  rm -rf "$root" "$out"
  mkdir -p "$root/DEBIAN" "$root/usr/bin" "$root/usr/share/applications" "$root/usr/share/icons/hicolor/512x512/apps"

  echo "[Linux ${debarch}] Building Ghost FTP"
  GOARCH="$goarch" go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o "$root/usr/bin/ghostftp" ./cmd/byftp
  chmod 0755 "$root/usr/bin/ghostftp"
  cp build/icon.png "$root/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png"
  cp linux/ghost-ftp.desktop "$root/usr/share/applications/ghost-ftp.desktop"
  sed -e "s/@VERSION@/${VERSION}/g" -e "s/@ARCH@/${debarch}/g" linux/debian/control.in > "$root/DEBIAN/control"

  dpkg-deb --root-owner-group --build "$root" "$out" >/dev/null
  rm -rf "$root"
  test -s "$out"
  echo "LINUX_PACKAGE_OK=${debarch}:$out"
}

build_arch amd64 amd64
build_arch arm64 arm64
build_arch 386 i386

echo "Ghost FTP ${VERSION} Linux packages built with ${raw_go} and telemetry=${telemetry}."
