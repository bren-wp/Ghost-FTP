# Settings

Ghost FTP settings are platform-appropriate and must never fabricate unsupported controls.

## Windows and Linux

Desktop settings cover maintained product options such as language, appearance and transfer behavior. The settings model is shared where the native surface implements the option.

## Android

Android exposes the settings appropriate to its mobile lifecycle/storage model. Desktop-only features are not shown merely for parity.

## Update actions

Update-related UI must be explicit about whether it is simulation, download navigation or actual installer behavior. A simulated progress surface must not claim that a binary was replaced when no installation occurred.

## Credential boundary

Ordinary settings persistence is separate from secret persistence. Failure to load or protect secret state must fail closed.

English is the canonical default/fallback. Localized strings should remain complete and must not be mixed with developer placeholders in the shipping UI.

macOS settings are retired with the removed macOS application.
