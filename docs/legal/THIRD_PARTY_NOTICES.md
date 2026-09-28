# Third-Party Notices

Ghost FTP includes and links third-party open-source libraries and operating-system components. Each component remains subject to its own licence terms; the Ghost FTP commercial licence does not replace those obligations.

## Canonical dependency sources

The exact dependency set for a release is determined from the source and lock/build files at that release tag, including:

- `ghostftp-desktop/package-lock.json` for JavaScript/TypeScript dependencies;
- `ghostftp-desktop/src-tauri/Cargo.lock` for Rust/Tauri dependencies;
- `android/app/build.gradle.kts` plus Gradle resolution for Android dependencies;
- Go module/build metadata for support tooling where applicable.

Current direct Android protocol dependencies include Apache Commons Net for FTP/FTPS and the maintained mwiede JSch distribution for SSH/SFTP.

Release/legal review must use the exact dependency graph of the source SHA being published rather than copying a stale manually curated version list.

## Trademarks and interoperability names

Names such as FileZilla, PuTTY, OpenSSH, Microsoft Windows, Linux distribution names, Android and other third-party products may appear only where required for interoperability/import/documentation. Those names and trademarks belong to their respective owners and do not imply endorsement, sponsorship or ownership by Ghost FTP, Brendigo LTD or Brendigo, obrt za programiranje.

## Distribution rule

Any third-party licence or notice that requires reproduction with binaries must be included in the corresponding release/documentation package before stable/FINAL distribution.
