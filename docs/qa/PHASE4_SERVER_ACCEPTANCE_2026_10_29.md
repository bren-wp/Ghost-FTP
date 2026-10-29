# Faza 4 — Ghost FTP Server acceptance matrix (29 October 2026)

## Source and release policy

- PR [#244](https://github.com/bren-wp/Ghost-FTP/pull/244) is the active, **unmerged Draft** source for **0.99.20 Preview**. Previous verified public edition is `v0.99.19`. No existing immutable release tag has been replaced.
- Server lives **only** in `ghost-server/` (Go 1.25). The existing Rust/Tauri/Android/Swift client architectures are not replaced. The server's unsigned Windows and Linux developer artifacts are GitHub Actions outputs, not a public signed release.
- Every pass below must be revalidated on the **final identical commit SHA** before merge/tag. Workflow status alone cannot establish a security-certified public service.

## Real implementation and regression gates

| Feature / risk | Concrete server implementation | Evidence and remaining gate |
| --- | --- | --- |
| FTP control | Real TCP listener, greeting, USER/PASS, PWD/CWD, MLSD/LIST/NLST, SIZE | Go local wire tests; standalone QA pipeline |
| TLS identity | Mandatory explicit AUTH TLS, TLS 1.2+, no plaintext credentials, no `PROT C` | Local self-signed trust rejection + independent curl with verified certificate |
| Data channels | Passive EPSV/PASV private TLS + peer IP match, bounded deadlines | End-to-end actual byte transfers, negative TLS cases |
| Root confinement | Go 1.25 `os.Root` handles for read, list, mkdir, uploads and mutations | Traversal / symlink tests; requires independently reviewed OS ACL boundaries |
| Create-only STOR | `-allow-uploads` default OFF, private random staging, fsync, `Root.Link` no-overwrite commit, bounded per-upload bytes | Go + curl upload, overwrite, quota, staged cleanup |
| MKD | Opt-in only with create permission, parent must exist, no overwrite | Real FTP socket test + path traversal assertion |
| DELE / RMD | Additional `-allow-destructive` opt-in, file-only DELE / empty-directory RMD | Real FTP socket regression tests; local OS ACL and retention policy remain operator duties |
| RNFR/RNTO | Additional opt-in, existing regular source, no-overwrite hardlink commit then source removal; never silently overwrites | Test occupied destination preserves both streams; cross-volume/link errors fail closed |
| Authentication anti-bruteforce | Per-session 3 strikes and bounded per-peer attempt tracker across reconnects | Auth tests; production network ingress policy and distributed limits unimplemented |
| Local administration | Optional token-gated loopback HTTP dashboard, real active-client/up-time/write-policy metrics, explicit Host/CSP checks | Admin API/HTML auth tests; CRUD for users/roles/certificates and remote administration intentionally absent |
| Windows SCM | Real native `-service`, manual LocalService registration, stop/shutdown and isolated cert/password file ACL handling | Native Windows CI installs/starts real service, opens TCP greeting, stops and unregisters |
| Portable Windows | Cross-compiled Windows x64 Go executable | Windows CI builds binary and checksums |
| Setup.exe | Real unsigned NSIS installer and uninstaller, no default credentials or firewall rules, uninstall leaves external data intact | CI silent install/uninstall roundtrip; code signing and upgrade/migration acceptance outstanding |
| Rust client ↔ server | Existing real `FtpFs`/native TLS client talks to standalone Ghost server on distinct localhost port | Newly added cross-product Rust protocol E2E; final result must be checked, not presumed |
| Linux portability | Go standalone Linux x64 executable | Build + race detector, vet, Linux E2E and checksum evidence |

## Security and release blockers

**Not yet demonstrated as a production-complete phase:**

- Third-party security assessment, Linux/Windows multi-user ACL roles and quota policy, encrypted private settings management, certificate renewal/revocation, persistent audit log with retention and redaction, and operational crash/recovery acceptance.
- Cross-platform installed client↔server matrix beyond the new CI Rust client and independent curl, including actual macOS and Android devices and network/firewall/NAT deployment.
- Signed Windows binaries and Authenticode signature verification, user-facing premium native server management application, installer update/rollback/uninstall across supported Windows versions.
- Final exact-HEAD green check of all required workflows, plus functional/UX operator acceptance. A passing earlier commit does not count.

**Operational default:** read-only, loopback listener; opt-in file creation and opt-in destructive mutations only for isolated test roots and dedicated restricted service identities. Do not expose the current Preview to untrusted public networks.

## Immediate next steps

Finish the full exact-head CI rerun, inspect actual Rust client↔Ghost Server errors if any, review the unsigned NSIS artifact and checksums, perform threat-model/security audit, and verify real Windows installer and service lifecycle manually before issuing a production release.
