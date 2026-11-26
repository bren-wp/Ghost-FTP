#!/usr/bin/env bash
# Mock only the provider identity boundary; this is NOT native KWallet acceptance.
set -euo pipefail
root="$(mktemp -d)"
trap 'rm -rf -- "$root"' EXIT
script="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/kwallet-acceptance.sh"
mkdir "$root/bin"
cat > "$root/bin/gdbus" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
case "${*: -2:1}" in
  org.freedesktop.DBus.GetNameOwner)
    n=$(cat "$MOCK_COUNT"); n=$((n+1)); echo "$n" > "$MOCK_COUNT"
    if [[ "$MOCK_MODE" == missing ]]; then exit 1; fi
    if [[ "$MOCK_MODE" == malformed ]]; then printf 'unexpected\n'; exit 0; fi
    if [[ "$MOCK_MODE" == race && "$n" -ge 2 || "$MOCK_MODE" == replaced && "$n" -ge 3 ]]; then
      printf "(':1.43',)\n"
    else printf "(':1.42',)\n"; fi ;;
  org.freedesktop.DBus.GetConnectionUnixProcessID)
    [[ "${*: -1}" == :1.42 || "${*: -1}" == :1.43 ]]
    printf '(uint32 1234,)\n' ;;
  *) exit 1 ;;
esac
MOCK
cat > "$root/bin/readlink" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$MOCK_MODE" == fallback ]]; then printf '/usr/bin/gnome-keyring-daemon\n'; else printf '/usr/bin/kwalletd6\n'; fi
MOCK
cat > "$root/probe" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1" == provider-crud && -d "$2" ]]
: > "$MOCK_PROBE"
[[ "$MOCK_MODE" != probe_failure ]]
MOCK
chmod +x "$root/bin/"* "$root/probe"
export PATH="$root/bin:$PATH" MOCK_COUNT="$root/count" MOCK_PROBE="$root/probe-ran"
export GHOSTFTP_VAULT_PROBE_OPT_IN=disposable-fixtures
for mode in stable fallback replaced race missing malformed probe_failure; do
    export MOCK_MODE="$mode"
    echo 0 > "$MOCK_COUNT"; rm -f "$MOCK_PROBE"
    if "$script" "$root/probe" > "$root/result" 2>&1; then
        [[ "$mode" == stable ]]
        grep -Fxq 'verified_provider=kwalletd6' "$root/result"
    else
        [[ "$mode" != stable ]]
        ! grep -q '^verified_provider=' "$root/result"
    fi
    case "$mode" in stable|replaced|probe_failure) [[ -f "$MOCK_PROBE" ]];; *) [[ ! -e "$MOCK_PROBE" ]];; esac
    printf 'mock_identity_case=%s PASS\n' "$mode"
done
