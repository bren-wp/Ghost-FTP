# Update center bootstrap

This is a manual Preview download center pinned to the published v0.99.89 release, not an automatic release resolver. The publish log for 45914252814 confirms the absence of signed updater artifacts. Both /latest.json and /updates/latest.json deliberately return HTTP 204 with no body and no-store. Never replace that response with a placeholder manifest.

Build the root ZIP from the repository root with:
    python3 updates/scripts/pack-web-root.py GhostFTP-Update-Web-Root-bootstrap-v0.99.89.zip

Create the update.ghostftp.com subdomain in your hosting panel, configure its DNS and valid TLS certificate, and extract the ZIP directly into the document root. PHP 8, Apache 2.4, mod_rewrite, mod_headers and AllowOverride for Options, FileInfo, Indexes and AuthConfig are required. Do not copy this operator document or the tests into the public directory.

Verify both endpoint aliases return 204 and no-store. Verify the EN/HR page, exact GitHub download URLs, CSP and directory listing protection. Keep the existing ghostftp.com/updates/latest.json endpoint unchanged until its live compatibility is verified. HSTS is a hosting-level step after verifying HTTPS.

Next gate: verified release-driven deployment or a bounded release resolver, exact-version signed manifest and signature checks, installed-client upgrade acceptance and old endpoint compatibility. Do not enable automatic installs or change the embedded verification key before these gates pass.
