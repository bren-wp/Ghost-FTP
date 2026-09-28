use crate::profiles::{AuthMethod, ConnectionProfile};
use anyhow::{anyhow, Context, Result};
use std::io::{Read, Write};
use std::net::{SocketAddr, TcpStream, ToSocketAddrs};
use std::sync::{Arc, Mutex as StdMutex};
use std::time::Duration;
use suppaftp::native_tls::TlsConnector;
use suppaftp::types::{FileType, Mode};
use suppaftp::{FtpError, FtpStream, NativeTlsConnector, NativeTlsFtpStream, Status};

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

/// Cooperative decision returned by Ghost FTP after each committed FTP chunk.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FtpTransferControl {
    Continue,
    Pause,
    Cancel,
}

/// Result of one FTP/FTPS stream attempt. transferred is the absolute
/// committed byte offset, including any prefix established via REST.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct FtpTransferOutcome {
    pub transferred: u64,
    pub control: FtpTransferControl,
}

const FTP_TRANSFER_CHUNK: usize = 64 * 1024;

/// SuppaFTP 6.x treats FTP 225 as an unexpected ABOR reply even though
/// RFC 959 defines it as "data connection open; no transfer in progress".
/// Some servers (including pyftpdlib) return 225 after the data socket is
/// closed as part of ABOR. At that point the reply has already been consumed,
/// so the control channel is synchronized and the abort is complete.
///
/// Keep every other SuppaFTP error fatal: only the exact 225 status is
/// normalized to success.
fn normalize_abort_result(result: std::result::Result<(), FtpError>) -> Result<()> {
    match result {
        Ok(()) => Ok(()),
        Err(FtpError::UnexpectedResponse(response))
            if response.status == Status::DataConnectionOpen =>
        {
            Ok(())
        }
        Err(error) => Err(into_anyhow(error)),
    }
}

/// Resolve the only upload offset that is safe to resume from after ABOR:
/// the size the server reports after the data channel has been closed.
///
/// A server that does not support SIZE cannot prove the committed prefix, so
/// Ghost FTP deliberately falls back to offset 0. Likewise, an impossible
/// remote size (before the requested REST floor or beyond bytes written) is
/// treated as unverified and forces a safe restart instead of risking
/// corruption.
fn verified_upload_prefix_after_abort(
    size_result: std::result::Result<usize, FtpError>,
    restart_floor: u64,
    attempted: u64,
) -> Result<u64> {
    match size_result {
        Ok(size) => {
            let size = size as u64;
            if size >= restart_floor && size <= attempted {
                Ok(size)
            } else {
                Ok(0)
            }
        }
        Err(FtpError::UnexpectedResponse(response))
            if matches!(
                response.status,
                Status::BadCommand
                    | Status::BadArguments
                    | Status::NotImplemented
                    | Status::NotImplementedParameter
                    | Status::FileUnavailable
            ) =>
        {
            Ok(0)
        }
        Err(error) => Err(into_anyhow(error).context("verify FTP upload prefix after ABOR")),
    }
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
    pub fn retr_to_writer<W: std::io::Write>(&mut self, path: &str, mut sink: W) -> Result<u64> {
        match self {
            Self::Plain(s) => s
                .retr(path, |r| {
                    std::io::copy(r, &mut sink).map_err(suppaftp::FtpError::ConnectionError)
                })
                .map_err(into_anyhow),
            Self::Tls(s) => s
                .retr(path, |r| {
                    std::io::copy(r, &mut sink).map_err(suppaftp::FtpError::ConnectionError)
                })
                .map_err(into_anyhow),
        }
    }
    pub fn put_from_reader<R: std::io::Read>(&mut self, path: &str, reader: &mut R) -> Result<u64> {
        match self {
            Self::Plain(s) => s.put_file(path, reader).map_err(into_anyhow),
            Self::Tls(s) => s.put_file(path, reader).map_err(into_anyhow),
        }
    }

    /// Download with explicit STREAM restart semantics. SuppaFTP 6.3.0 exposes
    /// REST plus a raw RETR data stream; Ghost FTP reads bounded chunks so
    /// Pause/Cancel can cooperatively ABOR instead of leaving RETR running.
    pub fn retr_resumable<W, C>(
        &mut self,
        path: &str,
        offset: u64,
        sink: &mut W,
        mut control: C,
    ) -> Result<FtpTransferOutcome>
    where
        W: Write,
        C: FnMut(u64) -> FtpTransferControl,
    {
        match self {
            Self::Plain(stream) => {
                if offset > 0 {
                    stream
                        .resume_transfer(
                            usize::try_from(offset).context("FTP resume offset too large")?,
                        )
                        .map_err(into_anyhow)?;
                }
                let mut data = stream.retr_as_stream(path).map_err(into_anyhow)?;
                let mut transferred = offset;
                let mut buf = vec![0u8; FTP_TRANSFER_CHUNK];
                loop {
                    match control(transferred) {
                        FtpTransferControl::Continue => {}
                        stop => {
                            normalize_abort_result(stream.abort(data))?;
                            return Ok(FtpTransferOutcome {
                                transferred,
                                control: stop,
                            });
                        }
                    }
                    let read = match data.read(&mut buf) {
                        Ok(read) => read,
                        Err(error) => {
                            let abort_error = normalize_abort_result(stream.abort(data)).err();
                            return Err(match abort_error {
                                Some(abort_error) => anyhow!(
                                    "read FTP data stream: {error}; FTP ABOR after read failure also failed: {abort_error}"
                                ),
                                None => anyhow!(error).context("read FTP data stream"),
                            });
                        }
                    };
                    if read == 0 {
                        stream.finalize_retr_stream(data).map_err(into_anyhow)?;
                        return Ok(FtpTransferOutcome {
                            transferred,
                            control: FtpTransferControl::Continue,
                        });
                    }
                    if let Err(error) = sink.write_all(&buf[..read]) {
                        let abort_error = normalize_abort_result(stream.abort(data)).err();
                        return Err(match abort_error {
                            Some(abort_error) => anyhow!(
                                "write FTP download destination: {error}; FTP ABOR after destination failure also failed: {abort_error}"
                            ),
                            None => anyhow!(error).context("write FTP download destination"),
                        });
                    }
                    transferred += read as u64;
                }
            }
            Self::Tls(stream) => {
                if offset > 0 {
                    stream
                        .resume_transfer(
                            usize::try_from(offset).context("FTPS resume offset too large")?,
                        )
                        .map_err(into_anyhow)?;
                }
                let mut data = stream.retr_as_stream(path).map_err(into_anyhow)?;
                let mut transferred = offset;
                let mut buf = vec![0u8; FTP_TRANSFER_CHUNK];
                loop {
                    match control(transferred) {
                        FtpTransferControl::Continue => {}
                        stop => {
                            normalize_abort_result(stream.abort(data))?;
                            return Ok(FtpTransferOutcome {
                                transferred,
                                control: stop,
                            });
                        }
                    }
                    let read = match data.read(&mut buf) {
                        Ok(read) => read,
                        Err(error) => {
                            let abort_error = normalize_abort_result(stream.abort(data)).err();
                            return Err(match abort_error {
                                Some(abort_error) => anyhow!(
                                    "read FTPS data stream: {error}; FTPS ABOR after read failure also failed: {abort_error}"
                                ),
                                None => anyhow!(error).context("read FTPS data stream"),
                            });
                        }
                    };
                    if read == 0 {
                        stream.finalize_retr_stream(data).map_err(into_anyhow)?;
                        return Ok(FtpTransferOutcome {
                            transferred,
                            control: FtpTransferControl::Continue,
                        });
                    }
                    if let Err(error) = sink.write_all(&buf[..read]) {
                        let abort_error = normalize_abort_result(stream.abort(data)).err();
                        return Err(match abort_error {
                            Some(abort_error) => anyhow!(
                                "write FTPS download destination: {error}; FTPS ABOR after destination failure also failed: {abort_error}"
                            ),
                            None => anyhow!(error).context("write FTPS download destination"),
                        });
                    }
                    transferred += read as u64;
                }
            }
        }
    }

    /// Upload continuation with an exact-prefix append. The caller validates
    /// that the remote file is exactly offset bytes before a non-zero resume.
    ///
    /// APPE is used for resumed uploads instead of REST + STOR. In practice this
    /// avoids server-specific REST/STOR truncation behavior (especially over
    /// explicit TLS) while remaining byte-accurate because Ghost FTP resumes
    /// only after the remote SIZE exactly matches the committed local offset.
    pub fn stor_resumable<R, C>(
        &mut self,
        path: &str,
        offset: u64,
        source: &mut R,
        mut control: C,
    ) -> Result<FtpTransferOutcome>
    where
        R: Read,
        C: FnMut(u64) -> FtpTransferControl,
    {
        match self {
            Self::Plain(stream) => {
                let mut data = if offset > 0 {
                    stream.append_with_stream(path).map_err(into_anyhow)?
                } else {
                    stream.put_with_stream(path).map_err(into_anyhow)?
                };
                let mut transferred = offset;
                let mut buf = vec![0u8; FTP_TRANSFER_CHUNK];
                loop {
                    match control(transferred) {
                        FtpTransferControl::Continue => {}
                        stop => {
                            normalize_abort_result(stream.abort(data))?;
                            let committed = verified_upload_prefix_after_abort(
                                stream.size(path),
                                offset,
                                transferred,
                            )?;
                            return Ok(FtpTransferOutcome {
                                transferred: committed,
                                control: stop,
                            });
                        }
                    }
                    let read = match source.read(&mut buf) {
                        Ok(read) => read,
                        Err(error) => {
                            let abort_error = normalize_abort_result(stream.abort(data)).err();
                            return Err(match abort_error {
                                Some(abort_error) => anyhow!(
                                    "read FTP upload source: {error}; FTP ABOR after source failure also failed: {abort_error}"
                                ),
                                None => anyhow!(error).context("read FTP upload source"),
                            });
                        }
                    };
                    if read == 0 {
                        stream.finalize_put_stream(data).map_err(into_anyhow)?;
                        return Ok(FtpTransferOutcome {
                            transferred,
                            control: FtpTransferControl::Continue,
                        });
                    }
                    if let Err(error) = data.write_all(&buf[..read]) {
                        let abort_error = normalize_abort_result(stream.abort(data)).err();
                        return Err(match abort_error {
                            Some(abort_error) => anyhow!(
                                "write FTP data stream: {error}; FTP ABOR after data-write failure also failed: {abort_error}"
                            ),
                            None => anyhow!(error).context("write FTP data stream"),
                        });
                    }
                    transferred += read as u64;
                }
            }
            Self::Tls(stream) => {
                let mut data = if offset > 0 {
                    stream.append_with_stream(path).map_err(into_anyhow)?
                } else {
                    stream.put_with_stream(path).map_err(into_anyhow)?
                };
                let mut transferred = offset;
                let mut buf = vec![0u8; FTP_TRANSFER_CHUNK];
                loop {
                    match control(transferred) {
                        FtpTransferControl::Continue => {}
                        stop => {
                            normalize_abort_result(stream.abort(data))?;
                            let committed = verified_upload_prefix_after_abort(
                                stream.size(path),
                                offset,
                                transferred,
                            )?;
                            return Ok(FtpTransferOutcome {
                                transferred: committed,
                                control: stop,
                            });
                        }
                    }
                    let read = match source.read(&mut buf) {
                        Ok(read) => read,
                        Err(error) => {
                            let abort_error = normalize_abort_result(stream.abort(data)).err();
                            return Err(match abort_error {
                                Some(abort_error) => anyhow!(
                                    "read FTPS upload source: {error}; FTPS ABOR after source failure also failed: {abort_error}"
                                ),
                                None => anyhow!(error).context("read FTPS upload source"),
                            });
                        }
                    };
                    if read == 0 {
                        stream.finalize_put_stream(data).map_err(into_anyhow)?;
                        return Ok(FtpTransferOutcome {
                            transferred,
                            control: FtpTransferControl::Continue,
                        });
                    }
                    if let Err(error) = data.write_all(&buf[..read]) {
                        let abort_error = normalize_abort_result(stream.abort(data)).err();
                        return Err(match abort_error {
                            Some(abort_error) => anyhow!(
                                "write FTPS data stream: {error}; FTPS ABOR after data-write failure also failed: {abort_error}"
                            ),
                            None => anyhow!(error).context("write FTPS data stream"),
                        });
                    }
                    transferred += read as u64;
                }
            }
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

const FTP_CONNECT_TIMEOUT: Duration = Duration::from_secs(15);
const FTP_IO_TIMEOUT: Duration = Duration::from_secs(60);

/// Resolve the host ourselves so every FTP/FTPS control connection gets a
/// bounded TCP connect plus read/write deadlines. Platform-default socket
/// waits can otherwise make an unreachable site look like a frozen app.
fn connect_control_socket(host: &str, port: u16) -> Result<TcpStream> {
    let addrs = (host, port)
        .to_socket_addrs()
        .with_context(|| format!("resolve FTP host {host}"))?;
    let mut last_error = None;
    for addr in addrs {
        match TcpStream::connect_timeout(&addr, FTP_CONNECT_TIMEOUT) {
            Ok(stream) => {
                stream
                    .set_read_timeout(Some(FTP_IO_TIMEOUT))
                    .context("set FTP control read timeout")?;
                stream
                    .set_write_timeout(Some(FTP_IO_TIMEOUT))
                    .context("set FTP control write timeout")?;
                stream
                    .set_nodelay(true)
                    .context("set FTP control TCP_NODELAY")?;
                return Ok(stream);
            }
            Err(error) => last_error = Some(error),
        }
    }
    match last_error {
        Some(error) => Err(error).with_context(|| format!("FTP connect {host}:{port}")),
        None => Err(anyhow!("FTP host {host} resolved to no addresses")),
    }
}

/// Build passive FTP/FTPS data sockets with the same bounded network waits as
/// the control channel. SuppaFTP's default passive builder uses an unbounded
/// TcpStream::connect, which can otherwise leave LIST/RETR/STOR stuck long
/// after the control connection itself is healthy.
fn connect_data_socket(addr: SocketAddr) -> std::result::Result<TcpStream, suppaftp::FtpError> {
    let stream = TcpStream::connect_timeout(&addr, FTP_CONNECT_TIMEOUT)
        .map_err(suppaftp::FtpError::ConnectionError)?;
    stream
        .set_read_timeout(Some(FTP_IO_TIMEOUT))
        .map_err(suppaftp::FtpError::ConnectionError)?;
    stream
        .set_write_timeout(Some(FTP_IO_TIMEOUT))
        .map_err(suppaftp::FtpError::ConnectionError)?;
    Ok(stream)
}

/// Keep passive data connections on the authenticated control peer. For IPv4
/// PASV this avoids stale/private advertised addresses and prevents a server
/// response from redirecting the client to an unrelated host. IPv6 requires
/// EPSV, which already reuses the control peer and only supplies a port.
fn configure_plain_data_channel(mut stream: FtpStream, peer_is_ipv6: bool) -> FtpStream {
    stream = stream.passive_stream_builder(connect_data_socket);
    if peer_is_ipv6 {
        stream.set_mode(Mode::ExtendedPassive);
    } else {
        stream.set_passive_nat_workaround(true);
    }
    stream
}

fn configure_tls_data_channel(
    mut stream: NativeTlsFtpStream,
    peer_is_ipv6: bool,
) -> NativeTlsFtpStream {
    stream = stream.passive_stream_builder(connect_data_socket);
    if peer_is_ipv6 {
        stream.set_mode(Mode::ExtendedPassive);
    } else {
        stream.set_passive_nat_workaround(true);
    }
    stream
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
            let mut g = inner
                .lock()
                .map_err(|_| anyhow!("FTP stream lock poisoned"))?;
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
        let tcp = connect_control_socket(&host_for_blocking, port)?;
        let peer_is_ipv6 = tcp
            .peer_addr()
            .with_context(|| format!("FTP peer address {addr}"))?
            .is_ipv6();
        if want_tls {
            // Explicit FTPS: connect as a NativeTlsFtpStream-typed stream
            // (still plain TCP at this point), then issue AUTH TLS via
            // into_secure. NativeTlsFtpStream and FtpStream are different
            // generic instantiations, so the type has to be picked up front.
            let s = NativeTlsFtpStream::connect_with_stream(tcp)
                .with_context(|| format!("FTP connect {addr}"))?;
            let s = configure_tls_data_channel(s, peer_is_ipv6);
            let tls_connector = TlsConnector::new().map_err(|e| anyhow!("TLS init: {e}"))?;
            let secured = s
                .into_secure(NativeTlsConnector::from(tls_connector), &host_for_blocking)
                .map_err(|e| anyhow!("FTPS AUTH TLS: {e}"))?;
            let mut tls = FtpStreamKind::Tls(secured);
            login(&mut tls, &username, &password)?;
            Ok(tls)
        } else {
            let s = FtpStream::connect_with_stream(tcp)
                .with_context(|| format!("FTP connect {addr}"))?;
            let s = configure_plain_data_channel(s, peer_is_ipv6);
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
        FtpStreamKind::Plain(s) => {
            s.login(user, password).map_err(into_anyhow)?;
            s.transfer_type(FileType::Binary)
                .map_err(into_anyhow)
                .context("FTP TYPE I")?;
        }
        FtpStreamKind::Tls(s) => {
            s.login(user, password).map_err(into_anyhow)?;
            s.transfer_type(FileType::Binary)
                .map_err(into_anyhow)
                .context("FTPS TYPE I")?;
        }
    }
    Ok(())
}
