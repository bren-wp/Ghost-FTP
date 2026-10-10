# Ghost FTP — platform-by-platform feature status and traceability matrix

This is the **candidate v0.93.0** feature inventory based on inspected repository source. The final authority on published versions is GitHub Releases; current source changes are in PR #125. A green unit test verifies a tested condition, not a full user experience. Distinguish **SRC** (source implementation present), **LIMITED** (restricted scope or protocol), **TODO** (not available), and **VERIFY** (targeted real-provider/device acceptance still required). Different platform statuses must not be collapsed into one generic marketing claim.

| Feature or action | Windows | Linux | macOS Preview | Android | Primary source and key restriction |
| --- | --- | --- | --- | --- | --- |
| Main app UI / workspace | SRC | SRC | SRC | SRC | Desktop React/Tauri vs Mac SwiftUI vs Kotlin Activity |
| Multiple layouts / panel controls | SRC | SRC | LIMITED | LIMITED | Desktop DualPaneBrowser, settings; Mac Files page |
| Files remote listing | SRC | SRC | LIMITED FTP | SRC FTP/FTPS/SFTP | Mac FTP MLSD only; Android ConnectionModel |
| FTP connect + transfer | SRC | SRC | SRC FTP | SRC | Verify server login and transfer E2E |
| FTPS authenticated transfer | SRC | SRC | TODO | SRC | Mac UI security summary is NOT FTPS engine |
| SFTP authenticated transfer | SRC | SRC | TODO | SRC | Mac UI security summary is NOT SFTP engine |
| SSH known-host verification | SRC | SRC | TODO | SRC | Must fail on mismatched key |
| Saved Site Manager | SRC | SRC | SRC simplified | LIMITED | Android has session-focused connection form |
| Favourites / tags / folders / bookmarks | SRC | SRC | TODO full parity | TODO full parity | Desktop SiteManagerDialog |
| Import profiles / formats | SRC | SRC | LIMITED JSON non-secret | LIMITED settings backup | Provider credentials must not leak |
| Duplicate site discovery | SRC | SRC | SRC advisory | TODO | Mac SavedSiteDuplicates |
| OS credential vault / Keychain | SRC | SRC availability depends | SRC | TODO account vault | Android password only in active memory |
| Temp Quick Connect | SRC | SRC | LIMITED | SRC session form | No unrequested persistent secret |
| Create remote folder | SRC | SRC | TODO | SRC | Mac FTP file ops currently narrow |
| Remote rename/delete | SRC | SRC | TODO site file controls | SRC | Confirm exact selected path |
| Folder upload/download | SRC | SRC | TODO | LIMITED | Android ordinary file picker and transfers |
| File properties / chmod | SRC where backend | SRC where backend | TODO | LIMITED | Cloud does not offer POSIX permissions |
| SHA-256 verification | SRC supported paths | SRC supported paths | TODO | TODO comprehensive | Must verify bytes |
| Concurrent queue / pause / resume / retry | SRC | SRC | TODO full queue | TODO full queue | Desktop TransferCenterDialog |
| Transfer/activity history | SRC | SRC | SRC FTP history | SRC bounded activity | Not automatically persistent on mobile |
| Directory sync/mirror | SRC | SRC | TODO | TODO | Mac/Android "Sync & Backup" is settings backup |
| Diff / disk scan / duplicate analysis | SRC | SRC | TODO | TODO | Desktop DirectoryDiff, DiskTreemap etc |
| Integrated SSH terminal | SRC | SRC | TODO | TODO | Desktop Terminal; agent authorization |
| Agent pairing/console | SRC | SRC | TODO | TODO | Desktop AgentBridge/AgentConsole |
| Command palette/snippets/skills | SRC | SRC | TODO | TODO | Desktop specific |
| Themes, density, languages | SRC | SRC | LIMITED | LIMITED | Desktop i18n selector != parity for mobile |
| Signed updater | SRC conditional signing | SRC conditional signing | TODO platform flow | TODO platform flow | Verified signed update metadata and package policy |
| Windows installers/portable | SRC | N/A | N/A | N/A | NSIS/MSI and portable smoke |
| Linux AppImage/DEB/RPM | N/A | SRC | N/A | N/A | Native build/install smoke |
| macOS SwiftUI application | N/A | N/A | SRC | N/A | Swift Package and macOS bundle |
| Android APK/AAB | N/A | N/A | N/A | SRC | Android Gradle CI and emulator |
| S3-compatible account browser | SRC | SRC | TODO | TODO | Desktop object_store adapters |
| AWS S3 + R2/B2 object operations | SRC | SRC | LIMITED signed URLs | LIMITED signed URLs | Provider header/method compatibility VERIFY |
| Azure Blob authenticated | SRC | SRC | TODO account access | TODO account access | Desktop Azure ObjectSession |
| GCS authenticated | SRC | SRC | TODO account access | TODO account access | GCS != Google Drive |
| WebDAV / Nextcloud | SRC | SRC | TODO | TODO | Desktop webdav adapter |
| Google Drive account adapter | SRC | SRC | TODO OAuth | TODO OAuth | Source gdrive adapter; provider acceptance VERIFY |
| Dropbox account adapter | SRC | SRC | TODO | TODO | Desktop dropbox adapter |
| OneDrive account adapter | SRC | SRC | TODO | TODO | Desktop onedrive adapter |
| Box account adapter | SRC | SRC | TODO | TODO | Desktop boxdrive adapter |
| Cloud signed PUT/GET links | Use auth adapters | Use auth adapters | SRC candidate | SRC candidate | Expiring S3/GCS/Azure URL; no permanent credentials |
| Cloud multi-part/resume | Backend dependent | Backend dependent | TODO | TODO | Do not claim universally |
| No required analytics | SRC policy | SRC policy | SRC policy | SRC policy | Verify dependencies and future telemetry |
| Backup without plaintext secrets | SRC policy | SRC policy | SRC | SRC settings | Test archive and import |
| TLS certificate validation | SRC | SRC | SRC for cloud; FTP plain | SRC | Mac FTPS not enabled |
| Private endpoint guard | SRC custom S3/Azure | SRC custom S3/Azure | LIMITED syntactic | LIMITED DNS precheck | Strong rebinding protections TODO |
| Live 1:1 approved pixel/device QA | VERIFY | VERIFY | VERIFY | VERIFY | No claim of 100% acceptance without captures |
| Official v0.93.0 published release | TODO | TODO | TODO | TODO | Wait all exact-SHA PR/main gates and asset hashes |

## How to read the matrix

**SRC** does not mean perfect or ready for every provider. **LIMITED** requires following the named restrictions. **TODO** means no implemented claim. **VERIFY** means QA evidence has not been established for this scope. "N/A" means a platform-specific installer/technology simply does not apply elsewhere. This matrix avoids claiming a single shared UI implementation across different toolkits.

## Source provenance

| Layer | Authoritative locations | Coverage |
| --- | --- | --- |
| Windows/Linux UI | ghostftp-desktop/src/App.tsx and src/components | Navigation, file browser, transfers, sync, settings |
| Desktop protocol core | ghostftp-desktop/src-tauri/src/session and remotefs | FTP/FTPS/SFTP, cloud and SaaS adapters |
| Desktop profile/secret storage | ghostftp-desktop/src-tauri/src/profiles, credentials and bridge | Serialized metadata, credential reference, OS keyring |
| Android | android/app/src/main/java/com/ghostftp/android/MainActivity.kt and ConnectionModel.kt | Workspaces and actual FTP/FTPS/SFTP behaviors |
| Android signed cloud | android/app/src/main/java/com/ghostftp/android/SignedCloudTransfer.kt | HTTPS signed file transport via SAF |
| macOS | macos/Sources/GhostFTPMacApp/Views and Services | SwiftUI workspace, FTP engine and profile persistence |
| macOS signed cloud | Services/SignedCloudTransfer.swift and Views/CloudWorkspace.swift | Signed PUT/GET, file panels and staging |
| Packaging/tests | .github/workflows, scripts and tests | Source formatting, compilation, packaging and E2E gates |
| Release bookkeeping | version.json, docs/releases, CHANGELOG.md | Candidate lineage and immutable release |

## Evidence needed before changing TODO/LIMITED to SRC

A pull request must add the working implementation plus unit tests, real protocol or provider integration tests, malformed-input/security cases, a verified UI click path and updated documentation. For features that can change files or remote permissions, test failure and rollback. Once production builds exist, verify the exact package on physical machines. Update this inventory with links to relevant test/CI runs and specific provider acceptance, rather than silently changing a marketing description.
