# Ghost FTP release process — Windows/Linux desktop

**Active source:** 0.99.55 Preview candidate. **Previous canonical public release:** v0.99.54. Check GitHub Releases for the actual newest published edition; development version numbers do not prove publication.

The single source of truth is `version.json`. A new version requires synchronized desktop metadata, a matching `docs/releases/<version>.md` file and exact-source CI evidence. If the source version has merged but is **not** yet published, the release-repair path may reuse that unpublished version only under the repository's immutable-lineage checks.

## Required exact-source gates

1. Build and run Windows/Linux native package and installer smoke checks.
2. Pass source quality, dead-code/security regression tests, frontend typecheck/build, Rust workspace tests/Clippy, Go tests and FTP/FTPS/SFTP protocol E2E.
3. Pass GitHub release governance and verified artifact inventory tests; never skip a failed gate by labeling a different source stable.
4. Confirm the source SHA is current main and equals the build/artifact provenance, and previousPublishedVersion matches the numerically newest immutable public edition.
5. A successful main-branch `Ghost FTP native build` can trigger `Ghost FTP release` through `workflow_run`. The workflow waits for all remaining successful, same-SHA gates before publishing.
6. Publish only after *all* artifacts have nonzero sizes, canonical filenames, signed-updater group coherence (if configured), SHA-256 checksums and independent GitHub asset digest checks.

## Release assets (current Windows/Linux)

Windows x64: Portable EXE, Setup EXE, MSI, ZIP bundle and native QA ZIP. Linux x86-64: native ELF, AppImage, DEB, RPM and tar.gz. Include full source ZIP, desktop-source ZIP, updates ZIP, documentation ZIP and SHA-256 manifest. The exact required inventory is defined by `.github/scripts/release-artifacts.json`.

Optional signed updater files are all-or-nothing: Windows installer signature, Linux AppImage signature, authenticated latest.json and Update-Service ZIP. If signing is unavailable, publish labeled unsigned **Preview** installable files with checksums, not a fictional signed automatic update.

Use the official [Releases index](https://github.com/bren-wp/Ghost-FTP/releases), not `/releases/latest` for Preview discovery. **Never delete or move published tags or change release asset bytes.** Historical editions retain their original platform assets and historical source notes as immutable evidence; subsequent editions are Windows/Linux only.

## Security and acceptance

Windows Authenticode, Linux package signing/compatibility, data-preserving upgrade/rollback, cloud-provider consent/refresh, stress/crash recovery and accessibility are independent stable-production gates. The old exact visual-reference comparison requirement is retired. A passing static scan is not a penetration test. Production signing and independently verified real-device visual acceptance are required before the stable channel can be enabled.

Every successfully verified merged change must have a new edition, but an unverified release must remain unpublished rather than bypass security and integrity gates. GitHub Actions failures must be diagnosed from actual failed steps before merging.

## Future Stable channel

The next production-stable transition targets v1.0.0 only after [Stable release policy](STABLE_RELEASE_POLICY.md) is satisfied. A green Preview CI run alone does not authorize relabeling to Stable. Historical published Preview releases/tags remain immutable. New Stable builds must be produced and checked from their exact source revision.
