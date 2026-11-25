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
  # Trigger D-Bus activation and wait for the actual provider registration.
  for _ in {1..100}; do
    if result="$(gdbus call --session --dest org.freedesktop.DBus \
      --object-path /org/freedesktop/DBus --method org.freedesktop.DBus.GetConnectionUnixProcessID org.freedesktop.secrets 2>/dev/null)" &&
      [[ "$result" =~ uint32[[:space:]]([0-9]+) ]]; then
      pid="${BASH_REMATCH[1]}"
      break
    fi
    sleep .1
  done
  [[ -n "${pid:-}" ]] || return 1
  [[ "$(basename "$(readlink -f "/proc/$pid/exe")")" == gnome-keyring-daemon ]] || return 1
  # Prove the daemon belongs to this disposable HOME, not another session.
  grep -zFxq "HOME=$HOME" "/proc/$pid/environ" || return 1
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
start_provider() {
  printf %s isolated-fixture-keyring | gnome-keyring-daemon --unlock --components=secrets >/dev/null
  timeout 10s gdbus call --session --dest org.freedesktop.secrets \
    --object-path /org/freedesktop/secrets --method org.freedesktop.DBus.Peer.Ping >/dev/null
  daemon_pid="$(owner_pid)"
}
start_provider
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
start_provider
[[ "$old" != "$daemon_pid" ]]
touch "$root/restart.go"
for operation in write delete; do
  wait_ready "$operation-commit"
  old="$daemon_pid"
  kill -KILL "$daemon_pid"
  daemon_pid=
  touch "$root/$operation-commit.go"
  wait_ready "$operation-restart"
  start_provider
  [[ "$old" != "$daemon_pid" ]]
  touch "$root/$operation-restart.go"
done
wait "$child"
child=
