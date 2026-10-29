# Phase 3 — actual v0.99.19 native Sync & Backup screenshots vs v8/v10 concepts (29 October 2026)

## Strict provenance

These **are genuine running-application screenshot pixels** from the earlier PR #243 head `ac1e34ed5d082bf69ef4bcba92c4090f8acbfd63`, captured in [successful Windows/Linux native-build run 40793685872](https://github.com/bren-wp/Ghost-FTP/actions/runs/40793685872). They predate the subsequent one-line Clippy fix; they do not verify the final v0.99.19 head until fresh exact-SHA CI passes.

- Windows artifact `12863646039` (`GhostFTP-Windows-x64-Preview-QA`), downloaded archive SHA-256 `d970f612c0d7eb162588eab12fcd1d41a69ebb06138b8f0f1ddd16fb7aa1f8fc`; screenshot `GhostFTP-Windows-Preview-Sync-Backup.png`, SHA-256 `a1ed9470557d3b78f6a4edb406955ea15464c6256da0cc444d2f1dfb171f74f9`. QA metadata: real `ghostftp.exe`, `WindowTitle=Ghost FTP`, `View=sync`, 1290×852, `PrintWindow(PW_RENDERFULLCONTENT)`, color/edge/opacity checks satisfied.
- Linux artifact `12862768368` (`GhostFTP-Linux-x86_64-Preview-QA`), downloaded archive SHA-256 `354e26c65616a56d5f79e0b9ecef9e36a9cc31927104da2da8544139e3bf4373`; screenshot `sync/Linux-Sync-1290x852.png`, SHA-256 `e84cedae8cbd10181873e1ac202572c425db47da53167c4b554954e6f0ee4a8e`. A dedicated, isolated real GTK/WebKit session had a functioning Xvfb display and WebKit content quality validator (`68` quantized colors, `0.8943` bright coverage and `0.0144` horizontal edge ratio).
- Reference: user-supplied `GhostFTP-Premium-Brand-UI(10).zip` SHA-256 `1cdaea3e60c0ec4b14c3e8fa515351ba0710ee9022060e7f107b8bb9fbf6e877`; exact paths `screens/windows/sync.png` and `screens/linux/sync.png`.
- Decoded all screenshots to RGBA with Pillow; pixel difference = any unequal 8-bit R/G/B/A channel at the same (x,y). Mean absolute RGB difference is on the 0–255 scale. Screenshots and concepts are all 1290×852, or **1,099,080 coordinate pixels** each.

| Platform / reference | Real screenshot | Different RGBA pixels | Fraction | Mean absolute RGB difference |
| --- | --- | ---: | ---: | ---: |
| Windows `sync` | `GhostFTP-Windows-Preview-Sync-Backup.png` | 1,068,415 | **97.21%** | 15.04 |
| Linux `sync` | `sync/Linux-Sync-1290x852.png` | 1,061,126 | **96.55%** | 14.78 |

## What the actual captures prove and do not prove

Both real apps open a functional **Sync & Backup** workspace from the actual navigation, display **New sync pair**, communicate the lack of configured pairs and show no fabricated credentials, scheduled backups, successful transfers or fake progress. They establish that the new view routing and native screenshot pipeline work on the existing prior commit.

The concept references depict Croatian text, four metric cards, three populated jobs, status history, quick tools and a claimed encrypted SFTP session. In an isolated unconfigured runtime, those are **illustrative data and must NOT be synthesized** as a fake authenticated app screenshot. The real screens show empty-state UI and English locale, so measured coordinate differences also reflect legitimate runtime-state and localization mismatches. Actual premium hierarchy/spacing/brand fidelity still requires design review and future CSS/layout improvements.

## Acceptance ledger (non-overlapping reference screens)

- Prior QA: **7 Windows** reference screens + **1 Linux** + **7 Android** = 15 reference-screen comparisons across previously released v0.99.17 and v0.99.18 builds.
- New real native Sync captures: **+1 Windows +1 Linux** = **17/75 reference comparisons** from mixed but traceable source builds.
- **0/75 accepted as pixel-perfect.** Completeness requires matching authentic app state, design geometry, keyboard/screen reader checks, interaction tests, an independent review and successful exact-head release QA. Do not mark the evidence manifest as accepted or infer 100% functional coverage from screenshot capture.
