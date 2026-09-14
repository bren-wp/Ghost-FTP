# Ghost FTP website and Web FTP

Ghost FTP **0.0.6** includes two maintained web surfaces under `web/`:

- `web/` — the self-contained product website intended for `https://ghostftp.com`;
- `web/ftp/` — a browser UI for FTP, explicit FTPS and fingerprint-pinned SFTP operations.

These are source/deployment surfaces and **do not add files to the 13 platform artifacts / 16 public files GitHub Release contract**.

## Product website

The marketing site is plain HTML/CSS/JavaScript with local assets only. It contains product positioning, authentic Windows/Linux/Android 0.0.6 screenshots, protocol/security explanations, canonical download links, privacy/security/legal pages and a Web FTP entry point.

The site intentionally avoids third-party fonts, analytics, advertising, remote JavaScript and remote CSS. `web/assets/images/` contains byte-identical copies of selected verified 0.0.6 runtime screenshots so the `web/` directory can be deployed on its own.

The supplied Apache `.htaccess` adds restrictive browser security headers and HTTPS redirection. Equivalent headers must be configured explicitly when deploying under nginx, Caddy or another server.

## Why Web FTP is server-assisted

Browser JavaScript cannot directly open arbitrary raw FTP, FTPS or SFTP TCP sockets. Therefore the `/ftp` frontend sends one requested operation over HTTPS to `web/ftp/api.php`. PHP opens the remote protocol connection, executes the operation and closes it.

The maintained design is deliberately truthful: this is an **ephemeral server-assisted transport**, not “direct browser FTP.” The web server is inside the trust boundary for that request.

## Credential lifecycle

The supplied web client:

- does not create Ghost FTP accounts;
- does not contain a user/profile database;
- does not use `localStorage`, IndexedDB or cookies to persist connection credentials;
- keeps connection data in JavaScript page memory;
- sends connection data only with the requested HTTPS operation;
- does not intentionally write passwords, listings or transfer contents to application logs;
- clears password/fingerprint fields on explicit disconnect/page unload where browser lifecycle allows.

Hosting infrastructure can still observe runtime request data and may create access/error logs. Configure reverse proxies and application servers so request bodies are never logged.

## SSRF boundary

`Security::publicTarget()` resolves the requested host and rejects private, loopback, link-local and reserved IP ranges by default. FTP/FTPS cURL requests pin the validated resolved address through `CURLOPT_RESOLVE` while preserving the original hostname for certificate verification.

A trusted private deployment may explicitly set:

```text
GHOSTFTP_WEB_ALLOW_PRIVATE=1
```

Do not enable that setting on an untrusted public deployment.

## FTP and explicit FTPS

FTP/FTPS use PHP cURL. Explicit FTPS uses `ftp://` plus `CURLUSESSL_ALL`, requires peer verification and requires hostname verification. Secure failure is never retried as plaintext FTP.

## SFTP

SFTP requires the PHP `ssh2` extension and an expected OpenSSH-style SHA-256 host-key fingerprint:

```text
SHA256:base64FingerprintWithoutPadding
```

The implementation reads the server host key with `ssh2_fingerprint`, compares it with `hash_equals`, and only then performs password authentication. Missing, malformed, unknown or mismatching fingerprints fail closed.

## File operations

The maintained browser UI exposes:

- remote listing and directory navigation;
- explicit local file selection;
- upload and download;
- create directory;
- rename/move within the configured root;
- file/directory delete;
- CHMOD where supported;
- bounded text Remote Edit.

Browser security prevents arbitrary local filesystem enumeration, so the “Local” pane represents files explicitly selected by the user rather than pretending to be a native filesystem browser.

## Deployment requirements

- PHP 8.2+;
- PHP cURL for FTP/FTPS;
- PHP ssh2 for SFTP;
- HTTPS;
- outbound network access to intended server ports;
- a web server configured not to expose `config.php` or Markdown source files.

Default source limits are defined in `web/ftp/config.php`: 256 MiB upload, 512 MiB download and 2 MiB Remote Edit. Hosting-level `upload_max_filesize`, `post_max_size`, request timeout and proxy limits may be lower and must be configured consistently.

## Verification

`scripts/check_web_contract.py` validates required source, no retired branding, no persistent browser credential storage, SSRF markers, explicit FTPS peer/hostname verification, SFTP host-key-before-auth ordering, no inline CSS/JS in product HTML, and PHP syntax when PHP CLI is available.

`scripts/test_web_contract.py` makes those boundaries part of the normal Python regression suite.

## Implementation prompt

For a complete build/audit brief covering the maintained browser client, use [`prompts/GHOST-FTP-WEB-APP-PROMPT.md`](prompts/GHOST-FTP-WEB-APP-PROMPT.md). The prompt preserves the same server-assisted transport, SSRF, FTPS certificate/hostname verification, SFTP fingerprint-before-authentication and no-persistent-credential boundaries documented here.
