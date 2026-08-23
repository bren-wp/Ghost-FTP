# Installation

## Windows

Use the generated setup executable for a normal per-user installation. Setup validates its embedded payload before changing the filesystem or registry, stages replacements, keeps rollback data during upgrades and creates Windows integration only after the payload passes integrity checks.

The first setup screen is the language selector. The chosen locale is used by setup and persisted as the initial ByFTP application language. The language can be changed later in application settings.

The installer does not require administrator rights for the normal per-user path. Do not bypass Windows path, reparse-point or integrity checks when packaging custom builds.

## Linux and macOS

Use the release packages produced by the repository build scripts. Linux packages are architecture-specific; macOS uses the Universal package produced by the gated build.

## Source builds

Read the product version from `VERSION`. Run tests and repository audits before distributing a local build. Production builds should keep Go telemetry disabled and use the same security defaults as release builds.
