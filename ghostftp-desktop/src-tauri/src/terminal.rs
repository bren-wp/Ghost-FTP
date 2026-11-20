use crate::session::SshSession;
use anyhow::{anyhow, Result};
use russh::ChannelMsg;
use serde::Serialize;
use std::collections::HashMap;
use std::sync::Arc;
use tauri::{AppHandle, Emitter};
use tokio::sync::{mpsc, Mutex, Semaphore};
use uuid::Uuid;

/// Defence-in-depth for Tauri IPC values, not just frontend widgets.
const MAX_PTY_SESSIONS: usize = 32;
const MAX_PTY_INPUT: usize = 128 * 1024;
fn validate_pty_size(cols: u32, rows: u32) -> Result<()> {
    if !(20..=500).contains(&cols) || !(5..=300).contains(&rows) {
        return Err(anyhow!("terminal dimensions outside allowed bounds"));
    }
    Ok(())
}

/// Preserve UTF-8 sequences split across SSH packets. Converting each packet
/// with from_utf8_lossy separately used to corrupt non-ASCII terminal output.
#[derive(Default)]
struct Utf8OutputDecoder {
    pending: Vec<u8>,
}

impl Utf8OutputDecoder {
    fn push(&mut self, chunk: &[u8]) -> String {
        if chunk.is_empty() {
            return String::new();
        }
        // Fast path for the common case of a complete valid UTF-8 packet.
        if self.pending.is_empty() {
            if let Ok(text) = std::str::from_utf8(chunk) {
                return text.to_owned();
            }
        }

        let mut bytes = Vec::with_capacity(self.pending.len() + chunk.len());
        bytes.append(&mut self.pending);
        bytes.extend_from_slice(chunk);
        let mut output = String::with_capacity(bytes.len());
        let mut offset = 0;

        while offset < bytes.len() {
            match std::str::from_utf8(&bytes[offset..]) {
                Ok(text) => {
                    output.push_str(text);
                    break;
                }
                Err(error) => {
                    let valid = error.valid_up_to();
                    output.push_str(
                        std::str::from_utf8(&bytes[offset..offset + valid])
                            .expect("validated UTF-8 prefix"),
                    );
                    offset += valid;
                    match error.error_len() {
                        Some(invalid) => {
                            output.push('\u{FFFD}');
                            offset += invalid;
                        }
                        None => {
                            // UTF-8 code points use at most four bytes.
                            // Only an incomplete 1–3-byte suffix is buffered.
                            self.pending.extend_from_slice(&bytes[offset..]);
                            break;
                        }
                    }
                }
            }
        }
        output
    }

    fn finish(&mut self) -> &'static str {
        if self.pending.is_empty() {
            ""
        } else {
            self.pending.clear();
            "\u{FFFD}"
        }
    }
}

enum PtyCommand {
    Input(Vec<u8>),
    Resize { cols: u32, rows: u32 },
    Close,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct TerminalDataEvent {
    terminal_id: String,
    data: String,
}

#[derive(Clone, Serialize)]
#[serde(rename_all = "camelCase")]
struct TerminalExitEvent {
    terminal_id: String,
    code: Option<i32>,
}

struct PtyHandle {
    tx: mpsc::Sender<PtyCommand>,
}

pub struct PtyManager {
    ptys: Arc<Mutex<HashMap<String, PtyHandle>>>,
    slots: Arc<Semaphore>,
}

impl PtyManager {
    pub fn new() -> Self {
        Self {
            ptys: Arc::new(Mutex::new(HashMap::new())),
            slots: Arc::new(Semaphore::new(MAX_PTY_SESSIONS)),
        }
    }

    pub async fn open(
        &self,
        session: &Arc<SshSession>,
        cols: u32,
        rows: u32,
        app: AppHandle,
    ) -> Result<String> {
        validate_pty_size(cols, rows)?;
        // Reserve before opening the remote channel: concurrent opens cannot
        // bypass the terminal session limit.
        let slot = self
            .slots
            .clone()
            .try_acquire_owned()
            .map_err(|_| anyhow!("too many active terminal sessions"))?;
        let id = Uuid::new_v4().to_string();

        let channel = {
            let h = session.handle.lock().await;
            h.channel_open_session().await?
        };
        channel
            .request_pty(true, "xterm-256color", cols, rows, 0, 0, &[])
            .await?;
        channel.request_shell(true).await?;

        let (tx, mut rx) = mpsc::channel::<PtyCommand>(64);
        let id_for_task = id.clone();
        let app_for_task = app.clone();
        let ptys = Arc::clone(&self.ptys);
        self.ptys.lock().await.insert(id.clone(), PtyHandle { tx });

        tokio::spawn(async move {
            let _slot = slot;
            let mut exit_code = None;
            let mut channel = channel;
            let mut stdout_decoder = Utf8OutputDecoder::default();
            let mut stderr_decoder = Utf8OutputDecoder::default();
            loop {
                tokio::select! {
                    cmd = rx.recv() => {
                        let Some(cmd) = cmd else { break };
                        match cmd {
                            PtyCommand::Input(bytes) => {
                                if channel.data(&bytes[..]).await.is_err() {
                                    break;
                                }
                            }
                            PtyCommand::Resize { cols, rows } => {
                                let _ = channel.window_change(cols, rows, 0, 0).await;
                            }
                            PtyCommand::Close => {
                                let _ = channel.eof().await;
                                break;
                            }
                        }
                    }
                    msg = channel.wait() => {
                        let Some(msg) = msg else { break };
                        match msg {
                            ChannelMsg::Data { ref data } => {
                                let text = stdout_decoder.push(data);
                                if !text.is_empty() {
                                    let _ = app_for_task.emit(
                                        "terminal://data",
                                        TerminalDataEvent {
                                            terminal_id: id_for_task.clone(),
                                            data: text,
                                        },
                                    );
                                }
                            }
                            ChannelMsg::ExtendedData { ref data, .. } => {
                                let text = stderr_decoder.push(data);
                                if !text.is_empty() {
                                    let _ = app_for_task.emit(
                                        "terminal://data",
                                        TerminalDataEvent {
                                            terminal_id: id_for_task.clone(),
                                            data: text,
                                        },
                                    );
                                }
                            }
                            ChannelMsg::ExitStatus { exit_status } => {
                                exit_code = Some(exit_status as i32);
                            }
                            ChannelMsg::Close | ChannelMsg::Eof => break,
                            _ => {}
                        }
                    }
                }
            }
            // Preserve the final incomplete code point as an explicit
            // replacement character before emitting terminal://exit.
            for decoder in [&mut stdout_decoder, &mut stderr_decoder] {
                let tail = decoder.finish();
                if !tail.is_empty() {
                    let _ = app_for_task.emit(
                        "terminal://data",
                        TerminalDataEvent {
                            terminal_id: id_for_task.clone(),
                            data: tail.to_string(),
                        },
                    );
                }
            }
            ptys.lock().await.remove(&id_for_task);
            let _ = app_for_task.emit(
                "terminal://exit",
                TerminalExitEvent {
                    terminal_id: id_for_task,
                    code: exit_code,
                },
            );
        });

        Ok(id)
    }

    pub async fn write(&self, id: &str, data: String) -> Result<()> {
        if data.len() > MAX_PTY_INPUT {
            return Err(anyhow!("terminal input exceeds size limit"));
        }
        let tx = self
            .ptys
            .lock()
            .await
            .get(id)
            .map(|h| h.tx.clone())
            .ok_or_else(|| anyhow!("terminal not found"))?;
        tx.send(PtyCommand::Input(data.into_bytes()))
            .await
            .map_err(|_| anyhow!("terminal closed"))
    }

    pub async fn resize(&self, id: &str, cols: u32, rows: u32) -> Result<()> {
        validate_pty_size(cols, rows)?;
        let tx = self
            .ptys
            .lock()
            .await
            .get(id)
            .map(|h| h.tx.clone())
            .ok_or_else(|| anyhow!("terminal not found"))?;
        tx.send(PtyCommand::Resize { cols, rows })
            .await
            .map_err(|_| anyhow!("terminal closed"))
    }

    pub async fn close(&self, id: &str) -> Result<()> {
        let h = self.ptys.lock().await.remove(id);
        if let Some(h) = h {
            let _ = h.tx.send(PtyCommand::Close).await;
        }
        Ok(())
    }
}

#[cfg(test)]
mod ipc_security_tests {
    use super::*;
    #[test]
    fn terminal_utf8_decoder_preserves_split_multibyte_output() {
        let mut decoder = Utf8OutputDecoder::default();
        assert_eq!(decoder.push(b"Hello \xC5"), "Hello ");
        assert_eq!(decoder.push(b"\xBE \xF0\x9F"), "ž ");
        assert_eq!(decoder.push(b"\x9A\x80!"), "🚀!");
        assert_eq!(decoder.finish(), "");
    }

    #[test]
    fn terminal_utf8_decoder_handles_invalid_bytes_without_unbounded_buffering() {
        let mut decoder = Utf8OutputDecoder::default();
        assert_eq!(decoder.push(b"a\xFFb"), "a�b");
        assert_eq!(decoder.push(b"\xE2\x82"), "");
        assert!(decoder.pending.len() <= 3);
        assert_eq!(decoder.push(b"\xAC!"), "€!");
        assert_eq!(decoder.push(b"\xE2"), "");
        assert_eq!(decoder.finish(), "�");
        assert!(decoder.pending.is_empty());
        assert_eq!(decoder.finish(), "");
    }

    #[test]
    fn backend_rejects_adversarial_terminal_dimensions() {
        assert!(validate_pty_size(80, 24).is_ok());
        for (cols, rows) in [(0, 0), (1, 24), (65535, 1), (80, 1000)] {
            assert!(validate_pty_size(cols, rows).is_err());
        }
    }

    #[test]
    fn terminal_capacity_is_atomic_and_released_when_task_ends() {
        let manager = PtyManager::new();
        let mut permits = (0..MAX_PTY_SESSIONS)
            .map(|_| manager.slots.clone().try_acquire_owned().unwrap())
            .collect::<Vec<_>>();
        assert!(manager.slots.clone().try_acquire_owned().is_err());
        permits.pop();
        assert!(manager.slots.clone().try_acquire_owned().is_ok());
    }
}
