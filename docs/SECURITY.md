# Security

ByFTP keeps transport and filesystem checks fail-closed.

FTP over TLS validates certificates. SFTP pins and verifies the host key. Uploads use a stable local snapshot before network transfer, remote writes use temporary staging and destination revalidation, and overwrite paths use backup/rollback logic. Local recursive operations guard against symlinks, junctions and reparse-point traversal.

Credentials must never be logged. Windows saved profile secrets are protected by DPAPI. External processes receive a minimized environment and bounded output handling.

Report security vulnerabilities through the repository Security policy rather than publishing working secrets or exploit details in a public issue.
