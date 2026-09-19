# Privacy

Ghost FTP is designed for direct user-controlled file transfer without application telemetry.

## What Ghost FTP does not require

- no mandatory Ghost FTP account;
- no advertising identifier;
- no behavioral analytics;
- no product telemetry;
- no hidden sync cloud;
- no Ghost FTP transfer proxy.

## Network traffic

Windows, Linux and Android send user-directed protocol traffic to the server selected by the user.

Browser helpers do not perform FTP transport.

## Saved data

Saved credentials are opt-in and platform-local.

Windows and Linux use their maintained local protected-secret boundaries. Android saved-site state remains intentionally non-secret unless a protected-secret path is explicitly implemented.

## Release infrastructure

Signing keys, PFX files, Android keystores and other publisher credentials are external release secrets and must never be committed or embedded in public artifacts.

GitHub, operating systems, certificate authorities, timestamp providers and remote servers have their own independent policies. Those services are not Ghost FTP application telemetry.

## Support

Never put real passwords, private keys, passphrases, server-private data or signing credentials in public bug reports.
