#!/usr/bin/env bash
# Build separately installable CLI and Demo Linux packages, no lifecycle side effects.
set -euo pipefail
COMPONENT="$1"; VERSION="$2"; SOURCE="$3"; OUTPUT="$4"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || exit 2
case "$COMPONENT" in
 cli) PKG=ghostftp-cli; PREFIX=GhostCLI; README=ghost-cli/README.md; DESC="Ghost FTP command-line client"; LICENSE=MIT ;;
 demo) PKG=ghostftp-demo; PREFIX=GhostFTP-Demo; README=ghost-demo/README.md; DESC="Ghost FTP offline simulated product demo"; LICENSE=Proprietary ;;
 *) echo "Unsupported component $COMPONENT" >&2; exit 2 ;;
esac
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
SOURCE="$(realpath "$SOURCE")"
OUTPUT="$(realpath -m "$OUTPUT")"
test -s "$SOURCE"; test -s "$README"
command -v dpkg-deb >/dev/null
command -v rpmbuild >/dev/null
command -v rpm >/dev/null
mkdir -p "$OUTPUT"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
BASE="$PREFIX-Linux-x86_64-v$VERSION"
install -m0755 "$SOURCE" "$OUTPUT/$BASE"
install -Dm0755 "$SOURCE" "$work/archive/$PKG"
install -Dm0644 "$README" "$work/archive/README.md"
tar -czf "$OUTPUT/$BASE.tar.gz" -C "$work/archive" "$PKG" README.md
test "$(tar -tzf "$OUTPUT/$BASE.tar.gz" | grep -Fxc "$PKG")" -eq 1
# No service, shell rc changes, execution, secrets, users or firewall mutations.
deb="$work/deb"
install -Dm0755 "$SOURCE" "$deb/usr/bin/$PKG"
install -Dm0644 "$README" "$deb/usr/share/doc/$PKG/README.md"
mkdir -p "$deb/DEBIAN"
cat > "$deb/DEBIAN/control" <<EOF
Package: $PKG
Version: $VERSION
Section: net
Priority: optional
Architecture: amd64
Maintainer: Brendigo <noreply@github.com>
Description: $DESC (Preview)
 Official Ghost FTP executable; no automatic services or credentials.
EOF
dpkg-deb --build --root-owner-group "$deb" "$OUTPUT/$PREFIX-Linux-amd64-v$VERSION.deb"
rpmroot="$work/rpmbuild"
mkdir -p "$rpmroot"/{SOURCES,SPECS,BUILD,RPMS,SRPMS}
install -m0755 "$SOURCE" "$rpmroot/SOURCES/$PKG"
install -m0644 "$README" "$rpmroot/SOURCES/README.md"
cat > "$rpmroot/SPECS/$PKG.spec" <<EOF
Name: $PKG
Version: $VERSION
Release: 1
Summary: $DESC (Preview)
License: $LICENSE
BuildArch: x86_64
AutoReqProv: no
%description
Official standalone Ghost FTP $COMPONENT package.
%install
install -Dm0755 %{_sourcedir}/$PKG %{buildroot}/usr/bin/$PKG
install -Dm0644 %{_sourcedir}/README.md %{buildroot}/usr/share/doc/$PKG/README.md
%files
/usr/bin/$PKG
/usr/share/doc/$PKG/README.md
EOF
rpmbuild --define "_topdir $rpmroot" -bb "$rpmroot/SPECS/$PKG.spec"
built="$(find "$rpmroot/RPMS" -type f -name '*.rpm' -print -quit)"
test -n "$built" && test -s "$built"
cp "$built" "$OUTPUT/$PREFIX-Linux-x86_64-v$VERSION.rpm"
for item in "$OUTPUT/$BASE" "$OUTPUT/$BASE.tar.gz" "$OUTPUT/$PREFIX-Linux-amd64-v$VERSION.deb" "$OUTPUT/$PREFIX-Linux-x86_64-v$VERSION.rpm"; do test -s "$item"; done
dpkg-deb -c "$OUTPUT/$PREFIX-Linux-amd64-v$VERSION.deb" | grep -Fq "./usr/bin/$PKG"
rpm -qpl "$OUTPUT/$PREFIX-Linux-x86_64-v$VERSION.rpm" | grep -Fxq "/usr/bin/$PKG"
echo "Verified $COMPONENT Linux standalone, archive, DEB and RPM v$VERSION."
