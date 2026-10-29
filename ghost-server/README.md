# Ghost FTP Server — Phase 4 security-first preview

This is a **new, standalone, real TCP server**, not a screenshot, mock admin
panel, or repackaged desktop client. The initial vertical slice implements
**read-only, authenticated explicit FTPS** with Go 1.25 (with pinned x/sys only for Windows SCM) and no
external runtime dependencies. It intentionally is **not** the completed
Windows FTP/FTPS Server application.

## Current protocol surface

A real TCP listener accepts FTP greetings, AUTH TLS, USER/PASS under TLS,
PBSZ 0, PROT P, PASV/EPSV, PWD/CWD/CDUP, TYPE I, SIZE, MLSD/LIST/NLST and
RETR through separately authenticated TLS data sockets. The served filesystem
root is bounded; symbolic links and parent-traversal operands fail closed.

The default preview rejects STOR, APPE, RNFR/RNTO, DELE, MKD, RMD, PORT/EPRT and
other mutations. An explicit \`-allow-uploads\` switch enables **new-file-only
STOR** with private TLS, a bounded per-file limit, a random staged file
(mode 0600), fsync and hardlink-based atomic create-if-absent commit.
Uploads never replace an existing name; unsupported filesystems fail closed.
Deletes, rename and directory creation remain unavailable. Passive TCP data connections require the same source IP as
the authenticated control connection. Passive ports are ephemeral; configure
local firewall rules accordingly. The default listener is **127.0.0.1:2121**
and never silently enables plaintext login. TLS 1.2+ is mandatory.

## Build and run for local testing

```sh
cd ghost-server
go test -race ./...
go build -o ghost-server ./cmd/ghost-server
# Create a private 0600 password file containing at least 12 characters.
./ghost-server -listen 127.0.0.1:2121 \
  -root /path/to/read-only-root \
  -cert /secure/server-fullchain.pem -key /secure/server-key.pem \
  -user operator -password-file /secure/ghost-server-password
```

For an isolated local write-path test only, add \`-allow-uploads -max-upload-bytes 67108864\`
and grant write permission **only to the intended subdirectory**. Never enable
writes on shared or adversarially mutable directory trees: traversal checks
are not yet race-free against privileged local filesystem modifications.

Run in a dedicated OS account without write permission to the served root
for the default read-only profile.
Use a publicly trusted certificate with the hostname your client connects to.
For Windows development, use `GOOS=windows GOARCH=amd64 go build
-o ghost-server.exe ./cmd/ghost-server`. The same executable now supports
`-service` to register a **real Windows Service Control Manager lifecycle**:
manual startup, running status, stop/shutdown and cancellation of the
server context. PowerShell `ghost-server/install/Install-GhostFTPServer.ps1`
and `Uninstall-GhostFTPServer.ps1` register/remove a **manual-start**
`GhostFTPServer` service using the restricted `NT AUTHORITY\\LocalService`
identity. Installation never starts a service with unconfigured credentials;
the operator must set explicit ACLs for read access to TLS keys, password
file and served root. No passwords are passed to the SCM command line.
Uninstallation preserves data. These scripts are not a signed `setup.exe`.

The Windows SCM integration is covered by native Windows compilation and
PowerShell parsing in CI but still needs real service start/stop field testing
with a correctly provisioned identity. Do not expose this Preview publicly.

## Honest blockers before Phase 4 acceptance

The default increment is **read-only**, with an explicitly gated create-only
STOR prototype. Full writable FTP semantics, deletion/rename authorization, explicit
administrator controls/UI, account management, rate limiting and lockout,
auditing, fully accepted Windows SCM service provisioning, signed installer/update/uninstall,
real client↔server E2E including negative TLS tests, TLS cert hot-reload and
signed binaries **do not yet exist**. Opt-in STOR now uses Go 1.25 \`os.Root\` handles to confine directory
traversal, random O_EXCL staged files and \`Root.Link\` for atomic
create-if-absent. Read-only listing and download currently retain legacy
path checks, which are not a complete TOCTOU defense. The service still
requires ACL isolation, quota/account design and independent adversarial
testing before public exposure.

No production security certification is claimed. Do not merge or expose the
service until integration, protocol fuzzing, Windows environment tests and
independent security review are complete.
