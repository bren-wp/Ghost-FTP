# Third-party notices

Ghost FTP desktop/core is built with Go and intentionally has **no external Go module dependency graph**. The application uses operating-system networking tools for protocol execution rather than bundling an untracked third-party FTP/SSH stack.

## Windows and Linux runtime tools

### FTP / FTPS

Ghost FTP uses an operating-system `curl` executable for FTP and explicit FTPS operations on the maintained desktop platforms.

`curl` is distributed and licensed by its respective operating-system/vendor package source. Ghost FTP does not claim ownership of curl and does not silently download a private bundled copy at runtime.

### SFTP

Ghost FTP uses operating-system OpenSSH `ssh` / `sftp` tooling for SFTP operations.

On Windows this normally means the Windows OpenSSH Client capability. On Linux it normally comes from the distribution's OpenSSH client package.

OpenSSH is distributed and licensed by its respective operating-system/vendor package source. Ghost FTP constrains the child-process configuration and credential handoff but does not relicense or impersonate OpenSSH.

## Linux package prerequisites

The canonical Debian packages declare the maintained runtime prerequisites, including:

- `ca-certificates`;
- `curl`;
- `openssh-client`.

These are system packages, not vendored Go dependencies.

## Web companion

The separate `GhostFTP WEB/` source surface has no third-party Composer runtime packages in its maintained `composer.json`. PHP extensions listed under `suggest` are operator-provided capabilities rather than Composer-installed libraries.

The Web companion is audited separately and is not a Windows/Linux desktop release artifact.

## GitHub Actions

Build workflows use pinned GitHub Actions revisions for repository checkout, Go/Python setup and artifact upload/download. Those actions are build-system dependencies executed by GitHub Actions; they are not installed runtime components of Ghost FTP.

## Signing material

Code-signing certificates and private keys are not third-party runtime libraries and must never be committed to this repository. A production Authenticode identity, when configured, is supplied externally through the protected release boundary.

The development self-signed code-signing helper exists only to validate the signing pipeline. It does not create a trusted publisher identity and its generated PFX/CER files are not source artifacts.

## Identity and historical provenance

Historical source paths or internal compatibility identifiers that contain `GhostFTP` are implementation details retained only where changing them could break upgrades or installed application identity. They do not represent a separate current public product.

Historical releases may also contain dependencies or platform artifacts that matched their original release matrix. Historical provenance is not the active Windows/Linux dependency contract.
