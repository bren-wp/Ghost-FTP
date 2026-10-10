# Ghost FTP 0.93.0 — production readiness and explicit blockers

**Decision:** **NOT READY FOR PRODUCTION PROMOTION** on source inspection alone. GitHub CI, device validation and cloud provider acceptance decide readiness; this document is not a substitute for those checks. The previous public release is 0.92.2; verify GitHub Releases again immediately before publishing.

## Platform status

| Platform | Existing product | Candidate 0.93.0 additions | Remaining release blockers |
| --- | --- | --- | --- |
| Windows | Tauri / Rust / React desktop, FTP/FTPS/SFTP and cloud adapters | Hardened HTTPS WebDAV and cloud endpoint rules | Exact-head build, MSI/NSIS install/uninstall/launch, cloud service E2E, file-integrity verification |
| Linux | Same desktop engine | HTTPS provider restrictions | Exact-head AppImage/DEB/RPM build and launch, keyring fallback, provider E2E |
| macOS | SwiftUI Preview, plain FTP file operations, Keychain | One-object provider-signed HTTPS upload/download | No complete FTPS/SFTP, no OAuth/bucket navigation, unsigned/notarization gaps, DNS/transport hardening, real macOS QA |
| Android | Kotlin FTP/FTPS/SFTP; document picker | One-object provider-signed HTTPS transfer; unsigned APK and AAB build pipeline | No cloud OAuth/browser, partial SAF output risk, DNS rebinding, cancellation/lifecycle testing, stable production signing identity and device testing |

## Mandatory P0 gates

1. **Source consistency:** root `version.json` and platform package versions agree; never rewrite published tags.
2. **Six CI workflows, exact PR SHA:** quality; protocol E2E; native Windows/Linux build; Android; macOS; Windows hardening. Each must conclude success for the *same* latest head commit, not an earlier PR commit.
3. **Merge:** only after all six PR gates succeed and independent code/security review. Confirm the merged `main` SHA and rerun equivalent gates on that SHA.
4. **Release:** allow only official `ghostftp-release.yml` automation after gating. Verify SHA-256, sizes, source SHA provenance and actual presence of NSIS EXE, MSI, Linux AppImage/DEB/RPM, Android unsigned APK, Android unsigned AAB, Android installable preview APK and labeled macOS Preview where the pipeline supports it.
5. **Signing distinction:** a CI-preview Android signing identity does **not** represent persistent production signing. Unsigned APK/AAB cannot be represented as ordinary production-installable packages. macOS Preview is not a notarized end-user product.
6. **Transfer integrity:** test interrupted upload/download, cancellation, conflict, empty/large/Unicode objects, TLS error, expired link and safe staging on real servers/providers.
7. **Privacy review:** token/credential/signed URL data flow audited, no secret-containing exception paths, no unrequested diagnostics/telemetry.
8. **UX:** click-through on all platform targets, minimum window sizes, accessibility, file picker recovery and cancellation. No untested 1:1 visual claim.

## Key unresolved security risks

- **P0 DNS rebinding:** Android DNS preflight runs before socket resolution and cannot guarantee destination binding; macOS signed URL validation also lacks transport-level target pinning. A trusted provider allowlist reduces exposure but is not a complete defense.
- **P0 SAF finalization:** Android may leave a partial document when writing a selected destination fails. Offer safer recoverable output semantics and verify digest before reporting success.
- **P0 OAuth and token lifecycle:** Android/macOS full cloud account sign-in, refresh-token protection, disconnect and revocation remain unimplemented.
- **P1 UX cancellation:** deterministic cancellation and status differentiation during real long uploads/downloads still require device QA.
- **P1 Service integration:** no disposable credentialed accounts/live vendor evidence is documented by the URL parser unit tests.

## Evidence links and updating policy

Use the PR [#125](https://github.com/bren-wp/Ghost-FTP/pull/125), the six workflow run URLs for its **latest** SHA and [GitHub Releases](https://github.com/bren-wp/Ghost-FTP/releases). Never copy a successful result from an earlier commit as proof for a new head. When a check fails, capture the failing job and exact error, fix and repeat. Update this document only when confirmed evidence changes.
