#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="$(tr -d '\r\n' < VERSION)"
MIN_GO_MAJOR=1
MIN_GO_MINOR=26
MIN_GO_PATCH=5
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid VERSION.' >&2; exit 1; }
for tool in go tar gzip; do command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }; done

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
  local goarch="$1" label="$2"
  local binary="dist/.ghostftp-linux-${label}"
  local portable_name="Ghost-FTP-${VERSION}-Linux-${label}"
  local portable_root="dist/${portable_name}"
  local portable_out="dist/${portable_name}.tar.gz"

  rm -rf "$binary" "$portable_root" "$portable_out"

  echo "[Linux ${label}] Building Ghost FTP portable runtime"
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
  echo "LINUX_PORTABLE_OK=${label}:$portable_out"

  rm -rf "$binary" "$portable_root"
}

# These architecture-specific tarballs are internal build/evidence inputs. They
# are not the public 0.0.6 Linux release surface. Public publication is produced
# by scripts/build_linux_distro_packages.py as one Installer and one Portable
# bundle for each Debian, Ubuntu and Fedora, with amd64/arm64/i386 payloads.
build_arch amd64 amd64
build_arch arm64 arm64
build_arch 386 i386

echo "Ghost FTP ${VERSION} internal Linux portable architecture bundles built with ${raw_go} and telemetry=${telemetry}."
