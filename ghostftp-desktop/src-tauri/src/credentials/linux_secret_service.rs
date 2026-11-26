//! Linux Secret Service over the official libsecret secret-tool client.
//!
//! No new Cargo lock dependencies; requires libsecret-tools plus a user-session
//! implementation of org.freedesktop.secrets (GNOME Keyring, KWallet).
//! Credentials are written via a private stdin pipe, NEVER via argv or env.
//! stdout contains a secret only during lookup, kept within Rust; stderr is
//! discarded to avoid echoing provider data into logs.
//! Legacy kernel keyutils entries are still resolved by the credentials layer.
//! This module must never erase the old backend during migration.

use anyhow::{anyhow, Result};
use std::io::{ErrorKind, Read, Write};
use std::os::fd::AsRawFd;
use std::process::{Child, Command, ExitStatus, Stdio};
use std::time::{Duration, Instant};
use zeroize::Zeroizing;

// Own captured data until it is explicitly transferred to a successful caller.
// Do not derive Debug: helper output may contain credentials or provider data.
struct HelperOutput {
    status: ExitStatus,
    stdout: Zeroizing<Vec<u8>>,
    stderr: Zeroizing<Vec<u8>>,
}

impl std::fmt::Debug for HelperOutput {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("HelperOutput")
            .field("status", &self.status)
            .finish_non_exhaustive()
    }
}

#[cfg(all(test, feature = "native-vault-candidate"))]
#[path = "credential_transaction_ledger.rs"]
mod credential_transaction_ledger;

#[cfg(all(test, feature = "native-vault-candidate"))]
#[path = "credential_mutation_journal.rs"]
mod credential_mutation_journal;

#[cfg(all(test, feature = "native-vault-candidate"))]
#[path = "native_vault_candidate.rs"]
mod native_vault_candidate;
#[cfg(all(test, feature = "native-vault-candidate"))]
pub(super) use native_vault_candidate::{clear_direct_candidate, configure_direct_candidate};

// GNOME libsecret's secret-tool currently reads at most 8192 bytes from a
// piped password; reject larger values instead of truncating private material.
const MAX_SECRET_BYTES: usize = 8191;
// A locked/unresponsive Secret Service must not indefinitely park a desktop
// command or leave a dangling credential helper on Windows/Linux clients.
const SECRET_TOOL_TIMEOUT: Duration = Duration::from_secs(30);
const CHILD_POLL_MIN_INTERVAL: Duration = Duration::from_millis(1);
const CHILD_POLL_MAX_INTERVAL: Duration = Duration::from_millis(20);

fn poll_delay(interval: Duration, remaining: Duration) -> Duration {
    interval.min(remaining)
}

fn next_poll_interval(interval: Duration) -> Duration {
    (interval * 2).min(CHILD_POLL_MAX_INTERVAL)
}

// Opt-in, test-only aggregate process timing. No credential bytes, identifiers
// or command arguments are recorded. Normal production builds retain no samples.
#[cfg(test)]
const PROFILE_CATEGORIES: [&str; 8] = [
    "transaction-key",
    "anchor",
    "manifest",
    "index-key",
    "blob",
    "gc-page",
    "gc-keys",
    "other",
];

#[cfg(test)]
fn profile_category(service: &str) -> usize {
    match service {
        "com.ghostftp.download-recovery.v1" => 0,
        "com.ghostftp.download-recovery-anchor.v1" => 1,
        "com.ghostftp.download-recovery-index.v1" => 2,
        "com.ghostftp.download-recovery-index-key.v1" => 3,
        "com.ghostftp.download-recovery-blob.v2" => 4,
        "com.ghostftp.download-recovery-gc.v1" => 5,
        "com.ghostftp.download-recovery-gc-keys.v1" => 6,
        _ => 7,
    }
}

#[cfg(test)]
struct HelperProfile {
    operations: [Vec<u128>; 3],
    categories: [[(usize, u128); 3]; 8],
}

#[cfg(test)]
std::thread_local! {
    static HELPER_PROFILE: std::cell::RefCell<Option<HelperProfile>> =
        const { std::cell::RefCell::new(None) };
}

#[cfg(test)]
pub(crate) fn start_linux_helper_profile() {
    HELPER_PROFILE.with(|profile| {
        *profile.borrow_mut() = Some(HelperProfile {
            operations: std::array::from_fn(|_| Vec::new()),
            categories: [[(0, 0); 3]; 8],
        });
    });
}

#[cfg(test)]
pub(crate) fn finish_linux_helper_profile() {
    HELPER_PROFILE.with(|profile| {
        if let Some(mut groups) = profile.borrow_mut().take() {
            for (category, counters) in PROFILE_CATEGORIES.into_iter().zip(groups.categories) {
                for (operation, (calls, total_us)) in
                    ["lookup", "store", "clear"].into_iter().zip(counters)
                {
                    if calls != 0 {
                        eprintln!(
                            "recovery_linux_helper_category category={category} operation={operation} calls={calls} total_us={total_us}"
                        );
                    }
                }
            }
            for (operation, samples) in ["lookup", "store", "clear"]
                .into_iter()
                .zip(&mut groups.operations)
            {
                if samples.is_empty() {
                    continue;
                }
                samples.sort_unstable();
                let count = samples.len();
                let total: u128 = samples.iter().sum();
                // Nearest rank, using the actual count and no interpolation.
                let p95 = samples[(count * 95).div_ceil(100) - 1];
                let p99 = samples[(count * 99).div_ceil(100) - 1];
                eprintln!(
                    "recovery_linux_helper operation={operation} calls={count} total_us={total} median_us={} p95_us={p95} p99_us={p99} max_us={}",
                    samples[count / 2],
                    samples[count - 1]
                );
            }
        }
    });
}

fn profile_helper(
    _operation: usize,
    _service: &str,
    run: impl FnOnce() -> Result<HelperOutput>,
) -> Result<HelperOutput> {
    #[cfg(test)]
    let started = Instant::now();
    let result = run();
    #[cfg(test)]
    HELPER_PROFILE.with(|profile| {
        if let Some(groups) = profile.borrow_mut().as_mut() {
            let elapsed = started.elapsed().as_micros();
            groups.operations[_operation].push(elapsed);
            let counter = &mut groups.categories[profile_category(_service)][_operation];
            counter.0 += 1;
            counter.1 += elapsed;
        }
    });
    result
}

// Drain both helper pipes before exit, or the OS pipe buffer can stall the child.
// Refuse an unbounded amount of subprocess output.
const MAX_HELPER_STDERR_BYTES: usize = 4096;

// Set O_NONBLOCK on the parent's pipe endpoints, not on the helper's
// separate stdin/stdout/stderr endpoints.
fn set_nonblocking(pipe: &impl AsRawFd) -> Result<()> {
    let fd = pipe.as_raw_fd();
    // SAFETY: the borrowed pipe owns a valid descriptor throughout both calls;
    // F_GETFL takes no third argument and F_SETFL takes integer status flags.
    let flags = unsafe { libc::fcntl(fd, libc::F_GETFL) };
    if flags == -1 {
        return Err(anyhow!("Linux Secret Service pipe flags unavailable"));
    }
    if unsafe { libc::fcntl(fd, libc::F_SETFL, flags | libc::O_NONBLOCK) } == -1 {
        return Err(anyhow!("Linux Secret Service nonblocking pipe unavailable"));
    }
    Ok(())
}

// Drain in bounded batches. Retain at most max_bytes + 1 to distinguish
// oversized output without prematurely closing a producing helper's pipe.
fn drain_pipe<P: Read>(
    pipe: &mut Option<P>,
    output: &mut Zeroizing<Vec<u8>>,
    max_bytes: usize,
) -> Result<bool> {
    let mut progressed = false;
    let Some(reader) = pipe.as_mut() else {
        return Ok(false);
    };
    let mut buffer = Zeroizing::new([0u8; 8192]);
    for _ in 0..8 {
        match reader.read(&mut buffer[..]) {
            Ok(0) => {
                *pipe = None;
                return Ok(true);
            }
            Ok(count) => {
                let retain = (max_bytes + 1 - output.len()).min(count);
                output.extend_from_slice(&buffer[..retain]);
                progressed = true;
            }
            Err(error) if error.kind() == ErrorKind::Interrupted => continue,
            Err(error) if error.kind() == ErrorKind::WouldBlock => break,
            Err(_) => return Err(anyhow!("Linux Secret Service output could not be read")),
        }
    }
    Ok(progressed)
}

// One deadline covers process exit AND pipe EOF. Inherited descriptors held
// by descendants cannot strand a scoped reader/writer thread after child exit.
fn supervise_helper(
    child: &mut Child,
    secret: Option<&[u8]>,
    timeout: Duration,
) -> Result<HelperOutput> {
    let started = Instant::now();
    let mut stdin = child.stdin.take();
    let mut stdout = child.stdout.take();
    let mut stderr = child.stderr.take();
    if secret.is_some() && stdin.is_none() {
        return Err(anyhow!("Linux Secret Service pipe unavailable"));
    }
    if secret.is_none() {
        stdin = None;
    }
    if let Some(pipe) = &stdin {
        set_nonblocking(pipe)?;
    }
    if let Some(pipe) = &stdout {
        set_nonblocking(pipe)?;
    }
    if let Some(pipe) = &stderr {
        set_nonblocking(pipe)?;
    }

    let input = secret.unwrap_or_default();
    let mut written = 0;
    // Reserve the entire bounded capture before reading any bytes, avoiding
    // reallocations that could leave earlier secret copies in freed memory.
    let mut out = Zeroizing::new(Vec::with_capacity(if stdout.is_some() {
        MAX_SECRET_BYTES + 1
    } else {
        0
    }));
    let mut err = Zeroizing::new(Vec::with_capacity(if stderr.is_some() {
        MAX_HELPER_STDERR_BYTES + 1
    } else {
        0
    }));
    let mut status = None;
    let mut interval = CHILD_POLL_MIN_INTERVAL;
    loop {
        if started.elapsed() >= timeout {
            return Err(anyhow!(
                "Linux Secret Service timed out; unlock the keyring"
            ));
        }
        let mut progressed = drain_pipe(&mut stdout, &mut out, MAX_SECRET_BYTES)?;
        progressed |= drain_pipe(&mut stderr, &mut err, MAX_HELPER_STDERR_BYTES)?;
        if let Some(pipe) = stdin.as_mut() {
            if written < input.len() {
                match pipe.write(&input[written..]) {
                    Ok(0) => return Err(anyhow!("Linux Secret Service refused credential input")),
                    Ok(count) => {
                        written += count;
                        progressed = true;
                    }
                    Err(error)
                        if matches!(
                            error.kind(),
                            ErrorKind::WouldBlock | ErrorKind::Interrupted
                        ) => {}
                    Err(_) => return Err(anyhow!("Linux Secret Service refused credential input")),
                }
            }
            if written == input.len() {
                stdin = None;
                progressed = true;
            }
        }
        if status.is_none() {
            status = child
                .try_wait()
                .map_err(|_| anyhow!("Linux Secret Service process failed"))?;
        }
        if let Some(exit) = status {
            if stdin.is_some() {
                return Err(anyhow!("Linux Secret Service refused credential input"));
            }
            if stdout.is_none() && stderr.is_none() {
                return Ok(HelperOutput {
                    status: exit,
                    stdout: out,
                    stderr: err,
                });
            }
        }
        if progressed {
            interval = CHILD_POLL_MIN_INTERVAL;
        } else {
            std::thread::sleep(poll_delay(
                interval,
                timeout.saturating_sub(started.elapsed()),
            ));
            interval = next_poll_interval(interval);
        }
    }
}

fn run_bounded(mut child: Child, secret: Option<&[u8]>, timeout: Duration) -> Result<HelperOutput> {
    let result = supervise_helper(&mut child, secret, timeout);
    if result.is_err() {
        // Pipe endpoints have already been dropped. Terminate/reap only our
        // direct child; never signal an unrelated process or credential daemon.
        let _ = child.kill();
        let _ = child.wait();
    }
    result
}

fn wait_bounded(child: Child, timeout: Duration) -> Result<HelperOutput> {
    run_bounded(child, None, timeout)
}

fn write_then_wait_bounded(child: Child, secret: &[u8], timeout: Duration) -> Result<HelperOutput> {
    run_bounded(child, Some(secret), timeout)
}

fn command(operation: &str, service: &str, account: &str) -> Command {
    let mut command = Command::new("secret-tool");
    command
        .arg(operation)
        .arg("--") // Prevent untrusted profile ids from becoming CLI options.
        .args([
            "application",
            "ghostftp",
            "service",
            service,
            "account",
            account,
        ])
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());
    command
}

pub fn lookup(service: &str, account: &str) -> Result<Option<String>> {
    #[cfg(all(test, feature = "native-vault-candidate"))]
    if native_vault_candidate::enabled() {
        return native_vault_candidate::lookup(service, account);
    }
    let output = profile_helper(0, service, || {
        let child = command("lookup", service, account)
            .spawn()
            .map_err(|_| anyhow!("Linux Secret Service client not installed or unavailable"))?;
        wait_bounded(child, SECRET_TOOL_TIMEOUT)
    })?;
    decode_lookup(output)
}

fn decode_lookup(output: HelperOutput) -> Result<Option<String>> {
    if output.status.success() {
        if output.stdout.len() > MAX_SECRET_BYTES {
            return Err(anyhow!(
                "Linux Secret Service credential exceeds maximum length"
            ));
        }
        return std::str::from_utf8(&output.stdout)
            .map(|value| Some(value.to_owned()))
            .map_err(|_| anyhow!("Linux Secret Service returned a non-textual credential"));
    }
    // secret-tool exits 1 with empty stderr when no entry exists. All other
    // failure causes (locked vault, D-Bus errors) fail closed, not as NoEntry.
    if output.status.code() == Some(1) && output.stderr.is_empty() {
        return Ok(None);
    }
    Err(anyhow!(
        "Linux Secret Service lookup failed or vault is locked"
    ))
}

pub fn store(service: &str, account: &str, secret: &str) -> Result<()> {
    #[cfg(all(test, feature = "native-vault-candidate"))]
    if native_vault_candidate::enabled() {
        return native_vault_candidate::store(service, account, secret);
    }
    if secret.is_empty() || secret.len() > MAX_SECRET_BYTES {
        return Err(anyhow!("Invalid Linux Secret Service credential length"));
    }
    let mut command = Command::new("secret-tool");
    command
        .arg("store")
        .arg("--label=Ghost FTP")
        .arg("--")
        .args([
            "application",
            "ghostftp",
            "service",
            service,
            "account",
            account,
        ])
        .stdin(Stdio::piped())
        .stdout(Stdio::null())
        .stderr(Stdio::null());
    let output = profile_helper(1, service, || {
        let child = command.spawn().map_err(|_| {
            anyhow!("Linux Secret Service client is unavailable; install libsecret-tools")
        })?;
        write_then_wait_bounded(child, secret.as_bytes(), SECRET_TOOL_TIMEOUT)
    })?;
    if !output.status.success() {
        return Err(anyhow!(
            "Linux Secret Service refused to save the credential"
        ));
    }
    match lookup(service, account)?.map(Zeroizing::new) {
        Some(saved) if saved.as_str() == secret => Ok(()),
        _ => Err(anyhow!("Linux Secret Service write could not be verified")),
    }
}

pub fn remove(service: &str, account: &str) -> Result<()> {
    #[cfg(all(test, feature = "native-vault-candidate"))]
    if native_vault_candidate::enabled() {
        return native_vault_candidate::remove(service, account);
    }
    let output = profile_helper(2, service, || {
        let child = command("clear", service, account)
            .spawn()
            .map_err(|_| anyhow!("Linux Secret Service client unavailable during deletion"))?;
        wait_bounded(child, SECRET_TOOL_TIMEOUT)
    })?;
    if !output.status.success() {
        return Err(anyhow!("Linux Secret Service refused credential deletion"));
    }
    if lookup(service, account)?.map(Zeroizing::new).is_some() {
        return Err(anyhow!(
            "Linux Secret Service credential still exists after deletion"
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn fixture_output(code: i32, stdout: &[u8], stderr: &[u8]) -> HelperOutput {
        use std::os::unix::process::ExitStatusExt;

        HelperOutput {
            status: ExitStatus::from_raw(code << 8),
            stdout: Zeroizing::new(stdout.to_vec()),
            stderr: Zeroizing::new(stderr.to_vec()),
        }
    }

    #[test]
    fn lookup_decode_preserves_exact_bytes_and_error_semantics() {
        let valid = fixture_output(0, "fixture-ž\n".as_bytes(), b"");
        assert_eq!(
            decode_lookup(valid).unwrap().as_deref(),
            Some("fixture-ž\n")
        );
        assert!(decode_lookup(fixture_output(0, &[0xff], b"")).is_err());
        assert!(decode_lookup(fixture_output(0, &vec![0; MAX_SECRET_BYTES + 1], b"")).is_err());
        assert!(decode_lookup(fixture_output(1, b"", b""))
            .unwrap()
            .is_none());
        assert!(decode_lookup(fixture_output(1, b"", b"locked")).is_err());
        assert!(decode_lookup(fixture_output(2, b"", b"")).is_err());
    }

    #[test]
    fn helper_output_debug_never_contains_captured_bytes() {
        let output = fixture_output(1, b"PRIVATE_STDOUT_SENTINEL", b"PRIVATE_STDERR_SENTINEL");
        let debug = format!("{output:?}");
        assert!(!debug.contains("PRIVATE_STDOUT_SENTINEL"));
        assert!(!debug.contains("PRIVATE_STDERR_SENTINEL"));
    }

    #[test]
    fn helper_profile_counts_only_fixed_categories_and_operations() {
        start_linux_helper_profile();
        let service = "com.ghostftp.download-recovery-blob.v2";
        assert!(profile_helper(0, service, || Err(anyhow!("fixture"))).is_err());
        assert!(profile_helper(1, "unrecognized-service", || Err(anyhow!("fixture"))).is_err());
        HELPER_PROFILE.with(|profile| {
            let profile = profile.borrow();
            let profile = profile.as_ref().unwrap();
            assert_eq!(profile.operations[0].len(), 1);
            assert_eq!(profile.operations[1].len(), 1);
            assert_eq!(profile.categories[4][0].0, 1);
            assert_eq!(profile.categories[7][1].0, 1);
            assert_eq!(profile.categories[4][1].0, 0);
        });
        finish_linux_helper_profile();
        HELPER_PROFILE.with(|profile| assert!(profile.borrow().is_none()));
    }

    #[test]
    #[ignore = "requires an isolated unlocked native Secret Service"]
    fn native_strict_delete_retains_verified_absence_with_two_lookups() {
        assert_eq!(
            std::env::var("GHOSTFTP_ACTIVE_RECOVERY_BENCH_OPT_IN")
                .ok()
                .as_deref(),
            Some("disposable-fixtures")
        );
        let root =
            std::env::temp_dir().join(format!("ghostftp-delete-bench-{}", uuid::Uuid::new_v4()));
        super::super::configure_native_recovery_test_vault(root.clone(), None);
        let result = std::panic::catch_unwind(|| {
            let service = "com.ghostftp.download-recovery.v1";
            let account = uuid::Uuid::new_v4().to_string();
            super::super::vault_set(service, &account, "test-only-value").unwrap();
            for expected_clears in [1, 0] {
                start_linux_helper_profile();
                super::super::vault_delete_strict(service, &account).unwrap();
                HELPER_PROFILE.with(|profile| {
                    let profile = profile.borrow();
                    let profile = profile.as_ref().unwrap();
                    assert_eq!(profile.operations[0].len(), 2);
                    assert_eq!(profile.operations[1].len(), 0);
                    assert_eq!(profile.operations[2].len(), expected_clears);
                });
                finish_linux_helper_profile();
                assert!(lookup(service, &account).unwrap().is_none());
            }
        });
        super::super::cleanup_native_recovery_test_vault(&root).unwrap();
        super::super::clear_recovery_test_vault();
        std::fs::remove_dir_all(root).unwrap();
        if let Err(error) = result {
            std::panic::resume_unwind(error);
        }
    }

    #[test]
    fn adaptive_polling_is_capped_and_respects_remaining_deadline() {
        let mut interval = CHILD_POLL_MIN_INTERVAL;
        for expected in [1, 2, 4, 8, 16, 20, 20] {
            assert_eq!(interval, Duration::from_millis(expected));
            interval = next_poll_interval(interval);
        }
        assert_eq!(
            poll_delay(interval, Duration::from_millis(3)),
            Duration::from_millis(3)
        );
        assert_eq!(poll_delay(interval, Duration::ZERO), Duration::ZERO);
    }

    #[test]
    fn invalid_secret_inputs_are_rejected_before_launching_any_process() {
        assert!(store("service", "account", "").is_err());
        assert!(store("service", "account", &"x".repeat(MAX_SECRET_BYTES + 1)).is_err());
        assert!(store("service", "account", &"x".repeat(MAX_SECRET_BYTES + 200)).is_err());
    }

    #[test]
    fn blocked_credential_helper_times_out_without_hanging() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "exec sleep 2"])
            .stdout(Stdio::null())
            .stderr(Stdio::null());
        let started = Instant::now();
        let child = helper.spawn().expect("launch temporary blocked command");
        let err = wait_bounded(child, Duration::from_millis(80)).unwrap_err();
        assert!(err.to_string().contains("timed out"));
        assert!(started.elapsed() < Duration::from_secs(2));
    }

    #[test]
    fn blocked_stdin_write_is_interrupted_by_the_process_deadline() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "exec sleep 2"])
            .stdin(Stdio::piped())
            .stdout(Stdio::null())
            .stderr(Stdio::null());
        let started = Instant::now();
        let child = helper.spawn().expect("launch unresponsive helper");
        // Larger than a typical Linux pipe buffer: the timeout must still fire.
        let input = vec![b'x'; 512 * 1024];
        let err = write_then_wait_bounded(child, &input, Duration::from_millis(80))
            .expect_err("hung helper must time out");
        assert!(err.to_string().contains("timed out"));
        assert!(started.elapsed() < Duration::from_secs(2));
    }

    #[test]
    fn credential_input_writer_closes_stdin_after_success() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "cat >/dev/null"])
            .stdin(Stdio::piped())
            .stdout(Stdio::null())
            .stderr(Stdio::null());
        let child = helper.spawn().expect("launch input consumer");
        let output = write_then_wait_bounded(child, b"test-secret", Duration::from_secs(2))
            .expect("finish writing helper input");
        assert!(output.status.success());
    }

    #[test]
    fn oversized_helper_output_is_drained_without_unbounded_allocation() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "head -c 8193 /dev/zero"])
            .stdout(Stdio::piped())
            .stderr(Stdio::null());
        let child = helper.spawn().expect("launch output fixture");
        let output =
            wait_bounded(child, Duration::from_secs(2)).expect("drain a large credential response");
        assert!(output.status.success());
        assert_eq!(output.stdout.len(), MAX_SECRET_BYTES + 1);
    }

    #[test]
    fn large_helper_output_is_fully_drained_while_memory_stays_bounded() {
        // A producer must not see a prematurely closed stdout/stderr pipe,
        // even after both credential-sized capture limits have been reached.
        // This also confirms the discarded data never accumulates in memory.
        let mut helper = Command::new("sh");
        helper
            .args([
                "-c",
                "head -c 1048576 /dev/zero; head -c 1048576 /dev/zero >&2",
            ])
            .stdout(Stdio::piped())
            .stderr(Stdio::piped());
        let child = helper.spawn().expect("launch large output fixture");
        let output = wait_bounded(child, Duration::from_secs(10)).expect("drain both output pipes");
        assert!(output.status.success());
        assert_eq!(output.stdout.len(), MAX_SECRET_BYTES + 1);
        assert_eq!(output.stderr.len(), MAX_HELPER_STDERR_BYTES + 1);
    }

    #[test]
    fn inherited_output_pipes_cannot_extend_helper_deadline() {
        for script in ["sleep 2 & exit 0", "sleep 2 >&2 & exit 0"] {
            let child = Command::new("sh")
                .args(["-c", script])
                .stdout(Stdio::piped())
                .stderr(Stdio::piped())
                .spawn()
                .unwrap();
            let started = Instant::now();
            let error = wait_bounded(child, Duration::from_millis(80)).unwrap_err();
            assert!(error.to_string().contains("timed out"));
            assert!(started.elapsed() < Duration::from_secs(2));
        }
    }

    #[test]
    fn inherited_stdin_cannot_block_an_incomplete_write_after_exit() {
        let child = Command::new("sh")
            .args(["-c", "sleep 2 <&0 & exit 0"])
            .stdin(Stdio::piped())
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .spawn()
            .unwrap();
        let input = vec![b'x'; 512 * 1024];
        let started = Instant::now();
        assert!(write_then_wait_bounded(child, &input, Duration::from_millis(80)).is_err());
        assert!(started.elapsed() < Duration::from_secs(2));
    }

    #[test]
    fn completed_credential_helper_retains_exact_stdout_bytes() {
        let mut helper = Command::new("sh");
        helper
            .args(["-c", "printf 'secret-with-newline\\n'"])
            .stdout(Stdio::piped())
            .stderr(Stdio::null());
        let child = helper.spawn().expect("launch local output fixture");
        let output = wait_bounded(child, Duration::from_secs(2)).unwrap();
        assert!(output.status.success());
        assert_eq!(output.stdout.as_slice(), b"secret-with-newline\n");
    }
}
