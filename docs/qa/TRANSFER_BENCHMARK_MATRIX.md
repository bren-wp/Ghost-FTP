# Windows/Linux desktop transfer acceptance — reproducible workload

This is a **test fixture and measurement procedure**, not evidence that any
cloud provider has passed. The script uses no credentials and no network.
Run the actual Ghost FTP desktop transfer separately against your *authorized*
test servers; never enter real users' buckets or disclose access keys in logs.

## Create sources and validate downloads

```bash
python tools/benchmarks/desktop-transfer-workload.py prepare --root /path/to/new/ghostftp-source --small-count 10000
python tools/benchmarks/desktop-transfer-workload.py prepare --root /path/to/new/ghostftp-large --small-count 1 --large
python tools/benchmarks/desktop-transfer-workload.py verify --root /path/to/downloaded-copy --manifest /path/to/ghostftp-source/manifest.json
```

* `--root` must not already exist. No files are removed or overwritten.
* The >4 GiB sparse file is **optional**; hashing/copying/transferring it reads
  real 4+ GiB and may incur cloud costs and substantial runtime.
* The prepared `manifest.json` records SHA-256 checksums. Verify each downloaded
  file against the original manifest, preserving names and bytes.
* Repeat each protocol/direction with concurrency **1, 4, 8, 16, 32**, and
  simulate disconnects, retries, 429, 5xx, authentication failures, disk full,
  remote conflicts and cancellations. Audit the final remote listing for orphans.

## Measure the desktop process

On the target machine, install optional `psutil` in the test Python environment.
Record actual Ghost FTP process PID, machine CPU/RAM, OS, backend, server version,
network link, TLS/SSH configuration, timestamps, and dataset hashes.

```bash
python tools/benchmarks/desktop-transfer-workload.py observe --pid 12345 --seconds 60 --parallelism 8
```

This samples **target process** CPU seconds and RSS, but it **does not** measure
GUI event latency and does not know real bytes transferred; manual byte counts
must be corroborated from validated source/destination manifests. For UI
responsiveness collect actual interactive input-to-paint traces separately.
The `--expected-bytes` derived throughput is only a theoretical label,
**not** verified transfer throughput.

## Acceptance matrix

| Backend | Live account and server required | Resume evidence | Integrity evidence | Status |
| --- | --- | --- | --- | --- |
| FTP / explicit FTPS | Authorized test server with controlled REST/STOR and TLS | Network abort + file identity/restart | SHA-256 before/after | Pending |
| SFTP | Authorized host with strict pinned known-host key | Interrupted read/write + restarted process | SHA-256 | Pending |
| AWS S3 / B2 (S3 API) | Separate test buckets, multipart and abort permissions | Failed part, orphan cleanup, new attempt | Whole-object SHA-256 | Pending |
| Google Cloud Storage | Authorized service account and bucket | Multipart error/restart | Whole-object SHA-256 | Pending |
| WebDAV / Nextcloud | Test endpoint/account | 429/503/timeout and conflict | SHA-256 | Pending |
| OneDrive / Box / Google Drive | Authorized test tenants | Upload session/retry and changed files | SHA-256 | Pending |

Other implemented provider integrations also require independent account fixtures
and provider-specific limits. Tests written without accounts are not completion.
Record each failing/successful run against a **specific PR SHA** and attach logs
with all tokens, key material and personal paths redacted.

## FileZilla comparison

Only compare with FileZilla on the same test machine, server, data set,
network, security settings, concurrency and cold/warm cache conditions.
Report repetitions, median and variability. **No FileZilla numbers are
supplied here because no real paired experiments have been conducted.**
