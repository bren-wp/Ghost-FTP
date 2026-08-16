# ByFTP build i verifikacijski alati

Ova mapa sadrži pomoćne alate produkcijskog pipelinea.

- `BUILD-LOCAL.sh` — lokalna offline Windows cross-build provjera
- `generate_brand_assets.py` — reproducibilno generiranje i provjera PNG/ICO resursa
- `audit_croatian.py` — provjera hrvatskih korisničkih/GitHub/release površina
- `audit_privacy.py` — privacy/network-policy provjera
- `make_payload.py` — izrada i integritet instalacijskog payloada
- `pe_resources.py` — PE ikona i VERSIONINFO resursi
- `verify_release.py` — PE i sigurnosna verifikacija binarija
- `release_notes.py` — hrvatske bilješke iz točnog CHANGELOG odjeljka

Puni proces izdavanja dokumentiran je u [`docs/IZDAVANJE-NA-GITHUBU.md`](../docs/IZDAVANJE-NA-GITHUBU.md), a kontrolna lista u [`docs/PROVJERA-IZDANJA.md`](../docs/PROVJERA-IZDANJA.md).
