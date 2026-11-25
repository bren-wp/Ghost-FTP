#!/usr/bin/env bash
# Run only in a fresh dbus-run-session and disposable HOME.
set -euo pipefail
[[ "${GHOSTFTP_VAULT_PROBE_OPT_IN:-}" == disposable-fixtures ]]
[[ "$HOME" == "$1/home" ]]
root="$1"
probe="$2"
daemon_pid=
child=
cleanup() {
  if [[ -n "$daemon_pid" ]]; then kill -CONT "$daemon_pid" 2>/dev/null || true; fi
  if [[ -n "$child" ]]; then kill "$child" 2>/dev/null || true; wait "$child" 2>/dev/null || true; fi
}
trap cleanup EXIT
owner_pid() {
  local result pid
  result="$(gdbus call --session --dest org.freedesktop.DBus \
    --object-path /org/freedesktop/DBus --method org.freedesktop.DBus.GetConnectionUnixProcessID org.freedesktop.secrets)"
  [[ "$result" =~ uint32[[:space:]]([0-9]+) ]]
  pid="${BASH_REMATCH[1]}"
  [[ "$(basename "$(readlink -f "/proc/$pid/exe")")" == gnome-keyring-daemon ]]
  # Prove the daemon belongs to this disposable HOME, not another session.
  grep -zFxq "HOME=$HOME" "/proc/$pid/environ"
  printf '%s\n' "$pid"
}
wait_ready() {
  for _ in {1..400}; do
    [[ ! -e "$root/$1.ready" ]] || return 0
    kill -0 "$child" 2>/dev/null || return 1
    sleep .05
  done
  return 1
}
printf %s isolated-fixture-keyring | gnome-keyring-daemon --unlock --components=secrets >/dev/null
daemon_pid="$(owner_pid)"
"$probe" gnome-faults "$root" &
child=$!
wait_ready block
kill -STOP "$daemon_pid"
touch "$root/block.go"
wait_ready resume
kill -CONT "$daemon_pid"
touch "$root/resume.go"
wait_ready interrupt
kill -STOP "$daemon_pid"
touch "$root/interrupt.go"
# The lookup is now allowed to enter the blocked provider call.
sleep .5
kill -KILL "$daemon_pid"
old="$daemon_pid"
daemon_pid=
wait_ready restart
printf %s isolated-fixture-keyring | gnome-keyring-daemon --unlock --components=secrets >/dev/null
daemon_pid="$(owner_pid)"
[[ "$old" != "$daemon_pid" ]]
touch "$root/restart.go"
wait "$child"
child=
