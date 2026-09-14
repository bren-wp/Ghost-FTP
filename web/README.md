# Ghost FTP website

This directory is the self-contained public website intended for `ghostftp.com`. Upload the contents of `web/` to the site document root. It does not depend on repository-relative assets outside this directory.

- `index.html` — product marketing page
- `download.html` — canonical 0.0.6 download matrix
- `security.html`, `privacy.html`, `legal.html` — trust and legal pages
- `ftp/` — server-assisted Web FTP client
- `assets/` — local CSS, JavaScript, logo and authentic 0.0.6 runtime captures

No external font, analytics, ad, JavaScript or CSS dependency is required. The supplied Apache `.htaccess` adds restrictive security headers and HTTPS redirection.

The runtime screenshots under `assets/images/` are byte-identical repository copies of the verified 0.0.6 evidence set; they are not generated marketing mockups.
