#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="$(tr -d '\r\n' < VERSION)"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo 'Invalid VERSION.' >&2; exit 1; }

for tool in go tar gzip sed awk tail mktemp; do
  command -v "$tool" >/dev/null || { echo "Missing required tool: $tool" >&2; exit 1; }
done

telemetry="$(go telemetry)"
[[ "$telemetry" == "off" ]] || {
  echo "Go telemetry must be disabled before a production build. Run: go telemetry off (current: $telemetry)" >&2
  exit 1
}

export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=linux
mkdir -p dist

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
PAYLOAD="$WORK/payload"
mkdir -p "$PAYLOAD/bin/amd64" "$PAYLOAD/bin/arm64" "$PAYLOAD/bin/i386"

build_arch() {
  local goarch="$1" public_arch="$2"
  local output="$PAYLOAD/bin/$public_arch/ghostftp"
  local native_host="$PAYLOAD/bin/$public_arch/ghostftp-native-host"
  echo "[Linux ${public_arch}] Building Ghost FTP payload"
  GOARCH="$goarch" go build -trimpath -buildvcs=false -ldflags "-s -w -X main.version=${VERSION}" -o "$output" ./cmd/ghostftp
  echo "[Linux ${public_arch}] Building Ghost FTP native browser host"
  GOARCH="$goarch" go build -trimpath -buildvcs=false -ldflags "-s -w -buildid= -X main.version=${VERSION}" -o "$native_host" ./cmd/ghostftp-native-host
  chmod 0755 "$output" "$native_host"
  test -s "$output"
  test -s "$native_host"
}

build_arch amd64 amd64
build_arch arm64 arm64
build_arch 386 i386

cat > "$PAYLOAD/ghostftp" <<'LAUNCHER'
#!/usr/bin/env sh
set -eu
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  i386|i486|i586|i686|x86) arch=i386 ;;
  *) echo "Ghost FTP: unsupported Linux CPU architecture: $(uname -m)" >&2; exit 64 ;;
esac
exec "$base_dir/bin/$arch/ghostftp" "$@"
LAUNCHER
chmod 0755 "$PAYLOAD/ghostftp"

cat > "$PAYLOAD/install.sh" <<'INSTALLER'
#!/usr/bin/env sh
set -eu
base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
for tool in install mkdir dirname uname command grep cat; do
  command -v "$tool" >/dev/null 2>&1 || { echo "Ghost FTP installer: missing required system tool: $tool" >&2; exit 69; }
done
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  i386|i486|i586|i686|x86) arch=i386 ;;
  *) echo "Ghost FTP: unsupported Linux CPU architecture: $(uname -m)" >&2; exit 64 ;;
esac

missing_runtime=""
for tool in curl ssh sftp; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    missing_runtime="$missing_runtime $tool"
  fi
done
ca_bundle=""
for candidate in \
  /etc/ssl/certs/ca-certificates.crt \
  /etc/pki/tls/certs/ca-bundle.crt \
  /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem; do
  if [ -s "$candidate" ]; then ca_bundle="$candidate"; break; fi
done
if [ -n "$missing_runtime" ] || [ -z "$ca_bundle" ]; then
  echo "Ghost FTP installer: required runtime dependencies are missing." >&2
  [ -z "$missing_runtime" ] || echo "Missing commands:$missing_runtime" >&2
  [ -n "$ca_bundle" ] || echo "Missing CA certificate bundle." >&2
  echo "Debian/Ubuntu: install ca-certificates curl openssh-client" >&2
  echo "Fedora: install ca-certificates curl openssh-clients" >&2
  exit 69
fi

prefix=${GHOSTFTP_PREFIX:-/usr/local}
bin_dir="$prefix/bin"
share_dir="$prefix/share"
libexec_dir="$prefix/libexec/ghostftp"

if [ ! -w "$prefix" ] && [ ! -w "$(dirname -- "$prefix")" ]; then
  echo "Ghost FTP: $prefix is not writable. Re-run with sudo or set GHOSTFTP_PREFIX to a writable location." >&2
  exit 77
fi

mkdir -p "$bin_dir" "$share_dir/applications" "$share_dir/icons/hicolor/512x512/apps" "$share_dir/doc/ghost-ftp" "$libexec_dir"
install -m 0755 "$base_dir/bin/$arch/ghostftp" "$bin_dir/ghostftp"
install -m 0755 "$base_dir/bin/$arch/ghostftp-native-host" "$libexec_dir/ghostftp-native-host"
install -m 0644 "$base_dir/ghost-ftp.desktop" "$share_dir/applications/ghost-ftp.desktop"
install -m 0644 "$base_dir/ghost-ftp.png" "$share_dir/icons/hicolor/512x512/apps/ghost-ftp.png"
install -m 0644 "$base_dir/LICENSE" "$share_dir/doc/ghost-ftp/LICENSE"
install -m 0644 "$base_dir/README.md" "$share_dir/doc/ghost-ftp/README.md"

browser_registration="not-installed"
if [ "$prefix" = "/usr/local" ]; then
  distro=$(cat "$base_dir/DISTRIBUTION" 2>/dev/null || true)
  case "$distro" in
    Fedora) firefox_manifest_dir=/usr/lib64/mozilla/native-messaging-hosts ;;
    Debian|Ubuntu) firefox_manifest_dir=/usr/lib/mozilla/native-messaging-hosts ;;
    *) firefox_manifest_dir= ;;
  esac
  if [ -n "$firefox_manifest_dir" ]; then
    if mkdir -p "$firefox_manifest_dir" 2>/dev/null; then
      firefox_manifest="$firefox_manifest_dir/com.ghostftp.bridge.json"
      manifest_tmp="$firefox_manifest.tmp.$$"
      if cat > "$manifest_tmp" <<'NATIVEHOST'
{
  "name": "com.ghostftp.bridge",
  "description": "Ghost FTP local browser bridge",
  "path": "/usr/local/libexec/ghostftp/ghostftp-native-host",
  "type": "stdio",
  "allowed_extensions": ["ghostftp-connection-helper@ghostftp.com"]
}
NATIVEHOST
      then
        chmod 0644 "$manifest_tmp"
        if mv -f -- "$manifest_tmp" "$firefox_manifest"; then
          browser_registration="$firefox_manifest"
        else
          rm -f -- "$manifest_tmp"
        fi
      fi
    fi
  fi
fi

cat > "$bin_dir/ghostftp-uninstall" <<'UNINSTALL'
#!/usr/bin/env sh
set -eu
prefix=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
bin_dir="$prefix/bin"
share_dir="$prefix/share"
libexec_dir="$prefix/libexec/ghostftp"
native_host="$libexec_dir/ghostftp-native-host"

if [ "$prefix" = "/usr/local" ]; then
  for manifest in \
    /usr/lib/mozilla/native-messaging-hosts/com.ghostftp.bridge.json \
    /usr/lib64/mozilla/native-messaging-hosts/com.ghostftp.bridge.json; do
    if [ -f "$manifest" ] \
      && grep -F '"name": "com.ghostftp.bridge"' "$manifest" >/dev/null 2>&1 \
      && grep -F '"path": "/usr/local/libexec/ghostftp/ghostftp-native-host"' "$manifest" >/dev/null 2>&1 \
      && grep -F 'ghostftp-connection-helper@ghostftp.com' "$manifest" >/dev/null 2>&1; then
      rm -f -- "$manifest"
    fi
  done
fi

rm -f -- "$native_host"
rmdir -- "$libexec_dir" 2>/dev/null || true
rm -f -- "$bin_dir/ghostftp"
rm -f -- "$share_dir/applications/ghost-ftp.desktop"
rm -f -- "$share_dir/icons/hicolor/512x512/apps/ghost-ftp.png"
rm -f -- "$share_dir/doc/ghost-ftp/LICENSE" "$share_dir/doc/ghost-ftp/README.md"
rmdir -- "$share_dir/doc/ghost-ftp" 2>/dev/null || true
rm -f -- "$bin_dir/ghostftp-uninstall"
printf 'Ghost FTP removed from %s. User configuration and server data were not touched.\n' "$prefix"
UNINSTALL
chmod 0755 "$bin_dir/ghostftp-uninstall"
printf 'Ghost FTP installed for %s at %s (CA bundle: %s)\n' "$arch" "$prefix" "$ca_bundle"
printf 'Native browser host installed at: %s\n' "$libexec_dir/ghostftp-native-host"
if [ "$browser_registration" != "not-installed" ]; then
  printf 'Firefox native messaging registered at: %s\n' "$browser_registration"
else
  printf 'Firefox native messaging registration was not installed. Browser integration requires the default /usr/local install with permission to write the system Firefox native-messaging directory.\n'
fi
printf 'Chrome/Edge/Opera registration remains disabled until exact official extension IDs are supplied; wildcard origins are never installed.\n'
printf 'Uninstall with: %s\n' "$bin_dir/ghostftp-uninstall"
INSTALLER
chmod 0755 "$PAYLOAD/install.sh"

cp linux/ghost-ftp.desktop "$PAYLOAD/ghost-ftp.desktop"
cp build/icon.png "$PAYLOAD/ghost-ftp.png"
cp LICENSE "$PAYLOAD/LICENSE"
cp linux/README.md "$PAYLOAD/README.md"
chmod 0644 "$PAYLOAD/ghost-ftp.desktop" "$PAYLOAD/ghost-ftp.png" "$PAYLOAD/LICENSE" "$PAYLOAD/README.md"

make_installer() {
  local distro="$1"
  local lower
  lower="$(printf '%s' "$distro" | tr '[:upper:]' '[:lower:]')"
  local stage="$WORK/${lower}-installer"
  local archive="$WORK/${lower}-installer-payload.tar.gz"
  local output="dist/Ghost-FTP-${VERSION}-Linux-${distro}-Installer.run"

  rm -rf "$stage" "$archive" "$output"
  mkdir -p "$stage"
  cp -a "$PAYLOAD/." "$stage/"
  printf '%s\n' "$distro" > "$stage/DISTRIBUTION"
  chmod 0644 "$stage/DISTRIBUTION"

  tar --sort=name --owner=0 --group=0 --numeric-owner --mtime='UTC 2020-01-01' -C "$stage" -cf - . | gzip -n -9 > "$archive"

  cat > "$output" <<'SELFEXTRACT'
#!/usr/bin/env sh
set -eu
for tool in awk tail gzip tar mktemp; do
  command -v "$tool" >/dev/null 2>&1 || { echo "Ghost FTP installer: missing required tool: $tool" >&2; exit 69; }
done
self=$0
payload_line=$(awk '/^__GHOSTFTP_PAYLOAD_BELOW__$/ { print NR + 1; exit }' "$self")
[ -n "$payload_line" ] || { echo 'Ghost FTP installer: embedded payload marker is missing.' >&2; exit 65; }
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
tail -n +"$payload_line" "$self" | gzip -dc | tar -xf - -C "$tmp"
set +e
"$tmp/install.sh" "$@"
status=$?
set -e
exit "$status"
__GHOSTFTP_PAYLOAD_BELOW__
SELFEXTRACT
  cat "$archive" >> "$output"
  chmod 0755 "$output"
  test -s "$output"
  echo "LINUX_${distro^^}_UNIVERSAL_INSTALLER=$output"
}

make_portable() {
  local distro="$1"
  local lower
  lower="$(printf '%s' "$distro" | tr '[:upper:]' '[:lower:]')"
  local name="Ghost-FTP-${VERSION}-Linux-${distro}-Portable"
  local stage="$WORK/${lower}-portable/$name"
  local output="dist/${name}.tar.gz"

  rm -rf "$WORK/${lower}-portable" "$output"
  mkdir -p "$stage"
  cp -a "$PAYLOAD/bin" "$stage/bin"
  cp "$PAYLOAD/ghostftp" "$stage/ghostftp"
  cp "$PAYLOAD/ghost-ftp.desktop" "$stage/ghost-ftp.desktop"
  cp "$PAYLOAD/ghost-ftp.png" "$stage/ghost-ftp.png"
  cp "$PAYLOAD/LICENSE" "$stage/LICENSE"
  cp "$PAYLOAD/README.md" "$stage/README.md"
  printf '%s\n' "$distro" > "$stage/DISTRIBUTION"
  chmod 0755 "$stage/ghostftp" \
    "$stage/bin/amd64/ghostftp" "$stage/bin/arm64/ghostftp" "$stage/bin/i386/ghostftp" \
    "$stage/bin/amd64/ghostftp-native-host" "$stage/bin/arm64/ghostftp-native-host" "$stage/bin/i386/ghostftp-native-host"
  chmod 0644 "$stage/ghost-ftp.desktop" "$stage/ghost-ftp.png" "$stage/LICENSE" "$stage/README.md" "$stage/DISTRIBUTION"

  tar --sort=name --owner=0 --group=0 --numeric-owner --mtime='UTC 2020-01-01' -C "$(dirname "$stage")" -cf - "$name" | gzip -n -9 > "$output"
  test -s "$output"
  echo "LINUX_${distro^^}_UNIVERSAL_PORTABLE=$output"
}

for distro in Debian Ubuntu Fedora; do
  make_installer "$distro"
  make_portable "$distro"
done

for forbidden in \
  "dist/Ghost-FTP-${VERSION}-Linux-Debian-amd64.deb" \
  "dist/Ghost-FTP-${VERSION}-Linux-Ubuntu-amd64.deb" \
  "dist/Ghost-FTP-${VERSION}-Linux-Fedora-x86_64.rpm" \
  "dist/Ghost-FTP-${VERSION}-Linux-Portable-amd64.tar.gz"; do
  test ! -e "$forbidden"
done

echo "Ghost FTP ${VERSION} Linux universal distro bundles built with amd64, arm64 and i386 application + native-browser-host payloads (telemetry=${telemetry})."
