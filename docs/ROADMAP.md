# Roadmap

Ghost FTP is currently focused on three maintained native applications: **Windows, Linux and Android**.

## 0.0.9 development focus

- finish Windows / Linux / Android visual parity against the supplied master references;
- keep every visible action engine-backed;
- remove developer-only labels and stale technical controls from product surfaces;
- align popup, More, Settings, connection and transfer states;
- improve readable labels, spacing and touch targets;
- preserve strict desktop SFTP trust and FTPS certificate validation;
- keep Android SFTP hidden until strict host-key verification is implemented there;
- improve README and product documentation;
- regenerate exact-head runtime evidence before release;
- keep packaging and release metadata synchronized with the three maintained native applications.

## Packaging

Windows remains Setup + Portable.

Linux remains Debian / Ubuntu / Fedora Installer + Portable.

Android remains an installable APK with explicit signer-state metadata.

Browser helpers remain supporting local packages, not a separate FTP engine.

## Retired surface

The former macOS application and distribution pipeline have been removed from the active repository surface.

