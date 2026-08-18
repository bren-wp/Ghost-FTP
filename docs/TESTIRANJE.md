# ByFTP — testiranje

## Obavezne provjere

```text
python scripts/generate_brand_assets.py --check
python scripts/audit_croatian.py
python scripts/audit_version.py
python scripts/audit_docs.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
go test ./...
go test -race ./...
go vet ./...
```

Na Windowsu dodatno:

```powershell
.\BUILD-WINDOWS.ps1
```

## Što testovi pokrivaju

- validaciju FTP/FTPS/SFTP veza i korisničkog unosa
- profile i DPAPI migracijske granice
- settings normalizaciju i cache
- transfer queue, parallelism, pause/resume, cancel/retry i auto-retry
- connection generation i cross-server retry blokadu
- worker panic containment
- završni status nakon kasnog cancel/disconnect racea
- active-operation/session-close lifecycle: disconnect mora otkazati poziv, pričekati njegov `release()` i tek zatim zatvoriti adapter
- bounded disconnect timeout koji vraća kontrolu pozivatelju bez prisilnog zatvaranja aktivnog adaptera
- deferred cleanup nakon timeouta i reconnect blokadu dok stara sesija još postoji
- ponovljeni disconnect nad istim close-stateom bez dvostrukog `Session.Close()`
- idempotentni `Operation` release
- path traversal, Windows rezervirane nazive, symlink/junction/reparse zaštite
- SFTP private-key regular-file/symlink/reparse granicu
- download staging `Lstat`/regular-file/reparse provjeru
- filesystem-root zabranu brisanja, uključujući Windows drive i UNC root
- Unix fallback listing s običnim nazivom koji sadrži ` -> ` i stvarnim symlink zapisom
- sigurno parsiranje prevelike veličine listinga bez `int64` wraparounda
- DOS/Windows-style listing veličinu datoteke
- rekurzivne upload/download planove i rollback
- installer payload integritet i upgrade rollback
- event stream fallback, deep-copy izolaciju i velike queue burstove
- velike lokalne i udaljene popise od 50.000 stavki kroz zajedničko stabilno sortiranje
- hrvatske UI/release/GitHub površine
- korisničke poruke za mrežni timeout, session-closing i deferred disconnect cleanup
- konzistentnost `VERSION` izvora bez ručnog semver drifta
- lokalne Markdown poveznice i potpunost dokumentacijskog indeksa
- determinističku generaciju PNG/ICO resursa
- Windows ZIP manifest, putanje, duplikate i SHA-256 nakon stvarnog pakiranja
- release tag/commit i postojeći asset digest fail-closed ugovor

## Stabilnosne regresije udaljene sesije

`internal/remote/manager_test.go` deterministički pokreće disconnect dok je remote `Operation` još aktivan. Osnovna regresija zahtijeva da context bude otkazan, ali adapter ne smije biti zatvoren prije `release()`.

Timeout regresija namjerno drži operaciju aktivnom dulje od disconnect deadlinea. Očekuje `ErrDisconnectTimeout`, potvrđuje da adapter i dalje nije zatvoren, da novi `Operation` nije dopušten i da `Connect` vraća `ErrSessionClosing`. Nakon `release()` test zahtijeva završni deferred `Session.Close()` i čišćenje close-statea.

Zasebna regresija pokreće drugi `Disconnect` dok isti deferred close još traje i potvrđuje da oba poziva koriste isti lifecycle umjesto dvostrukog zatvaranja adaptera. Idempotentni release dodatno se poziva dvaput kako WaitGroup brojač ne bi mogao postati negativan.

`internal/usererror/message_test.go` čuva hrvatske, korisnički razumljive poruke za session-closing i disconnect-cleanup stanje, odvojeno od običnog mrežnog timeouta.

## Ostale regresije 2.14.x

`internal/remote/listing_regression_test.go` čuva parser i sorting rubne slučajeve. Test s 50.000 stavki namjerno koristi isti javni limit udaljenog prikaza kako optimizacija velikih direktorija ne bi ostala samo mikro-test.

`internal/remote/private_key_validation_test.go` potvrđuje regularnu datoteku i symlink odbijanje; Windows produkcijski build i `audit_security.py` dodatno čuvaju reparse-point granu.

## Python release regresije

`test_release_tools.py` izrađuje privremene ZIP fixturee standardnom bibliotekom. Pozitivan fixture mora proći, a traversal, nepopisana datoteka i integritetska pogreška moraju biti odbijeni. Test ne treba mrežu ni vanjske Python pakete.

## CI

GitHub CI ima dva glavna joba:

1. Linux kvalitetu: Go unit/race/vet, privacy/security/version/docs/release/croatian/asset audite, Python release regresije i release metadata.
2. Windows produkcijski build: isti `BUILD-WINDOWS.ps1` put koji koristi službeno izdanje.

Merge se ne smatra spremnim dok oba joba nisu zelena.

## Release metadata i bundle

CI čita `VERSION`, generira bilješke iz odgovarajućeg `CHANGELOG.md` odjeljka i zahtijeva da izlaz nije prazan. Release workflow nakon kompresije dodatno pokreće `verify_bundle.py` nad konačnim Windows ZIP-om.

`BUNDLE-SHA256.txt` ne hashira samoga sebe, ali mora točno pokrivati svaku drugu datoteku u bundleu. Nepopisana, nestala, duplicirana, traversal ili hash-nepodudarna stavka zaustavlja izdanje.

## Konzistentnost verzije

`scripts/audit_version.py` provjerava da `VERSION` ostaje jedini produkcijski izvor broja verzije, da razvojni build ne glumi staro izdanje, da README/CHANGELOG/build skripte ostaju usklađeni te da workflow i bug predložak ne vrate ručno hardkodiranu aktualnu semantičku verziju.
