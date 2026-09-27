# Ghost FTP Update Service — Deployment

This document is for the person or automation that manages **ghostftp.com**. It is intentionally technical and must not be copied into user-facing application screens.

## Recommended production layout

Only one mutable machine-consumed file is required on the Ghost FTP website:

```text
public_html/
└── updates/
    └── latest.json
```

The Windows/Linux package URLs inside that file point to immutable, versioned GitHub Release assets. This keeps the Ghost FTP update-service host small and avoids duplicating large binaries on shared hosting.

The release workflow produces `GhostFTP-v<version>-Web-Update.zip` with exactly:

```text
updates/
└── latest.json
```

Extract that archive into the update-service hosting root so the resulting public URL is:

`https://ghostftp.com/updates/latest.json`

## What must NOT be uploaded

Never upload any of these to the website:

- Tauri private signing key;
- signing-key password;
- GitHub token;
- CI environment files;
- repository secrets;
- temporary build directories.

Signature text inside `latest.json` and public `.sig` release assets are safe to publish.

## Deployment order

1. Confirm the new GitHub Release exists and its tag points to the intended source SHA.
2. Confirm Windows Setup, Linux AppImage and their signature assets exist.
3. Verify release checksums/digests.
4. Download `GhostFTP-v<version>-Web-Update.zip`.
5. Extract it outside the live web root.
6. From a repository checkout, run `node updates/scripts/verify-manifest.mjs <path>/updates/latest.json --expected-version=<version>`.
7. Upload the new file as a temporary name, for example `/updates/latest.json.new`.
8. Atomically rename/replace it to `/updates/latest.json`.
9. Request the public URL over HTTPS and confirm HTTP 200.
10. Open Ghost FTP → Updates and run a manual check on Windows and Linux.

Do not update `latest.json` before the immutable release assets are available; clients could otherwise be offered an incomplete release.

## HTTP requirements

For `/updates/latest.json`:

- HTTPS only;
- HTTP 200 when present;
- `Content-Type: application/json; charset=utf-8`;
- `Cache-Control: no-cache, no-store, must-revalidate`;
- `X-Content-Type-Options: nosniff`;
- directory listing disabled for `/updates/`.

The desktop updater is a native client, so browser CORS is not required for the update check.

## Shared hosting / Apache

Use `updates/web/.htaccess.example` as the basis for `public_html/updates/.htaccess`.

Recommended deployed directory:

```text
public_html/updates/
├── .htaccess
└── latest.json
```

## Nginx

Use `updates/web/nginx.conf.example` inside the Ghost FTP server configuration.

## Rollback

Do **not** retarget or rewrite an existing Git tag.

If the latest release must stop being offered:

1. restore the previously known-good `latest.json`;
2. verify its package URLs/signatures still point to immutable published assets;
3. investigate/fix the new release in a new version.

This changes only what the update service offers; it does not delete installed files or rewrite release history.
