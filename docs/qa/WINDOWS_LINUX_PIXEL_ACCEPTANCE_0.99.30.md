# GhostFTP — Windows/Linux pixel-by-pixel acceptance protocol

**Status:** not yet accepted 1:1. A user-supplied ZIP contains concept illustrations,
not installed app screenshots: 30 Windows screens and 29 Linux screens. They
must be compared **individually**, using a real running native build at the
same 1290×852 logical/physical frame and aligned DPI, theme, locale and
authentic content state. Do not simulate a successful server transfer merely
to reproduce a marketing concept.

## Historical real app baselines (not a current-source claim)

The actual Linux build **v0.99.19** was captured in
[GitHub Actions run 40793685872](https://github.com/bren-wp/Ghost-FTP/actions/runs/40793685872)
and independently compared against the user-provided concept archive:

| Real Linux screen | Different exact RGBA pixels | Changed / total pixels | Mean absolute RGB channel difference (0–255) | 1:1 accepted |
| --- | ---: | ---: | ---: | --- |
| Files | 1,055,516 | 96.04% / 1,099,080 | 15.41 | No |
| Sync & Backup | 1,061,126 | 96.55% / 1,099,080 | 14.78 | No |

The large raw pixel mismatch includes different language, live filesystem
state, empty disconnected remote panes, OS font rendering and design layout.
It is **not** a 96% functional defect rate. Real-state desktop UI consistency,
component sizes and action behavior are separate gates.

## Reproducible measurements

With Pillow installed, extract the approved reference ZIP locally and run:

```bash
python3 tools/visual/pixel_compare.py \
  --reference /path/to/references/screens/linux/files.png \
  --actual /path/to/native-Linux-Files-1290x852.png \
  --json-output /tmp/ghostftp-files-parity.json
```

The script checks both dimensions, SHA-256 provenance and every RGBA pixel.
`--require-exact` fails on the first recorded mismatch when a truly exact
same-state fixture is expected; do not use it to declare invalid an honest
disconnected UI because a concept illustrates fake connected server data.

**Per-screen release acceptance** requires: real native capture, matched test
state, pixel report, review of geometry/typography/icons, all real interactive
actions, keyboard/screen-reader checks, responsive and DPI states, crash-free
native launch and OS keyring/protocol safety. Mark a screen accepted only
after these checks, never because a source component path exists. Save
screenshot PNGs into the repository only from verified **actual** app runs.
