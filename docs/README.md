# ByFTP dokumentacijski centar

**Od prvog spajanja do sigurnog svakodnevnog rada — sve na jednom mjestu.**

ByFTP dokumentacija organizirana je tako da prvo odgovori na korisničko pitanje **“kako da se spojim i napravim posao?”**, a tek zatim ulazi u tehničke detalje.

## Želim se odmah spojiti na hosting

Za najbrži početak otvorite [ByFTP za shared hosting](SHARED-HOSTING.md), a za instalaciju i platformske detalje [Instalaciju i prvi spoj](INSTALACIJA.md).

Za tipični shared hosting trebat će vam:

- FTP host koji je dao hosting provider;
- korisničko ime, uključujući oblik `korisnik@domena` ako ga hosting koristi;
- FTP lozinka;
- protokol i port — najčešće FTP ili explicit FTPS na portu 21.

Ako se ne možete spojiti, prije promjene postavki otvorite [Podršku i rješavanje problema](PODRSKA.md).

## Želim znati zašto je ByFTP siguran izbor

- [Sigurnost](SIGURNOST.md) — transfer staging, SFTP host-key zaštita, FTPS, no-follow datotečne granice i fail-closed ponašanje.
- [Privatnost](PRIVATNOST.md) — što ByFTP ne prikuplja, kako se tretiraju vjerodajnice i zašto nema aplikacijske telemetrije.
- [Testiranje i kvaliteta](TESTIRANJE.md) — što CI provjerava prije produkcijskog izdanja.
- [Provjera izdanja](PROVJERA-IZDANJA.md) — kako se potvrđuju paket, verzija i checksum.

## Želim razumjeti tehnologiju

- [Arhitektura](ARHITEKTURA.md) — zašto FTP/FTPS/SFTP dijele jedan engine i kako su slojevi odvojeni.
- [Plan razvoja](PLAN-RAZVOJA.md) — završene cjeline i smjer budućih poboljšanja.
- [Potpisivanje](POTPISIVANJE.md) — Authenticode, Apple Developer ID i stvarni status potpisivanja.

## Želim sudjelovati u razvoju

- [Doprinos projektu](DOPRINOS.md) — pravila, kvaliteta i sigurnosni standardi.
- [Izdavanje na GitHubu](IZDAVANJE-NA-GITHUBU.md) — kontrolirani release proces.
- [Obavijesti trećih strana](OBAVIJESTI-TRECIH-STRANA.md) — transparentnost o alatima i komponentama.

## Što ByFTP želi biti

ByFTP nije zamišljen kao “još jedan FTP prozor”. Cilj je spojiti tri stvari koje korisnicima hostinga stvarno trebaju:

1. **brzo spajanje** bez nepotrebne konfiguracije;
2. **jasno upravljanje datotekama** bez skrivanja što se događa;
3. **stroža zaštita transfera i vjerodajnica** nego kod običnog copy/overwrite modela.

Ako koristite shared hosting, WordPress, statičnu web stranicu, PHP aplikaciju ili vlastite servere, dokumentacija je organizirana tako da vas vodi od pristupnih podataka do stvarnog rada nad datotekama.

---

**Najbrži put:** [Shared hosting → prvi FTP spoj → `public_html`](SHARED-HOSTING.md)
