# Shared-hosting compatibility

ByFTP is designed to work with common shared-hosting FTP/FTPS accounts without assuming a Unix shell or administrator access.

## Recommended settings

Use the exact protocol, host, port and username supplied by the hosting provider. Typical values are FTP on port 21, explicit FTPS on port 21, implicit FTPS on port 990, and SFTP on port 22, but provider configuration always wins.

ByFTP uses passive FTP data transfers and does not place passwords on the process command line. FTP/FTPS credentials are delivered to curl through its controlled configuration input.

## Operations

The client supports listing, upload, download, directory creation, rename, delete and permission changes where the server supports them. Transfers use hidden/staged temporary objects and commit/revalidation logic where applicable to reduce partial-file exposure.

A hosting provider may still deny rename, CHMOD, delete, overwrite, hidden temporary files or concurrent sessions. Such restrictions are reported as server-side failures; ByFTP does not bypass them.

## Tested protocol path

The automated test suite contains a real loopback FTP integration test. It starts an authenticated FTP server on `127.0.0.1` and drives the production curl-backed adapter through actual FTP control and passive data connections. The workflow covers authentication, listing, directory operations, upload, rename, byte-for-byte download, delete and cleanup, plus a wrong-password negative case.

This deterministic test proves the client/server FTP path without storing public credentials or contacting an external service. It is not a claim that every hosting provider has identical policy.

## Troubleshooting shared hosting

If login works but listing or transfer fails, check passive FTP/firewall rules and provider session limits. If only overwrite/rename/delete fails, verify directory permissions and hosting policy. If FTPS fails, verify the provider certificate chain and correct TLS mode rather than disabling certificate checks.