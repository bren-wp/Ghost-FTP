# Ghost FTP — scoped repository audit and remediation (15 November 2026)

**Scope:** Release/source status, repository branch governance, and FTP/SFTP upload leaf-name validation. This is **not** a full security certification or evidence that native Windows/Linux acceptance has passed.

## Verified repository baseline

| Item | Evidence | Finding |
| --- | --- | --- |
| Main HEAD | GitHub branches/main API, 15 November 2026 | `af9c98104fcfbf9fcf6f333101cc87773777bdb7` |
| Published release | GitHub releases API and tag ref | [v0.99.67](https://github.com/bren-wp/Ghost-FTP/releases/tag/v0.99.67), published 14 November 2026, prerelease, 15 listed assets |
| Release tag | `refs/tags/v0.99.67` | Points at the above main SHA |
| Branch protection | GitHub branches/main API | `protected: false` — **P0 / OPEN** |
| Repository rulesets | GitHub repository rulesets API | `[]` — no active repository rulesets |
| Open PRs at start | GitHub PR search | None |
| Latest main workflows | GitHub Actions run summaries for the baseline SHA | Quality, protocol E2E, Windows hardening, native Windows/Linux builds reported success; installed-device, vault and 4+ GiB tests remain **NOT VERIFIED** |

## P0: GitHub branch governance — unresolved

The repository did not enforce PR review, mandatory status checks, deletion prevention or force-push restrictions on `main`. The available GitHub connector cannot write repository branch rulesets, so **no ruleset has been enabled in this iteration**.

Track enforcement and acceptance evidence in [Issue #296](https://github.com/bren-wp/Ghost-FTP/issues/296). Do not close the issue on documentation changes alone.

## P1: inaccurate publication status — corrected in source docs

The GitHub release, published 14 November 2026, carried a pre-publication description (`NOT PUBLISHED`). Several active documents repeated the outdated candidate status even though the tag, release and 15 listed assets existed.

- Corrected current publication descriptions in `README.md`, `CHANGELOG.md`, `docs/product/STATUS_AND_NEXT.md`, `docs/status/PRODUCTION_READINESS.md` and `docs/build/STATUS.md`.
- Added a dated erratum above the preserved pre-publication snapshot at `docs/releases/0.99.67.md`.
- Deliberately **did not modify** the published release description, existing tag, any binary asset, or historical release hashes. This respects immutable-release policy. The old release body can still show `NOT PUBLISHED`; source documentation now explains why it is obsolete.
- Publication means a Preview package exists, **not** that security, signing, credential storage or installed UI is certified.

## P2: defensive upload leaf-name validation

**Location:** `ghostftp-desktop/src-tauri/src/transfer/path_safety.rs` / `safe_upload_name`.

**Observed behavior before patch:** rejected slash, backslash, NUL, CR and LF but accepted other Unicode control code points such as TAB, ESC and DEL. These are not appropriate in protocol-facing upload leaf names. This is a validation hardening opportunity; no exploitable protocol injection has been demonstrated.

**Change:** reject every character satisfying Rust `char::is_control()` in addition to path separators. Preserve valid non-ASCII names. Add a focused unit regression covering C0 `U+0000..U+001F`, DEL/C1 `U+007F..U+009F`, traversal-like leaf values, path separators, and valid Croatian/German/Japanese file names.

**Compatibility note:** Previously accepted file names containing control characters are now rejected by this validator; this is intentional, and needs integration confirmation on all upload call paths.

**Test evidence:** source-level unit test committed, **CI result pending**. A passing Rust job on this PR SHA is required before marking automated regression PASS.

## Explicitly unverified / deferred

- Complete end-to-end transfer protection for FTP, explicit FTPS, SFTP and each cloud adapter.
- Full dead-code, dependency and threat-model audit of every active component.
- Native Windows Credential Manager and Linux Secret Service after reboot, locked-vault and migration scenarios.
- 0 B / 1 B / 64 KiB / 128 KiB / 1 GiB / >4 GiB transfer grid, 1/4/8 concurrency and crash/failure injection.
- Windows 10/11 and Linux X11/Wayland first-frame, responsive, accessibility and functional-control acceptance.
- Exact publisher signing and production Stable gates.

**Release rule:** Do not merge or publish a new Ghost FTP Preview solely on the basis of this scoped audit. Require the exact-head checks and appropriate native acceptance evidence. No new release is claimed in this document.
