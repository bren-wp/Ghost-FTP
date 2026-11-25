#!/usr/bin/env bash
# Run manually in a dedicated, unlocked KWallet test account / disposable VM.
# No provider restart or collection lock is attempted by this script.
set -euo pipefail
[[ "${GHOSTFTP_VAULT_PROBE_OPT_IN:-}" == disposable-fixtures ]]
[[ $# == 1 && -x "$1" ]]
result="$(gdbus call --session --dest org.freedesktop.DBus \
  --object-path /org/freedesktop/DBus --method org.freedesktop.DBus.GetConnectionUnixProcessID org.freedesktop.secrets)"
[[ "$result" =~ uint32[[:space:]]([0-9]+) ]]
pid="${BASH_REMATCH[1]}"
provider="$(basename "$(readlink -f "/proc/$pid/exe")")"
case "$provider" in kwalletd5|kwalletd6|ksecretd) ;; *) echo "KWallet provider not verified" >&2; exit 1;; esac
printf 'verified_provider=%s\n' "$provider"
root="$(mktemp -d)"
trap 'rm -rf -- "$root"' EXIT
timeout --kill-after=5s 30s "$1" provider-crud "$root"
