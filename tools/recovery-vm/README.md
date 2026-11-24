# Recovery storage hard-reset laboratory

Run only in a dedicated disposable Linux or Windows VM under libvirt. A forced power-off is destructive to the guest. Never use a workstation, shared VM, production domain, or a VM with real Ghost FTP credentials. The host requires Python 3, virsh, SSH with pinned guest host key and authorization to power off the designated test VM. The harness does not reboot the host.

Provision the same exact source revision and build the Rust test binary with `cargo test --locked -p ghostftp --lib --no-run`. Keep the compiled binary and dedicated existing fixture parent directory across reboots. Configure the test user to access its own persistent credential store after boot. On Linux install secret-tool and arrange an unlocked persistent Secret Service on the guest user bus (not a temporary in-memory keyring). Pass the user bus address with --linux-dbus-address when SSH does not inherit it. On Windows use an authenticated test-user session that can access that user's Credential Manager; verify this with the native probes before testing. Do not bypass these prerequisites or disable vault errors.

Name the VM `ghostftp-durability-linux` or `ghostftp-durability-windows` and record its libvirt UUID. For each case, use a fresh run and a separate report path. Example (replace the UUID, SSH target, and binary path with actual values):

```sh
python3 tools/recovery-vm/verify-hard-reset.py \
  --domain ghostftp-durability-linux \
  --uuid "$TEST_VM_UUID" --confirm-reset "$TEST_VM_UUID" \
  --ssh-target test@guest --platform linux \
  --guest-root /var/tmp/ghostftp-fixtures \
  --test-binary /opt/ghostftp/ghostftp_lib-test \
  --linux-dbus-address unix:path=/run/user/1000/bus \
  --case stage_published --report stage-published-linux.json
```

Windows: use --platform windows with absolute guest paths such as `C:\GhostFTPTest\fixtures` and the full test executable path. The SSH adapter invokes PowerShell with an encoded command and requires OpenSSH access. The test only cleans its registered native credentials after successful verification. The uniquely named guest fixture directory is retained for investigation. A failed run retains evidence; stop the test guest/process as needed before any further run. Restore a clean disposable guest snapshot between test series.

The cases are prepared_sql_committed, backup_renamed, stage_published, backup_removed, terminal_manifest_written and rollback:rollback_restored. The host observes the exact checkpoint, verifies the libvirt UUID/name again, forces power-off, starts the guest, verifies a changed guest boot identity, then runs the recovery verifier. The JSON report records these steps and errors. A test returning no matching success marker is a failure. The verifier does not automatically mutate user files; it checks authenticated ledger reconciliation, original/new byte preservation, absence of unexpected bytes, and repeatability.

A libvirt hard power-off tests abrupt **guest** shutdown. It does not prove loss of host power, volatile controller cache behavior, or durability of every filesystem. Use independent disposable disks and record hypervisor cache mode, filesystem, virtual disk driver, OS build, source SHA and storage configuration with the report. Repeat on the supported Windows and Linux filesystems and explicitly record failures. Real host power-loss testing requires a separately authorized laboratory and is not performed by this harness.

The native end-to-end benchmark in ordinary platform CI measures local journal-backed overwrites (1 MiB and 16 MiB, three samples per size), not network throughput. The manifest benchmark measures 1/8/32/128 synthetic index entries. Do not describe either as measured until logs exist.
