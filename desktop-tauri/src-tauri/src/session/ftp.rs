use crate::profiles::{AuthMethod, ConnectionProfile};
use anyhow::{anyhow, Context, Result};
use std::sync::{Arc, Mutex as StdMutex};
use suppaftp::native_tls::TlsConnector;
use suppaftp::{FtpStream, NativeTlsConnector, NativeTlsFtpStream};

/// One FTP control connection. suppaftp is synchronous; we wrap it in a
/// `std::sync::Mutex` and route every operation through `spawn_blocking` so
/// it cannot block tokio's runtime threads. Data transfers go through the
/// same stream (FTP has no native multiplexing — operations serialise on the
/// control connection by design).
pub struct FtpSession {
    pub id: String,
    pub profile: ConnectionProfile,
    inner: Arc<StdMutex<FtpStreamKind>>,
}

/// suppaftp ships two separate stream types depending on whether TLS is
/// involved. We keep them in an enum so all callsites can speak to either.
pub enum FtpStreamKind {
    Plain(FtpStream),
    Tls(NativeTlsFtpStream),
}

impl FtpStreamKind {
    pub fn list(&mut self, path: Option<&str>) -> Result<Vec<String>> {
        match self {
            Self::Plain(s) => s.list(path).map_err(into_anyhow),
            Self::Tls(s) => s.list(path).map_err(into_anyhow),
        }
    }
    pub fn rename(&mut self, from: &str, to: &str) -> Result<()> {
        match self {
            Self::Plain(s) => s.rename(from, to).map_err(into_anyhow),
            Self::Tls(s) => s.rename(from, to).map_err(into_anyhow),
        }
    }
    pub fn rm(&mut self, path: &str) -> Result<()> {
        match self {
            Self::Plain(s) => s.rm(path).map_err(into_anyhow),
            Self::Tls(s) => s.rm(path).map_err(into_anyhow),
        }
    }
    pub fn rmdir(&mut self, path: &str) -> Result<()> {
        match self {
            Self::Plain(s) => s.rmdir(path).map_err(into_anyhow),
            Self::Tls(s) => s.rmdir(path).map_err(into_anyhow),
        }
    }
    pub fn mkdir(&mut self, path: &str) -> Result<()> {
        match self {
            Self::Plain(s) => s.mkdir(path).map_err(into_anyhow),
            Self::Tls(s) => s.mkdir(path).map_err(into_anyhow),
        }
    }
    pub fn site(&mut self, cmd: &str) -> Result<()> {
        match self {
            Self::Plain(s) => s.site(cmd).map(|_| ()).map_err(into_anyhow),
            Self::Tls(s) => s.site(cmd).map(|_| ()).map_err(into_anyhow),
        }
    }
    pub fn size(&mut self, path: &str) -> Result<usize> {
        match self {
            Self::Plain(s) => s.size(path).map_err(into_anyhow),
            Self::Tls(s) => s.size(path).map_err(into_anyhow),
        }
    }
    pub fn retr_to_writer<W: std::io::Write>(
        &mut self,
        path: &str,
        mut sink: W,
    ) -> Result<u64> {
        match self {
            Self::Plain(s) => s
                .retr(path, |r| std::io::copy(r, &mut sink).map_err(suppaftp::FtpError::ConnectionError))
                .map_err(into_anyhow),
            Self::Tls(s) => s
                .retr(path, |r| std::io::copy(r, &mut sink).map_err(suppaftp::FtpError::ConnectionError))
                .map_err(into_anyhow),
        }
    }
    pub fn put_from_reader<R: std::io::Read>(
        &mut self,
        path: &str,
        reader: &mut R,
    ) -> Result<u64> {
        match self {
            Self::Plain(s) => s.put_file(path, reader).map_err(into_anyhow),
            Self::Tls(s) => s.put_file(path, reader).map_err(into_anyhow),
        }
    }
    pub fn quit(&mut self) {
        match self {
            Self::Plain(s) => {
                let _ = s.quit();
            }
            Self::Tls(s) => {
                let _ = s.quit();
            }
        }
    }
}

fn into_anyhow(e: suppaftp::FtpError) -> anyhow::Error {
    anyhow!(e.to_string())
}

impl FtpSession {
    /// Run a closure with mutable access to the underlying FTP stream on a
    /// blocking thread. Use this for any FTP operation — it ensures the
    /// blocking syscalls don't pin a tokio worker.
    pub async fn with_stream<F, T>(&self, f: F) -> Result<T>
    where
        F: FnOnce(&mut FtpStreamKind) -> Result<T> + Send + 'static,
        T: Send + 'static,
    {
        let inner = self.inner.clone();
        tokio::task::spawn_blocking(move || {
            let mut g = inner.lock().map_err(|_| anyhow!("FTP stream lock poisoned"))?;
            f(&mut g)
        })
        .await
        .map_err(|e| anyhow!("FTP task join failed: {e}"))?
    }
}

pub async fn ftp_connect(profile: &ConnectionProfile) -> Result<FtpSession> {
    let host = profile.host.clone();
    let port = profile.port;
    let username = profile.username.clone();
    let password = match &profile.auth {
        AuthMethod::Password { password } => password.clone(),
        AuthMethod::Key { .. } => {
            return Err(anyhow!(
                "FTP does not support private-key auth; switch to password"
            ))
        }
        AuthMethod::Agent => {
            return Err(anyhow!(
                "FTP does not support ssh-agent auth; switch to password"
            ))
        }
        AuthMethod::KeyRef { .. } => {
            return Err(anyhow!(
                "FTP does not support keychain key auth; switch to password"
            ))
        }
    };
    let want_tls = profile.protocol.eq_ignore_ascii_case("ftps");

    let id = uuid::Uuid::new_v4().to_string();
    let host_for_blocking = host.clone();
    let stream = tokio::task::spawn_blocking(move || -> Result<FtpStreamKind> {
        let addr = format!("{host_for_blocking}:{port}");
        if want_tls {
            // Explicit FTPS: connect as a NativeTlsFtpStream-typed stream
            // (still plain TCP at this point), then issue AUTH TLS via
            // into_secure. NativeTlsFtpStream and FtpStream are different
            // generic instantiations, so the type has to be picked up front.
            let s = NativeTlsFtpStream::connect(&addr)
                .with_context(|| format!("FTP connect {addr}"))?;
            let tls_connector = TlsConnector::new()
                .map_err(|e| anyhow!("TLS init: {e}"))?;
            let secured = s
                .into_secure(
                    NativeTlsConnector::from(tls_connector),
                    &host_for_blocking,
                )
                .map_err(|e| anyhow!("FTPS AUTH TLS: {e}"))?;
            let mut tls = FtpStreamKind::Tls(secured);
            login(&mut tls, &username, &password)?;
            Ok(tls)
        } else {
            let s = FtpStream::connect(&addr)
                .with_context(|| format!("FTP connect {addr}"))?;
            let mut plain = FtpStreamKind::Plain(s);
            login(&mut plain, &username, &password)?;
            Ok(plain)
        }
    })
    .await
    .map_err(|e| anyhow!("FTP connect task: {e}"))??;

    Ok(FtpSession {
        id,
        profile: profile.clone(),
        inner: Arc::new(StdMutex::new(stream)),
    })
}

fn login(stream: &mut FtpStreamKind, user: &str, password: &str) -> Result<()> {
    match stream {
        FtpStreamKind::Plain(s) => s.login(user, password).map_err(into_anyhow),
        FtpStreamKind::Tls(s) => s.login(user, password).map_err(into_anyhow),
    }
}
