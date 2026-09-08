#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="$(tr -d '\r\n' < VERSION)"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid VERSION.' >&2; exit 1; }

for tool in go tar gzip sed; do
  command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }
done

telemetry="$(go telemetry)"
[[ "$telemetry" == "off" ]] || {
  echo "Go telemetry must be disabled before a production build. Run: go telemetry off (current: $telemetry)" >&2
  exit 1
}

have_dpkg=0
if command -v dpkg-deb >/dev/null; then
  have_dpkg=1
elif [[ "${GHOSTFTP_REQUIRE_DEB:-0}" == "1" ]]; then
  echo 'Missing required tool for Debian/Ubuntu package build: dpkg-deb' >&2
  exit 1
fi

have_rpm=0
if command -v rpmbuild >/dev/null && command -v rpm >/dev/null; then
  have_rpm=1
elif [[ "${GHOSTFTP_REQUIRE_RPM:-0}" == "1" ]]; then
  echo 'Missing required tools for Fedora RPM build: rpmbuild and rpm' >&2
  exit 1
fi

export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=linux
mkdir -p dist

build_fedora_rpm() (
  set -euo pipefail
  local binary="$1" rpmarch="$2"
  local rpm_top rpm_spec rpm_out rpm_built

  rpm_top="$(mktemp -d)"
  trap 'rm -rf "$rpm_top"' EXIT

  rpm_spec="$rpm_top/SPECS/ghost-ftp.spec"
  mkdir -p "$rpm_top/BUILD" "$rpm_top/BUILDROOT" "$rpm_top/RPMS" "$rpm_top/SOURCES" "$rpm_top/SPECS" "$rpm_top/SRPMS"
  cp "$binary" "$rpm_top/SOURCES/ghostftp"
  cp linux/ghost-ftp.desktop "$rpm_top/SOURCES/ghost-ftp.desktop"
  cp build/icon.png "$rpm_top/SOURCES/ghost-ftp.png"
  cp LICENSE "$rpm_top/SOURCES/LICENSE"
  cp linux/README.md "$rpm_top/SOURCES/README.md"
  sed -e "s/@VERSION@/${VERSION}/g" linux/rpm/ghost-ftp.spec.in > "$rpm_spec"

  rpmbuild --define "_topdir $rpm_top" --target "$rpmarch" -bb "$rpm_spec" >/dev/null
  rpm_built="$(find "$rpm_top/RPMS" -type f -name '*.rpm' -print -quit)"
  [[ -n "$rpm_built" && -s "$rpm_built" ]] || { echo "Fedora RPM was not produced for $rpmarch" >&2; exit 1; }

  rpm_out="dist/Ghost-FTP-${VERSION}-Linux-Fedora-${rpmarch}.rpm"
  rm -f "$rpm_out"
  cp "$rpm_built" "$rpm_out"
  test -s "$rpm_out"
  echo "LINUX_FEDORA_RPM_OK=${rpmarch}:$rpm_out"
)

build_distro_arch() {
  local goarch="$1" debarch="$2" rpmarch="$3"
  local binary="dist/.ghostftp-distro-${debarch}"
  local portable_name="Ghost-FTP-${VERSION}-Linux-Portable-${debarch}"
  local portable_root="dist/${portable_name}"
  local portable_out="dist/${portable_name}.tar.gz"

  rm -rf "$binary" "$portable_root" "$portable_out"

  echo "[Linux ${debarch}] Building shared Ghost FTP binary"
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
    local distro slug deb_root deb_out
    for distro in Debian Ubuntu; do
      slug="${distro}"
      deb_root="dist/linux-${distro,,}-${debarch}-root"
      deb_out="dist/Ghost-FTP-${VERSION}-Linux-${slug}-${debarch}.deb"
      rm -rf "$deb_root" "$deb_out"
      mkdir -p "$deb_root/DEBIAN" "$deb_root/usr/bin" "$deb_root/usr/share/applications" "$deb_root/usr/share/icons/hicolor/512x512/apps"
      cp "$binary" "$deb_root/usr/bin/ghostftp"
      cp build/icon.png "$deb_root/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png"
      cp linux/ghost-ftp.desktop "$deb_root/usr/share/applications/ghost-ftp.desktop"
      chmod 0755 "$deb_root/usr/bin/ghostftp"
      chmod 0644 "$deb_root/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png" "$deb_root/usr/share/applications/ghost-ftp.desktop"
      sed -e "s/@VERSION@/${VERSION}/g" -e "s/@ARCH@/${debarch}/g" linux/debian/control.in > "$deb_root/DEBIAN/control"
      printf 'X-GhostFTP-Distribution: %s\n' "$distro" >> "$deb_root/DEBIAN/control"
      dpkg-deb --root-owner-group --build "$deb_root" "$deb_out" >/dev/null
      test -s "$deb_out"
      echo "LINUX_${distro^^}_DEB_OK=${debarch}:$deb_out"
      rm -rf "$deb_root"
    done
  fi

  if (( have_rpm )); then
    build_fedora_rpm "$binary" "$rpmarch"
  fi

  rm -rf "$binary" "$portable_root"
}

build_distro_arch amd64 amd64 x86_64
build_distro_arch arm64 arm64 aarch64
build_distro_arch 386 i386 i686

echo "Ghost FTP ${VERSION} Linux distro packages built from one verified binary per architecture (telemetry=${telemetry}; deb=${have_dpkg}; rpm=${have_rpm})."
