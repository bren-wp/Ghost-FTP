# Ghost FTP — future Stable release policy

**Owner:** BRENDIGO — https://brendigo.com  
**Target:** first Stable release v1.0.0; subsequent Stable bugfixes 1.0.x and feature releases 1.x.

## Product decision

The old design-reference 1:1 pixel comparison and fixed screen-count acceptance are no longer release requirements. Do not misrepresent archived concepts as live screenshots. Preserve genuine, sourced native screenshots for marketing and diagnosis.

**Stable is an independently validated distribution channel, not a renaming of a Preview build.** The current project security gate intentionally rejects `version.json.channel = "stable"` until mandatory production evidence is enforced by release CI. Do not delete or bypass that gate.

## Mandatory Stable acceptance

1. **Signed and verifiable Windows artifacts:** Authenticode signing with trusted identity and signature validation of distributed installer/portable executables; clean install and real previous-version upgrade/rollback without data loss.
2. **Linux distribution integrity:** documented and validated package signing / authenticity scheme, DEB/RPM/AppImage installation on supported distributions and tested uninstall/rollback; preserve application data.
3. **Update trust:** signatures verifiable by the embedded updater key; refuse incomplete or mismatched updater manifests, never silently fall back to unsigned auto-updates.
4. **Security:** exact-source security/dead-code checks, dependency and CVE review, threat review of credential storage and SSH/FTPS identity validation, and independent testing of critical threats.
5. **File transfer quality:** real FTP, explicit FTPS and SFTP success/error/cancel/resume cases; destination data integrity, no unintentional overwrites, temporary staging cleanup, large files, concurrent transfers and recovery after interruption.
6. **Cloud providers:** authorized account-based live tests for advertised integrations with upload/download/refresh and failure cases, token redaction and remote-change handling.
7. **Native app UX:** actual Windows/Linux installs, functional controls and dialogs, accessible keyboard/screen-reader navigation, zoom/display scaling and responsive sizing. No literal pixel-equality quotas.
8. **Release provenance:** exact Git commit and passing CI, immutable version and tag, reproducible asset inventory, SHA-256 validation and verified publish metadata. Archive test evidence per target platform.

## Promotion procedure

- Keep v0.99.x builds `preview` until acceptance is evidenced.
- Implement **enforced** signature, package, live-system and upgrade gates before enabling the `stable` branch of `.github/scripts/check-release-channel.mjs`.
- After those gates pass on the exact commit, advance monotonically from the last published Preview to v1.0.0, explicitly set channel `stable`, build new signed release artifacts and verify that the GitHub release is *not* marked prerelease.
- Do not retarget old tags, relabel prior unsigned artifacts or rewrite release history.

**Status of this PR:** policy and removal of obsolete visual gates only. It does **not** provide signing keys, third-party accounts, production certificates or independently validated installed-machine evidence.
