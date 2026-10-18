# Ghost FTP Android — production Play upload signing and identity (MOB-01)

**Status: guarded signing workflow implemented; production identity / Play Console acceptance NOT VERIFIED.**

## Immutable identity boundary

- Production application ID is `com.ghostftp.android` (no suffix).
- `com.ghostftp.android.preview` is a distinct, non-debuggable preview product whose ephemeral CI debug certificate **must never** become the Play production key.
- Google Play App Signing uses an **app-signing key** managed according to the Play Console enrollment decision and an **upload key** used for submitting Android App Bundles. The upload key is not automatically the app-signing key. Document both SHA-256 identities privately during enrollment; never derive or generate a replacement silently.
- If `com.ghostftp.android` already exists on Google Play, obtain the **existing expected upload-certificate SHA-256** and a matching authorized keystore from the current owner. Never generate a new random keystore to sign an upgrade. If this is a first-ever Play listing, explicitly choose/approve the signing strategy before recording the pinned certificate.
- Normal PR/main CI intentionally builds unsigned release APK/AAB and separately signed preview APK, unchanged from earlier releases. The protected manual signing workflow is distinct from publication and cannot itself upload to Google Play.

## Configure protected GitHub workflow

Create a GitHub Actions **environment named `play-upload-signing`**, restricted to `main` and protected by required reviewers. Configure these *environment secrets* (not repository plaintext files or PR-level variables):

| Name | Meaning |
| --- | --- |
| `GHOSTFTP_UPLOAD_KEYSTORE_B64` | Base64 encoding of the authorized existing Java PKCS12/JKS upload keystore |
| `GHOSTFTP_PLAY_STORE_PASSWORD` | Password of that keystore |
| `GHOSTFTP_PLAY_KEY_ALIAS` | Alias of the designated upload signing key |
| `GHOSTFTP_PLAY_KEY_PASSWORD` | Password of the upload key |

Configure the nonsecret but **reviewed and pinned** environment variable `GHOSTFTP_PLAY_UPLOAD_CERT_SHA256` with the expected 64-hex-digit SHA-256 upload certificate fingerprint (colons acceptable). Confirm it out-of-band from the existing Play Console upload certificate. Never use a test or debug fingerprint.

Invoke **Ghost FTP Android Play upload signing** manually from `main` only *after* the exact SHA passes all six required workflow families: quality, protocol E2E, native Windows/Linux builds, Android, macOS and Windows hardening. The workflow refuses a stale main HEAD, incomplete/missing secrets, missing or empty keystore, debug product identity, wrong versionCode/versionName, unverified APK/AAB signatures or mismatched fingerprint. There is no automatic key rotation, fallback or unauthenticated download of signing material.

The job writes the keystore as `0600` to the private ephemeral runner directory and removes it at the end. Gradle receives secrets through scoped process environment variables, not command-line arguments, Gradle properties in Git or artifacts. The signed AAB and public exact-SHA provenance are retained as an Actions artifact for **one day**. The protected GitHub environment approves access to secrets, not necessarily access to signed artifacts; use repository access controls appropriately.

## Negative / reproducible QA

The ordinary Android CI executes `gradle -p android --offline help` with a *single* signing input and then with all four inputs but an absent keystore. Both cases **must fail** with specific error text. It also verifies that the canonical ordinary release output remains unsigned and the isolated preview APK stays separately signed. Neither test uses the real signing keystore.

In the manual workflow:
1. Verify the exact SHA matches the current `main` branch and all six CI families returned success for that SHA.
2. Produce signed `assembleRelease` and `bundleRelease` using the *same* upload key.
3. Verify APK signature via `apksigner`, the pinned SHA-256 certificate, `com.ghostftp.android`, versionCode/versionName and nondebuggable state.
4. Independently verify AAB JAR signature via `jarsigner` and its certificate SHA-256 with `keytool`, matching the APK and reviewed pin.
5. Check `SHA256SUMS.txt` and `PROVENANCE.txt` against the exact build/PR SHA. No keys/passwords/tokens should appear in artifact or build logs.
6. Authorized maintainer uploads the AAB to the intended **Play Console internal/closed test track** and records Play's actual acceptance/rejection, app signing/upload certs and `com.ghostftp.android` identity.
7. On a device with the prior **same-package, same-signing-identity** production build, perform `adb install -r` upgrade and confirm saved user profiles and application data survive. A preview-to-production upgrade is **not** the same identity.

## Unmet release criteria

No production keystore or fingerprint was provided to this change. There is **no proof of a signed real AAB**, Google Play Console acceptance, distribution through Play, key rotation, device upgrade preserving data or installation on old users' devices. The workflow is manual by design, and must stay unused until signing identity is externally verified.

For security, do not add keystore files, `*.jks`, `*.keystore`, PKCS#12, base64 key blobs or Gradle signing passwords to the repository or GitHub Release assets.
