# Privacy

ByFTP contains no advertising, analytics SDK, application telemetry or mandatory cloud account. Connection details are used only to contact the server selected by the user.

## Desktop

Saved profiles/settings remain in the local application-data directory. Windows saved credentials use DPAPI. Production Go builds require Go telemetry to be disabled. External helper processes receive a minimized environment.

## Android

The Android client does not persist connection passwords, passphrases or SSH private-key secrets. It uses document-provider URIs for user-selected upload/download files and requests no broad device-storage permission.

Starting with 1.3.0, Android can remember the last successful **non-secret** connection metadata in app-private preferences: protocol, host, port, username and, for SFTP, the expected host-key fingerprint. Password/passphrase fields are structurally absent from that preset. Loaded values are validated again before use.

Cloud-backup and device-transfer rules exclude root, file, database, shared-preference and external app-data domains, so the local preset is not exported through Android backup/device transfer. Generic cleartext traffic is disabled for platform-aware networking. No Firebase Analytics, advertising SDK, project-controlled runtime API or ByFTP backend service is included.

The password field is cleared after every connection attempt and at Activity teardown. Active/pending clients and picker state are cleaned during lifecycle teardown. Release APK packaging does not change this privacy model.

## iOS

The native iOS client also has no ByFTP backend, analytics/advertising SDK or fixed runtime HTTP(S) endpoint.

Connection **secrets** remain session-only. The password field is cleared after each connection attempt, the FTP transport clears its own password copy after authentication and the app disconnects when it enters the background.

Starting with 1.3.0, iOS can remember only protocol, host, port and username as a non-secret `ConnectionPreset` in Keychain using `kSecAttrAccessibleWhenUnlockedThisDeviceOnly`. The preset has no password field, does not use `UserDefaults`, is device-bound by the selected Keychain accessibility class and is revalidated before restoration. The credential-bearing connection config is not retained merely to complete preset persistence.

Uploads use the system document picker and security-scoped access only for files explicitly selected by the user. Downloads are written to a private temporary directory for the system share/save workflow and can be explicitly cleared; the session store also removes the previous temporary download before creating a new one.

The iOS bundle contains no global App Transport Security bypass. Network traffic is limited to the endpoint entered by the user; PASV response hosts are not trusted as alternative destinations.

Unsigned IPA/app ZIP generation is a packaging/signing property only and does not introduce telemetry, cloud storage or secret persistence.

## Repository privacy gate

Version 1.6.0 extends release hygiene across every tracked repository file. The repository-wide integrity audit is entirely local to the checked-out Git tree: it enumerates tracked paths and file contents only. It does not upload source, credentials, filenames or audit results to a ByFTP service and does not add a runtime endpoint or telemetry path.

## Project communication

ByFTP does not send usage events to the repository, support endpoint or a project-operated service. Static repository/support URLs used as product metadata do not trigger automatic network requests.
