# Ghost FTP 0.95.53 — Full dead-code review and removals

**Evidence:** source-specific CI `tools/audits/full-dead-code-audit.mjs --json` on every pull request; previously completed v0.95.52 run `39493900524` published a machine-readable artifact with **687 working-tree files**, including 117 TypeScript, 87 Rust, 12 Android Kotlin, 16 Swift, 12 Go, 45 Android resources and 62 operational scripts.

The previous source scanner found **no unreferenced production Kotlin, Swift or Android resource candidates**, no missing core application entrypoints, and **one unreferenced developer helper**: `ghostftp-desktop/scripts/generate-funding-qr.mjs`. This QR generator is a manual branding/funding tool, *not* proved obsolete; it is preserved. These are reference-based candidates rather than an executable reachability proof.

## Verified removal in v0.95.53
Four *already completed and one-time* GitHub migration assets were removed after checking references from other tracked release scripts and CI:
- `.github/scripts/import-verified-windows-screenshots.mjs` — exact v0.95.51 screenshot import had completed successfully; original PNG SHA-256 hashes and source Actions run remain recorded in the current capture manifest.
- `.github/workflows/ghostftp-refresh-readme-09551.yml` — triggered only by old merged correction branch.
- `.github/scripts/repair-release-09551.sh` — source-locked one-time release repair completed; original `v0.95.51` Git release and its 21 artifacts are retained.
- `.github/workflows/ghostftp-repair-09551.yml` — executable trigger demanded a prior `main` head SHA that cannot recur, so it is permanently inactive.

Deletion is limited to these retired maintenance entrypoints; no end-user module, database migration, OS credential backend, release digest code or platform feature was removed.

Two additional temporary screenshot-import files were removed after the v0.95.52 verified ZIP importer succeeded (run 39499908379): `.github/scripts/import-verified-windows-09552.mjs` and `.github/workflows/ghostftp-readme-09552-refresh.yml`. Current PNGs and their manifest remain committed; removing retired import automation does not affect users.

## Future review
Review all React imports, Rust feature/cfg variants, Kotlin Manifest/Compose/resources, SwiftUI reflection and `@main`, Go OS build tags, Tauri command registration, SQLite migrations and native backups before any further deletion. Preserve historical credentials and signed-update lineage. Measure app bundle size and runtime memory before claiming any speedup or battery saving.
