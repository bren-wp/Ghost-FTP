# ByFTP Windows signing

ByFTP build namjerno ne generira lažni ili self-signed "verified publisher" identitet.

Za javni release:
1. Nabavi/verificiraj Brendigo code-signing identitet (npr. Microsoft Artifact Signing ili javno pouzdani code-signing certifikat).
2. Potpiši **Setup EXE i Portable EXE** nakon završnog PE/resource builda.
3. Koristi SHA-256 digest i pouzdani timestamp servis prema uputama izdavatelja certifikata/signing servisa.
4. Provjeri potpis Windows `signtool verify /pa /v` alatom na čistom Windows računalu.
5. Tek nakon toga objavi SHA-256 release manifest.

Privatni certifikat, tokeni i lozinke ne smiju biti u repozitoriju ili source paketu.
