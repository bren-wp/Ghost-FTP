# Cloud provider compatibility — 0.93.0 source candidate

This inventory describes **interfaces present in source**, not successful live authentication against every vendor. **IMPLEMENTED** means adapter/workflow source exists; **PARTIAL** means a narrowly defined operation; **BLOCKED** means the integration is not available. **VERIFIED** must be supported by a named provider test and exact-commit evidence. No provider is designated VERIFIED solely from compilation.

## Provider and platform matrix

| Provider | Windows / Linux desktop | macOS Preview | Android | Identity / operation boundary |
| --- | --- | --- | --- | --- |
| Amazon S3 | IMPLEMENTED adapter, provider E2E pending | PARTIAL signed object PUT/GET | PARTIAL signed object PUT/GET | S3-compatible SigV4 in desktop adapter; signed X-Amz URL on mobile |
| Cloudflare R2 | PARTIAL via S3-compatible adapter | PARTIAL signed object PUT/GET | PARTIAL signed object PUT/GET | R2 S3-compatible endpoint, provider-specific test pending |
| Backblaze B2 | PARTIAL via S3-compatible adapter | PARTIAL signed object PUT/GET | PARTIAL signed object PUT/GET | S3-compatible credentials on desktop; signed object links on mobile |
| Wasabi | PARTIAL via S3-compatible adapter | PARTIAL signed object PUT/GET | PARTIAL signed object PUT/GET | S3 endpoint support subject to region/header checks |
| DigitalOcean Spaces | PARTIAL via S3-compatible adapter | PARTIAL signed object PUT/GET | PARTIAL signed object PUT/GET | S3-compatible operations; signed-host validation |
| Scaleway Object Storage | PARTIAL via S3-compatible adapter | PARTIAL signed object PUT/GET | PARTIAL signed object PUT/GET | Region-appropriate S3 endpoints |
| Storj / Oracle Object Storage / IBM Object Storage | PARTIAL candidate via configurable S3 endpoint | BLOCKED arbitrary signed host | BLOCKED arbitrary signed host | No live provider acceptance evidence |
| MinIO / Ceph | PARTIAL configurable S3-compatible endpoint | BLOCKED custom/local signed endpoint | BLOCKED custom/local signed endpoint | No unsafe trust-all TLS or local-network bypass |
| Google Cloud Storage | IMPLEMENTED desktop adapter, live E2E pending | PARTIAL GCS signed object PUT/GET | PARTIAL GCS signed object PUT/GET | GCS signed query X-Goog-Signature; NOT Google Drive |
| Azure Blob | IMPLEMENTED desktop adapter, live E2E pending | PARTIAL Azure SAS object PUT/GET | PARTIAL Azure SAS object PUT/GET | Azure SAS sig and block-blob PUT requirement |
| WebDAV / Nextcloud / ownCloud | IMPLEMENTED WebDAV desktop adapter, provider E2E pending | BLOCKED | BLOCKED | Desktop HTTPS endpoint; Basic/Bearer redirects disabled |
| Google Drive | IMPLEMENTED desktop account adapter, OAuth acceptance pending | BLOCKED OAuth account access | BLOCKED OAuth account access | Drive OAuth and file APIs, separate from GCS |
| OneDrive | IMPLEMENTED desktop adapter, acceptance pending | BLOCKED | BLOCKED | Microsoft Graph, not generic signed S3 |
| Dropbox | IMPLEMENTED desktop adapter, acceptance pending | BLOCKED | BLOCKED | Dropbox-specific API and OAuth |
| Box | IMPLEMENTED desktop adapter, acceptance pending | BLOCKED | BLOCKED | Box-specific API and OAuth |

**Important:** A provider-specific signed URL is a short-lived bearer capability for **one object**, not an account connection, directory explorer, sync engine, bucket inventory, token vault or resumable queue. Mobile supported host families are deliberately restricted and must not be broadened to attacker-controlled domains for convenience.

## Windows/Linux cloud connection wizard — 0.95.48 Preview

In **Sites → Connect Cloud**, users choose browser-based account authorization (Google Drive, OneDrive, Dropbox, Box) or **S3 / other storage**. The latter provides guided credential-based forms for Amazon S3 (including custom HTTPS S3-compatible endpoints, e.g. Wasabi/MinIO), Backblaze B2 (automatically constructs `https://s3.<region>.backblazeb2.com`), Cloudflare R2, Google Cloud Storage (JSON service-account file picker), WebDAV and Nextcloud (constructs `/remote.php/dav/files/<username>/` when given the instance URL). No browser-OAuth behavior is falsely claimed for an S3 application key.

**Test Connection** uses a read-only provider listing for object storage, or depth-0 WebDAV PROPFIND, before metadata is saved; users may save unverified credentials for restricted write-only keys, with a conspicuous warning. Passwords/keys passed during saving are kept out of plaintext profile JSON by the existing OS-keychain backend; GCS stores a reference to a user-controlled JSON key file, not its private contents.

**MEGA and Proton Drive are not native adapters.** The wizard explains the limitation and can open official MEGAcmd and Proton Drive CLI instructions; no third-party login, encryption keys, FTP/WebDAV emulation, or remote operations are invented. The Proton Drive SDK presently says it is not ready for commercial/production third-party app use. Real installed Windows/Linux, account-specific E2E and vault-persistence acceptance are pending.

## Capability levels

| Capability | Windows / Linux | macOS | Android |
| --- | --- | --- | --- |
| Provider account connection | IMPLEMENTED adapters; verify each vendor | BLOCKED | BLOCKED |
| List buckets, folders or drives | IMPLEMENTED/adapter-dependent | BLOCKED | BLOCKED |
| Single upload/download | IMPLEMENTED/adapter-dependent | PARTIAL signed object | PARTIAL signed object via SAF |
| File/folder rename or move | Adapter-dependent | BLOCKED | BLOCKED |
| Multi-file copy/sync | Adapter-dependent | BLOCKED | BLOCKED |
| Multipart/resume for cloud | Adapter-dependent, inspect per backend | BLOCKED | BLOCKED |
| Third-party OAuth with PKCE | Verify each desktop provider | BLOCKED | BLOCKED |

## Evidence required to mark a provider VERIFIED

1. Disposable provider test account and minimum viable API permissions, stored only in protected CI secrets.
2. Authentic sign-in / credential lifecycle, disconnect and invalid credential cases where applicable.
3. List/upload/download (empty/Unicode/large files), overwrite conflict, delete with confirmation and post-download hash comparison where supported.
4. Expired, revoked and wrong-method token/link; 401/403/404/429/5xx; controlled DNS and redirect failures.
5. Source SHA, CI run URL, logs with redacted sensitive data, package hash and real-device/OS acceptance.
6. Provider billing/rate limits and API terms verified by the release owner.

Related: [cloud authentication](AUTHENTICATION.md), [cloud QA](../qa/CLOUD_E2E.md), [platform matrix](../handbook/FEATURE_MATRIX.md) and [signed-URL threat model](../security/CLOUD_SIGNED_URL_THREAT_MODEL.md).
