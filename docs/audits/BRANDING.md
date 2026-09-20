# Branding Audit

Scope: full source tree, runtime, installer, website, release-facing docs and generated binaries/source strings where inspectable.

- Product name, public host, bundle identifier and deep-link scheme are Ghost FTP / ghostftp.com / `com.ghostftp.desktop` / `ghostftp://` in the native source.
- Full-tree text scan found no legacy product identifier from the previous codebase in executable/source code.
- Product-facing ghostftp-runtime/site/setup pages contain no `example.com`, demo user, Production Server, Staging Server, Design Assets or Cloud Server seed profile.
- Site Manager initializes from actual stored profiles and is empty for a new profile store.
- Third-party names retained in native source are interoperability/import references (for example FileZilla/PuTTY) rather than product branding.
- Official public navigation in audited product-facing code targets ghostftp.com.

Status: **source branding audit passed for the scanned tree; binary GUI visual verification still requires target-OS execution.**
