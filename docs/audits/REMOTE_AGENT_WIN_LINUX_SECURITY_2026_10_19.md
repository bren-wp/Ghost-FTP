# Ghost FTP Remote Agent — Windows/Linux security audit
Date: 19 October 2026 · Status: **unreleased security preview; not certified**

## Repository and change-management facts
Original PR #214 was already **merged and closed** before this audit request. Per owner's prohibition, no new pull request has been opened and no new merge or GitHub Release was performed. New work is committed on the existing branch `fix/winlinux-stability-quality-v09548`, after the merged PR head, and is **not in main**. Development metadata remains `0.95.49` Preview, following immutable published `v0.95.48`.

## Scope actually inspected
`ghostftp-agentd/src/config.rs`, `server.rs`, `ops.rs`, `main.rs`, `jobs.rs`; `ghostftp-agent-proto` (`identity.rs`, `pairing.rs`, `secure.rs`, `msg.rs`, framing); desktop `agent_host.rs`, `terminal.rs`, `bridge.rs`, `grant.rs`, `deeplink.rs`, `capabilities/default.json`, `tauri.conf.json`, Settings/IPC/types; Go local runtime `main.go` and `security.go`. Static inspection of these files does **not** imply every path in the 7k-line bridge was exhaustively dynamically tested.

## Confirmed findings and implemented controls
| Area | Confirmed risk | Source-level remediation |
| --- | --- | --- |
| Pairing grants | New paired devices inherited global permissions with exec+write enabled | New daemon policy defaults both off. New peer gets explicit `read=true, write=false, delete=false, terminal=false`. Previous peers migrate from existing daemon global policy to explicit per-peer grants, preserving legacy effective access; the migration is saved before use |
| Privilege enforcement | Policy checked only global `allow_exec`/`allow_write` | Re-check pinned static key and per-peer authorization before **every** request; SystemInfo requires read, Delete requires delete; writes and shell require distinct privileges; global policy is an additional stop switch |
| Active revocation | Unpair affected only new connections; existing session could continue | Owner-side Tauri update/revocation notifies active sessions to close, and kills the revoked device's detached process jobs; disable-exec machine-wide kills all tracked jobs |
| Job isolation | Shared daemon JobStore used requester-provided IDs, potentially allowing one peer to poll/kill another peer's output | Prefix internal job IDs with pinned peer static key; return external ID to its rightful peer; kill by peer namespace |
| Pairing brute force | Six-digit Noise PSK attempts were limited by window TTL but not count | 6 handshake attempts maximum per pairing window, fail-closed when exhausted; handshake timeout 15s; initial length sniff 10s; max 32 simultaneous live connections |
| Memory exhaustion | Logical Noise messages could grow to a multi-megabyte frame sequence before validation | Enforce the same 16MiB logical-message cap on receive and send; validate continuation flags and reject overflow before appending |
| Network exposure | Both standalone daemon and in-app Tauri host bound `0.0.0.0` by default; mDNS published | Default both listeners to 127.0.0.1 and no mDNS on loopback; require explicit numeric `--bind IP` for headless LAN use or a visible network-access switch in desktop security settings; warn about firewall exposure |
| Controller UI | No separate owner switches by pinned controller | Settings → Security offers per-peer read, write, delete, terminal and revoke controls; standalone `ghostftp-agentd permit <pin> <category> <on|off>` updates saved device grants, requiring restart in headless mode |
| Identity disclosure | Deriving Rust Debug on `Identity` exposed raw private X25519 key in formatted logs | Custom Debug redacts private key. Owner-only 0600 identity file uses temp write/fsync/atomic rename; Unix agent directory chmod 0700 |
| Pin/config durability | JSON save directly truncated live policy/pin list | Private 0600 staged file + rename; bound config size and peer count; reject nonregular config; migrate legacy grants idempotently |
| Sensitive logging | Operation summary included raw shell command, remote file paths, and provider error bodies | Agent audit logs use operation category/status rather than paths, command or response payload |
| Grant token / SSRF | reqwest followed redirects for bearer-token URLs; error strings could echo those URLs; untrusted issuers could reach internal networks | Reject credential/query/fragment issuer URLs; resolve and pin a permitted destination; disallow redirects; reject reserved/private addresses; redact network and provider error details |
| Local Go runtime | GET `/api/session` gave local bearer token before rejecting hostile Host/Origin, enabling DNS rebinding scenarios | Check loopback-only Host, exact local Origin and cross-site Fetch Metadata **before every route**, including session-token bootstrap |
| Terminal IPC | Rust PTY accepted unbounded dimensions/input and unlimited opens | Bound sessions to 32, input to 128KiB per call, size 20–500 columns / 5–300 rows; no caller-supplied terminal ID in errors |

## Boundary and compatibility caveats
- Remote shell executes intentionally through PowerShell or /bin/sh under the current OS user. Shell command strings are **not escaped into inert data**: doing so would remove intended terminal functionality. Restrict terminal to explicit owner-approved peers, use least-privileged OS accounts, and do not expose shell permissions to untrusted devices.
- A remote **write** privilege includes replacement/truncation. The separate **delete** permission blocks explicit Delete operations, not every possible destructive consequence of writing/rename. Treat write access as high trust.
- Existing paired peers with old allow-all policy retain equivalent grants during migration; explicitly review/reduce them in Security settings. New peers default read-only. Headless CLI changes require daemon reload/restart to take effect (embedded Tauri changes are live).
- Active Tauri-host revocation interrupts current Noise channels and tracked subprocesses. An already-started uninterruptible filesystem operation can still have side effects. No guarantee of fully transactional rollback after remote revocation.
- Global six-attempt pairing limit prevents online brute force but allows an attacker to exhaust the active pairing window (availability DoS). The owner must explicitly reopen pairing.
- The Noise private identity remains a file in an OS-protected directory; it is not automatically secured by TPM/DPAPI/hardware key. Validate Windows NTFS ACL inheritance and Linux filesystem snapshots and backups.
- LAN binding needs host firewall policy. This project cannot programmatically guarantee correct Windows Firewall/ufw/firewalld rules on unmanaged devices.
- Webview capabilities still include updater and related plugin permissions required by shipped app functionality; capability minimization beyond existing main-window scope needs independent functionality/attack-surface review.
- Remote bridge uses a separate 127.0.0.1 bearer-token and interactive approval model; this audit does not claim all 7k-line bridge workflows have been fuzzed or Windows/Linux desktop acceptance-tested.
- `grant.rs` allows HTTP only for explicit loopback development issuers; publishing that mode without further consent is not recommended. Live DNS/rebinding races and malicious endpoints require OS/network testing.
- `secret-tool` migration and encrypted backups are addressed in the parallel credential lifecycle audit, not fully certified here.

## Negative / regression test evidence to collect
- Rust config: new-peer read-only, legacy peer migration, valid persisted grants and first-use rejection after denied permissions.
- Rust server: denies exec/write/delete, per-controller JobStore namespace, pairing attempt exhaustion / reset, Noise PSK/pinned-key mismatch, logical message bounds, active revoke during transfer and detached process cancellation.
- Go: malicious Host, DNS-rebound Host, forbidden Origin, cross-site Fetch Metadata, and normal 127.0.0.1 and localhost requests.
- Rust Identity: Debug cannot include a private sentinel; corrupt persisted identity fails closed.
- Rust grant: malformed private issuer address, URL credentials/query/fragment, redirect to off-origin, DNS rebinding and token-not-in-error.
- Rust PTY backend: invalid dimensions and oversized writes rejected independently of React UI.
- GitHub workflow verification on **exact last branch commit**: Rust fmt/check/test/Clippy, Go test/vet, frontend typecheck/build, native Windows/Linux packages, protocol E2E, Windows hardening and compatibility checks. **No green assertion without final SHA evidence.**

## Required real-platform acceptance before rollout
1. Windows 10/11: firewall prompts, bound-to-loopback vs LAN vs external interfaces, failed bind/rebind/port conflict, concurrent 33+ peers, mocked/real denied privilege, cancellation of PowerShell process trees on revoke, user account with restricted permissions, restart of pinned devices.
2. GNOME/KDE Linux: loopback vs mDNS, ufw/firewalld, rejected raw TCP connections, 6 failed Noise-PSK attempts, 100+ concurrent handshakes, malformed multipart encrypted frames, bash process group termination on revoke, restart and file permission/ACL inspection.
3. Both: unsafe grant redirects and attacker-controlled DNS, local Go runtime DNS rebinding in actual WebView, pairing QR/deeplink sanitization, log/crash traces verified without sentinel private keys or shell commands.
4. Fail if an untrusted controller can access another controller's remote job output, if a revoked session executes a new command, or if a new peer can write/delete/execute without explicit owner permission.

**Status:** source patches and tests are committed on the existing branch. Native OS/firewall/real multi-host/manual security acceptance is **not yet independently proven**.
