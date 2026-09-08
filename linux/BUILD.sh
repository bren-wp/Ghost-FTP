#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="$(tr -d '\r\n' < VERSION)"
MIN_GO_MAJOR=1
MIN_GO_MINOR=26
MIN_GO_PATCH=5
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid VERSION.' >&2; exit 1; }
for tool in go tar gzip sed; do command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }; done

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

have_dpkg=0
if command -v dpkg-deb >/dev/null; then
  have_dpkg=1
elif [[ "${GHOSTFTP_REQUIRE_DEB:-0}" == "1" ]]; then
  echo 'Missing required tool for DEB production build: dpkg-deb' >&2
  exit 1
fi

build_arch() {
  local goarch="$1" debarch="$2"
  local binary="dist/.ghostftp-linux-${debarch}"
  local deb_root="dist/linux-${debarch}-root"
  local deb_out="dist/Ghost-FTP-${VERSION}-Linux-${debarch}.deb"
  local portable_name="Ghost-FTP-${VERSION}-Linux-${debarch}"
  local portable_root="dist/${portable_name}"
  local portable_out="dist/${portable_name}.tar.gz"

  rm -rf "$binary" "$deb_root" "$deb_out" "$portable_root" "$portable_out"

  echo "[Linux ${debarch}] Building Ghost FTP"
  GOARCH="$goarch" go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o "$binary" ./cmd/ghostftp
  chmod 0755 "$binary"

  mkdir -p "$portable_root"
  cp "$binary" "$portable_root/ghostftp"
  cp linux/ghost-ftp.desktop "$portable_root/ghost-ftp.desktop"
  cp build/icon.png "$portable_root/ghost-ftp.png"
  cp LICENSE "$portable_root/LICENSE"
  cp linux/README.md "$portable_root/README.md"
  chmod 0755 "$portable_root/ghostftp"
  chmod 0644 "$portable_root/ghost-ftp.desktop" "$portable_root/ghost-ftp.png" "$portable_root/LICENSE" "$portable_root/README.md"

  tar --sort=name --owner=0 --group=0 --numeric-owner --mtime='UTC 2020-01-01' -C dist -cf - "$portable_name" | gzip -n -9 > "$portable_out"
  test -s "$portable_out"
  echo "LINUX_PORTABLE_OK=${debarch}:$portable_out"

  if (( have_dpkg )); then
    mkdir -p "$deb_root/DEBIAN" "$deb_root/usr/bin" "$deb_root/usr/share/applications" "$deb_root/usr/share/icons/hicolor/512x512/apps"
    cp "$binary" "$deb_root/usr/bin/ghostftp"
    chmod 0755 "$deb_root/usr/bin/ghostftp"
    cp build/icon.png "$deb_root/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png"
    cp linux/ghost-ftp.desktop "$deb_root/usr/share/applications/ghost-ftp.desktop"
    sed -e "s/@VERSION@/${VERSION}/g" -e "s/@ARCH@/${debarch}/g" linux/debian/control.in > "$deb_root/DEBIAN/control"

    dpkg-deb --root-owner-group --build "$deb_root" "$deb_out" >/dev/null
    test -s "$deb_out"
    echo "LINUX_DEB_OK=${debarch}:$deb_out"
  fi

  rm -rf "$binary" "$deb_root" "$portable_root"
}

build_arch amd64 amd64
build_arch arm64 arm64
build_arch 386 i386

if (( have_dpkg )); then
  echo "Ghost FTP ${VERSION} Linux DEB and portable packages built with ${raw_go} and telemetry=${telemetry}."
else
  echo "Ghost FTP ${VERSION} distro-neutral Linux portable packages built with ${raw_go} and telemetry=${telemetry}; dpkg-deb was not available, so DEB packaging was skipped."
fi
