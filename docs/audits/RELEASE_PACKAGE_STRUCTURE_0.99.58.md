# Ghost FTP 0.99.58 — release artifact structural hardening audit

**Date:** 13 November 2026
**Publisher:** BRENDIGO
**Scope:** automated release-package preflight before SHA-256 manifest generation; no changes to FTP/FTPS/SFTP transport semantics.

## Source review and confirmed issue

The previous format checker accepted a release ZIP if its first four bytes matched a ZIP local-header signature, even if no central directory existed. The Windows and Linux executables similarly passed superficial magic/architecture checks despite incomplete PE section tables or ELF program-header metadata. The checker also performed a path-based lstat before a separate open, permitting a race in which an unexpected symlink replaced the inspected file.

## Implemented mitigations

1. Validate PE architecture-consistent optional header and bounded section table.
2. Validate ELF bitness, little-endian format, version, executable type, architecture and program-header bounds.
3. Validate ZIP EOCD and central-directory entry signature using a separate bounded tail read. No archive entry extraction is performed.
4. Open via no-follow flags where supported; compare lstat and fstat identity and size, with a post-inspection size/mtime check.
5. Exercise the entire release asset inventory using synthetic format fixtures; reject corrupt ZIP structures and symlinks in unit regression tests.
6. Preserve SHA-256 publication checks, immutable Preview tags and mandatory signing gates for future Stable editions.

## Residual risks and validation limits

Content can be malicious despite structurally valid headers. This is neither an executable antivirus scan nor cryptographic signer verification. ZIP members are not recursively decompressed or verified for full content consistency; archive integrity remains a separate acceptance responsibility. Windows filesystem symlink semantics differ from Linux, and TOCTOU resistance is limited to this process's file handles. Exact-source CI, installed-platform tests, authenticated cloud-provider E2E and third-party security review are still required.
