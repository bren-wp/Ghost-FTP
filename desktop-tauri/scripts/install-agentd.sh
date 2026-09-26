#!/bin/sh
# Ghost FTP Agent — one-line installer for a headless Linux machine.
#
#   curl -fsSL https://ghostftp.com/downloads/install-agentd.sh | sh
#
# Downloads the right ghostftp-agentd for this OS/arch from the latest Ghost FTP
# release, drops it on PATH, and (unless --no-service) installs it as a
# background service that runs at boot/login. Then it opens a pairing window so
# you can pair from Ghost FTP right away.
#
# Flags (pass after `| sh -s --`):
#   --no-service      just install the binary; don't set up autostart
#   --read-only       serve browse/read/report only (no exec, no writes)
#   --dir <path>      install dir (default: /usr/local/bin, or ~/.local/bin)
#   --version <tag>   install a specific release tag (default: latest)
set -eu

BIN="ghostftp-agentd"
SERVICE=1
READONLY=""
VERSION="latest"
INSTALL_DIR=""

while [ $# -gt 0 ]; do
  case "$1" in
    --no-service) SERVICE=0 ;;
    --read-only) READONLY="--read-only" ;;
    --dir) INSTALL_DIR="$2"; shift ;;
    --version) VERSION="$2"; shift ;;
    -h|--help) sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown flag: $1" >&2; exit 1 ;;
  esac
  shift
done

# ---- Detect OS/arch and map to the release asset name ------------------------
os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Linux)  os_tag="linux" ;;
  *) echo "Unsupported OS '$os'. On Windows, download ghostftp-agentd from the release page." >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) arch_tag="x86_64" ;;
  arm64|aarch64)
      echo "No prebuilt Linux arm64 ghostftp-agentd yet — build from source (cargo build -p ghostftp-agentd)." >&2
      exit 1 ;;
  *) echo "Unsupported architecture '$arch'." >&2; exit 1 ;;
esac
asset="${BIN}-${os_tag}-${arch_tag}"

# ---- Resolve the download URL ------------------------------------------------
if [ "$VERSION" = "latest" ]; then
  url="https://ghostftp.com/downloads/${asset}"
else
  url="https://ghostftp.com/downloads/${VERSION}/${asset}"
fi

# ---- Choose an install dir (prefer a system dir, fall back to ~/.local/bin) --
if [ -z "$INSTALL_DIR" ]; then
  if [ -w /usr/local/bin ] 2>/dev/null; then INSTALL_DIR="/usr/local/bin"
  elif [ "$(id -u)" = "0" ]; then INSTALL_DIR="/usr/local/bin"
  else INSTALL_DIR="$HOME/.local/bin"
  fi
fi
mkdir -p "$INSTALL_DIR"
dest="${INSTALL_DIR}/${BIN}"

echo "Downloading ${asset} → ${dest}"
if command -v curl >/dev/null 2>&1; then
  curl -fSL "$url" -o "$dest"
elif command -v wget >/dev/null 2>&1; then
  wget -O "$dest" "$url"
else
  echo "Need curl or wget to download." >&2; exit 1
fi
chmod +x "$dest"


case ":$PATH:" in
  *":$INSTALL_DIR:"*) : ;;
  *) echo "Note: $INSTALL_DIR isn't on your PATH — add it or call $dest directly." ;;
esac

echo "Installed: $("$dest" info 2>/dev/null | head -1 || echo "$BIN")"

# ---- Install the service (optional) ------------------------------------------
if [ "$SERVICE" = "1" ]; then
  echo "Setting up the background service…"
  # shellcheck disable=SC2086
  "$dest" install $READONLY || {
    echo "Service install didn't complete — you can still run '$BIN run' yourself." >&2
  }
fi

# ---- Open a pairing window so the user can pair from Ghost FTP now -----------------
echo
echo "Opening a pairing window. In Ghost FTP: New Connection → Ghost FTP Agent → pick this"
echo "machine → enter the code below. (Ctrl-C when you've paired.)"
echo
# shellcheck disable=SC2086
exec "$dest" pair $READONLY
