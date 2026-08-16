# ByFTP dokumentacija

Ova mapa je jedino mjesto za detaljnu projektnu dokumentaciju. Root repozitorija namjerno ostaje fokusiran na glavni pregled, licencu, povijest promjena, verziju i build entrypointe.

## Sadržaj

- [ARHITEKTURA.md](ARHITEKTURA.md) — moduli, procesne granice, transfer/state lifecycle i release arhitektura
- [SIGURNOST.md](SIGURNOST.md) — model prijetnji, vjerodajnice, filesystem i release zaštite
- [PRIVATNOST.md](PRIVATNOST.md) — mrežna i lokalna politika privatnosti
- [TESTIRANJE.md](TESTIRANJE.md) — testni slojevi, auditi, CI i bundle regresije
- [PROVJERA-IZDANJA.md](PROVJERA-IZDANJA.md) — obavezna fail-closed release kontrolna lista
- [POTPISIVANJE.md](POTPISIVANJE.md) — Authenticode smjernice i granica signing tajni
- [PLAN-RAZVOJA.md](PLAN-RAZVOJA.md) — završene sigurnosne cjeline i sljedeći prioriteti
- [PODRSKA.md](PODRSKA.md) — prijava problema i privatnost podataka podrške
- [DOPRINOS.md](DOPRINOS.md) — pravila ovlaštenih doprinosa i quality gateovi
- [OBAVIJESTI-TRECIH-STRANA.md](OBAVIJESTI-TRECIH-STRANA.md) — sistemske komponente trećih strana
- [IZDAVANJE-NA-GITHUBU.md](IZDAVANJE-NA-GITHUBU.md) — GitHub CI/release proces, rerun i asset-integrity pravila
- [slike/](slike/) — službeni dokumentacijski vizualni resursi

## Automatizirana konzistentnost

`scripts/audit_docs.py` provjerava lokalne Markdown/HTML poveznice, zahtijeva da svaki detaljni dokument bude indeksiran ovdje i u glavnom README-u te zabranjuje ponovno uvođenje verzioniranih naslova poput `ByFTP 2.x.y — ...` koji bi zastarjeli pri sljedećem izdanju.

Povijest izdanja ostaje u root datoteci [`CHANGELOG.md`](../CHANGELOG.md), a licenca u [`LICENSE`](../LICENSE), jer su to standardne repozitorijske entrypoint datoteke.