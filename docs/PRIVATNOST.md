# Privacy

ByFTP is designed as a telemetry-free file-transfer client.

## No tracking

The application does not include analytics, advertising SDKs, user tracking or external crash-reporting services. It does not send entered FTP/SFTP credentials to Brendigo or an unrelated API. Network connections are made to the server the user configured and to the operating-system tools required to perform that protocol.

## Local data

Profiles, settings, known SFTP fingerprints and transfer state are stored locally. Saved credentials are optional and protected by the platform-specific storage model. On Windows, credential persistence uses Windows protection. Linux/macOS intentionally avoid pretending to provide equivalent encrypted persistent credential storage where the implementation cannot guarantee it.

## Logs and process arguments

Passwords, private-key passphrases and private keys must not appear in command-line arguments or persistent logs. Diagnostic and CI fixtures must not contain customer credentials or real production server data.

## Localization and privacy

Changing UI language changes presentation only. It does not change network destinations, protocol semantics, credential handling or security decisions.

## Removing local data

Profiles can be removed from the application. Uninstall behavior is intentionally separated from user-data ownership so application removal does not silently destroy unrelated filesystem content.

For support, share only sanitized errors and reproduction information.