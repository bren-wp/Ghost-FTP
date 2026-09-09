# Ghost FTP testing and quality gates

Ghost FTP **0.0.1** is validated through layered source, security, native build, packaging, UI-action and release-lifecycle gates.

## Core quality gate

The canonical Core gate requires:

```text
gofmt
go test -race ./...
go vet ./...
```

It also runs repository, platform, desktop-surface, dependency, version, localization, security, privacy, documentation and release audits plus the Python regression suite.

## Protocol and transfer regressions

Tests cover the maintained FTP/FTPS/SFTP engine contract, including:

- explicit FTPS verification and no silent downgrade;
- strict SFTP host-key verification/pinning;
- rooted local path and transfer confinement;
- transfer staging/activation/rollback behavior;
- connection-generation guards;
- privacy-safe diagnostics;
- bounded automatic retry policy;
- queue pause/resume/cancel/retry lifecycle;
- Remote Edit text/binary, size, revision/conflict, permission, read-back and metadata-refresh behavior.

## Settings regression contract

The settings suite verifies that visible runtime options remain bounded and migration-safe.

Examples include:

- parallelism accepts only **1–8** for explicit current values;
- an omitted legacy `parallelism=0` migrates to the canonical default **2**;
- negative or above-range explicit parallelism remains a validation error;
- connection timeout, retry count and retry delay stay inside documented bounds;
- unknown persisted conflict-policy state fails closed to conservative recovery behavior;
- one canonical conflict-policy field synchronizes legacy compatibility fields.

This prevents compatibility migration from becoming an excuse to silently accept arbitrary invalid settings.

## Desktop action wiring gate

`scripts/test_ui_action_wiring.py` prevents visible main controls from becoming decorative/dead UI.

Windows checks:

- enumerate main-window `mkButton(...)` command IDs from `createControls`;
- require every such ID to have a `case id...:` handler in `command()`;
- keep queue Cancel/Retry engine failures from being silently discarded.

Linux checks:

- require every maintained main control rectangle (connect/disconnect/profile/settings/navigation/file operations/upload/download/queue actions) to exist in layout construction;
- require the same control to have a mouse click handler;
- verify the Remote Edit button path;
- verify SFTP Trust/Cancel and prompt Apply/Cancel overlay controls are both rendered and handled.

The gate intentionally tests wiring, not just pixels. Runtime behavior remains covered by Go unit/integration tests and authentic UI smoke evidence.

## README/media integrity

README and active documentation use repository-local Ghost FTP icon/screenshot assets. Authentic screenshots are generated from the production Windows x64 Portable application. Remote tracking pixels, icon CDNs and mockup images are not accepted as release UI evidence.

## Windows production gate

The Windows production job builds and verifies:

```text
Ghost-FTP-0.0.1-Setup-x64.exe
Ghost-FTP-0.0.1-Setup-x86.exe
Ghost-FTP-0.0.1-Setup-x32.exe
Ghost-FTP-0.0.1-Portable-x64.exe
Ghost-FTP-0.0.1-Portable-x86.exe
```

It verifies release artifacts and exercises the Authenticode private-key pipeline policy. Production signing is optional; configured signatures must verify.

## Linux production gate

The Linux production job builds DEB and portable tar.gz packages for `amd64`, `arm64` and `i386` and compares the DEB/portable executable bytes for parity.

## Supplemental distro package gate

`.github/workflows/linux-distro-packages.yml` builds and verifies supplemental distro artifacts through:

```text
linux/BUILD-DISTROS.sh
```

Representative supplemental names include Debian, Ubuntu, Fedora and Portable families. These are CI verification artifacts, not additions to the canonical **12 platform artifacts / 15 public files** release allow-list.

## Native distro lifecycle gate

`.github/workflows/linux-distro-install.yml` verifies native install/remove/runtime/GUI behavior on:

- **Debian 13 amd64**;
- **Ubuntu 26.04 LTS amd64**;
- **Fedora 44 x86_64**.

**Native package-manager/runtime coverage is deliberately limited to x86-64.** Canonical production builds still include the documented additional Linux architectures.

## Authentic UI evidence

`.github/workflows/ui-screenshots.yml` builds the real Windows x64 Portable application and captures maintained Main Workspace, Site Manager, Settings and About windows. Mockups and generated approximations are not release evidence.

A release-prep change affecting `VERSION` or maintained desktop UI must obtain authentic evidence from the exact final source revision where the screenshot workflow is triggered.

## Exact-head and post-merge rule

**Exact-head and post-merge rule:** a PR is not merge-ready until every required workflow triggered for its exact final head is `completed/success`. After merge, required `push` workflows are identified by the exact merge SHA and must also finish `completed/success` before release preparation continues.

For a release-prep change that affects the canonical production build, the expected gates are:

1. Ghost FTP CI;
2. Ghost FTP Linux Distro Packages;
3. Ghost FTP Linux Distro Install Matrix;
4. Authentic UI Screenshots when its path/trigger contract applies.

A green run for an older commit does not satisfy a newer PR head.

## Release publication gate

0.0.1 publication additionally requires:

- exact current `main` release-branch validation;
- canonical release workflow quality/build jobs;
- exact 15-file GitHub Release allow-list;
- immediate and delayed remote release read-back;
- `prerelease=false` for the current 0.0.x release channel;
- verified `ghcr.io/bren-wp/ghost-ftp:0.0.1` distribution-bundle publication/read-back;
- successful latest-only retention cleanup after publication.

## Deterministic release-to-retention gate

The release branch trigger does not consider `gh workflow run release.yml` itself a successful release. Its regression/audit contract requires the trigger to:

1. record pre-existing release run IDs;
2. dispatch the canonical release workflow;
3. identify the newly created workflow-dispatch run whose `headSha` equals validated current `main`;
4. wait for that exact run with `gh run watch --exit-status`;
5. require terminal `success`;
6. only then dispatch canonical retention;
7. identify the new exact-main retention run;
8. wait for and require terminal retention `success`.

This closes the class of failure where publication succeeds but downstream `workflow_run` chaining is not emitted for a token-dispatched workflow. The retained `workflow_run` retention trigger remains defense in depth; successful release-branch orchestration requires explicit observed retention completion.

## Retention validation

The retention workflow must leave only the current `ghostftp-v0.0.1` release/tag, retain the current canonical release branch and exact-version GHCR package, remove superseded release branches/package versions, and leave `main` commit history untouched.

Before destructive cleanup it independently verifies current release identity, `draft=false`, `prerelease=false`, exactly **15 assets**, and current tag SHA equality with current `main`.

## Quality rule for new power-user features

A new feature such as search/filter, directory comparison, synchronized browsing, bandwidth limits, queue priority or bookmarks is not release-ready until all applicable layers exist:

- shared engine/runtime behavior;
- validation and safe defaults;
- Windows and Linux controls or an explicit documented platform limitation;
- localized user-facing copy;
- success/failure/cancel semantics;
- unit/integration/regression tests;
- action-wiring coverage for new controls;
- active documentation;
- authentic screenshot evidence when the maintained UI changes.

See [GitHub Releases](GITHUB-RELEASES.md), [Release verification](RELEASE-VERIFICATION.md), [Settings](SETTINGS.md), [Roadmap](ROADMAP.md) and [Versioning](VERSIONING.md).
