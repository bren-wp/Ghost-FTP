# Language Audit

The UI selector exposes 14 locales: English, Hrvatski, Deutsch, Français, Español, Italiano, Português, Nederlands, Polski, Slovenščina, Srpski, Bosanski, Македонски and Shqip. Embedded/web surfaces use `translate="no"` / `notranslate` and the browser runtime is launched with Chromium Translate UI disabled.

The current native dictionary has broader Croatian coverage than the other non-English dictionaries: the Croatian map contains 170 explicit keys, while each of the 12 `OTHER` locale maps currently contains 76 explicit keys. Because the requirement is that no advertised locale remain half-translated, this is an **open release blocker** and this package does not claim full 14-language parity.

Before FINAL, extract every visible English UI string (including buttons, tooltips, placeholders, errors, setup steps, statuses and accessibility labels), generate a canonical key set, require every non-English dictionary to cover that exact set, then manually review terminology and clipping in the target UI.
