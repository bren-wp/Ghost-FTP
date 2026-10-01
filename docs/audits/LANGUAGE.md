# Ghost FTP Language Audit — 0.20.9

The desktop UI advertises 14 locales: English, Hrvatski, Deutsch, Français, Español, Italiano, Português, Nederlands, Polski, Slovenščina, Srpski, Bosanski, Македонски and Shqip.

The existing `npm run check:i18n` contract keeps advertised non-English dictionaries aligned to the canonical key set and fails CI on missing/extra key drift. English remains the source language for current Transfer Center operational copy.

Key parity is not the same as linguistic acceptance. Target-OS QA must still review terminology, grammar, clipping, narrow-window layouts, dialogs, protocol/security terminology, placeholders and accessibility labels.

Status: source dictionary parity is CI-gated for the 0.20.9 line; manual linguistic/visual acceptance remains separate.
