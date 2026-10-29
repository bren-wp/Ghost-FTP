# Ghost FTP — Security policy

**Publisher:** Brendigo · **Product components:** Ghost FTP Client, Ghost FTP Server, Ghost CLI, Ghost FTP Demo.

This is the public, non-sensitive security policy for supported Ghost FTP editions. It deliberately does **not** include private keys, internal deployment addresses, administrative secrets, exploit instructions, customer environment identifiers or unreleased vulnerability information. Security is enforced by the code, permissions and tests—not by relying on undocumented or hidden weaknesses.

## Supported release and security status

The latest public release and its support classification are recorded on the [official GitHub Releases page](https://github.com/bren-wp/Ghost-FTP/releases). A `Preview` label means an evaluation/development build, **not** a production security certification. No claim is made that any release is free of vulnerabilities. Only use a release in an environment suited to its documented maturity and update it when supported security fixes are published.

Windows/Linux/macOS clients and the Android client, the standalone Go Ghost FTP Server, Ghost CLI and Ghost FTP Demo are distinct executables with distinct security boundaries. They can share the same brand, source-version policy and compatible protocol interfaces without automatically sharing device passwords or exposing remote administration.

## Security principles

- **Secure connections:** FTPS uses validated TLS identity; SFTP uses SSH host-key verification. Unsafe plaintext/downgrade paths must not be silently selected. Connection failures must be visible.
- **Credentials and privacy:** Prefer OS-protected credential storage and strong, distinct secrets. No passwords, signing material, private keys or tokens in plain-text diagnostics, URLs, process arguments or release artifacts. File paths and user-provided names must not be executed as shell instructions.
- **Server isolation:** Use a restricted service identity, filesystem permissions and a deliberately configured server root. Keep the local administrative interface bound to the local machine. Public network exposure, certificates, account policy, updates and backups require operator approval.
- **Data integrity:** Preserve user data on upgrade/uninstall unless explicitly authorized otherwise. Treat destructive operations, atomic writes, conflict resolution, transfer retries and storage quotas as security-relevant.
- **Release integrity:** Verify GitHub Release asset SHA-256 checksums and, where available, platform signatures/notarization separately. A checksum is not an Authenticode signature. Release gates must refuse missing or empty Client/Server/CLI/Demo artifacts; source tags and historical builds are immutable.
- **Demo separation:** Ghost FTP Demo uses synthetic files and local in-memory progress, with no real credentials, remote connections or disk mutations. Never use demo behavior as proof of live throughput or production transfers.
- **Cross-platform compatibility:** Shared protocol contracts and common source versions are not a license to synchronize secret material by plaintext replication. Any future cross-device credential transfer requires explicit consent, authenticated encryption, integrity validation, rollback and separate acceptance tests.

## Reporting a security issue

Use the repository's **private vulnerability reporting** feature when it is enabled, or the publisher's established private support channel. If neither is available, open a non-sensitive public issue requesting a private reporting method **without** including exploit details, credentials, log tokens, private customer files or endpoint information.

A useful private report contains the affected component, precise version, operating system, security impact, safely reproducible steps and a minimized sanitized sample. Do not access accounts or systems you do not own or have explicit permission to test. Do not disclose a working exploit publicly before coordination. Reports are assessed by severity and affected scope; a fix or disclosure date cannot be guaranteed in advance.

## Public notices and privacy

See [privacy policy](docs/legal/PRIVACY.md), [EULA](EULA.txt), [license](LICENSE.txt), [Ghost FTP Server guide](ghost-server/README.md) and [release history](CHANGELOG.md). Public release notes focus on user-visible security changes and limitations; vulnerability-specific technical details are shared responsibly when appropriate.

This policy is informational and does not replace the commercial licence, contractual support terms or applicable law.
