# ByFTP — plan razvoja

## Završene produkcijske cjeline

- izvorni Win32 desktop bez browser/localhost sloja
- zajednički tipizirani Engine za Windows, Linux i macOS frontend
- FTP, eksplicitni/implicitni FTPS i SFTP
- Windows DPAPI profili i endpoint/account/private-key binding
- SFTP host-key pinning, session-scoped trust i algoritmom vezani `known_hosts`
- uklonjen SFTP batch/AskPass konflikt; Windows password/passphrase AskPass ostaje funkcionalan
- fail-closed AskPass koji ne šalje tajnu MFA/OTP/nepoznatom promptu
- Linux/macOS terminalni app nad istim remote/transfer coreom
- Linux/macOS terminal sigurno parsira quoted lokalne i udaljene putanje s razmacima bez shell evaluacije
- terminalno čekanje transfera koristi autoritativni snapshot reda i ne ovisi o zadržavanju završnog eventa u bounded povijesti
- Linux/macOS procesno runtime spremište FTP/FTPS tajne bez disk persistencije
- process-level FTP/SFTP child-process smoke regresije na Linux i macOS runnerima
- Windows x64 + x86 produkcijski build
- Linux DEB amd64 + arm64 + i386 build
- macOS Universal Intel + Apple Silicon PKG build
- transfer queue, transient retry, skip-existing i cijela stabla mapa
- connection-generation i cross-server retry izolacija
- symlink/junction/reparse zaštite, transakcijski staging i rollback
- rekurzivni upload root revalidacija i filesystem-root delete blokada
- remote session active-operation lifecycle s bounded disconnectom
- state/config safe-open i provjerene prethodne generacije
- reproducibilni brand resursi
- hrvatski docs sustav i docs/security/privacy/release auditi
- fail-closed idempotentni GitHub Release publisher
- produkcijski release quality/race gate neovisan o PR CI-ju
- jedan autoritativni release okidač bez samookidanja vlastitim tagom
- serijalizirani release concurrency model
- točan staging allowlist za 10 platformskih paketa
- provjera konačnih Windows ZIP bundleova, Linux DEB metapodataka i macOS PKG strukture
- stvarno gašenje Go toolchain telemetrije u CI/releaseu uz fail-closed provjeru u build skriptama
- javna dokumentacija i distribucijski izlazi odvojeni od internih tehničkih build dokaza

## Sljedeći prioriteti

1. **Unix SFTP credential broker** — siguran password/passphrase tok na Linuxu/macOS-u bez plaintext argumenta, disk credential datoteke ili nekontroliranog environmenta.
2. **Kontrolirani interoperability test serveri** — automatizirani FTP/FTPS/SFTP testovi s jednokratnim testnim računima, potpuno odvojeni od produkcijskih credentiala.
3. **Windows Authenticode** — uvesti tek kada postoji stvarni Brendigo code-signing certifikat i siguran secret/signing workflow.
4. **macOS Developer ID i notarizacija** — uvesti tek kada postoji stvarni Apple Developer identitet i odgovarajući secrets.
5. **macOS/Linux UX** — dodati sigurnu command history/completion ergonomiju bez stvaranja lažnog GUI pariteta ili spremanja tajni.
6. **Pristupačnost Windows GUI-ja** — dodatna keyboard/focus navigacija, screen-reader oznake i DPI provjere.
7. **Transfer observability bez vanjske telemetrije** — stvarna lokalna brzina/ETA i statusi uz bounded event churn.
8. **Memory skaliranje** — daljnja optimizacija vrlo velikih rekurzivnih planova, queue eventa i directory prikaza.
9. **Kontrolirani reconnect** — recovery od mrežnog prekida bez slabljenja endpoint/account identity granice i bez retrya na drugi server.
10. **Supply-chain provenance** — dodatne build attestations i SBOM/provenance dokazi bez runtime telemetrije, bez tajni u repozitoriju i bez vanjskih Go ovisnosti.
11. **Linux distribucijski formati** — RPM/AppImage razmatrati tek nakon sigurnosnog, update i signing modela; ne širiti pakete samo radi broja formata.

## Neće se raditi prečacima

Sljedeće se ne smatra dovršenom funkcijom dok nema stvarnu sigurnosnu granicu i test:

- lažno označavanje „Povezano” prije autentikacije i udaljenog probea
- plaintext credential u command lineu
- automatsko prihvaćanje promijenjenog SFTP host ključa
- lažni Verified Publisher/Developer ID status
- paket za platformu koji samo preimenuje binarij druge platforme
- runtime telemetrija ili skriveni vanjski API
- release koji preskače production quality gate

Svaka nova funkcija mora očuvati privatnost, hrvatski korisnički sadržaj, tipizirani in-process API, platform-specific auth granice, filesystem/session zaštite, stvarno ugašenu build telemetriju i fail-closed release ugovor.
