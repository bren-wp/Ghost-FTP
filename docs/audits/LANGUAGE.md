# Ghost FTP Language Audit — RC10

The UI selector exposes 14 locales: English, Hrvatski, Deutsch, Français, Español, Italiano, Português, Nederlands, Polski, Slovenščina, Srpski, Bosanski, Македонски and Shqip. Embedded/web surfaces use `translate="no"` / `notranslate` where appropriate rather than relying on browser auto-translation.

## Dictionary parity

The current native dictionary defines a 185-key canonical non-English UI set. Source inspection on RC10 confirms that every advertised non-English locale contains the same 185 keys:

- Hrvatski
- Deutsch
- Français
- Español
- Italiano
- Português
- Nederlands
- Polski
- Slovenščina
- Srpski
- Bosanski
- Македонски
- Shqip

No locale is currently missing or adding keys relative to the canonical set.

RC10 now includes `npm run check:i18n`, and the quality workflow executes it before TypeScript/build validation. This prevents a future locale from silently drifting to partial key coverage.

## Remaining language QA before FINAL

Key parity is not the same as linguistic acceptance. FINAL still requires manual review of terminology and grammar, long translations and clipping, narrow-window layouts, dialogs/dropdowns, protocol/security terminology, tooltips/placeholders and accessibility labels.

Status: **source key parity is complete and CI-gated; visual/linguistic target-OS acceptance remains open before FINAL.**
