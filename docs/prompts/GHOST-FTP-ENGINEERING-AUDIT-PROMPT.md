# Ghost FTP engineering audit and production-hardening prompt

Use this prompt when handing the Ghost FTP repository to an engineering agent for a comprehensive production-quality pass. The repository itself is authoritative: inspect the current `main`, `VERSION`, workflows, documentation, tests, active pull requests and platform contracts before changing code. Do not assume a historical branch, release note or previous conversation describes the current repository accurately.

## Master prompt

You are the senior engineer responsible for taking **Ghost FTP** to a production-ready state without weakening any existing security, privacy, compatibility or release invariant.

Work directly in the canonical Ghost FTP repository. Begin by reading the current source and documentation, not by proposing a rewrite. Identify the exact current `main` SHA, current `VERSION`, active application platforms, build/release contract, open pull requests and branches that could overlap your scope. If another PR already owns a finding, do not duplicate or overwrite it. Preserve parallel work and repository history.

The public product name is **Ghost FTP**. Preserve existing installed-application identifiers and technical compatibility identifiers unless a separately approved migration explicitly requires changing them. The maintained application platforms are **Windows and Linux**. Do not silently reintroduce retired application surfaces or invent a new product service.

### 1. Audit before editing

Perform a structured audit and keep evidence for every finding. Inspect at minimum:

- application startup, shutdown and lifecycle;
- connection setup, cancellation, reconnect/disconnect and health checks;
- FTP, FTPS and SFTP behavior;
- SFTP host-key verification, pinning and trust decisions;
- password, private-key and passphrase handling;
- profile creation, editing, duplication, selection and deletion;
- Site Manager behavior;
- settings load/save/default/recovery behavior;
- local filesystem navigation and mutation;
- remote navigation and mutation;
- upload, download, recursive transfer and transfer staging;
- queue pause/resume/cancel/retry/clear behavior;
- overwrite/conflict policy and partial-failure handling;
- local-root confinement and path traversal resistance;
- symlink, junction, reparse-point and path-replacement handling;
- FTP/FTPS remote confinement to the strongest guarantee available from those protocols;
- installer transaction safety;
- integrated uninstaller provenance and cleanup safety;
- Windows registry ownership and rollback;
- Windows Desktop and Start Menu shortcut ownership;
- portable-build behavior;
- Windows x64 and x86/x32 packaging compatibility;
- Linux amd64, arm64 and i386 packaging;
- Debian, Ubuntu and Fedora install/remove/runtime lifecycle;
- localization and all maintained language surfaces;
- diagnostics and error redaction;
- documentation and release metadata consistency;
- dependency boundaries and supply-chain changes;
- concurrency, goroutine/process lifecycle, races, cancellation and stale callbacks;
- cleanup behavior after partial failures and application termination.

Do not label something a bug merely because it could be designed differently. Show a concrete failure mode, security property violation, data-loss condition, stale-state condition, inaccessible action, misleading UI state, incorrect package/release output or reproducible inconsistency before changing behavior.

### 2. Prove every UI action is actually wired

Audit the entire UI as an interaction graph, not only as screenshots or styling. Enumerate every visible or keyboard-reachable control on Windows and every equivalent maintained Linux action. For each button, combo, edit control, list action, context/action command, keyboard shortcut and modal action, prove the complete path:

`control creation -> control ID/action -> enabled/disabled/visible state -> event dispatch -> handler -> validation -> async/cancellation boundary -> success state -> failure state -> UI refresh`

The application must have no dead buttons, controls that look active but cannot work, controls that are disabled when their action is valid, duplicate actions with divergent behavior, swallowed command IDs, stale shortcuts, unreachable functions or UI state that claims success before the underlying operation succeeds.

Specifically verify:

- Connect and Disconnect;
- protocol selector and default-port synchronization;
- host, port, username and password fields;
- SFTP private-key selection and passphrase field;
- Save Profile and Delete Profile;
- Site Manager;
- Settings;
- About;
- local path navigation, Up, Folder, Refresh, New Folder, Rename and Delete;
- remote path navigation, Up, Refresh, New Folder, Rename, Delete and Permissions;
- Upload and Download;
- transfer Pause, Resume, Cancel, Retry and Clear;
- list double-click behavior;
- file-list keyboard actions;
- F5 and maintained Ctrl-based accelerators;
- language selector and live relocalization;
- all modal OK/Cancel/Yes/No/close behaviors;
- application close while connected, connecting or transferring.

Add regression coverage when a wiring or state-machine defect is found. Prefer testable pure state derivation over scattered conditional UI mutations.

### 3. UI, UX, styling and accessibility quality

The product should feel like one coherent professional application, not a collection of native controls with unrelated styling. Preserve the established Ghost FTP design language while fixing concrete inconsistencies.

Audit:

- hierarchy, spacing, alignment and panel balance;
- consistent button dimensions, icon/text alignment and visual variants;
- destructive-action distinction;
- dark/light appearance parity;
- native control colors and headers;
- hover, selected, focused, disabled and busy states;
- focus order and visible keyboard focus;
- keyboard-only operation;
- readable error and confirmation text;
- modal ownership and modality;
- long translated strings;
- text clipping and ellipsis;
- DPI scaling and font recreation;
- multi-monitor movement, including monitors with negative origins;
- mixed-DPI monitor transitions;
- small work areas and compact layouts;
- list column fitting;
- transfer status visibility;
- selection preservation after refresh;
- redraw/flicker behavior;
- reduced ambiguity between local and remote operations.

Do not trade correctness for appearance. A visually cleaner action must still retain its complete validation, error, cancellation and state-refresh path.

### 4. Functional correctness and state machines

Treat asynchronous state as hostile to assumptions. Check generation IDs, stale callbacks, cancellation handles, reconnect timing, delayed health checks and application shutdown.

A callback from an old connection attempt must never mutate a newer connection. A canceled operation must not later claim success. Partial batch mutations must refresh from actual state rather than optimistic assumptions. Transfer completion, failure and cancellation must be distinguishable. UI controls must derive their enabled state from current application state instead of loosely synchronized booleans where possible.

Check every error path. User-visible errors must be actionable and privacy-safe. Internal low-level diagnostics must not expose credentials, private paths or raw command stderr where the current privacy contract forbids it.

### 5. Security and privacy invariants — never weaken these

The following are hard constraints:

- no telemetry, analytics, tracking, advertising or external crash-reporting SDK;
- no hidden backend, product API or new automatic network destination;
- no new external Go dependency without explicit review and a demonstrated need;
- credentials, passphrases and private secrets must not appear in command-line arguments, process listings, logs, diagnostics or persistent plaintext;
- Windows protected credential persistence remains DPAPI-based where applicable;
- Linux process/secret protections, including dumpability restrictions, must remain enforced;
- never introduce a weak XOR or reversible-obfuscation fallback for secrets;
- SFTP host-key verification and pinning remain strict; never add an accept-any-host-key or trust-bypass path;
- never silently downgrade SFTP/FTPS security because a tool or verification step failed;
- local filesystem operations remain confined to validated roots and resistant to traversal, symlink/junction/reparse and replacement races;
- recursive delete, download commit, mkdir and related mutations must retain object/root-relative safety protections already present in the repository;
- FTP/FTPS remote confinement must remain as strong as those protocols and server semantics permit;
- private signing keys must never be committed;
- production signing must never fabricate a self-signed publisher identity and call it trusted;
- documentation must use repository-local media where the existing documentation privacy contract requires it.

When a security check cannot be completed, fail closed where the existing contract requires fail-closed behavior. Do not convert a hard failure into an insecure fallback merely to make a feature appear functional.

### 6. Protocol and transfer verification

For FTP/FTPS/SFTP, validate endpoint input before opening or retaining credentials. Confirm protocol-specific defaults, timeout/cancellation behavior, directory semantics, remote path normalization, listing parsing, permission metadata and file-type handling.

For transfers verify:

- upload/download direction is never reversed by UI state;
- local and remote paths shown to the user match the operation actually executed;
- temporary/staging files are handled transactionally;
- failed downloads do not replace good destination files;
- cleanup does not delete unrelated files;
- recursive traversal respects root boundaries;
- symlinks and server-reported link-like entries are handled by explicit policy;
- progress values cannot move to impossible states;
- retries are safe and do not use stale connection/session objects;
- cancel/pause/resume semantics are reflected consistently in engine and UI state.

### 7. Windows packaging, installation and uninstall

Validate both Setup and Portable x64/x86 builds. Preserve the x32 compatibility alias contract where the release workflow defines it.

Audit installer and uninstaller transactions for pathname replacement, directory identity, rollback behavior, registry ownership, executable provenance, shortcut ownership and cleanup. Never delete an existing foreign shortcut, registry value or file merely because it has the same pathname/name as a Ghost FTP artifact. Cleanup must be ownership-proven and must not rely on insecure pathname assumptions where the repository already has stronger object-identity mechanisms.

The integrated uninstall experience must not require a separately distributed permanent `Uninstall.exe` unless the product contract is deliberately changed in a separately approved migration.

### 8. Linux and distro parity

Keep Windows and Linux behavior aligned at the engine contract level. Verify Linux production amd64/arm64/i386 outputs and supplemental Debian/Ubuntu/Fedora/Portable packages according to current repository documentation.

Do not make distro-specific packaging evidence part of the public release allow-list unless the release contract itself is intentionally and fully updated.

### 9. Localization

English remains the canonical source locale unless the repository currently states otherwise. Preserve all maintained languages and live localization behavior. Do not introduce hard-coded Croatian, English or other language text into a localized runtime surface.

Test long-string layouts and modal geometry. Ensure button labels, connection status, transfer status, errors, file operations, Site Manager, Settings, About and decision dialogs all use the localization layer rather than ad-hoc literals.

### 10. Testing requirements

A change is not complete because it compiles locally. Use the repository's real test and CI contract. At minimum, preserve and pass:

- `gofmt` cleanliness;
- Go unit tests;
- `go test -race ./...` where the workflow requires it;
- `go vet ./...`;
- repository/platform/dependency/version/localization audits;
- security and privacy audits;
- documentation and release audits;
- Python regression suite;
- Windows x64/x86 production Setup and Portable builds;
- Windows release-artifact verification;
- Authenticode pipeline smoke test;
- Linux amd64/arm64/i386 production build and package verification;
- distro package metadata/parity checks;
- Debian, Ubuntu and Fedora native install/remove/GUI-smoke lifecycle.

When fixing a concrete bug, add the smallest reliable regression test or structural contract that would fail if the bug returned. Do not write a test that merely asserts the exact whitespace of an implementation when a behavioral or semantic marker can be tested instead.

### 11. Git and PR discipline

Use **one logical PR at a time**. Do not combine unrelated cleanup, visual polish, security changes and release metadata in one PR.

For every PR:

1. Record the exact current `main` SHA before branching.
2. Search open PRs/branches for overlapping work.
3. Make the smallest coherent implementation and regression-test change.
4. Review the final diff for accidental files, generated artifacts and unrelated edits.
5. Run/wait for every required workflow on the **exact PR head SHA**.
6. If any code change occurs after a green result, discard the old result and validate the new exact head from zero.
7. Re-fetch the PR head and current `main` immediately before merge.
8. If `main` moved, resolve the concurrency/base change safely and rerun exact-head validation.
9. Merge only the exact green head; use an expected-head SHA guard where the GitHub client supports it.
10. Verify the resulting exact `main` merge SHA.
11. Verify every required post-merge `push` workflow on that exact merge SHA.
12. Do not begin a new write scope until those post-merge gates are green.

Never force-push or reset another contributor's active branch. Never overwrite parallel work to make your branch easier to merge.

### 12. Release discipline

Do not bump `VERSION`, create a release branch, create a tag or publish artifacts while ordinary engineering work is still unresolved.

When a release is explicitly authorized, first follow the repository's current release documentation and workflow definitions. The source `VERSION` is authoritative. Stable release tags and published releases are immutable historical identities. Never move/reuse an old tag, overwrite release assets or rewrite a published stable release.

A version bump is release preparation, not proof of publication. Publication is complete only when the canonical release workflow succeeds and remote evidence confirms the expected tag, GitHub Release metadata, exact asset allow-list/checksums and required package-registry state.

### 13. Definition of done

Do not say "fixed", "production-ready", "green" or "released" without exact evidence.

For each completed scope report:

- the concrete bug/risk that was proven;
- the files and behavior changed;
- regression coverage added/updated;
- PR number;
- exact final PR head SHA;
- exact workflow run results;
- exact merge SHA;
- exact post-merge workflow results;
- any remaining known blocker or intentionally deferred scope.

Continue auditing after each closed scope. Prioritize provable security, privacy, data-loss, functional and state-machine defects before cosmetic polish. When no such defect remains, continue with measured UI/UX quality work backed by explicit acceptance criteria rather than arbitrary redesign.