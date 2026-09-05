# Dependencies and external-component policy

Ghost FTP 0.1.0 Beta minimizes third-party code, prohibits application tracking, and makes every remaining operating-system prerequisite explicit.

## Scope

The active desktop application targets are **Windows and Linux**. The existing Web companion source is audited separately and is not a Windows/Linux desktop release artifact.

## Desktop/core Go dependency contract

The root Go module is intentionally standard-library-only.

CI rejects:

- `require`, `replace`, `exclude` or `retract` directives in `go.mod`;
- an unexpected `go.sum` dependency graph;
- a vendored Go module tree;
- telemetry/analytics/advertising/crash-reporting dependency markers.

This means Ghost FTP does not pull a GUI toolkit, FTP library, SSH library, analytics library or updater framework into the desktop Go module.

## Runtime transport prerequisites

The current transport layer deliberately delegates protocol execution to mature operating-system tools instead of embedding third-party protocol stacks into the Go module.

### FTP and FTPS

Ghost FTP invokes OS `curl` using a generated stdin configuration. The application:

- starts curl with `-q` so ambient user curl configuration is not loaded;
- disables proxy use for the transfer session;
- sanitizes proxy-related environment state;
- passes credentials through protected runtime handling rather than command-line arguments;
- validates download staging paths before promotion;
- keeps FTPS certificate verification enabled;
- does not opt into a blanket `ssl-no-revoke` bypass.

### SFTP

Ghost FTP uses OS OpenSSH `ssh`/`sftp`. The application-created SSH configuration disables ambient features that would change the connection boundary, including:

- `ProxyCommand`;
- `ProxyJump`;
- global known-host inheritance;
- DNS host-key verification/update behavior;
- identity-agent inheritance;
- forwarding and agent forwarding.

SFTP credentials are exposed to the child process only through the bounded AskPass/runtime-secret mechanism. The application does not write an AskPass password/passphrase file to disk.

## GUI dependency boundary

The Windows reference workspace is implemented with native Win32 APIs and operating-system fonts/controls. Setup and Portable therefore do not bundle a third-party cross-platform GUI runtime merely to render the application shell.

Linux currently uses a hardened terminal frontend over the same shared engine. This is a presentation difference, not a second transfer/security implementation.

A future pixel-equivalent Linux GUI may use an operating-system display/toolkit prerequisite only after that dependency is explicitly reviewed and documented. It must not be described as “dependency-free” if it requires X11, Wayland, GTK, Qt, WebKit or another runtime component, even when that component is normally installed by the distribution.

Ghost FTP will not silently introduce a large GUI framework solely to make a visual-parity claim. The dependency policy requires the project to state the presentation gap accurately until a reviewed Linux graphical implementation exists.

See [Desktop reference UI](REFERENCE-UI.md) and [Platform parity](PLATFORM-PARITY.md).

## Windows prerequisites

Supported Windows installations normally provide the required curl/OpenSSH components, but the build does not silently bundle an untracked third-party copy.

If a required system component is unavailable, Ghost FTP must fail with an actionable connection error rather than downloading a dependency in the background.

The graphical application itself uses Win32/DWM/common-control facilities supplied by Windows.

## Linux prerequisites

Linux packages declare the operating-system packages required for the transport implementation. The canonical DEB build is the source of truth for package metadata.

At runtime Ghost FTP expects suitable `curl`, `ssh` and `sftp` executables to be available through the supported system paths/environment.

The maintained terminal frontend does not require a bundled GUI toolkit.

## Why this is not called “zero runtime dependencies”

The phrase would be inaccurate today. Ghost FTP has **zero external Go modules in the desktop/core module**, but protocol execution still depends on OS-provided `curl` and OpenSSH tools. Windows graphical presentation also necessarily uses operating-system Win32 APIs.

The repository audit intentionally preserves that distinction. A future embedded transport implementation or Linux graphical toolkit would require a separate security/dependency review because changing a protocol implementation or application runtime boundary is much more sensitive than replacing a small UI helper.

## Web companion

The Web companion has no third-party Composer runtime packages. Its `composer.json` contains only the supported PHP platform requirement and documents optional PHP extension capabilities.

Suggested extensions are capability declarations, not Composer-installed packages. The Web companion remains separate from the active desktop application release artifact count.

## Tracking/analytics policy

Ghost FTP must not add application dependencies for:

- analytics;
- advertising;
- behavioral tracking;
- remote crash collection;
- user/session replay;
- automatic marketing attribution;
- background application telemetry.

CI audits dependency surfaces and runtime source for known vendor markers and fixed telemetry-style network endpoints.

The local Diagnostics window is not a telemetry channel. It renders local application state and does not upload a report.

## Build-time actions

GitHub Actions uses pinned action revisions for checkout, Go setup, Python setup and artifact upload/download. These are build-system dependencies, not installed-application runtime dependencies.

Production builds also execute `go telemetry off` and verify that telemetry is disabled before compiling.

The authentic Windows screenshot workflow additionally uses Windows `PrintWindow(PW_RENDERFULLCONTENT)` and image verification on the CI runner. That tooling is build/test infrastructure and is not installed into Ghost FTP.

## Change-control rules

Any proposal that introduces a new runtime/library dependency must document:

1. why existing standard-library/OS facilities are insufficient;
2. exact package/component and version;
3. license and provenance;
4. security/update ownership;
5. telemetry/network behavior;
6. whether the dependency is bundled or system-provided;
7. rollback/removal strategy;
8. CI checks required to prevent unreviewed drift.

Dependencies must never be added merely to simplify a small UI or helper function when the current platform layer can implement it safely.
