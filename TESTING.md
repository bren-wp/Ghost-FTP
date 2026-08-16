# ByFTP 2.13.0 testiranje

Obavezne release provjere:

1. `python scripts/audit_privacy.py`
2. `go test ./...`
3. `go test -race ./...`
4. `go vet ./...`
5. `python scripts/release_notes.py --version <VERSION> --output <temp>/RELEASE-NOTES.txt`
6. Windows x64 `go vet ./...`
7. Windows x64 cross-build aplikacije, setupa i uninstallera
8. `scripts/verify_release.py` za Portable + Setup + Uninstaller: PE32+ GUI, VERSIONINFO, manifest i mitigacije
9. Windows/Source ZIP integrity, bundle/global SHA-256 i `BUILD-METADATA.txt` provenance
10. Windows 10/11 runtime smoke-test: FTP, FTPS explicit/implicit, SFTP password i private key, upload/download file/folder, reconnect, delete/rename/permissions, multi-select/batch queue cancel/retry, auto-retry i installer upgrade/rollback

Posebni regresijski testovi pokrivaju AskPass bez disk credential artefakta, SFTP metadata-minimizirani command line i host-key algorithm binding, remote double-click download path validaciju, no-follow symlink/reparse delete, stale connection-generation batch rezervacije, lokalni tree rollback, UI input limite, installer manifest/hash tampering, canonical uninstaller path, session disconnect cancellation, privacy env sanitization, SFTP direct routing bez implicitnih default identity ključeva, jednokratni/matching pending-trust credential, config traversal/symlink zaštitu, Windows reserved filenames, AskPass invocation, atomic transfer rollback i transfer queue lifecycle.

2.11 regresije dodatno pokrivaju settings hot-path cache, automatski transfer retry, atomski batch Cancel/Retry, cross-server Retry blokadu, FTP MLSD unsupported/transient klasifikaciju i potpuno uklanjanje generičkog in-process JSON dispatchera.

2.12/2.12.1 regresije pokrivaju transient-only retry, connection timeout, Skip Existing, LocalRoot runtime revalidaciju, redirected ByFTP-owned direktorije i kasnu symlink/junction zamjenu recursive-upload root direktorija.

## 2.13 regresije

- `internal/config/store_open_test.go` deterministički zamjenjuje state putanju nakon `Lstat`, a prije `Open`, jednom symlinkom i jednom drugim regularnim objektom; oba slučaja moraju biti odbijena.
- `internal/itemlist/sort_test.go` provjerava stabilno case-insensitive directory-first sortiranje i 50.000-stavki workload.
- `internal/desktop/transfer_state_test.go` provjerava state reset, update/append semantiku i batch od 20.000 poslova + 1.000 eventa.
- CI iz kanonskog `VERSION` generira release notes; nedostajući ili prazan odgovarajući CHANGELOG odjeljak ruši provjeru prije mergea.
- Windows release workflow mora proizvesti verzionirani Actions artefakt `byftp-<version>-complete-release` s točnim EXE/ZIP putanjama te `BUILD-METADATA.txt`.

## Release prihvatni kriteriji

Release se smatra spremnim tek kada su oba CI joba zelena, Windows production build i PE/security verifikacija prođu, a release workflow uspješno objavi GitHub Release, potpuni Actions artefakt i `ByFTP.Windows` paket. Nepotpisani CI/release binariji nisu zamjena za konačni Authenticode-signed javni build kada je produkcijski certifikat dostupan.
