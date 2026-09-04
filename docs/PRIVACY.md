# Ghost FTP privacy

Ghost FTP is designed without application telemetry in production builds. The application does not intentionally send product-usage analytics, advertising identifiers or crash telemetry to a Ghost FTP-operated collection service.

## Network traffic

Ghost FTP necessarily connects to servers selected by the user for FTP, FTPS or SFTP operations. Those servers and network intermediaries can observe connection metadata according to the selected protocol.

Plain FTP is unencrypted. Prefer FTPS or SFTP when the remote service supports them.

## Credentials and profiles

Connection profiles can contain hostnames, usernames, paths and authentication material. Secrets are handled as sensitive data and platform-specific protection is used where available. Do not commit exported profiles, configuration storage or credentials to source control.

SFTP host-key fingerprints should be verified before trusting a server. A fingerprint change should be treated as a security event until independently explained.

## Web deployment

The web/PWA client stores application configuration and per-user data on the operator's own hosting environment. Operators are responsible for protecting that hosting account, TLS configuration, backups and application storage permissions.

The web application is intentionally configured not to be indexed by search engines. It also uses restrictive session/cookie/security controls, but those controls do not replace secure server administration.

## Build privacy

Production build scripts disable Go telemetry and use controlled dependency behavior. CI contains privacy auditing intended to detect accidental telemetry or tracking additions.

## Third parties

FTP/FTPS/SFTP servers, hosting providers, operating-system vendors, GitHub and app-distribution platforms have their own privacy practices. Ghost FTP cannot control data processed independently by those services.
