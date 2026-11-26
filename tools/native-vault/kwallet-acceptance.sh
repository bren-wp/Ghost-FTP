#!/usr/bin/env bash
# Run manually in a dedicated, unlocked KWallet test account / disposable VM.
# No provider restart or collection lock is attempted by this script.
set -euo pipefail
[[ "${GHOSTFTP_VAULT_PROBE_OPT_IN:-}" == disposable-fixtures ]]
[[ $# == 1 && -x "$1" ]]

dbus_call() {
    gdbus call --session --dest org.freedesktop.DBus \
        --object-path /org/freedesktop/DBus --method "org.freedesktop.DBus.$1" "$2"
}

get_owner() {
    local result pattern
    pattern="^\\('(:[[:alnum:]_.-]+)',\\)$"
    result="$(dbus_call GetNameOwner org.freedesktop.secrets)" || return 1
    [[ "$result" =~ $pattern ]] || return 1
    printf '%s\n' "${BASH_REMATCH[1]}"
}

provider_identity() {
    local owner result pid executable provider confirmed
    owner="$(get_owner)" || return 1
    result="$(dbus_call GetConnectionUnixProcessID "$owner")" || return 1
    [[ "$result" =~ ^\(uint32[[:space:]]([0-9]+),\)$ ]] || return 1
    pid="${BASH_REMATCH[1]}"
    executable="$(readlink -f "/proc/$pid/exe")" || return 1
    provider="$(basename "$executable")" || return 1
    case "$provider" in kwalletd5|kwalletd6|ksecretd) ;; *) return 1;; esac
    confirmed="$(get_owner)" || return 1
    [[ "$confirmed" == "$owner" ]] || return 1
    printf '%s %s %s\n' "$owner" "$pid" "$executable"
}

before="$(provider_identity)" || { echo "KWallet provider not verified before CRUD" >&2; exit 1; }
root="$(mktemp -d)"
trap 'rm -rf -- "$root"' EXIT
timeout --kill-after=5s 30s "$1" provider-crud "$root"
after="$(provider_identity)" || { echo "KWallet provider not verified after CRUD" >&2; exit 1; }
[[ "$before" == "$after" ]] || { echo "KWallet provider identity changed during CRUD" >&2; exit 1; }
printf 'verified_provider=%s\n' "$(basename "${after#* * }")"
