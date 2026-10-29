# Ghost FTP Server — Linux Preview packages

The standalone server is separate from all Ghost FTP clients. These packages are **development Previews**, not security-certified production deployments. They install a real Go executable and a hardened systemd service, but never enable/start it or open a firewall port. FTPS-only, read-only and loopback-only remain the defaults.

Packages: standalone `GhostFTP-Server-Linux-x86_64-vVERSION`, `.tar.gz` (includes unit and documentation), Debian/Ubuntu `.deb`, and Fedora `.rpm`. Windows portable and unsigned NSIS setup are released alongside these.

For system packages, an unprivileged `ghostftp` system user and private `/var/lib/ghostftp-server/files` directory are created if absent. Create `/etc/ghostftp-server` (root-controlled); provision a publicly trusted TLS certificate/key and an unpredictable 12–256 byte password in an owner-private file at the paths used by the unit. **Do not use a test certificate or a shared/default password.** Ensure the restricted `ghostftp` identity has only the required read access to the credential material, and verify mode/ACLs. The package does not create, rotate, delete or overwrite certificates, secret files or served data.

Review `systemctl cat ghostftp-server`, `systemctl edit ghostftp-server` for safe server-specific paths and credentials, validate permissions, then run `sudo systemctl daemon-reload` and `sudo systemctl enable --now ghostftp-server` only after provisioning. The default listener binds `127.0.0.1:2121`; public exposure requires independent security review, certificate identity checks, firewall policy and explicit administrative approval.

Use `journalctl -u ghostftp-server` for service failures (never log secrets). The unit is fail-closed when certificates, key or password are missing. Avoid `-allow-uploads` or `-allow-destructive` on externally writable or unreviewed roots; the default is read-only.

Uninstalling the binary must not remove the account, private configuration, certificates, password, logs, or served data. Package upgrades must never silently start the service. No SFTP server is present in this Preview.
