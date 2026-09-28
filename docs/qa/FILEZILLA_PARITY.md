# Ghost FTP / FileZilla functional parity

Last reviewed: 2026-09-28
Benchmark: FileZilla Client
Reference: https://filezilla-project.org/client_features.php

FileZilla is used only as a functional benchmark. Ghost FTP must not copy FileZilla UI, branding or visual identity.

This tracker deliberately errs toward **Partial** or **Missing**. A capability moves to **Equivalent** only after the Ghost FTP implementation and regression tests cover the relevant behavior. A capability moves to **Ghost FTP better** only after reproducible evidence demonstrates a concrete advantage.

## Ghost FTP better

No capability is claimed here yet. Marketing superiority is not a substitute for measured evidence.

## Equivalent

No broad category is claimed equivalent yet. Individual protocol primitives may already work, but equivalence requires a sufficiently broad compatibility and recovery matrix.

## Partial

| Capability | Ghost FTP status | Evidence / work still required |
| --- | --- | --- |
| FTP client core | Partial | FTP connection and transfers exist and protocol E2E CI exists. Expand PASV/EPSV/PORT/EPRT, IPv4/IPv6, FEAT, UTF8, MLSD/MLST, LIST fallback, SIZE, MDTM, REST, keepalive, timeout and NAT coverage. |
| Explicit FTPS | Partial | Implemented; continue TLS certificate/hostname validation and real-server interoperability coverage. |
| Implicit FTPS | Partial | Desktop support must be exercised by dedicated regression tests; Android parity is not yet claimed. |
| SFTP | Partial | Password/key paths and host-key protections exist. Continue encrypted-key, keyboard-interactive, changed-host-key, SSH Agent and ProxyJump coverage where applicable. |
| Upload / download queue | Partial | Queue exists. Continue crash/restart/update persistence, retry/cancel races and atomic completion tests. |
| Pause / resume | Partial | Do not claim complete parity until byte-range resume is proven for both directions across every supported FTP/FTPS/SFTP path. |
| Interrupted transfer recovery | Partial | Network disconnect/server restart/timeout recovery requires broader fault-injection evidence. |
| Site management | Partial | Existing Sites/profile surfaces are functional; continue auth/profile migration and compatibility regression tests. |
| Remote file operations | Partial | Rename, delete, properties, folder creation and remote edit paths exist. Continue permission, Unicode, long-path and conflict coverage. |
| Remote editing | Partial | Ghost FTP downloads to a managed temporary location, opens an editor and watches for save/upload. Continue crash and atomic-save edge cases. |
| Search / sync utilities | Partial | Source modules exist, but feature-by-feature benchmark parity is not yet established. |
| Large-file / many-small-file behavior | Partial | Must be backed by repeatable stress tests and memory/resource measurements. |
| Cross-platform desktop packaging | Partial | Windows and Linux native bundles are CI-built and lifecycle-tested. Wider distro/runtime coverage remains incomplete. |
| Android client | Partial / product extension | Native Android app exists and exact release-candidate install/launch is tested. Protocol parity and cross-release signing continuity remain incomplete. |

## Missing

A feature belongs here when no production implementation and test evidence are present. Keep this list evidence-driven; do not infer absence merely because a feature was not reviewed in one cycle.

Current high-priority gaps to close before claiming full FileZilla-class parity:

- a proven full byte-range resume matrix for upload and download across FTP, explicit FTPS, implicit FTPS and SFTP;
- crash/restart/update-safe persistent queue recovery with no stored credentials;
- a broad active/passive FTP compatibility matrix including IPv6 and NAT edge cases;
- complete SSH Agent / ProxyJump coverage where desktop support is intended;
- representative real-server and OS compatibility evidence beyond the current CI fixtures;
- Android cross-release upgrade continuity using an existing secure persistent signing identity.

## Intentionally different

- Ghost FTP keeps its approved Ghost FTP UI/UX and branding; FileZilla UI is not copied.
- Ghost FTP may expose capabilities through different navigation or workflows when that preserves the approved product design.
- Native Android is treated as an additional Ghost FTP platform rather than a reason to imitate FileZilla desktop UI.

## Status discipline

For every parity change:

1. link the implementation path;
2. add or strengthen a regression / interoperability test;
3. run the relevant platform and protocol gates;
4. update this file only after the evidence exists;
5. never promote a status because a feature merely compiles or because a mock reports success.
