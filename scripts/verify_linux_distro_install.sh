#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "usage: $0 <debian|ubuntu|fedora> <expected-version> <expected-os-version> <package-path>" >&2
  exit 2
fi

target="$1"
expected_version="$2"
expected_os_version="$3"
package_path="$4"

[[ -s "$package_path" ]] || { echo "package not found: $package_path" >&2; exit 1; }

# The install gate must prove the package in the distro it claims to target.
# Fail closed if a workflow/container change silently substitutes another OS.
# shellcheck disable=SC1091
source /etc/os-release
case "$target" in
  debian)
    [[ "${ID:-}" == "debian" ]] || { echo "expected Debian, got ${ID:-unknown}" >&2; exit 1; }
    [[ "${VERSION_ID:-}" == "$expected_os_version" ]] || { echo "expected Debian $expected_os_version, got ${VERSION_ID:-unknown}" >&2; exit 1; }
    ;;
  ubuntu)
    [[ "${ID:-}" == "ubuntu" ]] || { echo "expected Ubuntu, got ${ID:-unknown}" >&2; exit 1; }
    [[ "${VERSION_ID:-}" == "$expected_os_version" ]] || { echo "expected Ubuntu $expected_os_version, got ${VERSION_ID:-unknown}" >&2; exit 1; }
    ;;
  fedora)
    [[ "${ID:-}" == "fedora" ]] || { echo "expected Fedora, got ${ID:-unknown}" >&2; exit 1; }
    [[ "${VERSION_ID:-}" == "$expected_os_version" ]] || { echo "expected Fedora $expected_os_version, got ${VERSION_ID:-unknown}" >&2; exit 1; }
    ;;
  *)
    echo "unsupported target: $target" >&2
    exit 2
    ;;
esac

verify_gui_smoke() {
  local smoke_home data_root runtime_dir xvfb_pid app_pid app_status=0
  # Keep the test HOME under a non-world-writable system path, then create the
  # standard Linux user-data root that normally already exists in a desktop
  # session. Ghost FTP deliberately treats LocalAppData as a trusted root and
  # only creates/verifies Ghost FTP-owned descendants beneath it.
  smoke_home="$(mktemp -d /var/lib/ghostftp-ci-home.XXXXXX)"
  chmod 0700 "$smoke_home"
  mkdir -p "$smoke_home/.local/share"
  chmod 0700 "$smoke_home/.local" "$smoke_home/.local/share"
  data_root="$smoke_home/.local/share"

  # Use a private runtime directory so the single-instance lock and any future
  # per-user runtime state do not share the container's global /tmp namespace.
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

  test -d "$data_root"
  test "$(stat -c '%a' "$data_root")" = "700"
  test "$(stat -c '%a' "$runtime_dir")" = "700"

  HOME="$smoke_home" XDG_RUNTIME_DIR="$runtime_dir" DISPLAY=:99 XAUTHORITY= /usr/bin/ghostftp >"$smoke_home/ghostftp.log" 2>&1 &
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
    if ! kill -0 "$app_pid" 2>/dev/null; then
      break
    fi
    sleep 0.1
  done
  if kill -0 "$app_pid" 2>/dev/null; then
    kill -KILL "$app_pid" 2>/dev/null || true
  fi
  wait "$app_pid" 2>/dev/null || true

  kill "$xvfb_pid" 2>/dev/null || true
  wait "$xvfb_pid" 2>/dev/null || true
  rm -rf "$smoke_home"
  trap - RETURN
  echo "GHOSTFTP_INSTALLED_GUI_SMOKE=PASS target=$target"
}

verify_common_runtime_tools() {
  command -v curl >/dev/null
  command -v ssh >/dev/null
  command -v sftp >/dev/null
  if [[ "$target" == "fedora" ]]; then
    test -s /etc/pki/tls/certs/ca-bundle.crt
  else
    test -s /etc/ssl/certs/ca-certificates.crt
  fi
}

if [[ "$target" == "debian" || "$target" == "ubuntu" ]]; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y --no-install-recommends xvfb >/dev/null

  expected_distro="Debian"
  [[ "$target" == "ubuntu" ]] && expected_distro="Ubuntu"
  [[ "$(dpkg-deb -f "$package_path" Package)" == "ghost-ftp" ]]
  [[ "$(dpkg-deb -f "$package_path" Version)" == "$expected_version" ]]
  [[ "$(dpkg-deb -f "$package_path" Architecture)" == "amd64" ]]
  [[ "$(dpkg-deb -f "$package_path" X-GhostFTP-Distribution)" == "$expected_distro" ]]
  [[ "$(dpkg-deb -f "$package_path" Depends)" == "ca-certificates, curl, openssh-client" ]]

  apt-get install -y --no-install-recommends "$package_path" >/dev/null
  [[ "$(dpkg-query -W -f='${Status}' ghost-ftp)" == "install ok installed" ]]
  [[ "$(dpkg-query -W -f='${Version}' ghost-ftp)" == "$expected_version" ]]
  [[ "$(dpkg-query -W -f='${Architecture}' ghost-ftp)" == "amd64" ]]
  dpkg-query -W ca-certificates curl openssh-client >/dev/null
  verify_common_runtime_tools

  test -x /usr/bin/ghostftp
  test -f /usr/share/applications/ghost-ftp.desktop
  test -f /usr/share/icons/hicolor/512x512/apps/ghost-ftp.png
  dpkg-query -L ghost-ftp | grep -Fx '/usr/bin/ghostftp' >/dev/null
  dpkg-query -L ghost-ftp | grep -Fx '/usr/share/applications/ghost-ftp.desktop' >/dev/null
  dpkg-query -L ghost-ftp | grep -Fx '/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png' >/dev/null

  verify_gui_smoke

  apt-get remove -y ghost-ftp >/dev/null
  if dpkg-query -W ghost-ftp >/dev/null 2>&1; then
    state="$(dpkg-query -W -f='${db:Status-Abbrev}' ghost-ftp 2>/dev/null || true)"
    [[ "$state" != ii* ]] || { echo "ghost-ftp remains installed after removal" >&2; exit 1; }
  fi
  test ! -e /usr/bin/ghostftp
  test ! -e /usr/share/applications/ghost-ftp.desktop
  test ! -e /usr/share/icons/hicolor/512x512/apps/ghost-ftp.png
else
  dnf install -y xorg-x11-server-Xvfb >/dev/null

  [[ "$(rpm -qp --qf '%{NAME}' "$package_path")" == "ghost-ftp" ]]
  [[ "$(rpm -qp --qf '%{VERSION}' "$package_path")" == "$expected_version" ]]
  [[ "$(rpm -qp --qf '%{ARCH}' "$package_path")" == "x86_64" ]]
  [[ "$(rpm -qp --qf '%{DISTRIBUTION}' "$package_path")" == "Fedora" ]]
  rpm -qp --requires "$package_path" | grep -Fx 'ca-certificates' >/dev/null
  rpm -qp --requires "$package_path" | grep -Fx 'curl' >/dev/null
  rpm -qp --requires "$package_path" | grep -Fx 'openssh-clients' >/dev/null

  dnf install -y "$package_path" >/dev/null
  [[ "$(rpm -q --qf '%{VERSION}' ghost-ftp)" == "$expected_version" ]]
  [[ "$(rpm -q --qf '%{ARCH}' ghost-ftp)" == "x86_64" ]]
  rpm -q ca-certificates curl openssh-clients >/dev/null
  verify_common_runtime_tools

  test -x /usr/bin/ghostftp
  test -f /usr/share/applications/ghost-ftp.desktop
  test -f /usr/share/icons/hicolor/512x512/apps/ghost-ftp.png
  test -f /usr/share/doc/ghost-ftp/LICENSE
  test -f /usr/share/doc/ghost-ftp/README.md
  rpm -ql ghost-ftp | grep -Fx '/usr/bin/ghostftp' >/dev/null
  rpm -ql ghost-ftp | grep -Fx '/usr/share/applications/ghost-ftp.desktop' >/dev/null
  rpm -ql ghost-ftp | grep -Fx '/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png' >/dev/null

  verify_gui_smoke

  dnf remove -y ghost-ftp >/dev/null
  ! rpm -q ghost-ftp >/dev/null 2>&1
  test ! -e /usr/bin/ghostftp
  test ! -e /usr/share/applications/ghost-ftp.desktop
  test ! -e /usr/share/icons/hicolor/512x512/apps/ghost-ftp.png
  test ! -e /usr/share/doc/ghost-ftp/LICENSE
  test ! -e /usr/share/doc/ghost-ftp/README.md
fi

echo "GHOSTFTP_DISTRO_INSTALL=PASS target=$target os=${ID}-${VERSION_ID} package_version=$expected_version"
