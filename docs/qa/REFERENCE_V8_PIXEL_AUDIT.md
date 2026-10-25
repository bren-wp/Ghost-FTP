# Ghost FTP reference v8 — 75-screen pixel comparison

**Source of truth:** user-provided `GhostFTP-Premium-Brand-UI(8).zip` (25 October 2026), SHA-256 `1cdaea3e60c0ec4b14c3e8fa515351ba0710ee9022060e7f107b8bb9fbf6e877`.

The ZIP contains **30 Windows** and **29 Linux** 1290×852 reference concepts and **16 Android** 480×960 concepts; **no approved macOS mockups**. Reference artwork is static illustrative design, NOT real FTP sessions or executable UI. The newly provided cyan-blue `#38ABFF` Android visuals supersede the 2026-10-16 violet UI mockups. The source code must still render real controls and genuine data.

## Automated measurement (no acceptance shortcuts)

Run `python -m pip install Pillow`, then:

```bash
python tools/qa/compare_reference_v8.py --reference-zip "GhostFTP-Premium-Brand-UI(8).zip" --captures-dir build/qa/captures --output build/qa/pixel-v8.json
```

Arrange **actual installed** screenshots as `build/qa/captures/windows/files.png`, `.../linux/files.png`, `.../android/files.png`, matching all 75 entries in `docs/qa/premium-75-screen-contract.json`. The script verifies the entire archive checksum, all reference names/dimensions, and records RGB mean absolute difference, changed pixels, exact equality, and missing captures per screen. `--require-all-captures` also fails on missing/mismatched captures. SHA verification protects against silently comparing with an older ZIP.

**Caveat:** different connected servers, dynamic file names, language, timestamps, platform chrome, font substitution, privacy protections and allowed error states can change legitimate pixels. A screenshot diff is diagnostic evidence, not proof of functional parity. No pixel-perfect acceptance is automatic.

## Verified six-screen baseline from v0.99.10 genuine emulator QA

Compared actual v0.99.10 Android 480×960 app screenshots from the user-associated GitHub QA artifact against v8:

| Screen | RGB mean absolute difference, 0–255 | Differing pixels |
| --- | ---: | ---: |
| about | 14.81 | 99.92% |
| files | 15.31 | 99.94% |
| settings | 13.88 | 99.94% |
| sites | 15.92 | 99.93% |
| sync | 21.55 | 99.94% |
| transfers | 18.36 | 99.94% |

These are **baseline measurements of v0.99.10, not of the newer Android cyan UI**. Ten other Android states require exact same-state captures. Real Windows/Linux captures also require source-registered acceptance. Nothing currently supports a 75/75 pixel-perfect claim. macOS requires its own future reference package, while existing native functional and security tests remain mandatory.

## Identity and functional constraints

- Android brand colors and icon glyph use the v8 cyan scheme; the sidebar/layout still needs independent screen-by-screen on-device verification and adjustment.
- Desktop Windows/Linux share 214 px sidebar, 52 px top chrome and 43 px navigation rows in the dark 1290×852 baseline; themes and small windows stay responsive.
- Native macOS uses semantic shared tokens but no fictitious Windows/Android pixel acceptance; identity-verified FTPS/SFTP issue #223 remains open.
- Never paste PNG concepts as app backgrounds, fake a connected profile, or bypass SSH host-key/certificate checks to match a mockup.
