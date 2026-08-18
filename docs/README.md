# ByFTP dokumentacija

Ova mapa je jedino mjesto za detaljnu projektnu dokumentaciju. Root repozitorija ostaje fokusiran na glavni pregled, licencu, povijest promjena, verziju i kanonske build ulaze.

## Sadržaj

- [INSTALACIJA.md](INSTALACIJA.md) — Windows x64/x86, Linux DEB, macOS Universal PKG, preduvjeti i SHA-256 provjera
- [ARHITEKTURA.md](ARHITEKTURA.md) — zajednički Engine, Windows GUI, Linux/macOS terminalno sučelje, remote/transfer lifecycle i release arhitektura
- [SIGURNOST.md](SIGURNOST.md) — model prijetnji, SFTP/AskPass, vjerodajnice, filesystem i release zaštite
- [PRIVATNOST.md](PRIVATNOST.md) — mrežna i lokalna politika privatnosti na svim platformama
- [TESTIRANJE.md](TESTIRANJE.md) — unit/race/vet, sigurnosni auditi i platform-specific CI buildovi
- [PROVJERA-IZDANJA.md](PROVJERA-IZDANJA.md) — obavezna fail-closed kontrolna lista za 13 javnih release asseta
- [POTPISIVANJE.md](POTPISIVANJE.md) — Windows Authenticode i macOS Developer ID granice
- [PLAN-RAZVOJA.md](PLAN-RAZVOJA.md) — završene cjeline i sljedeći prioriteti, uključujući Unix SFTP AskPass broker
- [PODRSKA.md](PODRSKA.md) — podržane platforme, auth matrica i sigurna prijava problema
- [DOPRINOS.md](DOPRINOS.md) — pravila ovlaštenih doprinosa i multi-platform quality gateovi
- [OBAVIJESTI-TRECIH-STRANA.md](OBAVIJESTI-TRECIH-STRANA.md) — curl/OpenSSH i sistemske komponente trećih strana
- [IZDAVANJE-NA-GITHUBU.md](IZDAVANJE-NA-GITHUBU.md) — GitHub CI/release proces, platform artefakti, rerun i asset-integrity pravila
- [slike/](slike/) — službeni dokumentacijski vizualni resursi

## Automatizirana konzistentnost

`scripts/audit_docs.py` provjerava lokalne Markdown/HTML poveznice, zahtijeva da svaki detaljni dokument bude indeksiran ovdje i u glavnom README-u te sprječava zastarjele verzionirane naslove za dokumente koji trebaju ostati aktualni između izdanja.

`scripts/audit_security.py` i `scripts/audit_release.py` dodatno provjeravaju da dokumentirani connect/release model nije odvojen od stvarnog koda i workflowa.

Povijest izdanja ostaje u root datoteci [`CHANGELOG.md`](../CHANGELOG.md), a licenca u [`LICENSE`](../LICENSE).
