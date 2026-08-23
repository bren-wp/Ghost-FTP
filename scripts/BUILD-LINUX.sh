#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="$(tr -d '\r\n' < VERSION)"
MIN_GO_MAJOR=1
MIN_GO_MINOR=26
MIN_GO_PATCH=5
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid VERSION.' >&2; exit 1; }
command -v go >/dev/null || { echo 'Go is not installed.' >&2; exit 1; }
command -v dpkg-deb >/dev/null || { echo 'dpkg-deb is not installed.' >&2; exit 1; }

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
  echo "ByFTP production builds require Go 1.26.5 or newer; current: $raw_go" >&2
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
  local out="dist/ByFTP-${VERSION}-Linux-${debarch}.deb"
  rm -rf "$root" "$out"
  mkdir -p "$root/DEBIAN" "$root/usr/bin" "$root/usr/share/applications" "$root/usr/share/icons/hicolor/512x512/apps"

  echo "[Linux ${debarch}] Building ByFTP"
  GOARCH="$goarch" go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o "$root/usr/bin/byftp" ./cmd/byftp
  chmod 0755 "$root/usr/bin/byftp"
  cp build/icon.png "$root/usr/share/icons/hicolor/512x512/apps/byftp.png"

  cat > "$root/usr/share/applications/byftp.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=ByFTP
Comment=Privacy-focused FTP, FTPS and SFTP client
Exec=/usr/bin/byftp
Icon=byftp
Terminal=true
Categories=Network;FileTransfer;
Keywords=FTP;FTPS;SFTP;ByFTP;
EOF

  cat > "$root/DEBIAN/control" <<EOF
Package: byftp
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${debarch}
Maintainer: ByFTP <https://github.com/bren-wp/by-ftp/issues>
Depends: ca-certificates, curl, openssh-client
Homepage: https://github.com/bren-wp/by-ftp
Description: Privacy-focused FTP, FTPS and SFTP client
 ByFTP is a file-transfer client without application telemetry. The Linux
 package uses a terminal interface and the same security and transfer core
 as the other supported platforms.
EOF

  dpkg-deb --root-owner-group --build "$root" "$out" >/dev/null
  rm -rf "$root"
  test -s "$out"
  echo "LINUX_PACKAGE_OK=${debarch}:$out"
}

build_arch amd64 amd64
build_arch arm64 arm64
build_arch 386 i386

echo "ByFTP ${VERSION} Linux packages built with ${raw_go} and telemetry=${telemetry}."
