# Settings

Ghost FTP Settings owns product configuration that does not belong in the Files workspace.

## Windows and Linux

Desktop settings include the maintained options for:

- language;
- appearance;
- transfer parallelism;
- upload/download limits;
- conflict policy;
- backup-before-overwrite;
- delete confirmation;
- retry behavior;
- connection timeout.

Language selection remains in Settings.

## Android

Android exposes platform-appropriate settings and does not fabricate unsupported desktop-only controls.

## Product actions

Settings may expose:

- local update simulation;
- Download latest;
- Premium;
- Official website.

The local update simulation must never rewrite the binary or claim a different installed version.

Real download/website actions open only the documented HTTPS product destinations and never append FTP credentials, remote paths or transfer data.

## Persistence

Non-secret settings persist separately from protected credentials.

## Active platforms

The maintained application targets are Windows, Linux and Android.
