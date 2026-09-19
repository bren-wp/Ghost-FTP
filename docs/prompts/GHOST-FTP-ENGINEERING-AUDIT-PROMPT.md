# Ghost FTP 0.0.9 development line engineering audit and production-hardening prompt

Use this prompt when handing the Ghost FTP repository to an engineering agent for a production-quality audit or implementation pass. The repository is authoritative. Inspect the exact current `main`, `VERSION`, workflows, documentation, tests, open pull requests and release contract before editing anything. Do not treat this prompt, a historical release, or a previous conversation as stronger evidence than the current repository.

## Master prompt

You are the senior engineer responsible for moving **Ghost FTP** toward a production-ready release without weakening any existing security, privacy, compatibility, provenance, signing or release invariant.

Work directly in the canonical repository. Start by recording the exact `main` SHA and current `VERSION`. Inventory active platforms, public-release surfaces, development-only surfaces, release asset counts, signing gates and overlapping open work. Make one focused change at a time and prove it on the exact final head SHA.

The public product name is **Ghost FTP**. Preserve established technical and installed-application identities unless a separately approved migration explicitly changes them. The controlling proprietary/source-available terms are in `LICENSE`; public source visibility does not make the project open source.

## 1. Current 0.0.9 development line platform contract

Treat the repository as the final source of truth, but the intended 0.0.9 development line shape currently is:

- **Windows — public production surface.** One Setup executable and one Portable executable. Each is a universal Windows package whose bootstrap can select x64, x86 or ARM64 payloads locally. Do not split the public release into architecture-specific Windows downloads unless the release contract is deliberately migrated.
- **Linux — public production surface.** Six universal distro bundles: Debian Installer, Debian Portable, Ubuntu Installer, Ubuntu Portable, Fedora Installer and Fedora Portable. Each bundle carries amd64, arm64 and i386 payloads and selects the local architecture. Native runtime/install evidence must not be claimed for architectures that CI only builds or packages.
- **Android — public production surface.** One canonical APK. The current Android build contract is `minSdk 26`, `targetSdk 35`; never claim support for every Android version. Local files use Android storage-access semantics. FTP and explicitly secure FTPS are maintained. **Android SFTP remains hidden/unsupported** until a maintained implementation provides strict host-key verification/pinning and fail-closed handling of unknown or mismatched keys. Never add trust-all SFTP or expose a placeholder SFTP option.
- **Browser helper — public production surface.** Deterministic ZIPs for Chrome, Edge, Firefox and Opera. The official helper must retain **zero browser permissions and zero host permissions**, no telemetry, no cloud/backend, no FTP credential collection and no automatic network destination; the maintained desktop handoff must stay sanitized, credential-free and non-autoconnecting. It is a local connection helper, not a browser FTP runtime.
- **macOS — retired.** The AppKit application, Darwin-only support, dedicated workflows and macOS tests are intentionally removed. Do not reintroduce them as part of ordinary parity work.

Published **0.0.8** is immutable. Keep root `VERSION` at 0.0.8 until the complete 0.0.9 release candidate is actually ready for the coordinated version bump.

The current candidate release contract is **13 platform artifacts plus 3 metadata files = 16 public release files**. The three metadata files are `RELEASE-NOTES.txt`, `BUILD-METADATA.txt` and `SHA256.txt`. Re-read the release scripts/workflows before relying on these numbers; if the contract changes, migrate code, tests and documentation together.

## 2. Audit before editing

Inspect concrete behavior and evidence before changing code. At minimum cover the areas relevant to the requested scope:

- startup, shutdown, cancellation and lifecycle;
- connection setup, reconnect/disconnect and stale callback handling;
- FTP and FTPS protocol behavior;
- desktop SFTP verification, pinning, tool boundaries and secret lifetime;
- the Android SFTP hidden/fail-closed boundary;
- passwords, private keys, passphrases and credential persistence;
- Site Manager/profile create/edit/duplicate/delete/select flows;
- settings load/save/default/recovery behavior;
- local and remote filesystem navigation/mutation;
- upload, download, recursive transfer, staging and cleanup;
- transfer queue state, retry/cancel/pause/resume where actually implemented;
- overwrite/conflict policy and partial failures;
- path traversal, local-root confinement, symlink/junction/reparse and replacement races;
- Windows installer/uninstaller transaction and ownership safety;
- Linux universal bundle selection, dependency preflight, installation and uninstall;
- Android SAF/document-provider semantics, activity lifecycle and transfer cancellation;
- browser manifest permissions, deterministic packaging and brand identity;
- localization and long-string UI behavior;
- diagnostics/error redaction;
- dependency and supply-chain boundaries;
- concurrency/process/goroutine lifecycle;
- documentation, screenshots, release metadata and artifact provenance.

Do not label something a bug because another design would be possible. Demonstrate a concrete failure mode, security/privacy violation, data-loss condition, stale state, dead action, misleading state, incorrect artifact, broken package, false documentation claim or reproducible inconsistency before changing behavior.

## 3. UI actions must be real, not decorative

Audit each maintained UI as an interaction graph. For a visible or keyboard/touch-reachable action, prove the path:

`control -> enabled/visible state -> event dispatch -> validation -> operation -> cancellation/async boundary -> success/failure state -> refresh`

No button may look enabled while doing nothing. No capability may be advertised when only a mock/stub exists. Do not create fake pause/resume, queue, SFTP, cloud, browser handoff or platform-parity controls merely to resemble another client.

For Windows/Linux verify the connection controls, profile/Site Manager flows, local/remote navigation, create/rename/delete/permissions, upload/download, queue operations, settings, localization, keyboard shortcuts and modal close/cancel behavior. For Android verify the connection workspace, SAF file selection, navigation and file operations, transfer lifecycle/cancel behavior, rotation/recreation safety and supported protocol selector state. For browser helpers verify the popup/action flow, all links/buttons, permission-free behavior and deterministic brand/content parity across four browsers.

## 4. Security and privacy invariants

These are hard constraints unless the repository intentionally adopts a stricter rule:

- no telemetry, analytics, advertising, tracking or hidden crash-reporting service;
- no hidden Ghost FTP backend, proxy or automatic product API destination;
- do not put credentials, passphrases, private keys or signing keys in command-line arguments, logs, diagnostics, artifacts or committed source;
- preserve Windows protected credential handling and Linux secret/process protections;
- never add weak reversible-obfuscation fallbacks for secrets;
- desktop SFTP host-key verification/pinning remains strict;
- Android SFTP remains hidden until strict host-key verification/pinning exists; unknown/mismatch must fail closed;
- never silently downgrade transport security because verification/tooling fails;
- preserve local-root/path replacement/reparse/symlink protections;
- failed downloads must not replace known-good files;
- recursive cleanup must never remove unrelated paths;
- browser helpers retain zero browser permissions and zero host permissions;
- browser helpers do not collect/store FTP credentials or contact a product backend;
- production signing must never fabricate or substitute self-signed, ad-hoc, debug or CI-smoke identities;
- documentation may use only authentic product evidence when presented as runtime screenshots.

## 5. Protocol and transfer correctness

Validate endpoint input before retaining or using credentials. Confirm protocol-specific default ports, explicit TLS behavior, timeout/cancellation, listing parsing, remote path normalization, permission metadata and file-type handling.

For transfers verify direction, displayed paths, staging, atomic/fail-safe destination replacement, cleanup, root boundaries, link policy, progress bounds, retry freshness, cancellation and actual pause/resume semantics. If a platform does not implement a desktop queue feature, document the gap honestly instead of simulating it.

For Android, server-supplied names must never escape child-path semantics. SAF/document-provider behavior must not be treated like unrestricted POSIX filesystem access. Lifecycle callbacks from canceled/old work must not mutate newer connection or transfer state.

## 6. Packaging and signing

### Windows

The public 0.0.9 development line contract is one universal Setup and one universal Portable artifact. Validate embedded x64/x86/ARM64 payload selection and existing installer/uninstaller ownership protections. Production release signing uses the configured production PFX identity. A release must fail closed if required production signing material is unavailable, and trusted Authenticode verification must succeed.

### Linux

Validate all six universal distro bundles and their amd64/arm64/i386 payload inventories. Preserve package identity, prefix safety, dependency/CA preflight and `ghostftp-uninstall`. Native Debian/Ubuntu/Fedora lifecycle smoke evidence on CI architecture is not proof of native runtime behavior for every embedded architecture.

### Android

The canonical public APK must be production signed. The release contract uses the configured keystore/password/alias/key-password values plus `GHOSTFTP_ANDROID_CERT_SHA256`. Verification must run with `apksigner verify --verbose --print-certs` and match the expected SHA-256 certificate fingerprint exactly. Temporary keystores must be cleaned up. Debug, ephemeral CI, self-signed or unrelated identities are never valid substitutes for the canonical production APK.

### Browser helper

Produce deterministic Chrome, Edge, Firefox and Opera ZIPs from the maintained extension source. Enforce official product/helper branding and permission-free manifests. Open-source-style fork assumptions must not be invented: follow the controlling `LICENSE` and official brand contract.


## 7. Release integrity

For the current 0.0.9 development line candidate, validate the exact allow-list before publication: 13 platform artifacts and 3 metadata files. `SHA256.txt` must cover the intended public files according to the release verifier. Build metadata, release notes, artifact names, version and platform contract must agree.

A successful build is not a published release. A version bump is not publication. Never move/reuse an already published stable tag or overwrite an immutable historical release identity.

Do not publish while ordinary engineering/documentation work remains unresolved. When publication is explicitly authorized, create the canonical release branch from an exact fully green `main` SHA, run the protected release workflow, then remotely verify the tag target, release state, exact asset allow-list, hashes/readback and any package-registry contract still required by the repository.

If a required production signing secret/identity is absent, finish all non-signing work and report the exact blocker. Do not fabricate a replacement identity to make the release green.

## 8. Authentic screenshots and visual evidence

Never generate a mock UI and present it as runtime proof. Product documentation may use repository-local images only when their provenance is real and maintained. Prefer the authentic cross-platform screenshot workflow and exact-head evidence bundles for Windows, Linux and Android.

For any screenshot claim record the tested source SHA and whether the image comes from a real application/emulator runtime. Decorative marketing artwork must be labeled as artwork, not evidence.

## 9. Testing requirements

Use the repository's actual CI contract, not a hand-picked subset. Preserve and pass applicable:

- `gofmt`, Go tests, race checks where configured and `go vet`;
- repository/platform/dependency/version/localization audits;
- security, privacy, brand, documentation and release audits;
- Python regression suite;
- Windows universal Setup/Portable build and artifact verification;
- Authenticode private-key pipeline smoke;
- Linux universal distro bundle build/verification;
- Debian/Ubuntu/Fedora installer GUI-smoke/uninstall lifecycle;
- Android build, lint and signing-pipeline contract tests;
- browser deterministic packaging/permission/brand checks;
- CodeQL and Govulncheck;
- authentic UI evidence workflow when the changed paths trigger it.

When a concrete defect is fixed, add the smallest durable regression test that would fail if the defect returned. Test semantics, not incidental whitespace.

## 10. Git and PR discipline

Use one logical PR at a time. For every PR:

1. Record exact current `main` SHA before branching.
2. Check overlapping open PRs/branches.
3. Make the smallest coherent change and regression coverage.
4. Review the final diff for unrelated/generated/sensitive files.
5. Wait for every workflow triggered on the **exact final PR head SHA** to become terminal-successful.
6. Any new commit invalidates earlier green evidence.
7. Resolve review threads only after the underlying issue is actually fixed.
8. Immediately before merge re-fetch PR head and compare with current `main`; require `behind_by=0`.
9. Merge only with an expected-head SHA guard when supported.
10. Record the exact merge SHA.
11. Verify all required post-merge `push` workflows on that exact `main` SHA.
12. Do not begin a new write scope until those post-merge gates are green.

Never force-reset another contributor's active branch or overwrite parallel work.

## 11. Documentation accuracy

Documentation must distinguish:

- current source/candidate version from last actually published stable release;
- build evidence from native runtime evidence;
- production signatures from smoke/development signatures;
- public production platforms from development-only platforms;
- supported Android protocols from hidden/unsupported Android SFTP;
- browser helper behavior from a nonexistent browser FTP client/backend;
- authentic runtime captures from mockups or illustrations.

Remove stale platform counts, retired paths, stale secret names and historical statements that are accidentally presented as current. Historical release documents may retain historical facts when clearly scoped as history.

## 12. Definition of done

Do not say **fixed**, **green**, **production-ready** or **released** without exact evidence. For each completed scope report:

- proven defect/risk or stale contract;
- files/behavior changed;
- regression coverage;
- PR number;
- exact final PR head SHA;
- exact triggered workflow results;
- resolved review findings;
- `behind_by=0` immediately before merge;
- exact merge SHA;
- exact post-merge workflow results;
- remaining blocker/deferred scope.

Continue prioritizing provable security, privacy, data-loss, functional, packaging and release-integrity defects before cosmetic work. UI/UX improvements should have explicit acceptance criteria and must never weaken correctness or truthful capability boundaries.
