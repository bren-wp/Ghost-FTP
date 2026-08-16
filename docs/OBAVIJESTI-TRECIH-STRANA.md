# ByFTP — obavijesti o komponentama trećih strana

ByFTP 2.14.0 runtime ne ugrađuje vanjske Go module ni web/runtime SDK-ove.

Koristi sistemske komponente operacijskog sustava Windows:

- Windows `curl.exe` za FTP/FTPS
- Windows OpenSSH Client za SFTP
- Segoe Fluent Icons na Windowsu 11 i Segoe MDL2 Assets kao fallback na Windowsu 10
- standardne Win32, Shell, Registry, Common Controls, DWM i DPAPI API-je

Te komponente ostaju dio Windows sustava i podliježu odgovarajućim Microsoftovim uvjetima. ByFTP ih ne redistribuira kao vanjske Go dependencies.
