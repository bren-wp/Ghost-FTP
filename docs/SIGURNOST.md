# Security model

ByFTP uses defense-in-depth around credentials, paths, protocol execution, remote commit state and process lifecycle. Security controls are tested as release invariants; UI convenience must not bypass them.

## Credentials

Passwords and private-key passphrases are not placed in curl/OpenSSH command-line arguments. Runtime secrets are protected/obfuscated in process memory where platform support allows and are explicitly forgotten after use. Saved Windows credentials use protected local storage and are bound to the matching profile identity. A changed server, username or private key does not silently inherit credentials from the old identity.

## SFTP trust

New SFTP server keys require an explicit fingerprint trust decision. Known host fingerprints are pinned. A changed key blocks the connection until the user verifies the change. Modern OpenSSH algorithm negotiation is used; obsolete RSA/SHA-1-only servers may be incompatible by design.

## Local filesystem

Path traversal, symlink, junction and reparse-point checks protect local transfer roots and destructive cleanup. Root-directory recursive deletion is blocked. Download staging is revalidated before commit.

## Remote transfer commit

Uploads use staged remote objects and revalidation before final placement. Cleanup uncertainty is propagated as an error and blocks unsafe automatic retry. No client can guarantee atomic no-replace behavior on every FTP/SFTP server; ByFTP therefore fails closed when it cannot prove the expected state.

## External processes

curl and OpenSSH children use bounded contexts, bounded output and sanitized environments. Linux/macOS use dedicated process groups for cancellation. Windows recursively terminates discovered descendants with a second snapshot to reduce orphan risk, including AskPass children. This materially reduces orphan processes but is not a mathematical guarantee against every OS scheduling race.

## Supply and release integrity

Go module network resolution is disabled in CI (`GOPROXY=off`, `GOSUMDB=off`) for the zero-external-module production build. GitHub Actions are pinned to commits. Release files are generated only after quality gates, exact asset staging and SHA-256 verification.

Report suspected security problems without including real secrets in public issues.