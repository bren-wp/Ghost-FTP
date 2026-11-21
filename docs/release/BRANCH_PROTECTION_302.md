# Ghost FTP by Brendigo — Protecting main (issue #302)

**Last observed:** `main.protected = false`, no active rulesets. Source CI cannot replace GitHub's server-side merge restrictions. This issue must remain **open** until protections are applied and verified by a repository administrator.

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
- `Windows filesystem security regressions`
- `Desktop security gate`
- `windows` (Validate Windows hardening)

Do not require skipped release-publishing jobs, schedule-only metadata jobs or matrix parent names. The required contexts must be present for ordinary PRs. Ensure `strict: true` or use a similarly rigorous exact-head merge queue. Review requirements should be strengthened to one approval where an independent reviewer is reliably available. A zero-approval PR rule still blocks direct pushes while permitting a sole maintainer to merge a green PR.

## Acceptance evidence (not yet completed)

1. GitHub API `GET /repos/bren-wp/Ghost-FTP/branches/main` reports `protected: true` and the protection endpoint confirms admin enforcement, strict required checks and conversations.
2. A deliberate disposable failing PR is blocked from merging, including an admin's ordinary merge workflow.
3. A current, fully green PR with all eight exact check contexts can merge, and main's post-merge checks complete.
4. Record GitHub settings evidence and dates in issue #302 before closure. The presence of this document or script is **not** evidence that the protections have been enabled.
