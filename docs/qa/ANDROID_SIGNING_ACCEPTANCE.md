# Android upload signing gate — Phase C / MOB-01

**Development source:** v0.95.42 Preview, following immutable published v0.95.41. No new release has been created by this signing PR.

**Scope:** Manual, release-owner-controlled AAB **upload-key** signing candidate only. This is **not** Play Console acceptance, installation/upgrade verification, or a change to existing APK/preview signing.

## Immutable identities

- Production `applicationId`: `com.ghostftp.android`. Do not rename it after publishing the app.
- Preview `applicationId`: `com.ghostftp.android.preview` — deliberately isolated; its debug signature is **never** the production upload identity.
- Google Play **app signing certificate** and developer **upload certificate** are different if Play App Signing is enabled. The new gate pins the **upload certificate**. The release owner must separately preserve and verify the Play app signing identity and package assignment.
- Play version codes must increase over versions previously published to **every relevant Play track**. The workflow validates the repository manifest, not remote Play track history.

## One-time owner setup (do not commit secret contents)

1. Inspect the existing Play Console app, package, app-signing certificate, **upload certificate**, rollout/track and version-code history. If an app already exists, reuse its accepted **upload** key; never generate a replacement casually.
2. Back up the existing upload keystore offline, restrict backup access and verify recovery under dual control. Store a Base64-encoded copy solely as the GitHub **environment secret** `ANDROID_UPLOAD_KEYSTORE_B64` within the protected `android-signing` environment.
3. Configure environment secrets `ANDROID_UPLOAD_STORE_PASSWORD`, `ANDROID_UPLOAD_KEY_PASSWORD`, `ANDROID_UPLOAD_KEY_ALIAS`. Configure environment variable `ANDROID_UPLOAD_CERT_SHA256` to the uppercase hex SHA-256 of the approved **upload** certificate. All six inputs are required; no defaults or debug-key fallback.
4. Protect the `android-signing` environment with required human reviewers and limit it to `main` / authorized operators. GitHub does **not** automatically enforce reviewer protection merely because the workflow refers to an environment.
5. Restrict repository Actions and artifact access. Treat signed AABs as sensitive release candidates even though no private keystore is uploaded as an artifact. Rotate and revoke compromised credentials according to Play's supported upload-key reset process.

## Invocation / evidence

Run workflow **Ghost FTP Android upload signing (manual)** from `main`; paste the exact current 40-character `main` SHA and confirmation `SIGN_WITH_EXISTING_UPLOAD_IDENTITY`.

The workflow checks both checked-out SHA and current remote main, then queries **all six required workflow families** (quality, protocol E2E, native packages, Android, macOS and Windows hardening) for **completed:success on that same exact SHA**, and refuses to sign when any is missing, pending, failed or stale. It then builds unsigned release APK/AAB, verifies package `com.ghostftp.android`, non-debuggable flag, `versionCode` and `versionName`, checks the keystore's certificate against the pinned SHA-256, signs with `jarsigner` without password CLI arguments, verifies AAB signature and emitted certificate, and retains the signed AAB plus SHA-256 and redacted evidence for seven days. Keystore bytes live only in a private temporary directory that is deleted on exit. Nothing is uploaded to Google Play automatically.

**Blockers before marking MOB-01 complete:** accepted AAB in the correct Play track, Google Play processing/result evidence, install previous approved version with test data, upgrade via Play to candidate, verify actual app-signing certificate, persisted user profiles and settings (excluding intentionally expired auth), and rollback/error recovery on devices. This workflow cannot execute those tests without the existing signing identity and Play Console owner access.

**Negative acceptance:** absent/incomplete secrets, wrong cert pin, nonmatching SHA, wrong package, debuggable APK, malformed/unsigned AAB, and signature verification failure must block distribution. The ordinary Android and release workflows remain unchanged and continue to label their outputs unsigned/preview.
