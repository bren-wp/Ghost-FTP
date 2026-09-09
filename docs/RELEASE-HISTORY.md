# Ghost FTP release history

## 0.0.1 — 2026-09-09

Ghost FTP 0.0.1 starts the current public release line.

### Application

- Native Windows and Linux desktop client using the same typed FTP/FTPS/SFTP engine.
- Local/Remote dual-pane file workflow, Site Manager, transfer queue and 24-language local UI contract.
- Built-in Remote Edit for supported remote text files on FTP, FTPS and SFTP.
- Remote Edit uses bounded text handling, revision/conflict detection, line-ending preservation, transactional upload/read-back verification and post-save metadata refresh.

### Security and privacy

- Strict FTPS certificate/hostname verification with no silent downgrade.
- Strict SFTP host-key verification/pinning.
- Trusted Linux transport and credential-bearing AskPass executable provenance.
- Rooted local file/transfer protections and staged activation/rollback.
- State-directory identity pinning.
- Windows installer/uninstaller/shortcut ownership and exact-object cleanup hardening.
- No telemetry, analytics, advertising, tracking, automatic crash upload or hidden product backend.

### UI and platform quality

- Focused, non-cluttered file-management layout.
- Responsive Windows work-area and mixed-DPI geometry.
- Remote Edit exposed as one clear action rather than a permanent additional pane.
- Windows and Linux behavior routed through the same engine contract where platform-native presentation permits.

### Release quality

- Windows Setup x64/x86, x32 alias and Portable x64/x86.
- Linux DEB/tar.gz for amd64/arm64/i386 plus multiarch ZIP.
- **12 platform artifacts / 15 public files**.
- Debian 13, Ubuntu 26.04 LTS and Fedora 44 native lifecycle/GUI smoke.
- Exact-head and exact post-merge workflow verification.
- Current release identity `ghostftp-v0.0.1` with `prerelease=false`.
- Verified GHCR distribution bundle `ghcr.io/bren-wp/ghost-ftp:0.0.1`.

## Public history retention policy

Only the latest public Ghost FTP version is retained after successful publication and remote verification. Older GitHub Releases, `ghostftp-v*` tags, superseded versioned release branches and obsolete package versions are removed by `.github/workflows/release-retention.yml` while the current release package and current canonical branch are retained.

The Git commit history on `main` is not rewritten by this policy.

The next release must start from the verified current source and advance through a separate release-prep change, beginning with 0.0.2 after 0.0.1 is fully published and retention cleanup succeeds.
