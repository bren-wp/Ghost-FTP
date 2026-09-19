# Release history

## 0.0.9

Ghost FTP 0.0.9 is the current source release line for Windows, Linux and Android.

This line retires the browser-extension distribution surface, keeps macOS retired, aligns the primary runtime workspaces more closely with the supplied master references, and changes the public release contract to **9 platform artifacts / 12 public files**.

Published versions are immutable. Any later source, UI, behavior, documentation, packaging or release-metadata change must use a higher version.

## 0.0.8

Ghost FTP 0.0.8 is the current published release.

Highlights:

- unified the maintained Files / Connections / Transfer Queue / Settings hierarchy;
- expanded Windows universal Setup + Portable packaging;
- expanded Linux Debian / Ubuntu / Fedora universal bundles;
- published the Android application with explicit signing-state metadata;
- published Chrome / Edge / Firefox / Opera local helper packages;
- added exact-head runtime evidence for Windows, Linux and Android;
- strengthened release verification, security, privacy and checksum contracts.

The historical 0.0.8 line previously carried a macOS validation surface. That application has since been retired from the active source tree and future release contracts.

## 0.0.7

0.0.7 remains an immutable protected baseline under tag `ghostftp-v0.0.7` where present. The `.github/workflows/release-retention.yml` policy preserves that historical tag/release identity and must not rewrite it.

## 0.0.6 and earlier

Earlier releases represent previous product and packaging contracts and must not be used to infer the current active platform set.
