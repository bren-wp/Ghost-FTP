# ByFTP 2.12.0 — sigurnost

## Vjerodajnice

- spremljene lozinke, zaporke privatnog ključa i aktivni session credential blobovi koriste Windows DPAPI
- password/passphrase ne idu u command-line argumente
- AskPass ne sprema credential blob na disk; DPAPI-zaštićeni blobovi predaju se samo kratkoživućem child procesu kroz sanitizirani environment
- AskPass način rada prihvaća samo vlastiti ByFTP executable, valjani jednokratni token i očekivani System32 OpenSSH parent proces
- spremljeni credential blob prolazi kroz connection manager bez ranog dešifriranja; plaintext nastaje tek neposredno prije curl/AskPass uporabe
- privremeni SFTP trust credential je DPAPI-zaštićen, vremenski ograničen, vezan uz točan identitet veze i jednokratan

## Mrežni procesi

- Windows build određuje stvarni System32 preko `GetSystemDirectoryW` i koristi samo taj `curl.exe`/OpenSSH; nema WINDIR/PATH fallbacka na Windowsu
- curl ne učitava `.curlrc`, ne nasljeđuje proxy/TLS override env i dobiva izričiti no-proxy
- SFTP koristi ByFTP-managed SSH config i blokira ProxyCommand, ProxyJump, agent, PKCS#11/security-key provider, KnownHostsCommand, PermitLocalCommand i forwarding
- bez eksplicitno odabranog ključa SFTP postavlja `IdentitiesOnly=yes` i `IdentityFile=none`, pa ne pokušava zadane privatne ključeve korisnika
- host key se provjerava preko zasebnog session-temporary ByFTP known_hosts zapisa i sesija se ograničava na točno potvrđeni host-key algoritam
- host/user/private-key metadata nisu na OpenSSH command lineu; nalaze se u nasumičnoj kratkoživućoj session config datoteci koja se uklanja na Disconnect/startup cleanupu
- FTPS certifikat i hostname i dalje se provjeravaju; online Schannel revocation dohvat je isključen radi stroge politike bez automatskih veza prema trećim CA servisima. To znači da ByFTP ne radi online provjeru je li inače valjan certifikat naknadno opozvan.

## Lifecycle veze

- remote operacije i transferi dobivaju session-scoped cancellation context
- Disconnect najprije zatvara prijem novih transfera i otkazuje aktivni rad, zatim prekida session
- installer/uninstaller rade samo nad kanonskom per-user ByFTP putanjom dobivenom preko Windows Known Folder API-ja

## Datoteke i putanje

- remote/local nazivi imaju traversal i control-character validaciju; dvoklik i gumb Preuzmi koriste isti server→lokalno sigurni path validator
- Windows rezervirani device nazivi i trailing dot/space se odbijaju za lokalni download
- download stabla ne smiju izaći iz odabrane mape kroz nested symlink/junction
- upload ne prati lokalne symlinkove
- lokalno rekurzivno brisanje i uninstall data cleanup ne slijede symlinkove ni Windows reparse/junction točke
- remote symlinkovi se ne preuzimaju rekurzivno
- atomski upload/download koriste privremenu datoteku i rollback prije zamjene originala
- direktorij ili symlink nikad se ne prepisuje običnom datotekom
- config i privatne helper datoteke koriste nasumične temp datoteke i atomic replace

## Runtime greške

- Windows Error Reporting je onemogućen za ByFTP proces
- production GUI hvata neočekivane Go panicove i ne prikazuje razvojni stack trace korisniku
- normalni runtime nema trajni activity/error log

## Ograničenja

Konačni release treba Authenticode potpis i runtime smoke-test na Windows 10/11 s stvarnim FTP/FTPS/SFTP serverima.

## Otpornost na reconnect i ekstreman unos

- transfer batch rezervacije su vezane uz generation aktivne veze; stara rezervacija ne može postati posao nakon Disconnect/reconnect ciklusa
- folder download rollback uklanja samo prazne direktorije koje je ByFTP upravo stvorio; nikad ne radi rekurzivni rollback tuđeg sadržaja
- Win32 input kontrole imaju ograničenja prije alokacije za host, user, password, passphrase, key i path vrijednosti

## 2.11 dodatno učvršćivanje

- transferi su vezani uz opaque identity aktivne veze; Retry na drugom hostu/accountu je blokiran
- batch Cancel/Retry prvo validira cijeli odabir kako zastarjeli redak ne bi ostavio polovicu odabira u neočekivanom stanju
- FTP MLSD capability cache razlikuje eksplicitno nepodržanu naredbu od transient mrežne/auth greške i ne zaključava pogrešno server na LIST fallback
- prethodni directory refresh se otkazuje pri novoj navigaciji, a subprocess wait cleanup je vremenski ograničen
- Windows proces uklanja current-directory DLL search putanju prije otvaranja GUI-ja
