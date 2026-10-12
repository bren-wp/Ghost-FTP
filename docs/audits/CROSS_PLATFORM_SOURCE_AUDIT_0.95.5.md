# Ghost FTP 0.95.5 — scoped source reachability, privacy and parity audit

Date: 12 October 2026. Source train **0.95.5**, branch `hardening/0.95.5-crossplatform-privacy-audit`. Previous published baseline: **v0.95.4**.

## Review coverage

| Platform | Executable QA / static reachability | Actual improvement | Evidence still absent |
| --- | --- | --- | --- |
| Windows / Linux | `npm run check:dead`, `npm run check:ui`, `npm run check`, Rust `cargo test`, Cargo Clippy, native bundle jobs and FTP/FTPS/SFTP protocol E2E | Redact signed-cloud bearer URLs in diagnostics, with regression tests | Independent full UI 1:1 screen comparison, production signed-updater and destructive-file-system lab |
| Android | Private Kotlin symbol audit, Gradle unit/lint/APK, emulator click smoke | Clear signed link before SAF picker and clean orphan document after recreation where provider permits | Real Android 8–15 device matrix, provider grant revocation, process death/disk-full/1GiB lab (#127) |
| macOS | Private Swift declaration audit, `swift test`, release build and pinned network transport tests | Clear bearer URL SwiftUI fields before file panel, hide bucket hostname | Real FTPS/SFTP implementation with trust prompts, provider DNS lab (#126), account OAuth (#128), notarized package |

**Dead-code scope:** The existing module-reachability checks cover source files and private declaration scans; they cannot automatically prove every exported symbol, conditional feature, dynamic binding or callback is unused. Unused source is not deleted without safe reference and build evidence. No unsupported sweeping removal is claimed.

**Security scope:** Signed links remain bearer secrets; Windows/Linux whole-URL diagnostics and mobile/macOS short-lived input fixes reduce incidental exposure. They do not replace actual provider network lab, certificate trust validation, authentication lifecycle or pentest.

**Visual scope:** All 75 supplied concept screen mappings remain pending real installed-build screenshot comparison; reference concepts are not proof of 1:1 production visuals.

## Release gates

New source is not a public version merely because metadata says 0.95.5. GitHub Actions on the exact PR HEAD and resulting `main` commit, real packages, signature verification and QA evidence decide publication. Open issues #126, #127 and #128 must remain open until their independent acceptance criteria are met.
