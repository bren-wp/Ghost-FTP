#!/usr/bin/env bash
set -euo pipefail
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo/ghostftp-desktop/src-tauri"
cargo build --release --locked -p ghostftp-cli
"$repo/ghostftp-desktop/src-tauri/target/release/ghostftp-cli" --version
