# Connection actions — click-by-click verification (0.95.10)

## Intent

A visible **Connect** must attempt an actual network session using entered credentials. **New connection** creates a reusable saved-site draft and must not connect or persist before an explicit **Save profile**. No duplicate prominent New Connection CTAs.

| Action | Windows / Linux | macOS | Android |
| --- | --- | --- | --- |
| Enter host, username, password, protocol/port | Inline Files Quick Connect | Inline Files temporary FTP only (FTPS/SFTP engine not ready) | Sites form; real FTP/FTPS/SFTP |
| Press Connect / Return | Immediate temporary backend session, no saved Site | Immediate real FTP session, no saved Site | Calls ConnectionController directly |
| New connection / Save profile | Sidebar saved Site editor; explicit save only | Unsaved draft; explicit save only | Separate Save new connection profile |
| Open a saved Site later | Sites inventory; backend connect | Sites inventory and FTP session editor | Select saved metadata, enter fresh password, tap Connect |
| Password persistence | Quick Connect never persists; saved Site uses existing vault | Temporary session never persists; opt-in Keychain in saved editor | Never included in saved metadata |

## Negative and privacy cases

1. Clicking Connect with a valid form never opens New Connection or saves a Site. A failed connection preserves masked input for a retry, but no credential is written to preferences/storage.
2. Invalid host, missing username or out-of-range port prevents a connection attempt. The error does not print the password.
3. New connection canceled without Save leaves no new persisted profile; creating a second profile must not create empty records. Test Connection does not write profile data.
4. Selecting an Android saved site clears the previous password. Android profile serialization stores only bounded nonsecret metadata and rejects duplicate names and over-limit inventory.
5. Windows/Linux test a real FTP, explicit FTPS and SFTP endpoint, including SSH host-key and FTPS TLS validation, reconnect/disconnect and saved Site selection. The macOS immediate form does **not** claim FTPS or SFTP support.
6. Verify click targets and keyboard/Android focus behavior in all target screen sizes; user-supplied reference art is not evidence of running-app parity.

**Status:** Source implementation candidate, not yet accepted by all installed-platform and physical-device checks.
