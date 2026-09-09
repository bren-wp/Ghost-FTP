# Ghost FTP support

Support for Ghost FTP **1.1.8 Stable** starts with the official product website at **https://ghostftp.com**. Reproducible software bugs and platform-specific technical problems may also be reported through the repository issue tracker:

https://github.com/bren-wp/Ghost-FTP/issues

Active support, package and runtime metadata use only the **Ghost FTP** product identity. Historical Stable releases remain immutable release identities.

## Before reporting

1. confirm installed version and architecture;
2. confirm the file came from the official Stable GitHub Release;
3. verify `SHA256.txt`;
4. on Windows, **inspect `WINDOWS_AUTHENTICODE` in `BUILD-METADATA.txt`**; verify Authenticode when it says `signed`, or record that the **official file is explicitly `unsigned`** when it says `unsigned`;
5. confirm intended protocol (FTP, FTPS or SFTP), host and port;
6. remember that a fresh connection defaults to explicit FTPS/21 and plain FTP is an explicit legacy compatibility choice;
7. reproduce with the smallest safe example possible.

An unsigned Stable artifact is not automatically corrupted. Its integrity must still match the official tag/release location and SHA-256 manifest. Conversely, **if metadata says `signed` and Windows signature verification fails**, treat that as a release-integrity problem.

## Bug report information

Include:

- Ghost FTP version/tag;
- Windows or Linux and architecture;
- Setup/Portable/DEB/tar.gz package used;
- Windows signing state from `BUILD-METADATA.txt` when relevant;
- protocol and authentication type;
- whether the connection was Quick Connect or a saved profile;
- exact reproduction steps;
- expected vs actual behavior;
- privacy-safe diagnostic category/message;
- sanitized screenshot/log excerpt if relevant.

For 1.1.8 Windows display issues, include monitor resolution/work area, scale/DPI, multi-monitor arrangement and whether the issue occurred at startup or while moving between monitors. Do not include unrelated private screen content.

For Windows installation/uninstall issues, identify Setup vs Portable, whether a foreign/modified shortcut already existed and the high-level operation that failed. Do not bypass ownership checks or manually delete unrelated same-name artifacts as a diagnostic step.

For Linux SFTP credential-helper issues, include distro/architecture and whether Ghost FTP/OpenSSH tools are system-installed or portable, but do not paste passwords, passphrases, private-key contents or complete child-process environments.

## Do not publish secrets

Never put these in public issues:

- passwords;
- private keys;
- private-key passphrases;
- protected saved-profile payloads;
- server confidential files;
- CI/signing secrets;
- recovery credentials/tokens.

Use synthetic values for reproductions.

## Connection failures

Ghost FTP uses privacy-safe connection diagnostics. Include the displayed category/remediation instead of raw `curl`/OpenSSH stderr, credentials or a complete private command environment.

For FTPS failures, do not bypass certificate/TLS errors by assuming plain FTP. Verify server protocol/port and certificate configuration. The client intentionally blocks silent secure-to-plain downgrade.

For SFTP host-key problems, provide only the public fingerprint if safe. Intentional server-key changes should be verified through an independent trusted channel.

On Linux, do not work around trusted transport/AskPass provenance failures by placing private copies of `ssh`, `sftp`, `ssh-keyscan` or `curl` earlier in `PATH`. The provenance boundary is a security property.

## Profile and credential issues

Main Save Profile and Windows Site Manager require explicit consent before newly entered credentials are persisted. If a profile reconnects without a password, confirm whether credentials were intentionally saved and whether the current OS user/protection context can decrypt them.

Changing server/account/private-key identity can intentionally clear credentials that no longer belong to that identity. This is a security safeguard rather than credential migration.

## Release/package problems

For GitHub Release issues include exact filename and SHA-256 value. For Windows artifacts include only public signing status (`signed`/`unsigned`) and verification result; never share certificate private material or Actions secrets.

For GitHub Packages issues include semantic tag/digest for `ghcr.io/bren-wp/ghost-ftp`. The GHCR object is a distribution bundle, not a runtime container.

For supplemental Debian/Ubuntu/Fedora/Portable CI packages, make clear that they are CI evidence rather than canonical 1.1.8 public assets.

## Security-sensitive reports

Avoid posting exploit-ready private details or real secrets publicly. Follow [Security](SECURITY.md) and provide only information needed to reproduce safely.

## Documentation/UI screenshot issues

Maintained release evidence under `docs/images/` must come from the authentic Windows screenshot workflow that launches the real x64 Portable executable. Report runtime/documentation mismatch rather than replacing evidence with a mockup.

Documentation corrections are welcome when current behavior, package names, security/privacy boundaries or release metadata are inaccurate. Historical release text should remain historical rather than being rewritten as current behavior.
