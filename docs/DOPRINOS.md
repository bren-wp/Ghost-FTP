# Doprinos projektu ByFTP

ByFTP je vlasnički/source-available softver tvrtke Brendigo. Objava izvornog koda ne daje opće pravo izmjene, redistribucije, rebrandinga ili izrade izvedenica.

## Potrebno je odobrenje

Issuei, prijave grešaka i prijedlozi funkcija su dobrodošli. Izmjene izvornog koda i pull requestovi dopušteni su samo kada ih Brendigo izričito zatraži ili odobri.

Otvaranje forka ili pull requesta samo po sebi ne daje pravo uporabe izvan ograničenih GitHub platformskih prava i [LICENSE](../LICENSE) uvjeta.

## Pravila za ovlaštene doprinose

- zadržati izvorni Win32 desktop; ne vraćati browser/localhost UI
- ne dodavati telemetriju, analitiku, oglašavanje, remote crash reporting ili automatske vanjske API pozive
- ne dodavati vanjski Go modul bez zasebne arhitekturne/sigurnosne provjere
- ne stavljati lozinke, passphrase ili privatni ključ u command-line argumente ili trajne logove
- očuvati SFTP host-key provjeru i lokalne path/reparse zaštite
- očuvati download staging, filesystem-root, state safe-open i cross-server retry granice
- preferirati tipizirane Go interfacee umjesto generičkog JSON dispatcha
- korisničke poruke i dokumentaciju pisati na hrvatskom
- novu detaljnu dokumentaciju dodati u `docs/README.md` i glavni README
- ne hardkodirati aktualni broj izdanja izvan `VERSION`, README/CHANGELOG prikaza koji audit izričito provjerava
- ne zaobilaziti `publish_release.ps1` ili `verify_bundle.py` u release workflowu
- ne uklanjati ByFTP/Brendigo identitet bez pisanog odobrenja
- ne uključivati kod treće strane bez potvrde licencne kompatibilnosti

## Prije ovlaštenog pull requesta

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

Na Windowsu pokrenite i `BUILD-WINDOWS.ps1`. Merge nije spreman ako bilo koji quality gate ili Windows produkcijski build nije zelen.