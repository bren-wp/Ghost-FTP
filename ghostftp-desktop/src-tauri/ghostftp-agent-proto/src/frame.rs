//! Length-prefixed framing over any async byte stream.
//!
//! Every frame is a big-endian `u32` length followed by that many bytes. These
//! carry Noise handshake messages and encrypted transport segments — both of
//! which are individually bounded well under 64 KiB — so we cap an incoming
//! frame at [`MAX_FRAME`] and treat anything larger as a protocol violation
//! (a corrupt or hostile peer) rather than trying to allocate it.

use anyhow::{bail, Result};
use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};

/// Hard ceiling on a single frame. A Noise transport message is at most 65535
/// bytes (ciphertext + tag); the handshake messages are smaller. 128 KiB leaves
/// generous headroom while still refusing an absurd length prefix outright.
pub const MAX_FRAME: usize = 128 * 1024;

/// Write one length-prefixed frame. `data` must fit in [`MAX_FRAME`].
pub async fn write_frame<W: AsyncWrite + Unpin>(w: &mut W, data: &[u8]) -> Result<()> {
    if data.len() > MAX_FRAME {
        bail!("frame too large to send: {} bytes", data.len());
    }
    w.write_all(&(data.len() as u32).to_be_bytes()).await?;
    w.write_all(data).await?;
    w.flush().await?;
    Ok(())
}

/// Read one length-prefixed frame. Returns the frame body. Errors if the peer
/// announces a length past [`MAX_FRAME`] or the stream ends mid-frame.
pub async fn read_frame<R: AsyncRead + Unpin>(r: &mut R) -> Result<Vec<u8>> {
    let mut len_buf = [0u8; 4];
    r.read_exact(&mut len_buf).await?;
    let len = u32::from_be_bytes(len_buf) as usize;
    if len > MAX_FRAME {
        bail!("peer announced oversized frame: {len} bytes (max {MAX_FRAME})");
    }
    let mut buf = vec![0u8; len];
    r.read_exact(&mut buf).await?;
    Ok(buf)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn roundtrips_frames() {
        let (mut a, mut b) = tokio::io::duplex(1024);
        let payload = b"hello ghostftp".to_vec();
        let p2 = payload.clone();
        let writer = tokio::spawn(async move {
            write_frame(&mut a, &p2).await.unwrap();
            write_frame(&mut a, &[]).await.unwrap(); // empty frame is legal
        });
        let got = read_frame(&mut b).await.unwrap();
        let empty = read_frame(&mut b).await.unwrap();
        writer.await.unwrap();
        assert_eq!(got, payload);
        assert!(empty.is_empty());
    }

    #[tokio::test]
    async fn rejects_oversized_length_prefix() {
        let (mut a, mut b) = tokio::io::duplex(64);
        tokio::spawn(async move {
            // Announce a length far past MAX_FRAME without sending the body.
            let _ = a.write_all(&(u32::MAX).to_be_bytes()).await;
        });
        let err = read_frame(&mut b).await.unwrap_err();
        assert!(err.to_string().contains("oversized"));
    }
}
