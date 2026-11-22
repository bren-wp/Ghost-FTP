# Ghost FTP by Brendigo — Protecting main (active issue #320)

> Legacy document filename retained for historical links. Original issue #302 is deleted (GitHub HTTP 410); active tracking and release blocker: [#320](https://github.com/bren-wp/Ghost-FTP/issues/320).

**Last observed:** `main.protected = false`, no active rulesets. Source CI cannot replace GitHub's server-side merge restrictions. The active issue #320 must remain **open** until protections are applied and verified by a repository administrator.

## Owner-authorized setup

An administrator can configure Settings → Rules → Rulesets for the `main` branch. Require pull requests (no direct push), passing up-to-date checks, conversation resolution, no force pushes, no branch deletion, administrator enforcement and no bypass actors. Alternatively, the repository contains `.github/scripts/enable-main-protection.sh` which configures equivalent classic branch protection using GitHub REST. **Do not paste an access token into a file or chat.**

With `gh`, `jq` and a properly authorised `gh auth login` session, execute from a trusted local clone:

```bash
bash .github/scripts/enable-main-protection.sh --apply
bash .github/scripts/enable-main-protection.sh --check
```

`--check` is the default and is read-only. `--apply` requires repository Administration:write and can fail if permission or plan policy is insufficient. Both modes refuse to claim success on missing checks.

## Mandatory status-check contexts

Require exactly the following current check-run names, observed on the verified v0.99.71 main SHA:
- `Rust workspace`
- `Product + compatibility tooling`
- `build (windows-latest, --bundles nsis,msi, GhostFTP-Windows-x64-Preview)`
- `build (ubuntu-22.04, --bundles deb,rpm,appimage, GhostFTP-Linux-x86_64-Preview)`
- `Real FTP / FTPS / SFTP roundtrip`
- `Real Windows FTP upload/download roundtrip`
- `Windows filesystem security regressions`
- `Desktop security gate`
- `windows` (Validate Windows hardening)

Do not require skipped release-publishing jobs, schedule-only metadata jobs or matrix parent names. The new Windows network-transfer test is a mandatory gate for releases after its CI verification. The required contexts must be present for ordinary PRs. Ensure `strict: true` or use a similarly rigorous exact-head merge queue. Review requirements should be strengthened to one approval where an independent reviewer is reliably available. A zero-approval PR rule still blocks direct pushes while permitting a sole maintainer to merge a green PR.

## Automatic release fail-closed check

The `Ghost FTP release` workflow reads the live branch API and permits either protected `main` **or verified exact merged-PR provenance** for the current-main SHA. On an unprotected branch it rejects direct pushes and unmerged/wrong-base or ambiguous PR associations. It still waits for all exact-source mandatory gates and validates signed-updater and artifact integrity before publication. This restores the previously working Preview release workflow without treating source-only controls as a substitute for repository-admin protection. Admins should still verify exact-head mandatory checks, a blocked failing-PR merge and a successful green merge before closing #320.

## Acceptance evidence (not yet completed)

1. GitHub API `GET /repos/bren-wp/Ghost-FTP/branches/main` reports `protected: true` and the protection endpoint confirms admin enforcement, strict required checks and conversations.
2. A deliberate disposable failing PR is blocked from merging, including an admin's ordinary merge workflow.
3. A current, fully green PR with all nine exact check contexts can merge, and main's post-merge checks complete.
4. Record GitHub settings evidence and dates in issue #320 before closure. The presence of this document or script is **not** evidence that the protections have been enabled.
