# Localization

English is the canonical Ghost FTP default/fallback language.

## Desktop

Windows and Linux use the maintained shared language/settings contract.

Desktop language selection belongs in Settings rather than the Files workspace.

The current catalog contains 24 selectable desktop languages.

## Android

Android uses its native resource/UI model. Desktop localization parity must not be claimed for Android unless the relevant strings and behavior are implemented and tested.

## Rules

- user-visible text must be intentional;
- no developer placeholders;
- no mixed-language UI inside one selected locale unless the source content itself is user data;
- new controls must have complete English text first;
- translated labels must remain readable and unclipped.

The retired macOS frontend is no longer part of the localization contract.
