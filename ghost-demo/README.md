# Ghost FTP Demo — isolated portable product walkthrough

Ghost FTP Demo is separately available for **Windows (portable EXE and unsigned Setup EXE)** and **Linux (standalone binary, tar.gz, DEB and RPM)**. Every package uses the same Go-embedded UI and fictional data; it is a **real self-contained executable**, not an FTP/SFTP network demo. It serves a native branded interface from its embedded assets on an ephemeral **127.0.0.1-only port**. No installer, remote host, credential, external service, browser plugin or network dependency is required.

Launch the EXE and open its local browser window. On remote/headless runners, use `-self-test`; the executable validates its own HTTP routes, fixtures and simulated lifecycle. `-version` reports the canonical CI-embedded version.

Demo contents are deliberately fictional: several connected-looking demonstration site profiles, project folders, example text/images/archives, a simulated upload/download queue, retry and cancellation and an explanatory sync preview. All transfer progress is simulated in memory; **no real FTP/FTPS/SFTP connection or filesystem mutation occurs**, and never silently touches saved production Ghost FTP credentials. Visual style shares the Ghost FTP approved desktop brand tokens and local SVG. UI prominently marks every simulated action as **DEMO / SIMULATION** to avoid misleading users.

The demo must never be treated as proof of actual live transfer throughput or platform implementation. Real client/server protocol QA remains in the native CI suite. Release pipeline verifies the binary, embedded version, self-test on Windows and SHA-256 like the other packages.

For development: `cd ghost-demo && go test ./... && go run .` (Go 1.23+).
