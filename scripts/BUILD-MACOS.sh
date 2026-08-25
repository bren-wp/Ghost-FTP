#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
[[ -f "$ROOT/VERSION" ]] || { echo 'VERSION is missing.' >&2; exit 1; }
exec bash "$ROOT/macos/BUILD.sh" "$@"
