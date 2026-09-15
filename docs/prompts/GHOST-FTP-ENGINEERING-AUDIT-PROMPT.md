# Ghost FTP engineering audit and production-hardening prompt

Use this prompt when handing the Ghost FTP repository to an engineering agent for a production-quality audit or implementation pass. The repository and current GitHub state are authoritative. Never rewrite an already published release identity merely because source development has moved on.

## Current published baseline

Ghost FTP **0.0.6** is a published non-prerelease release. Its canonical shape is **13 platform artifacts plus 3 metadata files = 16 public release files**. The published 0.0.6 tag/release is immutable history.

- **Windows — public production surface.** One universal Setup and one universal Portable package carrying x64/x86/ARM64 payloads.
- **Linux — public production surface.** Debian, Ubuntu and Fedora Installer + Portable bundles with amd64/arm64/i386 payloads.
- **Android — public production surface.** One production-signed APK. The current build contract is `minSdk 26`, `targetSdk 35`. Android SFTP remains hidden/unsupported until strict host-key verification is maintained.
- **Browser helper — public production surface.** Chrome, Edge, Firefox and Opera packages exist in 0.0.6. For 0.0.7, move toward a serious Ghost FTP extension experience through a secure local native companion/native-messaging bridge where raw FTP/SFTP access is required. Never simulate unsupported browser sockets or relay credentials through Ghost FTP servers.
- **macOS — active development/source surface only.** Public distribution remains gated on a real Developer ID Application identity and Apple notarization.

Production Android publication uses `GHOSTFTP_ANDROID_CERT_SHA256`. Production Windows publication is signed-only.

## Core engineering rules

Work on a dedicated branch from exact current `main`. Make focused commits. Preserve security/privacy tests and fix root causes rather than disabling tests or using allowed-failure shortcuts.

Audit actual behavior before changing it. Prioritize reproducible crashes, data-loss conditions, stale state, dead actions, misleading product copy, insecure credential handling, package/release defects, excessive idle work and duplicated/dead code.

User-visible actions must be real. No enabled button may be a decorative stub. No fake connection, simulated transfer, demo credentials, placeholder server or unsupported capability may appear as a production feature.

## Security and privacy invariants

- no telemetry, analytics, advertising, tracking, fingerprinting or hidden crash-upload service;
- no hidden Ghost FTP backend/proxy for user FTP/SFTP credentials or transfer bytes;
- no plaintext password/passphrase logging or plaintext credential files;
- no password in a URL;
- strict FTPS certificate/hostname validation;
- strict desktop SFTP host-key trust/pinning;
- Android SFTP remains hidden until strict host-key verification exists;
- no silent secure-to-plain downgrade;
- browser extensions use minimum required permissions and no remote executable JavaScript;
- production signing identities are never fabricated from CI/development credentials.

## Browser extension architecture

Browsers do not expose arbitrary raw FTP/FTPS/SFTP TCP sockets to extension JavaScript. Real protocol parity therefore requires a local Ghost FTP companion where supported.

A native-messaging design must:

1. use browser-supported native messaging rather than an unauthenticated localhost HTTP port;
2. register an explicit Ghost FTP host identity;
3. validate operation type, message size, host, port, protocol and paths;
4. keep credentials and transfer bytes on the user's device and selected server path;
5. expose only the least browser permission needed, normally `nativeMessaging` plus explicitly justified capabilities;
6. return privacy-safe user errors while retaining local diagnostic detail;
7. support disconnect/cancel cleanup when the browser side disappears;
8. include deterministic package and protocol-contract tests.

The browser UI should converge toward the Windows product language for connection management, navigation, transfer status, file operations and errors without pretending that browser-native filesystem authority equals desktop authority.

## Stability and performance

Review cancellation, timeouts, reconnect/disconnect, network loss, transfer interruption, disk full, permission denied, stale async completion, application shutdown during transfers and event/timer cleanup. Expected network/server failures must not crash Ghost FTP.

Avoid busy polling, unnecessary timers/threads/processes, duplicated caches, repeated parsing and retained large objects. Do not trade correctness for micro-optimizations.

## Product UI

Windows remains the reference UX. Polish navigation, file browser, connection manager, transfer queue, status/loading/empty/error states, keyboard focus, modals, confirmations, overwrite flow and resizing. Android and browser surfaces use equivalent product language adapted to their platform.

Remove end-user development vocabulary such as debug/test/demo/mock/prototype/internal/TODO/FIXME/WIP and raw exception/stack information. Internal tests may keep isolated test data.

## Quality gates

Applicable gates include Go formatting/tests/race/vet, Python regression tests, repository/platform/security/privacy/dependency/documentation/release audits, Windows builds, Linux builds/install lifecycle, Android build/lint/signing checks, browser deterministic builds, CodeQL, Govulncheck and exact-head runtime evidence.

A PR is not merge-ready until relevant workflows for its exact final head are successful. A green older SHA does not validate a newer commit.

## Release discipline

The existing `ghostftp-v0.0.6` release must not be changed, retagged or republished. Future release publication uses the repository's protected `workflow_dispatch` release path from an exact verified source. Missing production signing material is a blocker, not a reason to invent a replacement identity.

## Definition of done

Report concrete code changes, regression coverage, exact branch/PR head, relevant workflow results and any real platform limitation. Do not label work fixed, green, production-ready or released without corresponding evidence.
