#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <debian|ubuntu|fedora> <expected-version> <expected-os-version> <installer-path>" >&2
  exit 2
fi

target="$1"
expected_version="$2"
expected_os_version="$3"
installer_path="$4"
[[ "$expected_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "invalid expected version: $expected_version" >&2; exit 2; }
[[ -s "$installer_path" && -x "$installer_path" ]] || { echo "installer not found or executable: $installer_path" >&2; exit 1; }

# Fail closed if a workflow/container change silently substitutes another OS.
# shellcheck disable=SC1091
source /etc/os-release
case "$target" in
  debian)
    [[ "${ID:-}" == "debian" ]] || { echo "expected Debian, got ${ID:-unknown}" >&2; exit 1; }
    [[ "${VERSION_ID:-}" == "$expected_os_version" ]] || { echo "expected Debian $expected_os_version, got ${VERSION_ID:-unknown}" >&2; exit 1; }
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y --no-install-recommends ca-certificates curl openssh-client xvfb >/dev/null
    ;;
  ubuntu)
    [[ "${ID:-}" == "ubuntu" ]] || { echo "expected Ubuntu, got ${ID:-unknown}" >&2; exit 1; }
    [[ "${VERSION_ID:-}" == "$expected_os_version" ]] || { echo "expected Ubuntu $expected_os_version, got ${VERSION_ID:-unknown}" >&2; exit 1; }
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y --no-install-recommends ca-certificates curl openssh-client xvfb >/dev/null
    ;;
  fedora)
    [[ "${ID:-}" == "fedora" ]] || { echo "expected Fedora, got ${ID:-unknown}" >&2; exit 1; }
    [[ "${VERSION_ID:-}" == "$expected_os_version" ]] || { echo "expected Fedora $expected_os_version, got ${VERSION_ID:-unknown}" >&2; exit 1; }
    dnf install -y ca-certificates curl openssh-clients xorg-x11-server-Xvfb >/dev/null
    ;;
  *)
    echo "unsupported target: $target" >&2
    exit 2
    ;;
esac

for tool in curl ssh sftp; do
  command -v "$tool" >/dev/null || { echo "runtime dependency unavailable after installation: $tool" >&2; exit 1; }
done

ca_bundle=""
for candidate in \
  /etc/ssl/certs/ca-certificates.crt \
  /etc/pki/tls/certs/ca-bundle.crt \
  /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem; do
  if [[ -s "$candidate" ]]; then ca_bundle="$candidate"; break; fi
done
[[ -n "$ca_bundle" ]] || { echo 'runtime CA certificate bundle unavailable' >&2; exit 1; }

prefix=/usr/local
GHOSTFTP_PREFIX="$prefix" "$installer_path"

test -x "$prefix/bin/ghostftp"
test -x "$prefix/bin/ghostftp-uninstall"
test -f "$prefix/share/applications/ghost-ftp.desktop"
test -f "$prefix/share/icons/hicolor/512x512/apps/ghost-ftp.png"
test -f "$prefix/share/doc/ghost-ftp/LICENSE"
test -f "$prefix/share/doc/ghost-ftp/README.md"
grep -Fx 'Exec=ghostftp' "$prefix/share/applications/ghost-ftp.desktop" >/dev/null

verify_gui_smoke() {
  local smoke_home runtime_dir xvfb_pid app_pid app_status=0
  smoke_home="$(mktemp -d /var/lib/ghostftp-ci-home.XXXXXX)"
  chmod 0700 "$smoke_home"
  mkdir -p "$smoke_home/.local/share"
  chmod 0700 "$smoke_home/.local" "$smoke_home/.local/share"
  runtime_dir="$smoke_home/runtime"
  mkdir "$runtime_dir"
  chmod 0700 "$runtime_dir"

  Xvfb :99 -screen 0 1280x800x24 -nolisten tcp -ac >"$smoke_home/xvfb.log" 2>&1 &
  xvfb_pid=$!
  trap 'kill "$xvfb_pid" 2>/dev/null || true; rm -rf "$smoke_home"' RETURN
  for _ in $(seq 1 50); do
    [[ -S /tmp/.X11-unix/X99 ]] && break
    kill -0 "$xvfb_pid" 2>/dev/null || {
      cat "$smoke_home/xvfb.log" >&2 || true
      echo "Xvfb exited before becoming ready" >&2
      return 1
    }
    sleep 0.1
  done
  [[ -S /tmp/.X11-unix/X99 ]] || { echo "Xvfb socket did not become ready" >&2; return 1; }

  HOME="$smoke_home" XDG_RUNTIME_DIR="$runtime_dir" DISPLAY=:99 XAUTHORITY= "$prefix/bin/ghostftp" >"$smoke_home/ghostftp.log" 2>&1 &
  app_pid=$!
  sleep 2
  if ! kill -0 "$app_pid" 2>/dev/null; then
    wait "$app_pid" || app_status=$?
    cat "$smoke_home/ghostftp.log" >&2 || true
    echo "installed Ghost FTP exited during GUI startup smoke (status=$app_status)" >&2
    return 1
  fi

  kill -TERM "$app_pid" 2>/dev/null || true
  for _ in $(seq 1 30); do
    if ! kill -0 "$app_pid" 2>/dev/null; then break; fi
    sleep 0.1
  done
  if kill -0 "$app_pid" 2>/dev/null; then kill -KILL "$app_pid" 2>/dev/null || true; fi
  wait "$app_pid" 2>/dev/null || true
  kill "$xvfb_pid" 2>/dev/null || true
  wait "$xvfb_pid" 2>/dev/null || true
  rm -rf "$smoke_home"
  trap - RETURN
  echo "GHOSTFTP_INSTALLED_GUI_SMOKE=PASS target=$target"
}

verify_gui_smoke

"$prefix/bin/ghostftp-uninstall"
test ! -e "$prefix/bin/ghostftp"
test ! -e "$prefix/bin/ghostftp-uninstall"
test ! -e "$prefix/share/applications/ghost-ftp.desktop"
test ! -e "$prefix/share/icons/hicolor/512x512/apps/ghost-ftp.png"
test ! -e "$prefix/share/doc/ghost-ftp/LICENSE"
test ! -e "$prefix/share/doc/ghost-ftp/README.md"

echo "GHOSTFTP_DISTRO_INSTALL=PASS target=$target os=${ID}-${VERSION_ID} version=$expected_version ca_bundle=$ca_bundle"
