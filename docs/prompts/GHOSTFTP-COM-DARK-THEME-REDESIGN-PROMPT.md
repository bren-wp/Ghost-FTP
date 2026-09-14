# ghostftp.com 0.0.6 dark-theme production website prompt

Use this prompt to build or refactor the production website for **Ghost FTP** at `https://ghostftp.com`. The repository and the live site must be audited before implementation. Preserve valid URLs, legal information, redirects, SEO equity and working deployment behavior rather than replacing them blindly.

The website must visually belong to the same product as the real Ghost FTP applications while remaining privacy-first, fast, accessible, responsive and technically honest. The repository is authoritative for platform support, release state, screenshots, downloads, security claims and licensing.

## Master prompt

You are responsible for the production design and implementation of `ghostftp.com`.

Before changing public content, record the current repository `main` SHA, `VERSION`, last actually published stable release and the current candidate release contract. Inspect the live site and repository documentation. Do not present a candidate as already published, a development build as production, a CI artifact as an official release, or a mockup as an application screenshot.

Ghost FTP is proprietary/source-available software; the controlling terms are in the repository `LICENSE`. Do not describe it as open source unless the controlling license is deliberately changed.

## 1. Product and platform truth

Use the product identity exactly:

- Product: **Ghost FTP**
- Official website: `https://ghostftp.com`
- Positioning: privacy-first file-transfer client/tooling with local credential/data handling and secure defaults

The intended 0.0.6 public distribution currently contains:

- **Windows:** one universal Setup and one universal Portable executable, each carrying/selecting x64, x86 and ARM64 payloads.
- **Linux:** six universal distro bundles: Debian Installer + Portable, Ubuntu Installer + Portable, Fedora Installer + Portable. Each contains amd64, arm64 and i386 payloads and selects the local architecture.
- **Android:** one canonical production-signed APK. Current compatibility boundary is **Android API 26+ (`minSdk 26`)** with `targetSdk 35`; never claim support for every Android version.
- **Browser helper:** deterministic ZIPs for **Chrome, Edge, Firefox and Opera**. The official helper has **zero browser permissions and zero host permissions** and must not be represented as a browser FTP client.
- **macOS:** active development/source surface only. Do not offer a public production download until real Developer ID Application signing and Apple notarization are verified.

The current candidate allow-list is **13 platform artifacts plus 3 metadata files = 16 public release files**. Re-read the release workflow/scripts before hard-coding these numbers in production site data. The site must follow the current repository contract if it changes.

## 2. Protocol claims must be platform-specific

Do not flatten protocol support into one misleading global badge.

Desktop Windows/Linux may expose the maintained FTP/FTPS/SFTP capabilities defined by current source and documentation. Android currently maintains FTP and explicit secure FTPS behavior, but **Android SFTP is hidden/unsupported** until strict maintained host-key verification/pinning exists. Never advertise Android SFTP, a trust-all path or a placeholder protocol option.

The browser helper must not be described as performing FTP/FTPS/SFTP transfers itself. It has no host/network permissions, no product backend, no cloud relay and no browser-to-desktop connection handoff.

## 3. Audit the existing website first

Inventory the live site before implementation:

- routes and localized routes;
- canonical URLs and redirects;
- download/release links;
- privacy/security/legal/support pages;
- sitemap and robots policy;
- structured data;
- Open Graph/social metadata;
- current screenshots, icons, logos and fonts;
- forms and their destinations/data flow;
- cookies, storage, analytics, pixels and trackers;
- external scripts/assets/services;
- server configuration and HTTPS redirects;
- legacy routes with inbound value;
- broken links and redirect loops;
- accessibility, responsive and performance defects.

Preserve useful stable URLs where practical. When a route must change, use one deliberate permanent redirect and avoid redirect chains/loops.

## 4. Visual direction

Use the canonical Ghost FTP dark application palette as the primary reference unless current source has deliberately migrated it:

```text
Window/background: #0B0F17
Panel/surface:     #121824
List/surface-2:    #161D2A
Border:            #2C3648
Primary text:      #F2F5FA
Muted text:        #97A3B8
Accent:            #5B7CFA
Accent strong:     #7A98FF
Success:           #4AD79B
Warning:           #F2BA55
Danger:            #FF6878
Selection:         #202F50
```

Create a premium restrained workstation aesthetic: deep navy/charcoal surfaces, clear hierarchy, thin borders, Ghost FTP blue accents, high-quality typography and concise motion. Do not turn the site into a generic neon/cyberpunk/hacker theme. Avoid random purple/pink SaaS gradients, excessive glass, fake terminals and visual effects unrelated to the product.

Use application panel hierarchy, selected states, spacing and control language as design-system references, but keep the result a responsive website rather than a desktop-window clone.

## 5. Authentic media only

Use real repository-local product assets and authentic runtime screenshots. Current maintained documentation includes real product imagery under `docs/images/` and the authentic cross-platform screenshot workflow can provide exact-source evidence.

Never draw, AI-generate or manually fabricate a product UI and present it as a real screenshot. If decorative artwork is used, label/treat it as artwork. Do not use stock/watermarked UI imagery.

Each screenshot used as product evidence should have known provenance: platform, source/release version and real runtime/emulator capture where relevant.

## 6. Information architecture

Provide clear access to at least:

- Home
- Download
- Features
- Platforms
- Protocols
- Security
- Privacy
- Documentation / Help
- Browser Helper
- Support / Contact
- Release / Version information
- controlling license and required legal information

Recommended homepage sequence:

1. **Header** — logo/wordmark, Features, Platforms, Security, Download, Docs/Support, language control if implemented.
2. **Hero** — concise product statement, truthful current public platforms, primary download CTA and authentic main-workspace imagery.
3. **Trust facts** — factual privacy/security/platform statements only; no fake ratings, user counts, awards or certifications.
4. **Workflow** — connect, browse local/remote, transfer, manage profiles/queue where supported.
5. **Platforms** — Windows, Linux and Android production surfaces; browser helper separated from native clients; macOS visibly marked development-only if mentioned.
6. **Security** — desktop SFTP verification/pinning, FTPS/TLS, protected credential handling, local-path safety and release integrity; Android protocol limitations stated accurately.
7. **Authentic UI** — real Windows/Linux/Android captures and maintained screenshots where available.
8. **Downloads** — current stable release data from authoritative release metadata, with checksums/verification discoverable.
9. **Browser Helper** — Chrome/Edge/Firefox/Opera, permission-free contract, no network/backend/handoff claims.
10. **Privacy** — concise no-telemetry/no-hidden-backend explanation linked to full policy.
11. **FAQ/Support** — practical install, architecture, protocol, Android compatibility, portable/installer and troubleshooting answers.
12. **Footer** — product/docs/legal/privacy/security/support links and truthful publisher/legal attribution sourced from current legal docs.

## 7. Download experience

Never hard-code a candidate or stale binary as if it were the currently published stable release. The homepage and Download page must distinguish:

- current source/candidate version;
- last actually published stable release;
- public production artifacts;
- development-only or CI artifacts.

Use authoritative GitHub Release data or a deployment process deliberately synchronized with it. Do not invent mirrors.

For the 0.0.6 contract, the production UI should understand:

- Windows Setup vs Portable;
- six Linux universal distro choices;
- Android canonical APK with API 26+ compatibility statement;
- four browser helper packages;
- checksum/release metadata.

Do not expose internal payloads as separate public downloads merely because a universal bundle contains multiple architectures. Do not claim native ARM64/i386 Linux runtime evidence when only packaging/build evidence exists.

macOS development artifacts must not appear among normal production downloads until the signing/notarization gate is truly satisfied.

## 8. Website security and privacy

The site should follow the product's privacy-first posture:

- no analytics/telemetry by default;
- no advertising SDKs, tracking pixels or fingerprinting;
- no behavioral profiling;
- no invasive third-party chat widgets;
- no unnecessary third-party fonts/icon CDNs/decorative media;
- no hidden product API or proxy;
- no unnecessary cookies/localStorage;
- no consent banner if the site genuinely sets no non-essential cookies;
- never ask users to submit FTP/SFTP hostnames, usernames, passwords, private keys or server credentials to a marketing/support form;
- never proxy file-transfer sessions through the marketing website;
- never put product credentials into URLs, analytics events or logs by design.

If a contact form is maintained, collect only necessary contact/support fields, validate/encode input, implement CSRF/spam/rate/size protection without invasive tracking and clearly document where submissions go. Explicitly warn users not to send passwords or private keys. **Do not repurpose names, email addresses, support messages or any other contact submission data for marketing, profiling or unrelated secondary use without separate explicit opt-in consent.**

## 9. Browser helper presentation

Treat the browser helper as its own product surface, not as a web FTP client.

Public copy must preserve these invariants:

- Chrome, Edge, Firefox and Opera packages come from the official deterministic packaging contract;
- zero browser permissions;
- zero host permissions;
- no telemetry/tracking;
- no FTP/SFTP credential storage;
- no cloud/backend relay;
- no browser-to-desktop handoff;
- no claim that the extension can browse or transfer server files itself.

Do not add host permissions or external network behavior merely to make a demo appear more capable.

## 10. Android presentation

State the actual compatibility floor as API 26+ unless current build configuration changes after tested compatibility work. Do not say “all Android devices” or “all Android versions”.

Use real Android screenshots/emulator evidence. Describe local storage in user-facing terms consistent with Android's Storage Access Framework/document-provider model.

Do not expose or advertise Android SFTP until the implementation has strict maintained host-key verification/pinning and the repository changes its support contract. FTP and FTPS claims must reflect actual maintained behavior.

## 11. Front-end implementation

Prefer the existing maintainable production stack. Do not replace a simple static deployment with a large framework without a demonstrated need.

If no framework is required, prefer semantic HTML, maintainable external CSS and modular JavaScript only where interaction requires it.

Requirements:

- no inline CSS/JS unless a documented deployment/CSP reason requires a narrowly controlled exception;
- no maintained source minified into one line;
- no duplicate/unused CSS or JS;
- no unnecessary runtime framework dependency;
- progressive enhancement;
- HTTPS canonicalization without loops;
- stable caching for versioned/static assets;
- correct MIME types and compression where available;
- no mixed content;
- localized/subdirectory routes must resolve assets correctly.

## 12. Responsive and mobile behavior

Test real behavior at representative widths around 320, 360, 390, 412, 768, 1024, 1280, 1440 and wide desktop.

Require:

- no accidental horizontal overflow;
- fully working touch/keyboard mobile navigation;
- mobile menu open/close/Escape/focus restoration;
- no hidden content on small screens;
- non-overlapping hero/CTAs;
- screenshots keep usable aspect ratios;
- download/platform cards stack logically;
- responsive tables preserve labels;
- long translated copy wraps safely;
- practical touch targets;
- landscape mobile/tablet usability;
- `prefers-reduced-motion` support.

## 13. Accessibility

Target WCAG 2.2 AA-quality behavior where applicable:

- semantic landmarks/headings;
- real links/buttons;
- keyboard operation;
- visible focus;
- sufficient contrast;
- meaningful screenshot alt text;
- decorative assets hidden from the accessibility tree;
- programmatic form labels/errors/status;
- skip navigation where useful;
- no autoplay audio/video;
- no essential hover-only information.

Animation may explain hierarchy/state but must not delay access to content.

## 14. Localization

English is the canonical website source locale unless the deployed site defines another maintained contract. If multiple languages are offered, each public route must be completely and correctly translated rather than exposing translated navigation around English body copy.

Use correct `lang`, stable language URLs, `hreflang`, localized metadata and coherent canonical tags. Do not force IP/browser-language redirects that prevent manual language choice.

Do not claim the application language count unless sourced from current repository documentation.

## 15. SEO

Implement technical SEO without keyword stuffing:

- unique title/meta description per indexable page;
- one canonical URL per page;
- correct robots directives;
- XML sitemap containing public canonical pages only;
- Open Graph/social metadata using canonical authentic/local media;
- factual structured data only;
- meaningful headings/internal links;
- clean URLs and descriptive download link text;
- coherent localized canonicals/hreflang;
- permanent redirects for deliberate migrations;
- intentional 404 page;
- no indexable staging/debug/dev/admin routes.

Never fabricate stars, reviews, download counts, pricing/offers, customers, awards or certifications in visible copy or structured data.

## 16. Performance

Engineer for fast real-world performance rather than hiding content to chase a score:

- deliberately sized critical hero image;
- explicit dimensions/aspect ratios to control CLS;
- local/system font strategy;
- minimal main-thread JS;
- no render-blocking tracking tags;
- optimized local SVG/PNG/WebP/AVIF as appropriate;
- responsive raster images where useful;
- lazy-load below-the-fold media, not the critical hero;
- remove unused CSS/JS;
- immutable caching for fingerprinted assets;
- reasonable DOM complexity.

Use Lighthouse/PageSpeed-type checks as diagnostics, not proof that menus/downloads/content work correctly.

## 17. Security headers and hosting

Configure headers for the actual hosting environment and test them after deployment. Prefer a restrictive compatible policy:

- Content-Security-Policy without broad unsafe allowances;
- `X-Content-Type-Options: nosniff`;
- privacy-conscious `Referrer-Policy`;
- `Permissions-Policy` disabling unneeded capabilities;
- CSP `frame-ancestors` protection;
- HSTS only when HTTPS/subdomain readiness justifies the chosen scope;
- no debug/server-version leakage where configurable.

Do not copy a header template that breaks localized routes, downloads or forms.

## 18. Content accuracy and legal boundaries

Every feature/security/platform statement must map to maintained repository behavior. Avoid vague claims such as “military-grade”, “unhackable”, “100% secure”, “anonymous” or “zero knowledge”.

Do not expose internal development notes, stale `TODO` copy or fake “coming soon” labels for already implemented features. Do not describe source availability as open-source licensing. Link the controlling proprietary license and current privacy/security documentation where appropriate.

Historical release pages may keep historical facts. Current homepage/download/platform metadata must describe the current published release, not merely the current source candidate.

## 19. Acceptance testing

Before declaring the site complete, verify:

- all header/footer/mobile-nav links;
- every CTA;
- every download target, platform and artifact label;
- current stable vs candidate version state;
- checksums/verification links;
- Android API compatibility copy;
- browser helper permission/no-network claims;
- macOS development-only labeling;
- every language route;
- form success/error/validation paths where forms exist;
- 404 and redirects/no loops;
- canonical/robots/sitemap;
- structured data;
- no console errors/missing assets/mixed content;
- representative responsive widths/no overflow;
- keyboard/focus/reduced-motion/contrast behavior;
- no unauthorized trackers or network destinations;
- no FTP credential fields/proxy behavior;
- production CSP/security headers;
- real screenshots/logo render correctly;
- release/download data matches the authoritative repository/GitHub Release.

## 20. Delivery requirements

Deliver production source, not only design screenshots. Provide:

- final route map;
- migration/redirect map;
- asset inventory and provenance;
- deployment instructions;
- security-header configuration;
- SEO/sitemap/robots configuration;
- test checklist/results;
- external network-destination inventory;
- explicit confirmation of analytics/tracking/ads behavior;
- current download-data source and update process;
- intentionally preserved legacy compatibility behavior.

The finished site should feel unmistakably like Ghost FTP, truthfully present Windows/Linux/Android and the permission-free browser helper, keep macOS development-only until its real distribution gate is met, remain fast and accessible on mobile, and never turn the marketing website into a credential-handling file-transfer service.
