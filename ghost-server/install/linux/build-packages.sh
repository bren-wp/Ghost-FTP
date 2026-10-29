#!/usr/bin/env bash
# Build real offline-verifiable Linux server distribution assets.
set -euo pipefail
VERSION="${1:?canonical SemVer is required}"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || exit 1
cd "$(dirname "$0")/../../.."
ROOT="$PWD"
BIN="$ROOT/ghost-server/dist/ghost-server-linux-amd64"
UNIT="$ROOT/ghost-server/install/linux/ghostftp-server.service"
DOC="$ROOT/ghost-server/install/linux/README.md"
OUT="$ROOT/ghost-server/dist"
test -s "$BIN"; test -s "$UNIT"; test -s "$DOC"
command -v dpkg-deb >/dev/null
command -v rpmbuild >/dev/null
command -v rpm >/dev/null
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
install -Dm0755 "$BIN" "$work/archive/ghostftp-server"
install -Dm0644 "$UNIT" "$work/archive/ghostftp-server.service"
install -Dm0644 "$DOC" "$work/archive/README.md"
cp "$BIN" "$OUT/GhostFTP-Server-Linux-x86_64-v$VERSION"
chmod 0755 "$OUT/GhostFTP-Server-Linux-x86_64-v$VERSION"
tar -C "$work/archive" -czf "$OUT/GhostFTP-Server-Linux-x86_64-v$VERSION.tar.gz" ghostftp-server ghostftp-server.service README.md
tar -tzf "$OUT/GhostFTP-Server-Linux-x86_64-v$VERSION.tar.gz" | grep -Fxq ghostftp-server.service

# Debian: no auto-start, preserve existing root/data/credentials on upgrades.
deb="$work/deb"
install -Dm0755 "$BIN" "$deb/usr/bin/ghostftp-server"
install -Dm0644 "$UNIT" "$deb/usr/lib/systemd/system/ghostftp-server.service"
install -Dm0644 "$DOC" "$deb/usr/share/doc/ghostftp-server/README.md"
mkdir -p "$deb/DEBIAN"
cat > "$deb/DEBIAN/control" <<EOF
Package: ghostftp-server
Version: $VERSION
Section: net
Priority: optional
Architecture: amd64
Maintainer: Brendigo <noreply@github.com>
Description: Ghost FTP Server explicit FTPS standalone development Preview
 Standalone Go FTPS server with a manually provisioned, disabled-by-default systemd service.
EOF
cat > "$deb/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e
if ! getent group ghostftp >/dev/null 2>&1; then addgroup --system ghostftp; fi
if ! id -u ghostftp >/dev/null 2>&1; then adduser --system --ingroup ghostftp --home /var/lib/ghostftp-server --no-create-home --disabled-login --gecos 'Ghost FTP Server' ghostftp; fi
install -d -m 0750 -o ghostftp -g ghostftp /var/lib/ghostftp-server/files
# Do NOT enable/start service or create/overwrite any secrets.
exit 0
EOF
chmod 0755 "$deb/DEBIAN/postinst"
dpkg-deb --build --root-owner-group "$deb" "$OUT/GhostFTP-Server-Linux-amd64-v$VERSION.deb"
dpkg-deb --contents "$OUT/GhostFTP-Server-Linux-amd64-v$VERSION.deb" | grep -q 'ghostftp-server.service'

# RPM: same install/security lifecycle; no post-install auto-start.
rpmroot="$work/rpmbuild"
mkdir -p "$rpmroot/SOURCES" "$rpmroot/SPECS" "$rpmroot/BUILD" "$rpmroot/RPMS" "$rpmroot/SRPMS"
install -m 0755 "$BIN" "$rpmroot/SOURCES/ghostftp-server"
install -m 0644 "$UNIT" "$rpmroot/SOURCES/ghostftp-server.service"
install -m 0644 "$DOC" "$rpmroot/SOURCES/README.md"
cat > "$rpmroot/SPECS/ghostftp-server.spec" <<EOF
Name: ghostftp-server
Version: $VERSION
Release: 1
Summary: Ghost FTP Server standalone explicit FTPS development Preview
License: Proprietary
BuildArch: x86_64
AutoReqProv: no
%description
Standalone Go FTPS Preview; manually provision and enable systemd service.
%install
install -Dm0755 %{_sourcedir}/ghostftp-server %{buildroot}/usr/bin/ghostftp-server
install -Dm0644 %{_sourcedir}/ghostftp-server.service %{buildroot}/usr/lib/systemd/system/ghostftp-server.service
install -Dm0644 %{_sourcedir}/README.md %{buildroot}/usr/share/doc/ghostftp-server/README.md
%pre
getent group ghostftp >/dev/null 2>&1 || /usr/sbin/groupadd --system ghostftp
id -u ghostftp >/dev/null 2>&1 || /usr/sbin/useradd --system -g ghostftp -d /var/lib/ghostftp-server -s /sbin/nologin ghostftp
%post
install -d -m 0750 -o ghostftp -g ghostftp /var/lib/ghostftp-server/files
# Never auto-enable or start an unconfigured service.
%files
/usr/bin/ghostftp-server
/usr/lib/systemd/system/ghostftp-server.service
/usr/share/doc/ghostftp-server/README.md
EOF
rpmbuild --define "_topdir $rpmroot" -bb "$rpmroot/SPECS/ghostftp-server.spec"
built="$(find "$rpmroot/RPMS" -type f -name '*.rpm' -print -quit)"
test -n "$built"; test -s "$built"
cp "$built" "$OUT/GhostFTP-Server-Linux-x86_64-v$VERSION.rpm"
rpm -qpl "$OUT/GhostFTP-Server-Linux-x86_64-v$VERSION.rpm" | grep -Fxq /usr/lib/systemd/system/ghostftp-server.service
for f in "$OUT/GhostFTP-Server-Linux-x86_64-v$VERSION" "$OUT/GhostFTP-Server-Linux-x86_64-v$VERSION.tar.gz" "$OUT/GhostFTP-Server-Linux-amd64-v$VERSION.deb" "$OUT/GhostFTP-Server-Linux-x86_64-v$VERSION.rpm"; do test -s "$f"; done
echo "Verified native Linux server binary, archive, Debian and RPM payloads for v$VERSION."
