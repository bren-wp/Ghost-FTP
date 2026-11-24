#!/usr/bin/env bash
# Run only inside the disposable D-Bus/keyring session created by CI.
set -euo pipefail
: "${DBUS_SESSION_BUS_ADDRESS:?isolated D-Bus session required}"
: "${XDG_DATA_HOME:?isolated keyring data directory required}"
: "${XDG_RUNTIME_DIR:?isolated runtime directory required}"
export GHOSTFTP_NATIVE_RESTART_ROOT="$(mktemp -d)"
probe() {
  GHOSTFTP_NATIVE_RESTART_MODE="$1" cargo test --locked -p ghostftp --lib native_vault_restart_probe -- --ignored --nocapture
}
cleanup() {
  probe cleanup
  rm -rf -- "$GHOSTFTP_NATIVE_RESTART_ROOT"
}
trap cleanup EXIT
probe write
owner_pid() {
  gdbus call --session --dest org.freedesktop.DBus --object-path /org/freedesktop/DBus \
    --method org.freedesktop.DBus.GetConnectionUnixProcessID org.freedesktop.secrets |
    sed -n 's/^(uint32 \([0-9][0-9]*\),)$/\1/p'
}
pid="$(owner_pid)"
[[ "$pid" =~ ^[0-9]+$ ]]
test "$(stat -c %u "/proc/$pid")" = "$(id -u)"
test "$(basename "$(readlink "/proc/$pid/exe")")" = "gnome-keyring-daemon"
# Never target a system service or an unrelated session.
tr '\0' '\n' < "/proc/$pid/environ" | grep -Fx "DBUS_SESSION_BUS_ADDRESS=$DBUS_SESSION_BUS_ADDRESS" >/dev/null
kill -KILL "$pid"
for _ in $(seq 1 100); do
  if ! owner_pid >/dev/null 2>&1; then break; fi
  sleep 0.1
done
if owner_pid >/dev/null 2>&1; then
  echo "old Secret Service owner still present" >&2
  exit 1
fi
printf '%s' 'isolated-ci-keyring' | gnome-keyring-daemon --unlock --components=secrets >/dev/null
# The next probe must use the restarted daemon and persisted credential bytes.
new_pid="$(owner_pid)"
test -n "$new_pid"
test "$pid" != "$new_pid"
probe read
