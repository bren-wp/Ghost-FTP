// Ghost FTP by Brendigo
// Native real-socket regression helpers kept separate from protocol_e2e.rs.
// Test-only disposable paths and credentials; no production endpoint usage.
use super::*;

/// Exercise the same authenticated FTP/explicit-FTPS session instance across
/// reconnections and prove that explicit Disconnect is terminal. Opening the
/// saved *profile metadata* again must yield a different live session ID.
/// This is not a credential-vault persistence or native GUI acceptance test.
pub(super) async fn verify_ftp_reconnect_and_disconnect(
    protocol: &str,
    host: &str,
    port_number: u16,
    username: &str,
    password: &str,
) -> Result<()> {
    let p = profile(
        protocol,
        host,
        port_number,
        username,
        AuthMethod::Password {
            password: password.to_owned(),
        },
    );
    let session = open_session(&p, Arc::new(FixedVerifier(HostDecision::Accept)))
        .await
        .with_context(|| format!("{protocol} connect for lifecycle test"))?;
    let Session::Ftp(ftp) = session else {
        return Err(anyhow!("{protocol} did not create FTP/FTPS session"));
    };
    let original_id = ftp.id.clone();
    let base = format!("ghostftp-lifecycle-{}", Uuid::new_v4().simple());
    let remote = format!("{base}/sentinel.bin");
    let expected = simulated_transfer_data(128 * 1024 + 17);
    let fs = FtpFs::new(ftp.clone());
    fs.create_dir(&base)
        .await
        .with_context(|| format!("{protocol} lifecycle mkdir"))?;

    let upload_path = remote.clone();
    let upload_bytes = expected.clone();
    ftp.with_stream(move |stream| {
        let mut reader = Cursor::new(upload_bytes);
        stream.put_from_reader(&upload_path, &mut reader)?;
        Ok(())
    })
    .await
    .with_context(|| format!("{protocol} lifecycle upload"))?;

    for pass in 0..2 {
        ftp.reconnect()
            .await
            .with_context(|| format!("{protocol} authenticated reconnect pass {pass}"))?;
        anyhow::ensure!(
            ftp.id == original_id,
            "{protocol} reconnect incorrectly changed the existing session identity"
        );
        let read_path = remote.clone();
        let downloaded = ftp
            .with_stream(move |stream| {
                let mut bytes = Vec::new();
                stream.retr_to_writer(&read_path, &mut bytes)?;
                Ok(bytes)
            })
            .await
            .with_context(|| format!("{protocol} read original after reconnect pass {pass}"))?;
        anyhow::ensure!(
            downloaded == expected,
            "{protocol} file integrity failed after authenticated reconnect pass {pass}"
        );
    }

    ftp.shutdown()
        .await
        .with_context(|| format!("{protocol} explicit disconnect"))?;
    anyhow::ensure!(
        ftp.reconnect().await.is_err(),
        "{protocol} explicitly disconnected session was resurrected"
    );
    anyhow::ensure!(
        ftp.with_stream(|stream| {
            stream.noop()?;
            Ok(())
        })
        .await
        .is_err(),
        "{protocol} accepted a command after explicit disconnect"
    );

    // Same profile, new explicit Connect. This must not revive the old ID.
    let opened = open_session(&p, Arc::new(FixedVerifier(HostDecision::Accept)))
        .await
        .with_context(|| format!("{protocol} reconnect profile through fresh session"))?;
    let Session::Ftp(fresh) = opened else {
        return Err(anyhow!("{protocol} fresh profile login returned non-FTP"));
    };
    anyhow::ensure!(
        fresh.id != original_id,
        "{protocol} new explicit Connect reused an already disconnected session ID"
    );
    let read_path = remote.clone();
    let downloaded = fresh
        .with_stream(move |stream| {
            let mut bytes = Vec::new();
            stream.retr_to_writer(&read_path, &mut bytes)?;
            Ok(bytes)
        })
        .await
        .with_context(|| format!("{protocol} fresh connection download"))?;
    anyhow::ensure!(
        downloaded == expected,
        "{protocol} saved-profile new connection damaged original content"
    );
    let fresh_fs = FtpFs::new(fresh.clone());
    fresh_fs.delete(&remote, false).await?;
    fresh_fs.delete(&base, false).await?;
    fresh.shutdown().await?;
    Ok(())
}

/// Source-I/O fault during a real streaming STOR into a unique staging sibling:
/// verify an existing remote destination remains byte-for-byte intact after
/// control-channel recovery, remove the partial sibling, and prove no orphan
/// remains. This does not simulate a physical network drop or full process kill.
pub(super) async fn verify_staged_upload_source_fault(
    protocol: &str,
    host: &str,
    port_number: u16,
    username: &str,
    password: &str,
) -> Result<()> {
    let p = profile(
        protocol,
        host,
        port_number,
        username,
        AuthMethod::Password {
            password: password.to_owned(),
        },
    );
    let session = open_session(&p, Arc::new(FixedVerifier(HostDecision::Accept)))
        .await
        .with_context(|| format!("{protocol} staging-failure connect"))?;
    let Session::Ftp(ftp) = session else {
        return Err(anyhow!("{protocol} staging fixture did not open FTP/FTPS"));
    };
    let fs = FtpFs::new(ftp.clone());
    let base = format!("ghostftp-staging-fault-{}", Uuid::new_v4().simple());
    let destination = format!("{base}/existing.bin");
    let staging = format!("{base}/.ghostftp-upload-fault.part");
    let sentinel = simulated_transfer_data(128 * 1024 + 17);
    fs.create_dir(&base)
        .await
        .with_context(|| format!("{protocol} staging fixture mkdir"))?;
    let destination_for_seed = destination.clone();
    let sentinel_for_seed = sentinel.clone();
    ftp.with_stream(move |stream| {
        let mut reader = Cursor::new(sentinel_for_seed);
        stream.put_from_reader(&destination_for_seed, &mut reader)?;
        Ok(())
    })
    .await
    .with_context(|| format!("{protocol} seed existing destination"))?;

    let fault_path = staging.clone();
    let payload = simulated_transfer_data(2 * 1024 * 1024 + 257);
    let failed = ftp
        .with_stream(move |stream| {
            let mut source = FailAfterReader {
                inner: Cursor::new(payload),
                limit: 96 * 1024,
            };
            stream.stor_resumable(&fault_path, 0, &mut source, |_| {
                FtpTransferControl::Continue
            })
        })
        .await;
    anyhow::ensure!(
        failed.is_err(),
        "{protocol} truncated upload source unexpectedly committed staging payload"
    );
    ftp.reconnect()
        .await
        .with_context(|| format!("{protocol} reconnect after failed staging data socket"))?;

    let inspect_path = destination.clone();
    let actual = ftp
        .with_stream(move |stream| {
            let mut bytes = Vec::new();
            stream.retr_to_writer(&inspect_path, &mut bytes)?;
            Ok(bytes)
        })
        .await
        .with_context(|| format!("{protocol} read protected original after fault"))?;
    anyhow::ensure!(
        actual == sentinel,
        "{protocol} existing destination was modified by failed staging upload"
    );

    let listing = fs.list_dir(&base).await?;
    if listing.iter().any(|entry| entry.name == ".ghostftp-upload-fault.part") {
        fs.delete(&staging, false)
            .await
            .with_context(|| format!("{protocol} clean partial staging sibling"))?;
    }
    let after_cleanup = fs.list_dir(&base).await?;
    anyhow::ensure!(
        !after_cleanup
            .iter()
            .any(|entry| entry.name == ".ghostftp-upload-fault.part"),
        "{protocol} left an orphan partial staging sibling after cleanup"
    );
    fs.delete(&destination, false).await?;
    fs.delete(&base, false).await?;
    ftp.shutdown().await?;
    Ok(())
}
