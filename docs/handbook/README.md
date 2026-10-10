# Ghost FTP — complete product documentation and development status

**Development candidate:** 0.93.0 · **Previous verified public release:** v0.92.2 · **Current review:** PR #125. Documents are written in English to match the repository's existing engineering and marketing documentation. Status claims are scoped to inspected source and verified CI, not theoretical parity.

## Start here

| Document | Contents |
| --- | --- |
| [Windows app guide](WINDOWS.md) | Windows user options, protocols/cloud adapters, transfer queue, settings, installer, privacy and click-by-click test matrix |
| [Linux app guide](LINUX.md) | Linux app capabilities, WebKitGTK/keyring behavior, packages, desktop/network security and test matrix |
| [macOS app guide](MACOS.md) | Every existing SwiftUI workspace, saved-site/Keychain/FTP behaviors, signed cloud transfers and planned FTPS/SFTP |
| [Android app guide](ANDROID.md) | Every Kotlin workspace, connection/remote transfer actions, SAF cloud links, sensitive session state, QA and roadmap |
| [Platform feature matrix](FEATURE_MATRIX.md) | Each function classified SRC/LIMITED/TODO/VERIFY by platform, source mapping and status definitions |
| [Security and privacy model](SECURITY_PRIVACY.md) | Assets, threats, TLS/SSH, credentials, OAuth, signed links, storage, updates, file integrity, QA |
| [Detailed delivery roadmap](DELIVERY_ROADMAP.md) | P0/P1/P2 workstreams, implementation prerequisites, provider plans, test gates and release definition of done |
| [Cloud storage security scope](../product/CLOUD_STORAGE.md) | Signed links versus authenticated cloud accounts, risk boundaries and provider eligibility |
| [Implemented features](../product/FEATURES.md) | Repository's existing feature catalog; read platform-specific limitations in these guides |
| [QA documentation](../qa/README.md) | Existing production test and visual acceptance procedures |
| [Releases/versioning](../releases/VERSIONING.md) | Immutable tags and canonical source version |

## Source truth and how status changes

- **Published:** a GitHub Release exists for the exact commit with verified platform packages and checksums.
- **Implemented in source (SRC):** a reachable UI/transport handler exists in source. This is not a claim of complete device/provider acceptance.
- **Limited:** implementation intentionally covers only a platform/protocol, e.g. macOS FTP or mobile cloud signed-object URL.
- **In development:** a PR or code branch exists but has not passed all required gates; do not label stable.
- **Planned:** no complete implementation; clear work and acceptance criteria exist.
- **Not yet verified:** code exists but requires live provider/device and security tests.

## Cross-platform summary

Windows/Linux: broad cloud adapter implementation and advanced desktop file management, queue, analysis and terminal. macOS: SwiftUI saved sites, Keychain and FTP, with new signed cloud object transfers in candidate 0.93.0; FTPS/SFTP not yet active. Android: FTP/FTPS/SFTP app and file actions, with new signed cloud object transfers; not equivalent to desktop Site Manager/queue. Google Drive account access is in existing desktop adapters only. A GCS signed link is not a Google Drive sharing page.

## Update discipline

Update the platform guide, feature matrix, threat model, CHANGELOG, version notes and tests together with each code change. Mark a feature complete only when it has working source, UI, negative tests, protocol/provider integration tests and appropriate real-device acceptance. Record failed CI and known limitations honestly. Do not add marketing claims unsupported by the source.

## Production blockers

Cloud SSRF/DNS rebinding, credential lifecycle, secure file staging, a known transfer data-loss condition, failing CI, invalid release artifacts or wrong version lineage are P0 blockers. The current cloud signed-URL workflows are a restricted initial implementation; proper OAuth providers and full macOS/Android cloud browsing remain on the roadmap. Follow official provider security requirements and protect user files.
