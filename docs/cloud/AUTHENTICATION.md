# Cloud authentication and credential lifecycle

**Status:** Architectural contract for 0.94.0 (continuing the 0.93.0 cloud foundations). Presence of a desktop account adapter does not establish that every third-party identity flow has passed live integration testing.

## Identity classes

1. **Object-storage credentials:** S3-compatible SigV4 access keys, optional session credentials, region and endpoint. Do not embed static credentials in signed URLs or logs. Use the operating-system vault where supported. A custom endpoint must require secure TLS and deliberate configuration.
2. **Cloud API bearer credentials:** Google Drive, OneDrive, Dropbox and Box use vendor-specific API scopes, authorization, expiry and revocation. An S3 credential cannot be re-used to authorize Drive.
3. **One-object presigned capability:** Android/macOS currently accept selected S3-style, GCS-style and Azure SAS query parameters for one PUT or GET. No account is signed in. The URL is a secret and is not proof of identity just because its *shape* was accepted; the provider must validate it.
4. **WebDAV:** HTTPS URL plus correctly protected credentials/tokens. Never forward Authorization during an untrusted redirect.

## OAuth implementation requirements (macOS/Android: NOT IMPLEMENTED)

- System browser with OAuth Authorization Code + PKCE (S256), a per-session state value, exact registered redirect URI, minimal scopes and provider-appropriate endpoint validation.
- Exchange the authorization code only against the expected provider; reject state mismatch, reused authorization and untrusted app links.
- Protect refresh tokens using macOS Keychain or Android Keystore-backed secure storage, with access-control choices appropriate to the token threat model. Do not store OAuth client secrets in distributed apps.
- Refresh access tokens atomically; handle auth expiry, 401, revoked permission, account switch and user-driven disconnect. Delete local tokens on disconnect and revoke remotely when supported.
- Never place the token in an Activity Bundle, analytics event, exception/HTTP log, repository file, screenshot, crash report or default device backup.

## Signed-object workflow contract

**Android:** `SignedCloudTransfer.kt` accepts HTTPS with public provider hostnames and provider-specific, nonempty, nonduplicated signature query parameters; signed inputs stay transient. Android 0.94.0 routes signed requests through a direct-only OkHttp client whose DNS resolver refuses any mixed or non-public answer **at the actual HTTP transport's resolution point**. This closes the preflight/connection mismatch in the original HttpsURLConnection code, but does not replace real-device DNS rebinding, VPN or malicious network provider tests. Document picker grants narrow user-approved URI access. The output SAF document may be partially written after a failure.

**macOS:** `SignedCloudTransfer.swift` uses an ephemeral URLSession, no redirects, identity-encoded download requests, staged destination promotion and ephemeral in-memory URL state. Explicit DNS/socket binding and account OAuth are not yet complete.

**Windows/Linux:** Desktop cloud implementation resides under `ghostftp-desktop/src-tauri/src/session/`, including object and WebDAV adapters. Confirm per-provider key lifecycle in the associated vault code before claiming complete security parity.

## Disconnection acceptance tests

Disconnect must cancel outstanding token use, clear in-memory and at-rest secrets, sever account references without deleting user files and surface remote revocation errors separately from local deletion. A crash/restart must not auto-replay a destructive upload with a stale signed URL.

See [providers](PROVIDERS.md), [security review](../handbook/SECURITY_PRIVACY.md), [privacy policy](../legal/PRIVACY.md).
