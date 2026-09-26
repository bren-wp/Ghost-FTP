//! The SCP wire protocol (Plan 13 Phase 3).
//!
//! Some hosts disable the SFTP subsystem but still allow an interactive/exec
//! shell (classic busybox appliances, locked-down boxes). Those can't be browsed
//! or transferred over `russh-sftp`, but they *can* run `scp`, whose old
//! rcp-derived wire protocol moves bytes over a plain exec channel's stdin/stdout.
//! This module implements that protocol against a generic `AsyncRead + AsyncWrite`
//! so the exact byte exchange is unit-testable over `tokio::io::duplex` with no
//! server — the only way to trust a legacy protocol short of a live busybox.
//!
//! Two directions, matching the two `scp` server modes we exec:
//! - **Download** ([`download_to`]): we exec `scp -f <path>` (the remote is the
//!   *source*) and receive a `C`-record header + the bytes.
//! - **Upload** ([`upload_from`]): we exec `scp -t <path>` (the remote is the
//!   *sink*) and send a `C`-record header + the bytes.
//!
//! Every step is gated by a single **status byte**: `0` = ok, `1` = warning,
//! `2` = fatal — a `1`/`2` is followed by a message terminated by `\n`. We treat
//! any non-zero status as an error carrying that message, which is how a
//! "no such file" or a permission denial surfaces.
//!
//! Path safety: the caller must build the exec command with a single-quoted path
//! ([`quote_path`]) so the remote shell never word-splits or glob-expands it —
//! `scp -f 'dir/my file'`, never `scp -f dir/my file`.

use anyhow::{bail, Context, Result};
use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};

/// The header of a file as announced by an SCP `C` record.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ScpHeader {
    /// POSIX mode bits (e.g. `0o644`), as parsed from the octal in the record.
    pub mode: u32,
    /// File length in bytes — exactly how many content bytes follow.
    pub size: u64,
    /// The basename the server reported.
    pub name: String,
}

/// Single-quote a remote path for a POSIX shell so `scp -f`/`-t` receives it as
/// one literal argument (no word-splitting, globbing, or metacharacter surprises).
pub fn quote_path(path: &str) -> String {
    format!("'{}'", path.replace('\'', r"'\''"))
}

/// The `scp` command to exec on the server to *send us* `path` (download).
pub fn source_command(path: &str) -> String {
    format!("scp -f {}", quote_path(path))
}

/// The `scp` command to exec on the server to *receive* into `path` (upload).
pub fn sink_command(path: &str) -> String {
    format!("scp -t {}", quote_path(path))
}

/// Drive the source (download) side of the protocol: read the file the server is
/// offering and stream its bytes into `out`. `stream` must be an exec channel
/// already running [`source_command`]. Returns the parsed header.
///
/// `T` (mtime) records are accepted and skipped; only the first `C` record's file
/// is read. `D`/`E` (directory) records are rejected — callers list directories
/// over the shell, not by recursively draining an SCP stream.
pub async fn download_to<S, W>(stream: &mut S, out: &mut W) -> Result<ScpHeader>
where
    S: AsyncRead + AsyncWrite + Unpin,
    W: AsyncWrite + Unpin,
{
    // Tell the source to proceed.
    write_ok(stream).await?;

    loop {
        let line = read_record_line(stream).await?;
        match line.as_bytes().first() {
            Some(b'T') => {
                // Modification-time record (only with -p) — acknowledge, ignore.
                write_ok(stream).await?;
                continue;
            }
            Some(b'C') => {
                let header = parse_c_record(&line)?;
                // Ready for the content.
                write_ok(stream).await?;
                copy_exact(stream, out, header.size).await?;
                // The trailing byte is the file's own transfer status.
                let status = stream.read_u8().await.context("read file status")?;
                if status != 0 {
                    let msg = read_message(stream).await.unwrap_or_default();
                    bail!("scp transfer failed: {msg}");
                }
                write_ok(stream).await?;
                return Ok(header);
            }
            Some(b'D') | Some(b'E') => {
                bail!("scp returned a directory; browse over the shell, not scp -r")
            }
            Some(1) | Some(2) => {
                // Error/warning record: the rest of the line is the message.
                bail!("scp: {}", &line[1..]);
            }
            _ => bail!("unexpected scp record: {line:?}"),
        }
    }
}

/// Drive the sink (upload) side: announce a `C` record for `header` then stream
/// exactly `header.size` bytes from `data`. `stream` must be an exec channel
/// already running [`sink_command`]. The server's target path is fixed by the
/// exec command; `header.name` is the basename it records.
pub async fn upload_from<S, R>(stream: &mut S, header: &ScpHeader, data: &mut R) -> Result<()>
where
    S: AsyncRead + AsyncWrite + Unpin,
    R: AsyncRead + Unpin,
{
    // The sink sends an initial status once it's ready. `expect_ok` carries the
    // server's own message on failure (e.g. "Permission denied"); we deliberately
    // don't wrap it in a `.context(...)`, because anyhow's Display shows only the
    // outermost context and callers stringify via `to_string()` — a wrapper would
    // hide the actual reason from the user.
    expect_ok(stream).await?;

    let record = format!("C{:04o} {} {}\n", header.mode & 0o7777, header.size, header.name);
    stream
        .write_all(record.as_bytes())
        .await
        .context("write scp C record")?;
    stream.flush().await?;
    expect_ok(stream).await?;

    copy_exact(data, stream, header.size).await?;

    // A zero byte terminates the file content.
    stream.write_all(&[0]).await.context("write scp end-of-file")?;
    stream.flush().await?;
    expect_ok(stream).await?;
    Ok(())
}

// ---------- protocol primitives ----------

/// Write a single `0` status byte ("proceed / ok").
async fn write_ok<S: AsyncWrite + Unpin>(stream: &mut S) -> Result<()> {
    stream.write_all(&[0]).await.context("write scp ack")?;
    stream.flush().await?;
    Ok(())
}

/// Read one status byte and fail (carrying the server's message) if it's non-zero.
async fn expect_ok<S: AsyncRead + AsyncWrite + Unpin>(stream: &mut S) -> Result<()> {
    let status = stream.read_u8().await.context("read scp status")?;
    if status == 0 {
        return Ok(());
    }
    let msg = read_message(stream).await.unwrap_or_default();
    bail!("scp error: {msg}");
}

/// Read a control-record line (up to and consuming the `\n`), returned without the
/// newline. The first byte may be `T`/`C`/`D`/`E` or a `1`/`2` error marker.
async fn read_record_line<S: AsyncRead + Unpin>(stream: &mut S) -> Result<String> {
    let first = stream.read_u8().await.context("read scp record")?;
    let mut bytes = vec![first];
    // Error markers are 0x01/0x02; a normal record starts with an ASCII letter.
    loop {
        let b = stream.read_u8().await.context("read scp record")?;
        if b == b'\n' {
            break;
        }
        bytes.push(b);
    }
    Ok(String::from_utf8_lossy(&bytes).into_owned())
}

/// Read a message terminated by `\n` (the text after a non-zero status byte).
async fn read_message<S: AsyncRead + Unpin>(stream: &mut S) -> Result<String> {
    let mut bytes = Vec::new();
    loop {
        let b = match stream.read_u8().await {
            Ok(b) => b,
            Err(_) => break,
        };
        if b == b'\n' {
            break;
        }
        bytes.push(b);
    }
    Ok(String::from_utf8_lossy(&bytes).trim().to_string())
}

/// Parse a `C<mode> <size> <name>` record into an [`ScpHeader`].
fn parse_c_record(line: &str) -> Result<ScpHeader> {
    // "C0644 1234 name with spaces"
    let body = &line[1..];
    let mut it = body.splitn(3, ' ');
    let mode_s = it.next().context("scp C record missing mode")?;
    let size_s = it.next().context("scp C record missing size")?;
    let name = it.next().context("scp C record missing name")?.to_string();
    let mode = u32::from_str_radix(mode_s, 8)
        .with_context(|| format!("parse scp mode {mode_s:?}"))?;
    let size: u64 = size_s.parse().with_context(|| format!("parse scp size {size_s:?}"))?;
    Ok(ScpHeader { mode, size, name })
}

/// Copy exactly `n` bytes from `src` to `dst`, erroring on early EOF.
async fn copy_exact<R, W>(src: &mut R, dst: &mut W, n: u64) -> Result<()>
where
    R: AsyncRead + Unpin,
    W: AsyncWrite + Unpin,
{
    let mut remaining = n;
    let mut buf = vec![0u8; 64 * 1024];
    while remaining > 0 {
        let want = remaining.min(buf.len() as u64) as usize;
        let read = src.read(&mut buf[..want]).await.context("read scp body")?;
        if read == 0 {
            bail!("scp stream ended {remaining} bytes early");
        }
        dst.write_all(&buf[..read]).await.context("write scp body")?;
        remaining -= read as u64;
    }
    dst.flush().await?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;
    use tokio::io::{AsyncReadExt, AsyncWriteExt};

    #[test]
    fn quote_path_escapes_single_quotes_and_spaces() {
        assert_eq!(quote_path("a/b.png"), "'a/b.png'");
        assert_eq!(quote_path("dir/my file"), "'dir/my file'");
        assert_eq!(quote_path("o'brien"), r"'o'\''brien'");
        assert_eq!(source_command("/x y"), "scp -f '/x y'");
        assert_eq!(sink_command("/x y"), "scp -t '/x y'");
    }

    #[test]
    fn parses_c_records() {
        let h = parse_c_record("C0644 1234 photo.png").unwrap();
        assert_eq!(h, ScpHeader { mode: 0o644, size: 1234, name: "photo.png".into() });
        // Names may contain spaces (splitn(3) keeps the tail intact).
        let h = parse_c_record("C0755 7 my file").unwrap();
        assert_eq!(h.name, "my file");
        assert_eq!(h.mode, 0o755);
    }

    // A minimal in-memory SCP *server* side, so the real client code runs the
    // real byte exchange over an in-process duplex — no network, no daemon.

    /// Play the source (server sends a file) side of a download.
    async fn serve_source<S: AsyncRead + AsyncWrite + Unpin>(
        stream: &mut S,
        header: &ScpHeader,
        body: &[u8],
    ) {
        // Client's initial "proceed" ack.
        assert_eq!(stream.read_u8().await.unwrap(), 0);
        let rec = format!("C{:04o} {} {}\n", header.mode, header.size, header.name);
        stream.write_all(rec.as_bytes()).await.unwrap();
        stream.flush().await.unwrap();
        // Client acks the C record.
        assert_eq!(stream.read_u8().await.unwrap(), 0);
        stream.write_all(body).await.unwrap();
        stream.write_all(&[0]).await.unwrap(); // end-of-file status
        stream.flush().await.unwrap();
        // Client's final ack.
        assert_eq!(stream.read_u8().await.unwrap(), 0);
    }

    /// Play the sink (server receives a file) side of an upload; return what it got.
    async fn serve_sink<S: AsyncRead + AsyncWrite + Unpin>(stream: &mut S) -> (ScpHeader, Vec<u8>) {
        // Sink is ready.
        stream.write_all(&[0]).await.unwrap();
        stream.flush().await.unwrap();
        // Read the C record line.
        let line = super::read_record_line(stream).await.unwrap();
        let header = super::parse_c_record(&line).unwrap();
        stream.write_all(&[0]).await.unwrap(); // accept the record
        stream.flush().await.unwrap();
        let mut body = vec![0u8; header.size as usize];
        stream.read_exact(&mut body).await.unwrap();
        // Trailing zero byte.
        assert_eq!(stream.read_u8().await.unwrap(), 0);
        stream.write_all(&[0]).await.unwrap(); // accept the body
        stream.flush().await.unwrap();
        (header, body)
    }

    #[tokio::test]
    async fn download_round_trips_a_file() {
        let (mut client, mut server) = tokio::io::duplex(4096);
        let header = ScpHeader { mode: 0o644, size: 13, name: "hello.txt".into() };
        let body = b"hello, world!".to_vec();

        let server_task = {
            let header = header.clone();
            let body = body.clone();
            tokio::spawn(async move {
                serve_source(&mut server, &header, &body).await;
            })
        };

        let mut out = Vec::new();
        let got = download_to(&mut client, &mut out).await.unwrap();
        server_task.await.unwrap();

        assert_eq!(got, header);
        assert_eq!(out, body);
    }

    #[tokio::test]
    async fn download_skips_a_leading_mtime_record() {
        let (mut client, mut server) = tokio::io::duplex(4096);
        let body = b"data".to_vec();
        let server_task = tokio::spawn(async move {
            assert_eq!(server.read_u8().await.unwrap(), 0);
            server.write_all(b"T1700000000 0 1700000000 0\n").await.unwrap();
            server.flush().await.unwrap();
            assert_eq!(server.read_u8().await.unwrap(), 0); // ack the T
            server.write_all(b"C0600 4 f\n").await.unwrap();
            server.flush().await.unwrap();
            assert_eq!(server.read_u8().await.unwrap(), 0); // ack the C
            server.write_all(b"data").await.unwrap();
            server.write_all(&[0]).await.unwrap();
            server.flush().await.unwrap();
            assert_eq!(server.read_u8().await.unwrap(), 0);
        });
        let mut out = Vec::new();
        let got = download_to(&mut client, &mut out).await.unwrap();
        server_task.await.unwrap();
        assert_eq!(out, body);
        assert_eq!(got.name, "f");
    }

    #[tokio::test]
    async fn download_surfaces_a_server_error() {
        let (mut client, mut server) = tokio::io::duplex(4096);
        tokio::spawn(async move {
            assert_eq!(server.read_u8().await.unwrap(), 0);
            // 0x01 warning marker + message.
            server.write_all(&[1]).await.unwrap();
            server.write_all(b"scp: /nope: No such file or directory\n").await.unwrap();
            server.flush().await.unwrap();
        });
        let mut out = Vec::new();
        let err = download_to(&mut client, &mut out).await.unwrap_err();
        assert!(err.to_string().contains("No such file"), "{err}");
    }

    #[tokio::test]
    async fn upload_round_trips_a_file() {
        let (mut client, mut server) = tokio::io::duplex(4096);
        let header = ScpHeader { mode: 0o644, size: 5, name: "up.bin".into() };
        let data = b"12345".to_vec();

        let server_task = tokio::spawn(async move { serve_sink(&mut server).await });

        let mut reader = Cursor::new(data.clone());
        upload_from(&mut client, &header, &mut reader).await.unwrap();
        let (got_header, got_body) = server_task.await.unwrap();

        assert_eq!(got_header.size, 5);
        assert_eq!(got_header.name, "up.bin");
        assert_eq!(got_body, data);
    }

    #[tokio::test]
    async fn upload_fails_when_sink_rejects_the_record() {
        let (mut client, mut server) = tokio::io::duplex(4096);
        tokio::spawn(async move {
            server.write_all(&[0]).await.unwrap(); // ready
            server.flush().await.unwrap();
            let _ = super::read_record_line(&mut server).await.unwrap();
            // Reject with a fatal status + message.
            server.write_all(&[2]).await.unwrap();
            server.write_all(b"scp: /ro/up.bin: Permission denied\n").await.unwrap();
            server.flush().await.unwrap();
        });
        let header = ScpHeader { mode: 0o644, size: 3, name: "up.bin".into() };
        let mut reader = Cursor::new(b"abc".to_vec());
        let err = upload_from(&mut client, &header, &mut reader).await.unwrap_err();
        assert!(err.to_string().contains("Permission denied"), "{err}");
    }
}
