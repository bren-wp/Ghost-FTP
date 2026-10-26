# Android filename search — bounded glob audit (26 October 2026)

## Scope and root cause

The Android Files search previously translated user-supplied `*` and `?` patterns to a Java/Kotlin regular expression and called `Regex.matches` on remote inventory from the UI thread, or on local entries from a background thread. Alternating wildcard/literal sequences can cause disproportionate regex backtracking on long filenames; limiting the query to 80 characters does not guarantee a bound on matching time.

This change replaces regex matching with `FilenameGlob`, a filename-only matcher that treats every other character literally, collapses adjacent `*` characters, and has worst-case time proportional to pattern length times filename length. `*` still matches zero or more characters and `?` matches exactly one character. Case-sensitive/case-insensitive search settings remain supported.

## Covered surfaces

- Android Files remote inventory: matching no longer executes user-derived regex on the UI thread.
- Android Files local scoped search: retains background enumeration, visited/matches/depth bounds, symlink checks, cancellation generation checks and existing unavailable/error behavior.
- JVM regressions: ordinary wildcard cases, literal regex metacharacters, case folding, repeated ambiguous wildcard patterns and maximum length.
- No change to FTP/FTPS/SFTP transport, stored profiles, credentials, navigation, visual tokens or packaging.

## Verification and remaining acceptance

Standalone Kotlin compiler smoke tests exercised wildcard/literal/case/ambiguous input and passed. The GitHub Android JVM, lint and emulator suites are the authoritative integration gates; check their status for the PR head commit before merging. Manual device UI testing and all 75 v8 real-capture pixel comparisons are **not** claimed. Native macOS FTPS/SFTP identity verification remains open in issue #223. Do not promote a production release based on this isolated hardening change.
