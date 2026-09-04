# Ghost FTP Web

This directory contains the shared-hosting web/PWA edition of **Ghost FTP**.

> The source directory is still named `GhostFTP WEB` as a migration/compatibility detail. The public product, UI and release package are Ghost FTP.

## Requirements

- supported PHP runtime with the extensions required by the configured FTP/SFTP adapters;
- HTTPS for production use;
- writable protected application storage;
- web-server configuration that prevents direct access to sensitive application/storage files.

## Deployment

Use the web package from GitHub Releases:

```text
Ghost-FTP-X.Y.Z-Web.zip
```

Extract it into the intended web application directory, confirm the server honors the included access-control configuration, open the application over HTTPS and complete the setup flow.

Do not expose storage/configuration files through the web server. Do not reuse a compromised application secret; restore from a trusted backup or rotate credentials according to the recovery procedure.

## Security controls

The web client includes defensive controls for:

- CSRF validation;
- strict session/cookie handling;
- HTTPS/HSTS behavior where HTTPS is active;
- Content Security Policy and related security headers;
- login/request rate limiting;
- remote-host and path validation;
- bounded temporary/download operations;
- cleanup of stale temporary files;
- protected JSON storage and recovery behavior;
- `noindex`/`nofollow` directives so the private file client is not intended for search indexing.

Plain FTP is unencrypted. Prefer FTPS or SFTP whenever the server supports it.

## PWA

The manifest identifies the application as **Ghost FTP** and supports standalone installation behavior on compatible browsers/mobile devices. The web edition remains a private application surface rather than an SEO landing page.

## Internal compatibility names

PHP namespaces, helper functions, constants and some storage/session keys may retain the legacy `GhostFTP`/`GhostFTP` prefix so existing deployments, sessions and stored configuration can migrate safely. These are implementation identifiers only. New user-visible strings and downloadable packages must use **Ghost FTP**.

## Version

The web `VERSION` file must match the repository root `VERSION`. Ghost FTP begins at `1.0.0` and follows the repository Semantic Versioning policy.
