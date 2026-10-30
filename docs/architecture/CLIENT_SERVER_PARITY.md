> **Historical architecture note only (v0.99.22):** This file describes the discontinued standalone Ghost FTP Server. Beginning with v0.99.24, the Server, Demo and standalone CLI have been removed from active source and current release packages. All paths, workflows and packaging requirements below are archival, **not current instructions**. For current supported products consult [client-only packaging](../distribution/PLATFORM_PACKAGES.md), [security policy](../../SECURITY.md) and [release notes](../releases/0.99.24.md).

# Ghost FTP Client + Server — cross-platform parity and synchronization contract

**Development scope:** v0.99.22 Preview, 29 October 2026. This record is evidence of source alignment and real feature availability, **not** a claim of fully approved 1:1 visual parity or production readiness.

## Design: one canonical identity

The canonical design source is [premium-ui-tokens.json](../branding/premium-ui-tokens.json). All platforms use Ghost FTP brand colors `#070E1A` canvas, `#0D192B` panel, `#38ABFF` accent, `#EAF6FF` text, `#26DC8F` success and `#FF7284` danger. Desktop/server workspaces target 214 px navigation, 43 px rows and 52 px chrome at the supplied 1290×852 reference viewport. Android uses its own 480×960 reference and physical-dp/insets, without forcing a cropped desktop UI onto a phone.

Server admin at `ghost-server/admin-assets/` embeds **exactly the same ghost icon SVG bytes** as `ghostftp-desktop/branding/ghostftp-symbol.svg`. It has active Overview, Sessions, Security, Limits and Diagnostics navigation and responsive layouts. It must never expose fictional users, traffic graphs, permissions, file writes, fake transfer progress or editable server settings without a working server API. All displayed metrics come from its own real authenticated server response.

Source-level regressions are checked by `.github/scripts/check-server-premium-parity.test.mjs` and the existing `.github/scripts/check-premium-ui-tokens.mjs`. Neither is a screenshot-diff engine. **Pixel-perfect acceptance remains pending** for all real Windows, Linux, Android, macOS and Server screens, including 1290×852 reference dimensions, dark/light themes, keyboard/screen-reader interactions, 200% DPI and mobile breakpoints.

## Synchronization: one actual server, not duplicated fake stores

| Area | Contract | Current acceptance |
| --- | --- | --- |
| FTP/FTPS file operations | Clients connect to the real shared Ghost FTP Server or supported third-party FTPS server; remote filesystem is the source of truth | Existing FTPS interoperability and safety CI; broader real-platform matrix pending |
| Server monitoring | Local `GET /api/v1/status` is the source of truth for live connections, uptime, TLS, upload/destructive policy, upload ceiling, server version and contract version | Implemented read-only/local-only; authenticated HTTP tests |
| Old local tooling | `GET /api/status` aliases the same live handler | Retained, tested |
| Build versions | `version.json` controls all client packages; Linux and Windows Server embed identical version in CLI and monitoring API through `-ldflags` | CI build version check and source contract gate; production upgrade parity pending |
| Secrets | No admin tokens in localStorage, cookies, URL, remote payloads or logs; no credentials in release assets | Static source tests and unit tests; independent penetration testing pending |
| User/session administration | Only existing runtime metrics are readable; no remote write API, user RBAC or full audit history | **Not yet supported** |
| Cross-device settings and sync job definitions | Must use explicitly authenticated, consented, end-to-end secure interchange with conflict resolution, not an unencrypted shared cache | **Not yet supported** |
| Windows/Linux Server distribution | Portable/setup and standalone/tarball/DEB/RPM included with each new release | Mandatory in draft v0.99.22 release contract; publication pending |
| Android/macOS Server daemon | No native Android/macOS *server daemon* is implemented; Android/macOS remain clients | **Not yet supported**; do not fabricate packages |

## Security boundary

The administration endpoint is bound to loopback, requires a distinct strong bearer token, validates the HTTP Host header against the local configured endpoint and sends `Cache-Control: no-store`, restrictive CSP and anti-framing headers. Opening port 2821 remotely or embedding the token in a client installer would violate the contract. Shared status does not authorize remote administration.

Private server credential and certificate management remains the job of the local OS. The UI explicitly shows disabled or unavailable controls instead of exposing unsupported, unsafe operations. Automatic refresh clears stale telemetry when network calls fail, and locks when the token is rejected or the page closes.

## Next acceptance milestones

1. Green exact-SHA Go/Windows NSIS/Linux DEB/RPM and shared design API CI, no bypasses.
2. Capture **actual** Server shell and native Windows/Linux/Android/macOS client screens, systematically compare reference colors, placements, interactions, scales and responsive layouts.
3. Build a real, authenticated service-management API (accounts, groups, roles, ACL, quotas, audit, process lifecycle and certificate rotation) with authorization tests and secure IPC/local access; then create UI screens for implemented operations.
4. Design an explicit cross-device pairing/import/export mechanism to synchronize saved sites and sync jobs without leaking passwords/private keys and without silently overwriting user data.
5. Verify client ↔ server protocol E2E across supported OSes and real devices. Only then consider production signing, notarization and final approval.

**Release discipline:** A green layout-source test is not visual sign-off. A locally functioning admin screen is not a production server or a remote administration feature.
