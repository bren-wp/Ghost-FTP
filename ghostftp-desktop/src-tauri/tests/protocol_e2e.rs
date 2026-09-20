use anyhow::{anyhow, Context, Result};
use async_trait::async_trait;
use ghostftp_lib::profiles::{AuthMethod, ConnectionProfile};
use ghostftp_lib::remotefs::{ftp::FtpFs, sftp::SftpFs, RemoteFs};
use ghostftp_lib::session::{
    open_session, HostDecision, HostKeyVerifier, HostPromptKind, Session,
};
use std::io::Cursor;
use std::sync::Arc;
use tokio::io::{AsyncReadExt, AsyncWriteExt};
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

    fs.rename(&upload, &renamed)
        .await
        .context("FTP rename")?;

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

    fs.delete(&renamed, false).await.context("FTP delete file")?;
    fs.delete(&base, false).await.context("FTP delete directory")?;

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
        return Err(anyhow!("SFTP unknown host key was accepted after explicit rejection"));
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
    fs.rename(&upload, &renamed)
        .await
        .context("SFTP rename")?;

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
        return Err(anyhow!("SFTP chmod mismatch: expected 0640, got {mode:04o}"));
    }

    let mut downloaded = Vec::new();
    remote
        .read_to_end(&mut downloaded)
        .await
        .context("SFTP download")?;
    if downloaded != payload {
        return Err(anyhow!("SFTP download content mismatch"));
    }

    fs.delete(&renamed, false).await.context("SFTP delete file")?;
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
    if open_session(
        &bad_ftps,
        Arc::new(FixedVerifier(HostDecision::Accept)),
    )
    .await
    .is_ok()
    {
        return Err(anyhow!(
            "FTPS accepted a certificate whose identity does not match 127.0.0.1"
        ));
    }

    let sftp_port = port("GHOSTFTP_E2E_SFTP_PORT")?;
    sftp_password_roundtrip(
        "127.0.0.1",
        sftp_port,
        &username,
        &password,
    )
    .await?;

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
