# Ghost FTP Server — Phase 4 security-first preview

This is a **new, standalone, real TCP server**, not a screenshot, mock admin
panel, or repackaged desktop client. The initial vertical slice implements
**read-only, authenticated explicit FTPS** with standard-library Go and no
external runtime dependencies. It intentionally is **not** the completed
Windows FTP/FTPS Server application.

## Current protocol surface

A real TCP listener accepts FTP greetings, AUTH TLS, USER/PASS under TLS,
PBSZ 0, PROT P, PASV/EPSV, PWD/CWD/CDUP, TYPE I, SIZE, MLSD/LIST/NLST and
RETR through separately authenticated TLS data sockets. The served filesystem
root is bounded; symbolic links and parent-traversal operands fail closed.

The preview rejects STOR, APPE, RNFR/RNTO, DELE, MKD, RMD, PORT/EPRT and
other mutations. Passive TCP data connections require the same source IP as
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

Run in a dedicated OS account without write permission to the served root.
Use a publicly trusted certificate with the hostname your client connects to.
For Windows development, use `GOOS=windows GOARCH=amd64 go build
-o ghost-server.exe ./cmd/ghost-server`. This is a **portable executable
prototype**, not a registered Windows service, production installer, or
signed release. Do not expose this preview publicly.

## Honest blockers before Phase 4 acceptance

The first increment is **read-only**. FTP writes, secure uploads with
temporary-file atomic promotion, deletion/rename authorization, explicit
administrator controls/UI, account management, rate limiting and lockout,
auditing, Windows Service Control Manager lifecycle, installer/update/uninstall,
real client↔server E2E including negative TLS tests, TLS cert hot-reload and
signed binaries **do not yet exist**. Symlink/path checks are non-atomic
against an untrusted concurrent local filesystem modifier; production needs
OS-level handle-relative traversal, ACL isolation and adversarial testing.

No production security certification is claimed. Do not merge or expose the
service until integration, protocol fuzzing, Windows environment tests and
independent security review are complete.
