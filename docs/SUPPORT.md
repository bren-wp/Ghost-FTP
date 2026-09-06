# Ghost FTP support

Support for the current Ghost FTP **1.1.1 Stable** candidate and published stable versions should begin with the repository issue tracker for reproducible bugs and platform-specific problems:

https://github.com/bren-wp/Ghost-FTP/issues

Do not treat the 1.1.1 candidate as published until the official tag/Release/package read-back has succeeded. Published 1.1.0 and 1.0.0 remain historical stable releases.

## Before reporting

1. confirm the installed version and architecture;
2. confirm the file came from the official stable GitHub Release;
3. verify `SHA256.txt`;
4. on Windows, inspect `WINDOWS_AUTHENTICODE` in `BUILD-METADATA.txt`; verify Authenticode when it says `signed`, or record that the official file is explicitly `unsigned` when it says `unsigned`;
5. confirm the intended protocol (FTP, FTPS or SFTP), host and port;
6. for a fresh connection, remember that Ghost FTP defaults to explicit FTPS/21; select plain FTP only when the server intentionally requires legacy unencrypted FTP;
7. reproduce with the smallest safe example possible.

An unsigned Stable artifact is not automatically a corrupted artifact. Its integrity must still match the official tag/release location and SHA-256 manifest. Conversely, if metadata says `signed` and Windows signature verification fails, treat that as a release-integrity problem.

## Bug report information

Include:

- Ghost FTP version/tag;
- Windows or Linux and architecture;
- Setup/Portable/DEB package used;
- Windows signing state from `BUILD-METADATA.txt` when relevant;
- protocol and authentication type;
- whether the connection was Quick Connect or a saved profile;
- exact reproduction steps;
- expected vs actual behavior;
- privacy-safe diagnostic category/message;
- sanitized screenshot/log excerpt if relevant.

If the issue involves appearance, state whether Classic Light was fresh/default state or Dark was explicitly selected. If it involves a saved credential, describe whether the user consented to persistence without providing the credential itself.

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

Ghost FTP uses privacy-safe connection diagnostics. Include the displayed category/remediation instead of pasting raw credentials or a complete private command environment.

For FTPS failures, do not “fix” certificate/TLS errors by assuming plain FTP. Verify the server protocol/port and certificate configuration. The maintained client intentionally blocks silent secure-to-plain downgrade.

For SFTP host-key problems, provide only the public fingerprint if it is safe to disclose. Verify intentional server-key changes through an independent trusted channel.

## Profile and credential issues

The main Save Profile flow and Windows Site Manager require explicit consent before newly entered credentials are persisted. If a profile reconnects without a password, confirm whether credentials were intentionally saved and whether the current OS user/protection context can still decrypt them.

Changing the server/account/private-key identity can intentionally clear credentials that no longer belong to that identity. This is a security safeguard rather than automatic credential migration.

## Release/package problems

For GitHub Release issues include the exact filename and SHA-256 value. For Windows artifacts also include only the public signing status (`signed`/`unsigned`) and signature-verification result when applicable; never share certificate private material or Actions secrets.

For GitHub Packages issues include the semantic tag/digest for:

```text
ghcr.io/bren-wp/ghost-ftp
```

Remember that the GHCR object is a distribution bundle, not the normal desktop runtime container.

## Security-sensitive reports

Avoid posting exploit-ready private details or real secrets publicly. Follow the repository security policy in [Security](SECURITY.md) and provide only the information needed to reproduce the issue safely.

## Documentation/UI screenshot issues

Maintained screenshots under `docs/images/` must come from the authentic Windows screenshot workflow that launches the real x64 Portable executable. Report a mismatch between documentation and runtime rather than replacing a screenshot with a mockup or generated approximation.

Documentation corrections are welcome when current behavior, package names, security/privacy boundaries or release metadata are inaccurate. Historical release text should remain historical rather than being rewritten as current product behavior.
