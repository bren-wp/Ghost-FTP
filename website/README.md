# Ghost FTP website

Dependency-free Ghost FTP loading/landing page implemented with real HTML and CSS components. The supplied artwork is used only as a visual reference and as individual brand/background assets; the page is not a full-screen screenshot with hotspots.

- Primary language: English
- Additional language: Hrvatski
- Official links: `https://ghostftp.com/`
- Responsive desktop/mobile layout
- No external runtime dependencies


## Desktop update service

The public website content and the desktop update service share the same domain but have different responsibilities. Human-facing pages describe updates in product language only. Technical deployment details, response structure, signing requirements and server configuration live under `/updates/` in the repository.

The production desktop application checks the official Ghost FTP update service under `https://ghostftp.com/updates/`. The release workflow prepares a web-ready update package; hosting operators deploy it according to `updates/WEB_DEPLOYMENT.md`.
