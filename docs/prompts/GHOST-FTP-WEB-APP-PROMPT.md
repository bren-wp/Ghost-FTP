# Ghost FTP Web App implementation prompt

Use this prompt to build, audit or harden the maintained Ghost FTP browser client under `web/ftp/`. The repository is authoritative for release identity, supported protocols, security boundaries, branding, screenshots and licensing. Do not replace those contracts with assumptions from a prior branch or conversation.

## Product identity and deployment

Build **Ghost FTP Web** as the browser-accessible companion at `https://ghostftp.com/ftp/` while keeping the public product name **Ghost FTP**. The complete website source lives under `web/`; the web app lives under `web/ftp/` and must be deployable as part of that tree on ordinary HTTPS PHP hosting with the documented extensions.

The web app should visually follow the Windows reference application: recognizable two-pane file-management structure, restrained workstation styling, Ghost FTP dark palette by default, the same spacing/control language, clear status and transfer feedback, and a responsive mobile transformation. It is not a fake desktop screenshot and must remain usable as a browser application.

## Non-negotiable transport truth

Browser JavaScript cannot open arbitrary raw FTP, FTPS or SFTP TCP sockets. Never claim direct browser-to-server transport when the implementation is server assisted.

The maintained model is:

1. the user enters connection values in the browser;
2. the page sends one requested operation to `web/ftp/api.php` over HTTPS;
3. the trusted Ghost FTP web host opens the selected FTP/FTPS/SFTP connection;
4. the requested operation completes or fails;
5. that transport is closed and request-scoped sensitive values are discarded.

Do not add a registration database, synchronized account, cloud drive, analytics backend, credential vault, hidden transfer history service or product telemetry. Do not persist FTP/SFTP passwords, private-key passphrases or connection profiles in cookies, localStorage, IndexedDB or a server-side product database. Page-memory convenience state is acceptable only for the current browser session and must never be represented as secure durable storage.

## Required protocol boundaries

### FTP

Plain FTP is an explicit unencrypted compatibility mode. UI copy must identify that fact. Do not silently choose it after a secure transport failure.

### Explicit FTPS

Use explicit FTPS with TLS certificate and hostname verification enabled. No `CURLOPT_SSL_VERIFYPEER=false`, `CURLOPT_SSL_VERIFYHOST=0`, trust-all certificate path or silent fallback to FTP. Resolve and validate the host before connection, reject disallowed/reserved/private targets by default, then pin the selected validated address for the request to limit DNS-rebinding/TOCTOU exposure.

### SFTP

SFTP requires an expected `SHA256:...` server host-key fingerprint supplied by the user or an explicitly trusted configuration. Verify the connected server key **before password/key authentication**. Unknown, malformed or mismatched identity fails closed. Never add first-contact auto-accept, `StrictHostKeyChecking=no`, trust-all behavior or a decorative fingerprint field that is not enforced.

## SSRF and network boundary

Treat every host and port as untrusted input. The default public deployment must reject loopback, link-local, multicast, unspecified, documentation/test-only, carrier-grade/private/reserved ranges and other non-public destinations for both IPv4 and IPv6. If private-host access is deliberately enabled for a controlled self-hosted deployment, that must be an explicit server configuration opt-in and documented as widening the trust boundary.

Validate the host syntax, port range, protocol-specific limits and DNS result set before opening a socket. Do not follow redirects to a new network identity. Do not let user-controlled URLs select arbitrary HTTP endpoints.

## File/path safety

Remote paths and names are untrusted. Normalize separators, reject NUL/control characters and traversal components, preserve the protocol root semantics and prevent a child operation from escaping the requested remote root. Never concatenate an unchecked user name directly into a filesystem or protocol command.

Uploads must use server/PHP temporary-file primitives safely and enforce configured size limits. Downloads and Remote Edit reads must be bounded; abort when limits are exceeded. Remote Edit must reject binary/NUL content, enforce a practical text-size ceiling and avoid writing partially validated content.

## Functional surface

Implement real working paths for the supported controls; do not ship decorative actions:

- connect / disconnect state;
- FTP / explicit FTPS / SFTP selector with accurate fields;
- remote listing and navigation;
- refresh;
- upload;
- download;
- create directory;
- rename;
- delete file/directory;
- CHMOD when supported by the selected protocol/server;
- bounded Remote Edit read/save;
- clear errors and completion feedback;
- keyboard-accessible dialogs/actions;
- mobile-safe responsive navigation and controls.

If a desktop feature such as a durable queue, synchronized directory comparison or protected saved profile store is not implemented in the web app, do not add a fake button merely for visual parity. Document the difference.

## UI and theme

Default dark values should remain visually aligned with the maintained app palette:

- background `#0B0F17`;
- primary surface `#121824`;
- secondary/list surface `#161D2A`;
- border `#2C3648`;
- primary text `#F2F5FA`;
- muted text `#97A3B8`;
- accent `#5B7CFA` / `#7A98FF`;
- success `#4AD79B`;
- warning `#F2BA55`;
- danger `#FF6878`.

The light theme must be intentionally soft/off-white rather than stark white: use warm/cool gray-tinted surfaces, strong readable text and equivalent hierarchy. Both themes must meet practical contrast requirements.

Use external CSS/JS files, no third-party CDN fonts/icons/scripts, no unnecessary framework and no inline application JavaScript. Use semantic HTML, visible keyboard focus and `prefers-reduced-motion`. On small screens stack connection controls and file panels without horizontal overflow; touch targets must remain usable.

## Privacy and logging

The supplied app has no analytics, ads, fingerprinting or behavioral tracking. Never send credentials or remote filenames to an analytics endpoint. Application errors returned to the browser must be conservative and must not echo passwords, passphrases, private keys or raw server authentication responses.

The hosting operator is inside the web-app trust boundary because it receives request-scoped credentials and opens the remote connection. State this clearly in documentation and privacy copy. Do not describe the web app as zero-knowledge, peer-to-peer or direct-from-browser.

Configure PHP/runtime logging so secrets are not intentionally logged. Production deployment should disable display_errors and avoid request-body logging at reverse-proxy/WAF/application layers where credentials may be present.

## Security headers and deployment

Serve only over HTTPS. Maintain a restrictive CSP compatible with local assets and required upload/download behavior. Include at least `X-Content-Type-Options: nosniff`, a privacy-conscious `Referrer-Policy`, restrictive `Permissions-Policy` and frame-ancestor protection. HSTS is a deployment-level decision and should be enabled only after the HTTPS/subdomain scope is known to be correct.

`web/` must remain self-contained: local logo/icon/CSS/JS/runtime screenshots, stable relative paths, no build-time CDN dependency. The `web/README.md` and `web/ftp/README.md` must document PHP requirements, required extensions, HTTPS, upload/body size alignment, temporary-directory requirements, private-host opt-in and production logging considerations.

## Release/download relationship

The web app itself is a maintained source/deployment surface, not an additional GitHub 0.0.6 binary release asset. The canonical 0.0.6 GitHub release remains the repository-defined 13 platform artifacts plus 3 metadata files unless the release contract is deliberately migrated.

Website download links may point to canonical `ghostftp-v0.0.6` asset URLs, but the public copy must not say the release is published until the protected release transaction actually succeeds. Do not expose internal per-architecture Windows payloads or CI-only APKs.

## Tests and acceptance criteria

Before calling the web app production-ready:

- run PHP syntax lint for every tracked `web/**/*.php` file;
- run the web contract audit/regression tests;
- verify JS syntax and no console errors;
- verify no credential persistence APIs (`localStorage`, IndexedDB, cookies) are introduced;
- verify private/reserved hosts fail closed by default;
- verify explicit FTPS keeps peer/hostname verification enabled;
- verify SFTP fingerprint comparison occurs before authentication;
- test malformed host, port, path, filename, permission and fingerprint inputs;
- test upload/download/read size bounds;
- test connect/list/upload/download/mkdir/rename/delete/CHMOD/Remote Edit against disposable synthetic servers where CI infrastructure permits;
- test 320/360/390/412/768/1024/1440 widths, keyboard navigation and reduced motion;
- verify CSP/security headers and no third-party runtime requests;
- verify product/download/legal links and canonical ghostftp.com routes;
- verify documentation states the server-assisted trust boundary plainly.

A green syntax check is not sufficient evidence that remote operations work. Where CI cannot run disposable protocol servers, keep that limitation explicit and add the strongest deterministic contract tests available.

## Definition of done

The finished `web/ftp` is acceptable only when it is a real, deployable Ghost FTP browser client with implemented actions, accurate transport/security copy, fail-closed identity verification, bounded file operations, no hidden persistence/telemetry, responsive parity-oriented UI, maintained documentation and exact-head CI evidence. Never trade these boundaries for a visually impressive demo.
