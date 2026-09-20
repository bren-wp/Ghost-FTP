# Ghost FTP 2.1.1 RC7

Ghost FTP 2.1.1 RC7 is a release-candidate build of the Ghost FTP desktop file-transfer client for Windows and Linux.

## Highlights

- Full Ghost FTP branding with the approved Electric Blue / Deep Navy / Slate Blue / Ice White visual system.
- Frameless custom Ghost FTP titlebar in the native Tauri application.
- Main File Manager, Site Manager, New Connection, Preferences, Transfer Center, File Properties and About/Updates surfaces aligned to the approved reference layouts.
- Responsive/adaptive rules around the 1290×852 desktop reference surface.
- FTP, FTPS and SFTP native protocol/session/transfer implementation in the Tauri/Rust source.
- Local/remote file operations, transfer queue, retry/reconnect, checksum and permissions support.
- English plus 13 additional interface languages.
- Privacy-first configuration with no analytics or telemetry enabled by default.
- Hardened local state/profile persistence and credential handling.
- Windows setup flow without a separate uninstall.exe.
- Source audit, branding guard and native Windows/Linux CI.

## Release status

RC7 is a **pre-release**, not the final production designation. The source audit passes in GitHub Actions. Native Windows/Linux artifacts are attached only when the corresponding native Tauri build completes successfully.

The project release gate still requires the final Windows/Linux titlebar screenshot acceptance test, end-to-end FTP/FTPS/SFTP testing against authorised test servers, installer upgrade/uninstall acceptance testing and final pixel/responsive comparison before a future build is labelled FINAL.

## Official project

Product: Ghost FTP  
Publisher: Brendigo LTD / Brendigo, obrt za programiranje  
Website: https://ghostftp.com/

See BUILD_STATUS.md, QA.md, TITLEBAR_QA.md, PIXEL_QA.md and SECURITY_AUDIT.md for the detailed status.
