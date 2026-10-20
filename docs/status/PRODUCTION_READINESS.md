# Ghost FTP 0.95.51 — production readiness and explicit blockers

**Decision:** **NOT READY FOR PRODUCTION PROMOTION** on source inspection alone. GitHub CI, device validation and cloud provider acceptance decide readiness; this document is not a substitute for those checks. v0.95.50 was public on 20 October 2026, but its Android and macOS Preview/unsigned assets do not demonstrate stable release acceptance; 0.95.51 remains unpublished.

## Platform status

| Platform | Existing product | Current 0.95.51 source scope | Remaining release blockers |
| --- | --- | --- | --- |
| Windows | Tauri / Rust / React desktop, FTP/FTPS/SFTP and cloud adapters | Hardened HTTPS WebDAV and cloud endpoint rules | Exact-head build, MSI/NSIS install/uninstall/launch, cloud service E2E, file-integrity verification |
| Linux | Same desktop engine | HTTPS provider restrictions | Exact-head AppImage/DEB/RPM build and launch, keyring fallback, provider E2E |
| macOS | SwiftUI Preview, plain FTP file operations, Keychain | One-object provider-signed HTTPS upload/download | No complete FTPS/SFTP, no OAuth/bucket navigation, unsigned/notarization gaps, DNS/transport hardening, real macOS QA |
| Android | Kotlin FTP/FTPS/SFTP; document picker | Direct-only OkHttp with connecting-DNS address guard, cancelable signed transfers, best-effort incomplete SAF cleanup, APK/AAB pipeline | No cloud OAuth/browser, incomplete SAF cleanup on hostile providers, physical-device DNS/VPN/rebinding and cancellation tests, production signing identity |

## Mandatory P0 gates

1. **Source consistency:** root `version.json` and platform package versions agree; never rewrite published tags.
2. **Six CI workflows, exact PR SHA:** quality; protocol E2E; native Windows/Linux build; Android; macOS; Windows hardening. Each must conclude success for the *same* latest head commit, not an earlier PR commit.
3. **Merge:** only after all six PR gates succeed and independent code/security review. Confirm the merged `main` SHA and rerun equivalent gates on that SHA.
4. **Release:** require manually dispatched `ghostftp-release.yml` Preview promotion after gating. Verify SHA-256, sizes, source SHA provenance and actual presence of NSIS EXE, MSI, Linux AppImage/DEB/RPM, Android unsigned APK, Android unsigned AAB, Android installable preview APK and labeled macOS Preview where the pipeline supports it.
5. **Signing distinction:** a CI-preview Android signing identity does **not** represent persistent production signing. Unsigned APK/AAB cannot be represented as ordinary production-installable packages. macOS Preview is not a notarized end-user product.
6. **Transfer integrity:** test interrupted upload/download, cancellation, conflict, empty/large/Unicode objects, TLS error, expired link and safe staging on real servers/providers.
7. **Privacy review:** token/credential/signed URL data flow audited, no secret-containing exception paths, no unrequested diagnostics/telemetry.
8. **UX:** click-through on all platform targets, minimum window sizes, accessibility, file picker recovery and cancellation. No untested 1:1 visual claim.

## Key unresolved security risks

- **P0 network routing:** Android replaces a disconnected DNS preflight with validation inside its HTTP connecting DNS selection and forces direct networking. This is source-level mitigation; prove DNS rebinding, private IPv4/IPv6, proxy/VPN routing and provider compatibility on devices. macOS still lacks equivalent transport-bound destination enforcement.
- **P0 SAF finalization:** Android now attempts removal of the newly created document after failed/canceled downloads and warns if cleanup fails. Some providers can leave partial content; strong rollback guarantees, destination integrity and realistic provider tests are still blocked.
- **P0 OAuth and token lifecycle:** Android/macOS full cloud account sign-in, refresh-token protection, disconnect and revocation remain unimplemented.
- **P1 UX cancellation:** deterministic cancellation and status differentiation during real long uploads/downloads still require device QA.
- **P1 Service integration:** no disposable credentialed accounts/live vendor evidence is documented by the URL parser unit tests.

## Evidence links and updating policy

Use the PR [#125](https://github.com/bren-wp/Ghost-FTP/pull/125), the six workflow run URLs for its **latest** SHA and [GitHub Releases](https://github.com/bren-wp/Ghost-FTP/releases). Never copy a successful result from an earlier commit as proof for a new head. When a check fails, capture the failing job and exact error, fix and repeat. Update this document only when confirmed evidence changes.

## 20 October 2026 governance finding

- Public v0.95.50 has draft=false, prerelease=false despite unsigned Android production APK/AAB, an installable Android Preview APK and a macOS Preview archive. No stable certification follows.
- Version 0.95.51 is a release-governance change only. Android Play signing and upgrade validation, macOS FTPS/SFTP and notarization, Windows production signing and full device QA remain blocked.
