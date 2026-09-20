//! The encrypted channel: a Noise handshake, then message-framed transport.
//!
//! Two handshake shapes, both XX-based so the driver is shared:
//!   * **pairing** — `Noise_XXpsk3` with a PSK derived from the 6-digit code.
//!     After it completes, each side reads the other's static public key
//!     ([`SecureChannel::remote_static`]) and pins it for next time.
//!   * **paired** — plain `Noise_XX`; after it completes the controller verifies
//!     the daemon's static key equals the pin (and vice-versa), refusing an
//!     unrecognised peer.
//!
//! Transport messages: a logical payload (serialised JSON) is split into Noise
//! segments that each fit under the 65535-byte per-message limit, every segment
//! carrying a 1-byte continuation flag, so an arbitrarily large listing or file
//! chunk streams over a sequence of frames.

use crate::frame::{read_frame, write_frame};
use anyhow::{anyhow, bail, Context, Result};
use serde::de::DeserializeOwned;
use serde::Serialize;
use snow::{HandshakeState, TransportState};
use tokio::io::{AsyncRead, AsyncWrite};

/// Noise pattern for a first-time pairing handshake (PSK from the code).
const PAIR_PATTERN: &str = "Noise_XXpsk3_25519_ChaChaPoly_SHA256";
/// Noise pattern for a normal, already-paired handshake (pinned keys).
const PAIRED_PATTERN: &str = "Noise_XX_25519_ChaChaPoly_SHA256";

/// A Noise transport message is capped at 65535 bytes (ciphertext incl. the
/// 16-byte tag). Leave room for the tag and our 1-byte continuation flag.
const MAX_SEGMENT: usize = 65535 - 16 - 1;

/// Which end of the handshake this peer is. The controller (Ghost FTP) initiates; the
/// daemon responds.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Role {
    Initiator,
    Responder,
}

/// How to authenticate this handshake.
pub enum Auth {
    /// First-time pairing: mix this PSK (from the code) into `XXpsk3`.
    Pairing { psk: [u8; 32] },
    /// Already paired: plain XX, then require the remote static key to equal
    /// `expect_remote` (base64-decoded pin). `None` accepts any key (used only
    /// where the caller pins afterwards itself).
    Paired { expect_remote: Option<Vec<u8>> },
}

/// An established, encrypted, message-framed connection to the peer.
pub struct SecureChannel<S> {
    stream: S,
    noise: TransportState,
    remote_static: Vec<u8>,
}

impl<S> SecureChannel<S>
where
    S: AsyncRead + AsyncWrite + Unpin,
{
    /// Drive the Noise handshake to completion and wrap the stream. On the
    /// paired path this also enforces the pin; on the pairing path the caller
    /// pins [`Self::remote_static`] afterwards.
    pub async fn establish(
        mut stream: S,
        role: Role,
        local_private: &[u8],
        auth: Auth,
    ) -> Result<Self> {
        let mut builder = snow::Builder::new(
            match &auth {
                Auth::Pairing { .. } => PAIR_PATTERN,
                Auth::Paired { .. } => PAIRED_PATTERN,
            }
            .parse()
            .context("parse noise params")?,
        )
        .local_private_key(local_private);
        if let Auth::Pairing { psk } = &auth {
            builder = builder.psk(3, psk);
        }
        let mut hs = match role {
            Role::Initiator => builder.build_initiator(),
            Role::Responder => builder.build_responder(),
        }
        .context("build noise handshake")?;

        drive_handshake(&mut stream, &mut hs).await?;

        let remote_static = hs
            .get_remote_static()
            .ok_or_else(|| anyhow!("handshake produced no remote static key"))?
            .to_vec();

        if let Auth::Paired {
            expect_remote: Some(expected),
        } = &auth
        {
            if &remote_static != expected {
                bail!(
                    "peer identity mismatch — expected pinned key {}, got {}. \
                     Refusing (possible impersonation); re-pair if this machine's key legitimately changed.",
                    crate::identity::encode_public(expected),
                    crate::identity::encode_public(&remote_static),
                );
            }
        }

        let noise = hs.into_transport_mode().context("enter transport mode")?;
        Ok(Self {
            stream,
            noise,
            remote_static,
        })
    }

    /// The peer's static public key (raw 32 bytes). After a pairing handshake
    /// the caller persists this as the pin.
    pub fn remote_static(&self) -> &[u8] {
        &self.remote_static
    }

    /// The peer's static key as a base64 pin string.
    pub fn remote_static_b64(&self) -> String {
        crate::identity::encode_public(&self.remote_static)
    }

    /// Encrypt and send one logical message, split across as many Noise segments
    /// as its length requires.
    pub async fn send<T: Serialize>(&mut self, msg: &T) -> Result<()> {
        let plaintext = serde_json::to_vec(msg).context("serialize message")?;
        let mut segments = plaintext.chunks(MAX_SEGMENT).peekable();
        // An empty payload still needs one (final) segment.
        if plaintext.is_empty() {
            self.send_segment(&[], true).await?;
            return Ok(());
        }
        while let Some(seg) = segments.next() {
            let last = segments.peek().is_none();
            self.send_segment(seg, last).await?;
        }
        Ok(())
    }

    async fn send_segment(&mut self, data: &[u8], last: bool) -> Result<()> {
        // Prefix the plaintext segment with its continuation flag, then encrypt.
        let mut plain = Vec::with_capacity(data.len() + 1);
        plain.push(if last { 0u8 } else { 1u8 });
        plain.extend_from_slice(data);
        let mut out = vec![0u8; plain.len() + 16];
        let n = self
            .noise
            .write_message(&plain, &mut out)
            .context("encrypt segment")?;
        write_frame(&mut self.stream, &out[..n]).await
    }

    /// Receive and decrypt one logical message, reassembling its segments.
    pub async fn recv<T: DeserializeOwned>(&mut self) -> Result<T> {
        let mut payload = Vec::new();
        loop {
            let frame = read_frame(&mut self.stream).await?;
            let mut buf = vec![0u8; frame.len()];
            let n = self
                .noise
                .read_message(&frame, &mut buf)
                .context("decrypt segment")?;
            if n == 0 {
                bail!("decrypted segment missing continuation flag");
            }
            let last = buf[0] == 0;
            payload.extend_from_slice(&buf[1..n]);
            if last {
                break;
            }
            if payload.len() > crate::frame::MAX_FRAME * 256 {
                bail!("reassembled message exceeded sanity cap");
            }
        }
        serde_json::from_slice(&payload).context("deserialize message")
    }

    /// Consume the channel and return the underlying stream (rarely needed).
    pub fn into_inner(self) -> S {
        self.stream
    }
}

/// Shared handshake loop: alternate write/read per `snow`'s turn tracking until
/// the pattern completes. Handshake messages carry no payload here.
async fn drive_handshake<S>(stream: &mut S, hs: &mut HandshakeState) -> Result<()>
where
    S: AsyncRead + AsyncWrite + Unpin,
{
    let mut buf = vec![0u8; 1024];
    while !hs.is_handshake_finished() {
        if hs.is_my_turn() {
            let n = hs
                .write_message(&[], &mut buf)
                .context("write handshake message")?;
            write_frame(stream, &buf[..n]).await?;
        } else {
            let frame = read_frame(stream).await?;
            let mut scratch = vec![0u8; frame.len().max(1)];
            hs.read_message(&frame, &mut scratch)
                .context("read handshake message")?;
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identity::Identity;
    use crate::msg::{Request, Response};

    /// End-to-end: pair over a duplex pipe, then exchange app messages both ways.
    #[tokio::test]
    async fn pairing_handshake_then_message_exchange() {
        let server_id = Identity::generate().unwrap();
        let client_id = Identity::generate().unwrap();
        let psk = crate::pairing::psk_from_code("424242");

        let (c, s) = tokio::io::duplex(64 * 1024);
        let sk = server_id.private_bytes().unwrap();
        let server_pub = server_id.public_bytes().unwrap();

        let server = tokio::spawn(async move {
            let mut ch = SecureChannel::establish(s, Role::Responder, &sk, Auth::Pairing { psk })
                .await
                .unwrap();
            let req: Request = ch.recv().await.unwrap();
            assert!(matches!(req, Request::Ping));
            ch.send(&Response::Pong).await.unwrap();
            ch.remote_static().to_vec()
        });

        let ck = client_id.private_bytes().unwrap();
        let mut ch = SecureChannel::establish(c, Role::Initiator, &ck, Auth::Pairing { psk })
            .await
            .unwrap();
        // Client learns + could pin the server's key.
        assert_eq!(ch.remote_static(), server_pub.as_slice());
        ch.send(&Request::Ping).await.unwrap();
        let resp: Response = ch.recv().await.unwrap();
        assert!(matches!(resp, Response::Pong));

        let seen_client = server.await.unwrap();
        assert_eq!(seen_client, client_id.public_bytes().unwrap());
    }

    /// A paired handshake with the correct pin succeeds; a wrong pin is refused.
    #[tokio::test]
    async fn paired_handshake_enforces_pin() {
        let server_id = Identity::generate().unwrap();
        let client_id = Identity::generate().unwrap();
        let server_pub = server_id.public_bytes().unwrap();
        let wrong = Identity::generate().unwrap().public_bytes().unwrap();

        // Correct pin → connects.
        {
            let (c, s) = tokio::io::duplex(64 * 1024);
            let sk = server_id.private_bytes().unwrap();
            let srv = tokio::spawn(async move {
                let _ = SecureChannel::establish(
                    s,
                    Role::Responder,
                    &sk,
                    Auth::Paired {
                        expect_remote: None,
                    },
                )
                .await;
            });
            let ck = client_id.private_bytes().unwrap();
            let ch = SecureChannel::establish(
                c,
                Role::Initiator,
                &ck,
                Auth::Paired {
                    expect_remote: Some(server_pub.clone()),
                },
            )
            .await;
            assert!(ch.is_ok(), "correct pin should connect");
            let _ = srv.await;
        }

        // Wrong pin → refused after the handshake completes.
        {
            let (c, s) = tokio::io::duplex(64 * 1024);
            let sk = server_id.private_bytes().unwrap();
            let srv = tokio::spawn(async move {
                let _ = SecureChannel::establish(
                    s,
                    Role::Responder,
                    &sk,
                    Auth::Paired {
                        expect_remote: None,
                    },
                )
                .await;
            });
            let ck = client_id.private_bytes().unwrap();
            let ch = SecureChannel::establish(
                c,
                Role::Initiator,
                &ck,
                Auth::Paired {
                    expect_remote: Some(wrong),
                },
            )
            .await;
            assert!(ch.is_err(), "wrong pin must be refused");
            let _ = srv.await;
        }
    }

    /// A payload larger than one Noise segment survives the multi-segment split.
    #[tokio::test]
    async fn large_message_survives_segmentation() {
        let server_id = Identity::generate().unwrap();
        let client_id = Identity::generate().unwrap();
        let psk = crate::pairing::psk_from_code("000000");
        let (c, s) = tokio::io::duplex(1024 * 1024);
        let sk = server_id.private_bytes().unwrap();

        let big = "x".repeat(200_000);
        let big2 = big.clone();
        let server = tokio::spawn(async move {
            let mut ch = SecureChannel::establish(s, Role::Responder, &sk, Auth::Pairing { psk })
                .await
                .unwrap();
            let got: Response = ch.recv().await.unwrap();
            match got {
                Response::File { data, .. } => assert_eq!(data, big2),
                _ => panic!("wrong message"),
            }
        });
        let ck = client_id.private_bytes().unwrap();
        let mut ch = SecureChannel::establish(c, Role::Initiator, &ck, Auth::Pairing { psk })
            .await
            .unwrap();
        ch.send(&Response::File {
            data: big,
            bytes: 200_000,
            truncated: false,
        })
        .await
        .unwrap();
        server.await.unwrap();
    }
}
