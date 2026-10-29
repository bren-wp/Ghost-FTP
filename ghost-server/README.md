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

## Windows unsigned setup.exe (developer artifact)

The separate Windows server CI job now compiles a real NSIS installer and
uploads it together with the portable Windows executable and SHA-256 checksums.
The installer copies only the binary, documentation and explicit PowerShell
SCM registration scripts. It **does not** automatically start a service,
create login credentials, open firewall rules or rewrite existing data.
The uninstaller refuses to remove program files while GhostFTPServer is still
registered to prevent orphaned service entries. Service unregistering is an
explicit, separately approved administrative action.

This installer is **not code signed** and has not yet completed a full
interactive installation/upgrade/uninstallation acceptance cycle on managed
Windows versions. Do not deploy publicly until TLS identity, ACL isolation,
least-privilege accounts and service lifecycle have been verified.

## Local administration dashboard

The server now has an optional, **real read-only local administration UI**.
The web listener stays disabled unless a separate administrator token of
32–256 bytes is supplied from an owner-private file. Example:

```sh
umask 077
openssl rand -hex 32 > /secure/ghostftp-admin-token
chmod 600 /secure/ghostftp-admin-token
./ghost-server -root /srv/ftps -cert /secure/ftps.crt \
  -key /secure/ftps.key -user operator \
  -password-file /secure/ftps-password \
  -admin-token-file /secure/ghostftp-admin-token \
  -admin-listen 127.0.0.1:2821
```

Open `http://127.0.0.1:2821/admin/` **on the server machine**.
The interface displays real service state, active sessions, TLS policy,
whether create-only uploads are enabled and uptime. No mock profiles or
fake transfers are inserted. API status is protected by a separate
constant-time bearer token check. The endpoint enforces loopback address,
an exact Host header, restrictive CSP/no-store/no-frame headers, no
cookies and no permissive CORS. The login token stays in tab memory;
no browser storage is used. Management is intentionally read-only:
configuring accounts, quotas, certificates or remote administration
through the dashboard is **not implemented**.

If `-admin-token-file` is omitted, **no HTTP admin listener is opened**.
Windows deployments must explicitly restrict read access to the token
file to their dedicated service identity; POSIX mode 0600 is mandatory.

## Additional destructive FTP operations (explicit opt-in only)

For a restricted local development account, both `-allow-uploads` and
`-allow-destructive` must be present to allow **DELE of regular files**,
**RMD of empty directories** and **RNFR/RNTO for regular files**. Renames
are limited to same-filesystem hardlinks with a no-overwrite commit,
followed by removal of the source. A failed source removal can leave both
names intact and returns failure rather than risking data loss.

Destructive operations are disabled unless **both** flags are set.
`RNTO` without `RNFR`, root removal, symlink targets, internal staging
names and overwrite attempts are rejected. Directory rename, overwrite,
append and recursive deletion are not supported. Operators must use
dedicated accounts and protected on-disk ACLs; this Preview is not an
unattended public file-sharing service.

### Native FTPS data termination compatibility

Some native-TLS clients close the data TCP connection without a TLS
`close_notify` alert. Even a bare TCP EOF may be reported as a normal end
of stream, so accepting it without the original file length would risk a
truncated upload.

Every create-only `STOR` now requires a single-use `ALLO <bytes>`
declaration made over the authenticated FTPS control channel **before STOR**.
The server compares the complete stream against that byte count, even when
TLS read returns a clean EOF. Undeclared uploads are rejected with `550`
before data transfer. Declared lengths that do not match actual bytes are
rejected with `451`; quota violations return `552`. Partial files are
never promoted and the staged files are removed before a terminal reply.

This is a deliberate Preview interoperability constraint: third-party FTP
clients must support sending an explicit `ALLO` command before each upload.
The Ghost FTP desktop client integration test and the independent curl FTPS
test both exercise this authenticated length announcement.
