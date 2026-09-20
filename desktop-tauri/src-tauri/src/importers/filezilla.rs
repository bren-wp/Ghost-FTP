use super::ProfilePreview;
use anyhow::{Context, Result};
use quick_xml::events::Event;
use quick_xml::Reader;
use std::path::PathBuf;

/// Locate FileZilla's `sitemanager.xml` for the current user. FileZilla
/// stores its config under platform-conventional config dirs:
///   - Windows: %APPDATA%\FileZilla\sitemanager.xml
///   - Linux:   ~/.config/filezilla/sitemanager.xml (or ~/.filezilla/ on
///              older installs — we check both)
pub fn default_path() -> Option<PathBuf> {
    if cfg!(target_os = "windows") {
        if let Some(appdata) = std::env::var_os("APPDATA") {
            return Some(
                PathBuf::from(appdata)
                    .join("FileZilla")
                    .join("sitemanager.xml"),
            );
        }
    }
    // Linux + fallback
    let cfg = dirs::config_dir().map(|c| c.join("filezilla").join("sitemanager.xml"));
    if let Some(p) = cfg.as_ref() {
        if p.exists() {
            return cfg;
        }
    }
    if let Some(home) = dirs::home_dir() {
        let legacy = home.join(".filezilla").join("sitemanager.xml");
        if legacy.exists() {
            return Some(legacy);
        }
    }
    cfg
}

pub fn parse_file(path: &PathBuf) -> Result<Vec<ProfilePreview>> {
    let text =
        std::fs::read_to_string(path).with_context(|| format!("reading {}", path.display()))?;
    parse(&text)
}

/// FileZilla's sitemanager.xml is a nested tree of <Folder> and <Server>
/// nodes. We walk it depth-first and collect every <Server>, attaching the
/// folder breadcrumb to the preview's `note` field.
pub fn parse(xml: &str) -> Result<Vec<ProfilePreview>> {
    let mut reader = Reader::from_str(xml);
    reader.config_mut().trim_text(true);
    let mut buf = Vec::new();

    let mut out: Vec<ProfilePreview> = Vec::new();
    let mut folder_stack: Vec<String> = Vec::new();
    let mut current_server: Option<ServerAccum> = None;
    let mut current_field: Option<String> = None;
    let mut expecting_folder_name = false;

    loop {
        match reader.read_event_into(&mut buf) {
            Ok(Event::Start(e)) => {
                let name = String::from_utf8_lossy(e.name().as_ref()).into_owned();
                match name.as_str() {
                    "Folder" => {
                        // The Folder element contains a text node *before*
                        // its child <Server>/<Folder> entries — that text is
                        // the folder's display name. We can't easily peek;
                        // we set a flag and grab the next Text event.
                        expecting_folder_name = true;
                        folder_stack.push(String::new());
                    }
                    "Server" => {
                        current_server = Some(ServerAccum::default());
                    }
                    _ => {
                        current_field = Some(name);
                    }
                }
            }
            Ok(Event::Text(t)) => {
                let txt = t
                    .unescape()
                    .map(|c| c.into_owned())
                    .unwrap_or_else(|_| String::new());
                if expecting_folder_name && !txt.trim().is_empty() {
                    if let Some(slot) = folder_stack.last_mut() {
                        *slot = txt.trim().to_string();
                    }
                    expecting_folder_name = false;
                    continue;
                }
                if let (Some(server), Some(field)) =
                    (current_server.as_mut(), current_field.as_ref())
                {
                    server.set(field, txt.trim());
                }
            }
            Ok(Event::End(e)) => {
                let name = String::from_utf8_lossy(e.name().as_ref()).into_owned();
                match name.as_str() {
                    "Folder" => {
                        folder_stack.pop();
                    }
                    "Server" => {
                        if let Some(server) = current_server.take() {
                            if let Some(preview) = server.into_preview(&folder_stack) {
                                out.push(preview);
                            }
                        }
                    }
                    _ => {
                        current_field = None;
                    }
                }
            }
            Ok(Event::Eof) => break,
            Ok(_) => {}
            Err(e) => {
                return Err(anyhow::anyhow!(
                    "sitemanager.xml parse error at {}: {e}",
                    reader.buffer_position()
                ));
            }
        }
        buf.clear();
    }
    Ok(out)
}

#[derive(Default)]
struct ServerAccum {
    name: Option<String>,
    host: Option<String>,
    port: Option<u16>,
    protocol: Option<u8>,
    user: Option<String>,
}

impl ServerAccum {
    fn set(&mut self, field: &str, value: &str) {
        if value.is_empty() {
            return;
        }
        match field {
            "Name" => self.name = Some(value.to_string()),
            "Host" => self.host = Some(value.to_string()),
            "Port" => self.port = value.parse().ok(),
            "Protocol" => self.protocol = value.parse().ok(),
            "User" => self.user = Some(value.to_string()),
            _ => {}
        }
    }

    fn into_preview(self, folder_stack: &[String]) -> Option<ProfilePreview> {
        let host = self.host?;
        // FileZilla protocol numeric ids (see FileZilla source, ServerProtocol):
        //   0  = FTP
        //   1  = SFTP
        //   3  = FTPES (explicit FTP over TLS)
        //   4  = FTPS  (implicit FTP over TLS)
        //   6  = STORJ + various — out of scope for us
        let (proto, default_port): (&str, u16) = match self.protocol.unwrap_or(0) {
            1 => ("sftp", 22),
            3 | 4 => ("ftps", 21),
            0 => ("ftp", 21),
            _ => return None, // unsupported protocol
        };

        let mut p = ProfilePreview::new(self.name.unwrap_or_else(|| host.clone()));
        p.protocol = proto.into();
        p.host = host;
        p.port = self.port.unwrap_or(default_port);
        p.username = self.user.unwrap_or_default();
        let breadcrumb = folder_stack
            .iter()
            .filter(|s| !s.is_empty())
            .cloned()
            .collect::<Vec<_>>()
            .join(" / ");
        p.note = if breadcrumb.is_empty() {
            Some("from FileZilla".to_string())
        } else {
            Some(format!("FileZilla: {breadcrumb}"))
        };
        Some(p)
    }
}
