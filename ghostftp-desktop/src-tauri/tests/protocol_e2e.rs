use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use ghostftp_lib::profiles::{AuthMethod, ConnectionProfile};
use ghostftp_lib::remotefs::{ftp::FtpFs, sftp::SftpFs, RemoteFs};
use ghostftp_lib::session::ftp::FtpTransferControl;
use ghostftp_lib::session::{open_session, HostDecision, HostKeyVerifier, HostPromptKind, Session};
use std::io::{Cursor, Read, SeekFrom, Write};
use std::sync::Arc;
use tokio::io::{AsyncReadExt, AsyncSeekExt, AsyncWriteExt};
use uuid::Uuid;

#[derive(Clone, Copy)]
struct FixedVerifier(HostDecision);

#[async_trait]
impl HostKeyVerifier for FixedVerifier {
    async fn decide(
        &self,
        _host: &str,
        _port: u16,
        _key_type: &str,
        _fingerprint: &str,
        _stored_fingerprint: Option<&str>,
        _kind: HostPromptKind,
    ) -> Result<HostDecision, russh::Error> {
        Ok(self.0)
    }
}

fn enabled() -> bool {
    std::env::var("GHOSTFTP_PROTOCOL_E2E").as_deref() == Ok("1")
}

fn required(name: &str) -> Result<String> {
    std::env::var(name).with_context(|| format!("missing required E2E variable {name}"))
}

fn port(name: &str) -> Result<u16> {
    required(name)?
        .parse()
        .with_context(|| format!("invalid port in {name}"))
}

fn byte_mismatch(actual: &[u8], expected: &[u8]) -> String {
    let first = actual
        .iter()
        .zip(expected.iter())
        .position(|(actual, expected)| actual != expected)
        .or_else(|| (actual.len() != expected.len()).then_some(actual.len().min(expected.len())));
    format!(
        "actual_len={}, expected_len={}, first_mismatch={first:?}",
        actual.len(),
        expected.len()
    )
}

struct FailAfterWriter {
    limit: usize,
    written: usize,
}

impl Write for FailAfterWriter {
    fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
        if self.written >= self.limit {
            return Err(std::io::Error::other(
                "intentional Ghost FTP E2E destination failure",
            ));
        }
        let allowed = (self.limit - self.written).min(buf.len());
        self.written += allowed;
        Ok(allowed)
    }

    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

struct FailAfterReader {
    inner: Cursor<Vec<u8>>,
    limit: u64,
}

impl Read for FailAfterReader {
    fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
        if self.inner.position() >= self.limit {
            return Err(std::io::Error::other(
                "intentional Ghost FTP E2E source failure",
            ));
        }
        let remaining = (self.limit - self.inner.position()) as usize;
        let allowed = buf.len().min(remaining);
        std::io::Read::read(&mut self.inner, &mut buf[..allowed])
    }
}

fn profile(
    protocol: &str,
    host: &str,
    port: u16,
    username: &str,
    auth: AuthMethod,
) -> ConnectionProfile {
    ConnectionProfile {
        id: format!("e2e-{}", Uuid::new_v4()),
        name: format!("Ghost FTP {protocol} E2E"),
        protocol: protocol.to_string(),
        host: host.to_string(),
        port,
        username: username.to_string(),
        auth,
        default_remote_path: Some(".".to_string()),
        description: None,
        color: None,
        auto_connect: Some(false),
        bucket: None,
        region: None,
        endpoint: None,
        account: None,
        agent_key: None,
        group: None,
        favorite: None,
        bookmarked: None,
        tags: None,
        last_used: None,
        sort_order: None,
        icon: None,
        jump_host: None,
        jump_port: None,
        jump_username: None,
    }
}

async fn ftp_roundtrip(
    protocol: &str,
    host: &str,
    port: u16,
    username: &str,
    password: &str,
) -> Result<()> {
    let p = profile(
        protocol,
        host,
        port,
        username,
        AuthMethod::Password {
            password: password.to_string(),
        },
    );
    let session = open_session(&p, Arc::new(FixedVerifier(HostDecision::Accept))).await?;
    let Session::Ftp(ftp) = session else {
        return Err(anyhow!("{protocol} did not open an FTP session"));
    };
    let fs = FtpFs::new(ftp.clone());

    let base = format!("ghostftp-e2e-{}", Uuid::new_v4().simple());
    let upload = format!("{base}/upload.txt");
    let renamed = format!("{base}/renamed.txt");
    let payload = format!("Ghost FTP {protocol} E2E payload\n").into_bytes();

    fs.create_dir(&base).await.context("FTP mkdir")?;

    let upload_path = upload.clone();
    let upload_payload = payload.clone();
    ftp.with_stream(move |stream| {
        let mut reader = Cursor::new(upload_payload);
        stream.put_from_reader(&upload_path, &mut reader)?;
        Ok(())
    })
    .await
    .context("FTP upload")?;

    let listing = fs.list_dir(&base).await.context("FTP list")?;
    if !listing.iter().any(|entry| entry.name == "upload.txt") {
        return Err(anyhow!("{protocol} LIST did not return upload.txt"));
    }

    fs.rename(&upload, &renamed).await.context("FTP rename")?;

    let download_path = renamed.clone();
    let downloaded = ftp
        .with_stream(move |stream| {
            let mut bytes = Vec::new();
            stream.retr_to_writer(&download_path, &mut bytes)?;
            Ok(bytes)
        })
        .await
        .context("FTP download")?;
    if downloaded != payload {
        return Err(anyhow!("{protocol} download content mismatch"));
    }

    // Exercise the same SuppaFTP 12 restart + transfer-stream primitives production
    // pause/resume uses. An interrupted transfer deliberately reconnects before
    // the next command: if the server completed the data socket just before ABOR,
    // SuppaFTP can consume the queued 226 as the ABOR reply and leave the ABOR
    // 225 queued. A fresh authenticated control connection removes that race
    // while preserving the server-confirmed byte offset.
    let resume_path = format!("{base}/resume.bin");
    let resume_payload: Vec<u8> = (0..(512 * 1024))
        .map(|index| ((index * 19 + 23) % 251) as u8)
        .collect();
    let pause_after = 128 * 1024u64;

    let upload_path = resume_path.clone();
    let upload_payload = resume_payload.clone();
    let upload_pause = ftp
        .with_stream(move |stream| {
            let mut reader = Cursor::new(upload_payload);
            stream.stor_resumable(&upload_path, 0, &mut reader, |transferred| {
                if transferred >= pause_after {
                    FtpTransferControl::Pause
                } else {
                    FtpTransferControl::Continue
                }
            })
        })
        .await
        .with_context(|| format!("{protocol} ABOR partial upload"))?;
    // Bytes accepted by the local data socket can be ahead of bytes the FTP
    // server has durably committed when ABOR closes the transfer. The
    // production resumable path therefore reports the server-confirmed SIZE,
    // which may legitimately be below the local pause threshold. Some servers
    // discard the interrupted STOR entirely, in which case zero is the only
    // safe restart offset and Ghost FTP deliberately restarts from the start.
    if upload_pause.control != FtpTransferControl::Pause || upload_pause.transferred > pause_after {
        return Err(anyhow!(
            "{protocol} upload did not report a valid server-committed pause offset"
        ));
    }
    ftp.reconnect()
        .await
        .with_context(|| format!("{protocol} reconnect after paused upload"))?;

    let upload_path = resume_path.clone();
    let upload_payload = resume_payload.clone();
    let upload_offset = upload_pause.transferred;
    let (mut upload_done, needs_fresh_restart) = ftp
        .with_stream(move |stream| {
            if upload_offset > 0 {
                let remote_size = stream.size(&upload_path)? as u64;
                if remote_size != upload_offset {
                    return Err(anyhow!(
                        "remote prefix mismatch before FTP resume: {remote_size} != {upload_offset}"
                    ));
                }
            }

            let expected_len = upload_payload.len() as u64;
            let mut reader = Cursor::new(upload_payload);
            std::io::Seek::seek(&mut reader, SeekFrom::Start(upload_offset))?;
            let outcome =
                stream.stor_resumable(&upload_path, upload_offset, &mut reader, |_| {
                    FtpTransferControl::Continue
                })?;

            let needs_fresh_restart = if outcome.control == FtpTransferControl::Continue {
                match stream.size(&upload_path).ok().map(|size| size as u64) {
                    Some(size) => size != expected_len,
                    None => upload_offset > 0,
                }
            } else {
                false
            };

            Ok((outcome, needs_fresh_restart))
        })
        .await
        .with_context(|| format!("{protocol} verify resumed upload result"))?;

    if needs_fresh_restart {
        ftp.reconnect()
            .await
            .with_context(|| format!("{protocol} reconnect before full upload retry"))?;

        let restart_path = resume_path.clone();
        let restart_payload = resume_payload.clone();
        upload_done = ftp
            .with_stream(move |stream| {
                let expected_len = restart_payload.len() as u64;
                let mut reader = Cursor::new(restart_payload);
                let written = stream.put_from_reader(&restart_path, &mut reader)?;
                if written != expected_len {
                    return Err(anyhow!(
                        "fresh-session FTP recovery wrote {written} bytes, expected {expected_len}"
                    ));
                }
                let verified = stream.size(&restart_path)? as u64;
                if verified != expected_len {
                    return Err(anyhow!(
                        "fresh-session FTP restart verification mismatch: {verified} != {expected_len}"
                    ));
                }
                Ok(ghostftp_lib::session::ftp::FtpTransferOutcome {
                    transferred: expected_len,
                    control: FtpTransferControl::Continue,
                })
            })
            .await
            .with_context(|| format!("{protocol} trusted full upload retry on fresh session"))?;
    }
    if upload_done.control != FtpTransferControl::Continue
        || upload_done.transferred != resume_payload.len() as u64
    {
        return Err(anyhow!("{protocol} resumed upload length mismatch"));
    }

    // Verify upload resume independently before using the same file to test
    // download resume. This keeps a corrupt resumed upload result from being
    // misdiagnosed later as a REST+RETR failure.
    let verify_upload_path = resume_path.clone();
    let uploaded_bytes = ftp
        .with_stream(move |stream| {
            let mut bytes = Vec::new();
            stream.retr_to_writer(&verify_upload_path, &mut bytes)?;
            Ok(bytes)
        })
        .await
        .with_context(|| format!("{protocol} verify resumed upload content"))?;
    if uploaded_bytes != resume_payload {
        return Err(anyhow!(
            "{protocol} resumed upload content mismatch ({})",
            byte_mismatch(&uploaded_bytes, &resume_payload)
        ));
    }

    let download_path = resume_path.clone();
    let download_pause = ftp
        .with_stream(move |stream| {
            let mut bytes = Vec::new();
            let outcome = stream.retr_resumable(&download_path, 0, &mut bytes, |transferred| {
                if transferred >= pause_after {
                    FtpTransferControl::Pause
                } else {
                    FtpTransferControl::Continue
                }
            })?;
            Ok((outcome, bytes))
        })
        .await
        .with_context(|| format!("{protocol} ABOR partial download"))?;
    if download_pause.0.control != FtpTransferControl::Pause
        || download_pause.0.transferred != download_pause.1.len() as u64
    {
        return Err(anyhow!(
            "{protocol} download did not stop at its committed local prefix"
        ));
    }
    ftp.reconnect()
        .await
        .with_context(|| format!("{protocol} reconnect after paused download"))?;

    let download_path = resume_path.clone();
    let download_offset = download_pause.0.transferred;
    let mut rebuilt = download_pause.1;
    let download_done = ftp
        .with_stream(move |stream| {
            stream
                .retr_resumable(&download_path, download_offset, &mut rebuilt, |_| {
                    FtpTransferControl::Continue
                })
                .map(|outcome| (outcome, rebuilt))
        })
        .await
        .with_context(|| format!("{protocol} REST + RETR download resume"))?;
    if download_done.0.control != FtpTransferControl::Continue || download_done.1 != resume_payload
    {
        return Err(anyhow!(
            "{protocol} resumed download content mismatch ({})",
            byte_mismatch(&download_done.1, &resume_payload)
        ));
    }

    // Cancel uses the same cooperative ABOR path as Pause but intentionally
    // keeps a distinct control result. Production retires the interrupted
    // control connection before any subsequent command, so verify the fresh
    // authenticated session can immediately inspect the canceled target.
    let cancel_path = format!("{base}/cancel.bin");
    let cancel_payload = resume_payload.clone();
    let cancel_path_for_transfer = cancel_path.clone();
    let cancel_outcome = ftp
        .with_stream(move |stream| {
            let mut reader = Cursor::new(cancel_payload);
            stream.stor_resumable(&cancel_path_for_transfer, 0, &mut reader, |transferred| {
                if transferred >= 64 * 1024 {
                    FtpTransferControl::Cancel
                } else {
                    FtpTransferControl::Continue
                }
            })
        })
        .await
        .with_context(|| format!("{protocol} cooperative upload cancel"))?;
    if cancel_outcome.control != FtpTransferControl::Cancel {
        return Err(anyhow!("{protocol} cancel did not return Cancel control"));
    }
    ftp.reconnect()
        .await
        .with_context(|| format!("{protocol} reconnect after canceled upload"))?;
    let cancel_path_for_size = cancel_path.clone();
    ftp.with_stream(move |stream| {
        let _ = stream.size(&cancel_path_for_size)?;
        Ok(())
    })
    .await
    .with_context(|| format!("{protocol} fresh control channel after ABOR"))?;

    // I/O failures must retire the raw data stream and the control connection
    // before the next command. This avoids carrying an ABOR/completion reply
    // race into unrelated work on the session.
    let download_failure_path = resume_path.clone();
    let download_failure = ftp
        .with_stream(move |stream| {
            let mut writer = FailAfterWriter {
                limit: 96 * 1024,
                written: 0,
            };
            stream.retr_resumable(&download_failure_path, 0, &mut writer, |_| {
                FtpTransferControl::Continue
            })
        })
        .await;
    if download_failure.is_ok() {
        return Err(anyhow!(
            "{protocol} download destination failure unexpectedly succeeded"
        ));
    }
    ftp.reconnect()
        .await
        .with_context(|| format!("{protocol} reconnect after download I/O failure"))?;
    let resume_path_after_download_error = resume_path.clone();
    ftp.with_stream(move |stream| {
        let _ = stream.size(&resume_path_after_download_error)?;
        Ok(())
    })
    .await
    .with_context(|| format!("{protocol} fresh control channel after download I/O failure"))?;

    let io_failure_path = format!("{base}/io-failure.bin");
    let io_failure_target = io_failure_path.clone();
    let io_failure_payload = resume_payload.clone();
    let upload_failure = ftp
        .with_stream(move |stream| {
            let mut reader = FailAfterReader {
                inner: Cursor::new(io_failure_payload),
                limit: 96 * 1024,
            };
            stream.stor_resumable(&io_failure_target, 0, &mut reader, |_| {
                FtpTransferControl::Continue
            })
        })
        .await;
    if upload_failure.is_ok() {
        return Err(anyhow!(
            "{protocol} upload source failure unexpectedly succeeded"
        ));
    }
    ftp.reconnect()
        .await
        .with_context(|| format!("{protocol} reconnect after upload I/O failure"))?;
    let io_failure_path_for_size = io_failure_path.clone();
    ftp.with_stream(move |stream| {
        let _ = stream.size(&io_failure_path_for_size)?;
        Ok(())
    })
    .await
    .with_context(|| format!("{protocol} fresh control channel after upload I/O failure"))?;

    fs.delete(&renamed, false)
        .await
        .context("FTP delete file")?;
    fs.delete(&resume_path, false)
        .await
        .context("FTP delete resume file")?;
    fs.delete(&cancel_path, false)
        .await
        .context("FTP delete canceled file")?;
    fs.delete(&io_failure_path, false)
        .await
        .context("FTP delete I/O failure file")?;
    fs.delete(&base, false)
        .await
        .context("FTP delete directory")?;

    ftp.with_stream(|stream| {
        stream.quit();
        Ok(())
    })
    .await?;

    Ok(())
}

async fn sftp_password_roundtrip(
    host: &str,
    port: u16,
    username: &str,
    password: &str,
) -> Result<()> {
    let p = profile(
        "sftp",
        host,
        port,
        username,
        AuthMethod::Password {
            password: password.to_string(),
        },
    );

    // Unknown host keys must really pass through the verifier.
    let rejected = open_session(&p, Arc::new(FixedVerifier(HostDecision::Reject))).await;
    if rejected.is_ok() {
        return Err(anyhow!(
            "SFTP unknown host key was accepted after explicit rejection"
        ));
    }

    let session = open_session(&p, Arc::new(FixedVerifier(HostDecision::Accept)))
        .await
        .context("SFTP password connect")?;
    let Session::Ssh(ssh) = session else {
        return Err(anyhow!("SFTP did not open an SSH session"));
    };
    let fs = SftpFs::new(ssh.clone());

    let base = format!("ghostftp-e2e-{}", Uuid::new_v4().simple());
    let upload = format!("{base}/upload.txt");
    let renamed = format!("{base}/renamed.txt");
    let payload = b"Ghost FTP SFTP E2E payload\n".to_vec();

    fs.create_dir(&base).await.context("SFTP mkdir")?;

    let cell = ssh.ensure_sftp().await?;
    let mut remote = {
        let sftp = cell.lock().await;
        sftp.create(&upload).await.context("SFTP create upload")?
    };
    remote.write_all(&payload).await.context("SFTP upload")?;
    remote.flush().await?;

    let listing = fs.list_dir(&base).await.context("SFTP list")?;
    if !listing.iter().any(|entry| entry.name == "upload.txt") {
        return Err(anyhow!("SFTP list did not return upload.txt"));
    }

    fs.chmod(&upload, 0o640).await.context("SFTP chmod")?;
    fs.rename(&upload, &renamed).await.context("SFTP rename")?;

    let cell = ssh.ensure_sftp().await?;
    let (mut remote, mode) = {
        let sftp = cell.lock().await;
        let mode = sftp
            .metadata(&renamed)
            .await
            .context("SFTP stat after chmod")?
            .permissions
            .unwrap_or(0)
            & 0o777;
        let file = sftp.open(&renamed).await.context("SFTP open download")?;
        (file, mode)
    };
    if mode != 0o640 {
        return Err(anyhow!(
            "SFTP chmod mismatch: expected 0640, got {mode:04o}"
        ));
    }

    let mut downloaded = Vec::new();
    remote
        .read_to_end(&mut downloaded)
        .await
        .context("SFTP download")?;
    if downloaded != payload {
        return Err(anyhow!("SFTP download content mismatch"));
    }

    // Prove the exact SFTP primitives used by production pause/resume against
    // a real OpenSSH internal-sftp server: reopen a partial remote file without
    // truncating it, seek both sides to the committed byte and append the rest.
    let resume_path = format!("{base}/resume.bin");
    let resume_payload: Vec<u8> = (0..(512 * 1024))
        .map(|index| ((index * 31 + 17) % 251) as u8)
        .collect();
    let resume_offset = 128 * 1024;

    let cell = ssh.ensure_sftp().await?;
    let mut partial = {
        let sftp = cell.lock().await;
        sftp.create(&resume_path)
            .await
            .context("SFTP create resume target")?
    };
    partial
        .write_all(&resume_payload[..resume_offset])
        .await
        .context("SFTP seed partial upload")?;
    partial.flush().await?;
    drop(partial);

    let cell = ssh.ensure_sftp().await?;
    let mut resumed_remote = {
        let sftp = cell.lock().await;
        let metadata = sftp
            .metadata(&resume_path)
            .await
            .context("SFTP stat partial upload")?;
        if metadata.size.unwrap_or(0) != resume_offset as u64 {
            return Err(anyhow!("SFTP partial upload size mismatch before resume"));
        }
        sftp.open_with_flags(&resume_path, russh_sftp::protocol::OpenFlags::WRITE)
            .await
            .context("SFTP reopen partial upload for resume")?
    };
    resumed_remote
        .seek(SeekFrom::Start(resume_offset as u64))
        .await
        .context("SFTP seek remote upload to committed offset")?;
    resumed_remote
        .write_all(&resume_payload[resume_offset..])
        .await
        .context("SFTP append resumed upload")?;
    resumed_remote.flush().await?;
    drop(resumed_remote);

    // Download from a non-zero remote offset too. This is the inverse primitive
    // used when a local partial download already contains the committed prefix.
    let cell = ssh.ensure_sftp().await?;
    let mut resumed_download = {
        let sftp = cell.lock().await;
        sftp.open(&resume_path)
            .await
            .context("SFTP reopen completed upload for ranged download")?
    };
    resumed_download
        .seek(SeekFrom::Start(resume_offset as u64))
        .await
        .context("SFTP seek remote download to committed offset")?;
    let mut rebuilt = resume_payload[..resume_offset].to_vec();
    resumed_download
        .read_to_end(&mut rebuilt)
        .await
        .context("SFTP ranged download remainder")?;
    if rebuilt != resume_payload {
        return Err(anyhow!("SFTP byte-range resume content mismatch"));
    }

    fs.delete(&renamed, false)
        .await
        .context("SFTP delete file")?;
    fs.delete(&resume_path, false)
        .await
        .context("SFTP delete resume file")?;
    fs.delete(&base, false)
        .await
        .context("SFTP delete directory")?;

    Ok(())
}

async fn sftp_key_auth(
    host: &str,
    port: u16,
    username: &str,
    key_path: &str,
    passphrase: &str,
) -> Result<()> {
    let p = profile(
        "sftp",
        host,
        port,
        username,
        AuthMethod::Key {
            path: key_path.to_string(),
            passphrase: Some(passphrase.to_string()),
        },
    );
    let session = open_session(&p, Arc::new(FixedVerifier(HostDecision::Accept)))
        .await
        .context("SFTP encrypted private-key connect")?;
    let Session::Ssh(ssh) = session else {
        return Err(anyhow!("SFTP key auth did not open an SSH session"));
    };
    let fs = SftpFs::new(ssh);
    fs.list_dir(".")
        .await
        .context("SFTP key-auth directory listing")?;
    Ok(())
}

#[tokio::test(flavor = "multi_thread", worker_threads = 2)]
async fn real_ftp_ftps_sftp_roundtrips() -> Result<()> {
    if !enabled() {
        eprintln!("Ghost FTP protocol E2E skipped (set GHOSTFTP_PROTOCOL_E2E=1 to enable)");
        return Ok(());
    }

    let username = required("GHOSTFTP_E2E_USERNAME")?;
    let password = required("GHOSTFTP_E2E_PASSWORD")?;

    ftp_roundtrip(
        "ftp",
        "127.0.0.1",
        port("GHOSTFTP_E2E_FTP_PORT")?,
        &username,
        &password,
    )
    .await?;

    let ftps_port = port("GHOSTFTP_E2E_FTPS_PORT")?;
    ftp_roundtrip("ftps", "localhost", ftps_port, &username, &password).await?;

    // The local FTPS certificate is trusted for DNS:localhost only. Connecting by
    // IP must fail, proving hostname/certificate verification is not bypassed.
    let bad_ftps = profile(
        "ftps",
        "127.0.0.1",
        ftps_port,
        &username,
        AuthMethod::Password {
            password: password.clone(),
        },
    );
    if open_session(&bad_ftps, Arc::new(FixedVerifier(HostDecision::Accept)))
        .await
        .is_ok()
    {
        return Err(anyhow!(
            "FTPS accepted a certificate whose identity does not match 127.0.0.1"
        ));
    }

    let sftp_port = port("GHOSTFTP_E2E_SFTP_PORT")?;
    sftp_password_roundtrip("127.0.0.1", sftp_port, &username, &password).await?;

    sftp_key_auth(
        "127.0.0.1",
        sftp_port,
        &username,
        &required("GHOSTFTP_E2E_KEY_PATH")?,
        &required("GHOSTFTP_E2E_KEY_PASSPHRASE")?,
    )
    .await?;

    Ok(())
}
