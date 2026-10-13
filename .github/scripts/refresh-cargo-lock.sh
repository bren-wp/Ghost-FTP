#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TAURI_DIR="$ROOT/ghostftp-desktop/src-tauri"

cd "$TAURI_DIR"

# Verify the existing committed lock graph rather than regenerating it.
# Re-resolving all registry packages can silently upgrade transitive
# dependencies and previously tried to downgrade Tauri below the exact
# version required by Cargo.toml (2.11.6). A development version bump
# must not change third-party dependency versions or published hashes.
# If workspace package versions drift, fail closed and require a reviewed
# lockfile update committed with the manifest changes.
cargo metadata --locked --format-version 1 >/dev/null

check_locked() {
  local package="$1"
  local expected="$2"
  local actual
  actual="$(awk -v pkg="$package" '
    $0 == "name = \"" pkg "\"" { found=1; next }
    found && /^version = / { gsub(/version = |"/, ""); print; exit }
  ' Cargo.lock)"
  if [[ "$actual" != "$expected" ]]; then
    echo "Cargo.lock mismatch for $package: expected $expected, got ${actual:-missing}" >&2
    exit 1
  fi
}

# The Tauri core lock must follow the exact pinned version in the manifest;
# do not hardcode an older unrelated version in the refresh workflow.
TAURI_PIN="$(sed -nE 's/^tauri = \{ version = "=([^"]+)".*/\1/p' Cargo.toml)"
if [[ -z "$TAURI_PIN" ]]; then
  echo "Cannot resolve the exact Tauri dependency pin from Cargo.toml." >&2
  exit 1
fi

check_locked tauri "$TAURI_PIN"
check_locked tauri-runtime 2.11.3
check_locked tauri-runtime-wry 2.11.4
check_locked tauri-macros 2.6.3
check_locked tauri-utils 2.9.3
check_locked tauri-codegen 2.6.3
check_locked tauri-plugin 2.6.3
check_locked tauri-build 2.6.3

echo "Canonical Cargo.lock metadata and dependency versions verified."
