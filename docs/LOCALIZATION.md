# Localization

English is the canonical Ghost FTP default/fallback language.

## Active surfaces

Localization applies to:

- Windows native desktop UI;
- Linux native desktop UI;
- Android native UI where translated resources are implemented.

A platform must not claim translation parity for strings it does not actually expose.

## Product terminology

The following product nouns should stay consistent:

- Files
- Connections / Sites
- Transfer Queue / Transfers
- Settings
- Local Files
- Remote Files
- Quick Connect
- Bookmarks
- More

## Quality rules

- no untranslated developer placeholders in shipping UI;
- no mixed-language button rows caused by missing resources;
- text must fit or wrap according to the platform layout contract;
- English remains available as a recovery fallback.

macOS localization coverage is retired with the removed macOS application.
