# ByFTP 2.12.0 testiranje

Obavezne release provjere:

1. `python scripts/audit_privacy.py`
2. `go test ./...`
3. `go test -race ./...`
4. `go vet ./...`
5. Windows x64 `go vet ./...`
6. Windows x64 cross-build aplikacije, setupa i uninstallera
7. `scripts/verify_release.py` za Portable + Setup + Uninstaller: PE32+ GUI, VERSIONINFO, manifest i mitigacije
8. ZIP integrity i SHA-256
9. Windows 10/11 runtime smoke-test: FTP, FTPS explicit/implicit, SFTP password i private key, upload/download file/folder, reconnect, delete/rename/permissions, multi-select/batch queue cancel/retry, auto-retry i installer upgrade/rollback

Posebni regresijski testovi pokrivaju AskPass bez disk credential artefakta, SFTP metadata-minimizirani command line i host-key algorithm binding, remote double-click download path validaciju, no-follow symlink/reparse delete, stale connection-generation batch rezervacije, lokalni tree rollback, UI input limite, installer manifest/hash tampering, canonical uninstaller path, session disconnect cancellation, privacy env sanitization, SFTP direct routing bez implicitnih default identity ključeva, jednokratni/matching pending-trust credential, config traversal/symlink zaštitu, Windows reserved filenames, AskPass invocation, atomic transfer rollback i transfer queue lifecycle.

2.11 regresije dodatno pokrivaju settings hot-path cache, automatski transfer retry, atomski batch Cancel/Retry, cross-server Retry blokadu, FTP MLSD unsupported/transient klasifikaciju i potpuno uklanjanje generičkog in-process JSON dispatchera.
