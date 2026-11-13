# Ghost FTP — Windows/Linux cloud provider inventory

> **Active source:** Ghost FTP 0.99.59 Preview. **Previous canonical release:** v0.99.58. Supported platforms: Windows and Linux.

**SOURCE PRESENT** means implementation files exist; it is not live vendor certification. Features depend on account scopes and adapter capabilities. No provider receives a LIVE VERIFIED label without genuine authenticated end-to-end evidence.

| Provider family | Desktop source | External acceptance |
| --- | --- | --- |
| S3-compatible object stores (Amazon S3, R2, B2-compatible, Wasabi, Spaces) | `remotefs/object.rs`, `session/object.rs` | Pending per-vendor region, signing and multipart tests |
| Google Drive | `remotefs/gdrive.rs`, `session/gdrive.rs` | Pending authorized OAuth lifecycle |
| OneDrive | `remotefs/onedrive.rs`, `session/onedrive.rs` | Pending authorized Graph API tests |
| Dropbox | `remotefs/dropbox.rs`, `session/dropbox.rs` | Pending authorized API tests |
| Box | `remotefs/boxdrive.rs`, `session/boxdrive.rs` | Pending authorized API tests |
| WebDAV, Nextcloud, ownCloud | `remotefs/webdav.rs`, `session/webdav.rs` | Pending real TLS, lock and conditional-write tests |
| Additional services (Dynamics, HubSpot, Shopify) | `remotefs/dynamics.rs`, `remotefs/hubspot.rs`, `remotefs/shopify.rs` | Pending per-service permissions and operations |

Paths are relative to `ghostftp-desktop/src-tauri/src/`. This inventory does not assert that folder sync, delete, move, multipart or token refresh work for every backend.

## Security requirements

Never log tokens/signed URLs, silently bypass TLS identity, assume a multipart ETag is a SHA-256 digest, accept unverified transfer success, or overwrite existing local data without consent. Validate limits, redirects, HTTP ranges and complete disk writes before promotion.

[Live acceptance plan](../qa/CLOUD_E2E.md) · [Provider authentication](AUTHENTICATION.md) · [Historical multi-platform matrix](../audits/ARCHIVED_PROVIDER_PLATFORM_MATRIX_0.93.md).
