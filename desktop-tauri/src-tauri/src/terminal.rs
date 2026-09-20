use crate::session::SshSession;
use anyhow::{anyhow, Result};
use russh::ChannelMsg;
use serde::Serialize;
use std::collections::HashMap;
use std::sync::Arc;
use tauri::{AppHandle, Emitter};
use tokio::sync::{mpsc, Mutex};
use uuid::Uuid;

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
    ptys: Mutex<HashMap<String, PtyHandle>>,
}

impl PtyManager {
    pub fn new() -> Self {
        Self {
            ptys: Mutex::new(HashMap::new()),
        }
    }

    pub async fn open(
        &self,
        session: &Arc<SshSession>,
        cols: u32,
        rows: u32,
        app: AppHandle,
    ) -> Result<String> {
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

        tokio::spawn(async move {
            let mut channel = channel;
            loop {
                tokio::select! {
                    biased;
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
                                let s = String::from_utf8_lossy(data).to_string();
                                let _ = app_for_task.emit(
                                    "terminal://data",
                                    TerminalDataEvent {
                                        terminal_id: id_for_task.clone(),
                                        data: s,
                                    },
                                );
                            }
                            ChannelMsg::ExtendedData { ref data, .. } => {
                                let s = String::from_utf8_lossy(data).to_string();
                                let _ = app_for_task.emit(
                                    "terminal://data",
                                    TerminalDataEvent {
                                        terminal_id: id_for_task.clone(),
                                        data: s,
                                    },
                                );
                            }
                            ChannelMsg::ExitStatus { exit_status } => {
                                let _ = app_for_task.emit(
                                    "terminal://exit",
                                    TerminalExitEvent {
                                        terminal_id: id_for_task.clone(),
                                        code: Some(exit_status as i32),
                                    },
                                );
                            }
                            ChannelMsg::Close | ChannelMsg::Eof => break,
                            _ => {}
                        }
                    }
                }
            }
            let _ = app_for_task.emit(
                "terminal://exit",
                TerminalExitEvent {
                    terminal_id: id_for_task,
                    code: None,
                },
            );
        });

        self.ptys
            .lock()
            .await
            .insert(id.clone(), PtyHandle { tx });
        Ok(id)
    }

    pub async fn write(&self, id: &str, data: String) -> Result<()> {
        let map = self.ptys.lock().await;
        let h = map
            .get(id)
            .ok_or_else(|| anyhow!("terminal {id} not found"))?;
        h.tx.send(PtyCommand::Input(data.into_bytes()))
            .await
            .map_err(|_| anyhow!("terminal closed"))
    }

    pub async fn resize(&self, id: &str, cols: u32, rows: u32) -> Result<()> {
        let map = self.ptys.lock().await;
        let h = map
            .get(id)
            .ok_or_else(|| anyhow!("terminal {id} not found"))?;
        h.tx.send(PtyCommand::Resize { cols, rows })
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
