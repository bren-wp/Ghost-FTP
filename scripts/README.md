# ByFTP build i verifikacijski alati

Ova mapa sadrži pomoćne alate produkcijskog pipelinea. Svi alati koriste standardne mogućnosti Go/Python/PowerShell okruženja i ne dodaju runtime ovisnosti ByFTP aplikaciji.

- `BUILD-LOCAL.sh` — lokalna offline Windows cross-build provjera
- `generate_brand_assets.py` — reproducibilno generiranje i provjera PNG/ICO resursa
- `audit_croatian.py` — provjera hrvatskih korisničkih/GitHub/release površina
- `audit_version.py` — provjera `VERSION` kao jedinog produkcijskog izvora broja verzije
- `audit_docs.py` — lokalne dokumentacijske poveznice, indeks i verzijski neutralni naslovi
- `audit_security.py` — ključne filesystem/transfer/state sigurnosne invarijante i regresijski testovi
- `audit_privacy.py` — privacy/network-policy provjera
- `audit_release.py` — statički ugovor release workflowa, bundlea i idempotentnog publishera
- `test_release_tools.py` — stdlib regresijski testovi release ZIP verifikacije
- `make_payload.py` — izrada i integritet instalacijskog payloada
- `pe_resources.py` — PE ikona i VERSIONINFO resursi
- `verify_release.py` — PE i sigurnosna verifikacija binarija
- `verify_bundle.py` — provjera konačnog Windows ZIP-a, putanja i `BUNDLE-SHA256.txt`
- `release_notes.py` — hrvatske bilješke iz točnog CHANGELOG odjeljka
- `publish_release.ps1` — idempotentna GitHub Release objava s tag/commit i asset SHA-256 provjerom

Puni proces izdavanja dokumentiran je u [`docs/IZDAVANJE-NA-GITHUBU.md`](../docs/IZDAVANJE-NA-GITHUBU.md), kontrolna lista u [`docs/PROVJERA-IZDANJA.md`](../docs/PROVJERA-IZDANJA.md), a testni slojevi u [`docs/TESTIRANJE.md`](../docs/TESTIRANJE.md).