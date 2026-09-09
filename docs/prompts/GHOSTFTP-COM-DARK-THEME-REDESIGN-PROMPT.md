# ghostftp.com dark-theme production website prompt

Use this prompt to build a new `ghostftp.com` product website or to refactor the existing site in place. The website must visually belong to the same product as the Ghost FTP desktop application while preserving the product's privacy-first and dependency-minimal principles.

## Master prompt

You are responsible for designing and implementing the production website for **Ghost FTP** at **https://ghostftp.com**.

The repository and the current production site must be audited before implementation. Do not blindly replace working URLs, legal content, download links, redirects, search-engine metadata or useful information. If an existing site is present, inventory it first, identify what should be preserved, migrated, redirected or removed, and then implement the redesign without creating broken inbound links.

This website is the **official product, download, documentation, privacy/security and support website** for the maintained Windows/Linux Ghost FTP application. It is not authorization to create a browser-based FTP client, proxy user FTP traffic, collect FTP credentials or restore a retired application runtime in the browser.

### 1. Product identity

Use the public identity exactly:

- Product: **Ghost FTP**
- Official website: **https://ghostftp.com**
- Maintained desktop platforms: **Windows and Linux**
- Protocols: **FTP, FTPS and SFTP**
- Product positioning: modern, privacy-first FTP client with local credential/data handling and strong secure-default behavior

The site must not invent a new company/product name, fake sub-brand, fake certification, fake award, user count, review score, customer logo, testimonial or usage metric.

Use the real repository-local Ghost FTP logo/icon and authentic application screenshots. Do not draw a fake application UI and present it as a screenshot. Do not use AI-generated application screenshots as product evidence.

### 2. Visual direction — match the real Ghost FTP dark application theme

The primary website experience should use the canonical Ghost FTP dark palette from the application source. Treat these values as design tokens, not loose inspiration:

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

The aesthetic should be premium, restrained and workstation-oriented: deep navy/charcoal backgrounds, layered panels, clean thin borders, Ghost FTP blue accents, strong typography, concise motion and authentic product imagery.

Do **not** turn it into a generic black/neon "hacker" site. Avoid glowing green terminals, excessive gradients, glass everywhere, cyberpunk clichés, random purple/pink SaaS gradients or oversized decorative effects that do not exist in the application.

Use the application as the design-system reference: its panel hierarchy, navigation rail ideas, list surfaces, subtle selected states, button variants and spacing rhythm should inform the website. The site should still be a modern responsive website rather than a literal desktop-window clone.

### 3. Audit the existing site before changing it

If `ghostftp.com` already contains production files, first produce an inventory of:

- current routes and language routes;
- canonical URLs;
- downloads and release links;
- legal/privacy/security pages;
- support/contact routes;
- existing redirects and `.htaccess`/server configuration where relevant;
- sitemap and robots policy;
- structured data;
- Open Graph/Twitter metadata;
- current images/icons/fonts;
- forms and their data flow;
- external dependencies;
- cookies/storage/analytics/tracking;
- legacy routes that still receive inbound traffic;
- performance and accessibility problems;
- broken links and redirect loops.

Preserve useful stable URLs where possible. When a route must change, use a deliberate permanent redirect where appropriate. Do not create redirect chains or loops.

### 4. Information architecture

Build a clear production information architecture. The final implementation may use separate pages or carefully designed sections, but it must provide obvious paths to at least:

- Home;
- Download;
- Features;
- Protocols;
- Security;
- Privacy;
- Documentation / Help;
- Support / Contact;
- Release / version information;
- legal information required by the actual operating entity and jurisdiction.

Recommended homepage sequence:

1. **Header/navigation** — Ghost FTP logo/wordmark, Features, Security, Download, Docs/Support, language control if localization is implemented.
2. **Hero** — concise product statement, Windows/Linux availability, primary Download action, secondary View Features/Documentation action, authentic main-workspace screenshot.
3. **Trust strip** — truthful facts only: FTP / FTPS / SFTP, Windows / Linux, local-first/privacy-first principles, maintained language count if sourced from the current product contract.
4. **Product workflow** — connect, browse local/remote, transfer, manage queue, profiles/Site Manager.
5. **Security section** — SFTP host-key verification/pinning, FTPS/TLS, protected credential handling, path/root protections, no hidden telemetry claim only where supported by current documentation.
6. **Authentic UI showcase** — repository-local main workspace, Site Manager, Settings and About screenshots from real production builds where maintained by the repository.
7. **Platform/download section** — architecture-appropriate Windows and Linux choices sourced from the current canonical release.
8. **Privacy section** — concise explanation with link to full privacy policy.
9. **FAQ / support** — practical install, portable, protocol, security and troubleshooting questions based on current documentation.
10. **Footer** — product navigation, legal/privacy/security/support links, current truthful product/publisher attribution, no fake social links.

### 5. Download experience

Do not hard-code stale binaries as if they are forever current. The download surface must be driven by, generated from or deliberately synchronized with the repository's authoritative release information.

The canonical release may expose Windows Setup and Portable variants plus Linux package/archive variants. Present architecture labels clearly. Never label x86 as x64, never hide the Portable/Setup distinction, and never imply supplemental CI-only distro packages are canonical public release artifacts unless the repository's current release contract actually says so.

When linking to GitHub Releases, point to the canonical official repository/release destination. Do not mirror binaries to an unverified third-party host. Where checksums are available, make verification discoverable.

The UI should make the primary recommended download obvious without removing access to other supported architectures. Detecting a browser platform may improve the suggested button, but the complete supported list must remain accessible and user-controlled.

### 6. Security and privacy requirements for the website

The site itself must follow the same privacy-first philosophy as the application:

- no analytics or telemetry by default;
- no advertising SDKs;
- no tracking pixels;
- no fingerprinting;
- no behavioral profiling;
- no third-party chat widget that tracks visitors;
- no external webfonts when system/local fonts are sufficient;
- no remote icon CDN when repository-local SVG/icon assets can be used;
- no remote decorative media dependency;
- no hidden product API;
- no unnecessary cookie/localStorage use;
- no cookie-consent banner if the site genuinely does not set non-essential cookies;
- never ask for FTP/SFTP hostname, username, password, private key or server credentials on the marketing website;
- never proxy FTP/SFTP connections through the website;
- never store product credentials in a form, query parameter, analytics event or server log by design.

If a contact form is required, collect only the fields necessary for support/contact, document where the submission goes, use CSRF/spam protection that does not introduce invasive tracking, validate and encode all input, apply size/rate limits and never repurpose submissions for marketing without explicit consent.

### 7. Front-end implementation constraints

Prefer the existing production stack if it is maintainable and secure. Do not replace a simple working static site with a large framework merely for fashion.

If building from scratch and no framework is genuinely required, prefer a minimal static implementation with semantic HTML, maintainable external CSS and modular JavaScript only where interaction requires it. Keep assets repository-local and make deployment straightforward on ordinary HTTPS hosting.

Requirements:

- no inline CSS or JavaScript unless the existing CSP/deployment architecture has a documented reason requiring a narrowly controlled exception;
- no minified-one-line source files in the maintained source tree;
- no duplicated CSS/JS components;
- no unused libraries;
- no runtime dependency on a JavaScript framework for content that works as HTML/CSS;
- progressive enhancement for optional interactions;
- server configuration that does not create redirect loops;
- HTTPS-only canonical URLs;
- stable cache headers for hashed/static assets;
- correct MIME types;
- compression where hosting supports it;
- no mixed content;
- no broken asset-relative paths on localized routes/subdirectories.

### 8. Responsive behavior

The design must be fully usable from small mobile screens through large desktop monitors. Test actual layout behavior; do not merely add one media query and call it responsive.

Verify at representative widths including approximately 320, 360, 390, 412, 768, 1024, 1280, 1440 and wide desktop.

Required behavior:

- no horizontal scrolling from layout overflow;
- navigation converts to a fully working keyboard/touch-accessible mobile menu;
- menu can open, close, escape and restore focus correctly;
- hero copy and CTAs do not overlap;
- screenshots keep usable aspect ratio and readable framing;
- download cards stack in a sensible order;
- tables become responsive without losing labels;
- long translated text wraps safely;
- buttons remain at least practical touch-target size;
- no content is hidden merely because viewport width is small;
- landscape mobile/tablet layouts remain usable;
- reduced-motion preference is honored.

### 9. Accessibility and UX

Target WCAG 2.2 AA-quality behavior where applicable.

Use:

- semantic landmarks;
- one clear page-level heading hierarchy;
- real buttons and links rather than clickable generic containers;
- descriptive link labels;
- keyboard-operable navigation and dialogs/menus;
- visible focus indication compatible with the dark palette;
- sufficient color contrast;
- meaningful `alt` text for product imagery;
- decorative imagery excluded from the accessibility tree where appropriate;
- form labels, errors and status messages tied programmatically to fields;
- skip navigation where useful;
- `prefers-reduced-motion` support;
- no autoplay video/audio;
- no essential information that appears only on hover.

Animations should communicate hierarchy or state, not delay access to content.

### 10. Localization

English is the primary canonical site language unless the current production site/repository specifies another source-locale contract. If the site supports multiple languages, translations must be complete and human-readable rather than partially translated navigation around English body copy.

Use stable language-specific URLs where practical, correct `lang` attributes, `hreflang` relationships, localized metadata and self-consistent canonical tags. Do not auto-redirect users solely by IP or browser language in a way that prevents them from choosing another language.

The product's maintained application language count may be stated only when sourced from the current application/release documentation. Do not assume the website must expose every application locale unless the implementation scope explicitly requires it; if all are exposed, they must all be properly maintained.

### 11. SEO and discoverability

Implement technical SEO without keyword stuffing:

- unique meaningful `<title>` and meta description per indexable route;
- one canonical URL per page;
- correct robots directives;
- valid XML sitemap containing public canonical pages only;
- Open Graph and social metadata using repository-local/public canonical media;
- structured data only when factually supported (for example software application/product/organization data with real values);
- meaningful headings and internal links;
- descriptive download/link text;
- clean URLs;
- no duplicate language/canonical combinations;
- redirects from migrated legacy URLs;
- intentional 404 page with normal navigation;
- no indexable staging/debug/admin/dev routes.

Do not fabricate review stars, pricing, offers, download counts or software ratings in structured data.

### 12. Performance targets

Engineer for excellent real-world performance, not a synthetic score obtained by hiding content.

Goals:

- fast LCP with a deliberately sized hero asset;
- near-zero CLS through explicit image dimensions/aspect ratios and stable font strategy;
- minimal main-thread JavaScript;
- no render-blocking third-party tags;
- local optimized WebP/AVIF/PNG/SVG assets as appropriate while retaining source-quality originals where the project needs them;
- responsive `srcset`/sizes for photographic/raster UI screenshots where beneficial;
- lazy-load below-the-fold imagery, not the critical hero image;
- avoid enormous background videos;
- remove unused CSS/JS;
- cache immutable static assets;
- keep DOM complexity reasonable.

Run Lighthouse/PageSpeed-type checks for mobile and desktop, but also manually inspect layout and interaction because a score cannot prove a working menu or accurate download path.

### 13. Security headers and hosting

Configure headers according to the actual hosting platform. Prefer a restrictive policy compatible with the implementation:

- `Content-Security-Policy` with no broad `unsafe-*` allowances unless specifically justified;
- `X-Content-Type-Options: nosniff`;
- `Referrer-Policy` with privacy-conscious behavior;
- `Permissions-Policy` disabling unneeded browser capabilities;
- frame-ancestor protection via CSP;
- HSTS only when HTTPS and all relevant subdomains are ready for the chosen scope;
- no server version/debug leakage where configuration allows it.

Do not copy a header template that breaks downloads, localized routes or required forms. Test the deployed policy.

### 14. Content accuracy

Use the current Ghost FTP repository documentation as the source for supported platforms, protocols, security claims, release state, installation choices and privacy behavior.

Every feature claim on the website must map to a real maintained capability. Avoid claims such as "military-grade", "zero knowledge", "unhackable", "100% secure" or "anonymous" unless there is an exact, defensible technical definition in the product contract.

Release/version text must be updated from an authoritative source rather than copied from an old mockup. Historical release pages may retain historical versions; current download/home metadata must agree with the current published stable release.

### 15. Required pages/content quality

Write production-ready content rather than placeholder copy. Remove lorem ipsum, `TODO`, "coming soon" for functionality that already exists, internal developer comments and implementation explanations from public copy.

Security and privacy pages should be detailed enough to explain the actual model without exposing secrets. Download/help pages should answer common architecture, Setup-vs-Portable and package-selection questions. Contact/support copy should explain what diagnostic information is safe to share and explicitly discourage sending passwords or private keys.

### 16. Testing and acceptance criteria

Before declaring the website complete, verify at minimum:

- every header/footer/mobile-nav link;
- every CTA;
- every download link and architecture label;
- every language selector route;
- every form and validation/error path if forms exist;
- 404 handling;
- redirect rules and absence of redirect loops;
- canonical, robots and sitemap output;
- page titles/descriptions;
- structured data validity;
- no console errors;
- no missing assets;
- no mixed content;
- no horizontal overflow at representative viewport widths;
- full keyboard navigation;
- visible focus states;
- mobile menu open/close/Escape/outside behavior;
- reduced-motion behavior;
- contrast and accessible names;
- no external tracking/network requests beyond explicitly approved functional destinations;
- no credential fields or hidden FTP proxy path;
- CSP/security headers on the deployed environment;
- production HTTPS canonicalization;
- real product screenshots and logo render correctly;
- current release/version/download data is correct.

Where tooling exists, run HTML/CSS/JS validation, link checking, accessibility checks and performance audits. Manually verify the critical flows even when automation passes.

### 17. Delivery requirements

Deliver production source, not only screenshots or a visual concept. Keep the code readable and documented where deployment behavior is non-obvious.

Provide:

- final route map;
- migration/redirect map for changed legacy URLs;
- asset inventory and source provenance;
- deployment instructions for the real hosting environment;
- security-header/server configuration;
- SEO/sitemap/robots configuration;
- testing checklist/results;
- list of external network destinations, ideally limited to canonical user-initiated destinations such as official download/documentation links;
- explicit confirmation that analytics/tracking/ads were not introduced;
- any intentionally preserved legacy compatibility behavior.

The finished `ghostftp.com` should look unmistakably related to the Ghost FTP application's dark theme, remain fast and readable on mobile, provide trustworthy current downloads and documentation, and preserve the product's privacy-first behavior without turning the website into a credential-handling FTP service.
