# ByFTP 2.12.0 — privatnost

ByFTP nema telemetriju, analytics, oglašivačke SDK-ove, vanjske API-je, crash-reporting prema internetu, tracking ni automatski update-check.

## Mrežna politika

- ByFTP uspostavlja FTP, FTPS ili SFTP vezu samo prema poslužitelju i portu koje korisnik unese ili lokalno spremi u profil.
- Hostname se razrješava preko standardnog Windows DNS mehanizma; ByFTP nema vlastiti DNS-over-HTTPS servis.
- Proxy varijable okruženja se ne nasljeđuju. curl dobiva izričito `proxy=""` i `noproxy="*"`.
- OpenSSH ne čita korisnički SSH config, ProxyJump, ProxyCommand, SSH agent, tuđi AskPass, PKCS#11/security-key provider ni local-command hookove. Ako privatni ključ nije izričito odabran, ByFTP isključuje i zadane SSH identity datoteke.
- Windows release koristi samo sistemski curl/OpenSSH iz stvarnog System32 direktorija.
- Za FTPS se provjeravaju certifikat i naziv poslužitelja, dok je Schannel online revocation dohvat isključen kako ByFTP ne bi automatski kontaktirao CA/CRL infrastrukturu trećih strana.
- Windows Error Reporting se za ByFTP proces onemogućuje pri startupu, pa ByFTP crash ne pokreće WER slanje memorijskog izvještaja.

## Lokalni podaci

- Normalni runtime nema trajni activity/error log. Stari `byftp.log` i `byftp.log.1` ostaci iz ranijih verzija brišu se pri pokretanju.
- Spremljeni profil je na Windowsu u cijelosti zaštićen DPAPI-jem: naziv profila, host, port, korisničko ime, lokalne/udaljene putanje, fingerprint i credential blobovi nisu zapisani kao čitljiv plaintext profil.
- Legacy `profiles.json` migrira se u DPAPI-zaštićeni `profiles.secure.json`, nakon čega se stara plaintext datoteka uklanja.
- Spremanje novih lozinki/passphrasea u profil traži izričitu potvrdu korisnika.
- Aktivne FTP/SFTP credential vrijednosti drže se DPAPI-zaštićeno između pojedinačnih protocol poziva. Spremljeni DPAPI blob ne dešifrira se u connection manageru; otključava se tek neposredno prije konkretne mrežne uporabe. Kratkoživući plaintext byte buffer briše se nakon predaje sistemskom alatu.
- Lozinka/passphrase se ne stavljaju u command-line argumente. FTP credential config ide curlu preko stdin-a; SFTP AskPass dobiva samo DPAPI-zaštićeni blob kroz kratkoživući child environment i ne stvara credential datoteku na disku.
- SFTP `known_hosts` i session config postoje samo tijekom aktivne sesije i brišu se pri Disconnectu/startup cleanupu. OpenSSH command line koristi nasumični alias umjesto hosta/usernamea/key patha. Host-key sesija se ograničava na točno onaj ključ/algoritam čiji fingerprint korisnik potvrdi. Privremena potvrda novog host ključa čuva eventualnu nespremljenu vjerodajnicu samo kao kratkoživući DPAPI blob vezan uz taj točan server/fingerprint i odbacuje je nakon prihvaćanja, odbijanja ili novog pokušaja povezivanja.
- Odabrani privatni ključ ne dodaje se u Windows Recent listu iz ByFTP file pickera.
- Settings datoteka sadrži samo lokalne preference aplikacije, bez vjerodajnica.
- Uspješno povezivanje i spremanje profila prazne vidljiva password/passphrase polja.
- Uninstaller po izboru korisnika može obrisati lokalne ByFTP profile i postavke; zadano ih čuva radi moguće ponovne instalacije.

## Što ByFTP ne može kontrolirati

Windows, DNS resolver, antivirus/EDR, firewall, backup/sync softver ili sam odredišni FTP/FTPS/SFTP poslužitelj mogu imati vlastita pravila zapisivanja ili mrežne politike. ByFTP im ne šalje telemetriju niti koristi njihove vanjske API-je; odredišni poslužitelj nužno prima podatke potrebne za vezu i prijenos koji korisnik zatraži. Brisanje lokalne datoteke nije isto što i forenzičko sigurno brisanje fizičkih blokova SSD-a niti može ukloniti kopije koje je drugi backup/sync softver već napravio izvan ByFTP-a.

Release build pokreće `scripts/audit_privacy.py`; build pada ako se uvedu runtime web/API importi, fiksni HTTP(S) endpointi, telemetry vendor markeri, vanjski Go moduli, trajno runtime logiranje ili ponovno JSON-serijaliziranje osjetljivih Connect/Profile operacija.

## Dodatne 2.9 lokalne zaštite

- server-controlled download naziv prolazi istu Windows sigurnu validaciju bez obzira koristi li korisnik gumb Preuzmi ili dvoklik
- lokalno rekurzivno brisanje i opcionalno uklanjanje korisničkih podataka pri uninstallu ne prolaze kroz symlink/reparse/junction odredište
- osjetljiva Win32 polja imaju ograničen broj znakova prije nego aplikacija pročita njihov sadržaj, čime se izbjegava nepotrebno velika memorijska alokacija kroz paste/input
- transfer rezervacija stare sesije ne može se commitati u novu sesiju nakon reconnecta

## Dodatne 2.11 zaštite

- cijeli desktop→engine put sada je tipiziran; nema generičkog JSON command dispatchera koji bi nepotrebno kopirao korisničke putanje ili druge podatke unutar procesa
- svaki transfer dobiva samo interni hash identiteta aktivnog protokola/hosta/porta/korisnika/fingerprinta; hash ostaje isključivo u memoriji i služi za blokiranje ponavljanja starog transfera prema drugom odredištu
- settings cache sadrži samo lokalne preference aplikacije; nema hosta, korisničkog imena, lozinke, ključa ni udaljenih putanja
- automatski retry i batch retry nikada ne mijenjaju mrežno odredište; Retry starog posla prema drugoj vezi je odbijen
- current-directory DLL search je uklonjen za ByFTP proces kako lokalna radna mapa ne bi bila izvor neočekivanog DLL učitavanja
