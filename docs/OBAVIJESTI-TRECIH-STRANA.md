# Third-party notices

ByFTP production Go code is intentionally built without external Go module dependencies. CI sets the Go proxy/checksum database to off so an unnoticed module download cannot become part of the release build.

The application relies on operating-system or bundled command-line tooling for protocols: curl for FTP/FTPS and OpenSSH for SFTP. Those tools remain subject to their own licenses and platform distribution terms.

GitHub Actions used for CI/release are pinned to immutable commit SHAs. Packaging environments include Microsoft, Linux distribution and Apple tooling supplied by the corresponding hosted runner images.

Brand names such as Windows, macOS, GitHub, curl and OpenSSH belong to their respective owners. Their mention describes compatibility and does not imply endorsement.