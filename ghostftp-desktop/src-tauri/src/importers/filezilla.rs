use super::ProfilePreview;
use anyhow::{bail, Context, Result};
use quick_xml::events::Event;
use quick_xml::Reader;
use std::io::Read;
use std::path::PathBuf;

const MAX_XML_BYTES: usize = 8 * 1024 * 1024;
const MAX_IMPORT_SITES: usize = 2_048;
const MAX_FOLDER_DEPTH: usize = 32;

/// Locate FileZilla's `sitemanager.xml` for the current user. FileZilla
/// stores its config under platform-conventional config dirs:
///   - Windows: %APPDATA%\FileZilla\sitemanager.xml
/// - Linux: ~/.config/filezilla/sitemanager.xml (or ~/.filezilla/ on
///   older installs — we check both)
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
    // FileZilla profiles can come from an arbitrary user-selected XML file.
    // Bound allocations before parsing so a huge file cannot freeze the UI.
    let mut text = String::new();
    std::fs::File::open(path)
        .with_context(|| format!("opening {}", path.display()))?
        .take((MAX_XML_BYTES + 1) as u64)
        .read_to_string(&mut text)
        .with_context(|| format!("reading {}", path.display()))?;
    parse(&text)
}

/// FileZilla's sitemanager.xml is a nested tree of <Folder> and <Server>
/// nodes. We walk it depth-first and collect every <Server>, attaching the
/// folder breadcrumb to the preview's `note` field.
pub fn parse(xml: &str) -> Result<Vec<ProfilePreview>> {
    if xml.len() > MAX_XML_BYTES {
        bail!("FileZilla import exceeds the 8 MiB safety limit");
    }
    let mut reader = Reader::from_str(xml);
    reader.config_mut().trim_text(true);
    let mut buf = Vec::new();

    let mut out: Vec<ProfilePreview> = Vec::new();
    let mut folder_stack: Vec<String> = Vec::new();
    let mut current_server: Option<ServerAccum> = None;
    let mut current_field: Option<String> = None;
    let mut opened_root = false;
    let mut closed_root = false;
    let mut opened_servers = false;
    let mut closed_servers = false;

    loop {
        match reader.read_event_into(&mut buf) {
            Ok(Event::Start(e)) => {
                let name = String::from_utf8_lossy(e.name().as_ref()).into_owned();
                match name.as_str() {
                    "FileZilla3" => {
                        if opened_root || closed_root {
                            bail!("Unexpected FileZilla document root");
                        }
                        opened_root = true;
                    }
                    "Servers" => {
                        if !opened_root || opened_servers {
                            bail!("Unexpected FileZilla Servers element");
                        }
                        opened_servers = true;
                    }
                    "Folder" => {
                        if !opened_servers || closed_servers {
                            bail!("FileZilla folder is outside the Servers section");
                        }
                        if current_server.is_some() {
                            bail!("A FileZilla site may not contain nested folders");
                        }
                        if folder_stack.len() >= MAX_FOLDER_DEPTH {
                            bail!("FileZilla folder hierarchy exceeds the safe depth limit");
                        }
                        folder_stack.push(String::new());
                        current_field = None;
                    }
                    "Server" => {
                        if !opened_servers || closed_servers {
                            bail!("FileZilla site is outside the Servers section");
                        }
                        if current_server.is_some() {
                            bail!("Nested FileZilla server entries are not supported");
                        }
                        current_field = None;
                        current_server = Some(ServerAccum::default());
                    }
                    _ if current_server.is_some() => {
                        current_field = Some(name);
                    }
                    _ => {}
                }
            }
            Ok(Event::Text(t)) => {
                // A malformed XML escape must never silently erase a host
                // or turn a partial site into a plausibly valid import.
                let txt = t.unescape().context("invalid FileZilla XML text escape")?;
                let trimmed = txt.trim();
                if trimmed.is_empty() {
                    buf.clear();
                    continue;
                }
                if let Some(server) = current_server.as_mut() {
                    if let Some(field) = current_field.as_ref() {
                        server.set(field, trimmed);
                    } else if server.name.is_none() {
                        // FileZilla can save the site label as trailing direct
                        // text in <Server>, not only as a <Name> element.
                        server.name = Some(trimmed.to_owned());
                    }
                } else if let Some(folder) = folder_stack.last_mut() {
                    // Only the first non-empty direct text names this folder.
                    // Child <Server> field text must never overwrite its name.
                    if folder.is_empty() {
                        *folder = trimmed.to_owned();
                    }
                }
            }
            Ok(Event::End(e)) => {
                let name = String::from_utf8_lossy(e.name().as_ref()).into_owned();
                match name.as_str() {
                    "FileZilla3" => {
                        closed_root = true;
                    }
                    "Servers" => {
                        if current_server.is_some() || !folder_stack.is_empty() {
                            bail!("FileZilla Servers closed with unfinished entries");
                        }
                        closed_servers = true;
                    }
                    "Folder" => {
                        if folder_stack.pop().is_none() {
                            bail!("Unexpected end of FileZilla folder");
                        }
                        current_field = None;
                    }
                    "Server" => {
                        let server = current_server
                            .take()
                            .context("Unexpected end of FileZilla server")?;
                        current_field = None;
                        if let Some(preview) = server.into_preview(&folder_stack) {
                            if out.len() >= MAX_IMPORT_SITES {
                                bail!("FileZilla import exceeds the 2048-site safety limit");
                            }
                            out.push(preview);
                        }
                    }
                    _ => {
                        current_field = None;
                    }
                }
            }
            Ok(Event::Empty(e)) => {
                // Empty server collections are valid exported FileZilla sites.
                if e.name().as_ref() == b"Servers" {
                    if !opened_root || opened_servers {
                        bail!("Unexpected empty FileZilla Servers element");
                    }
                    opened_servers = true;
                    closed_servers = true;
                }
            }
            Ok(Event::DocType(_)) => bail!("FileZilla imports must not contain XML DTDs"),
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
    if current_server.is_some()
        || !folder_stack.is_empty()
        || !opened_root
        || !closed_root
        || !opened_servers
        || !closed_servers
    {
        bail!("Truncated or invalid FileZilla saved-site XML");
    }
    Ok(out)
}

/// Reject untrusted relative and network key paths from an imported profile.
/// Relative paths resolve against Ghost FTP's process working directory, while
/// Windows UNC paths may access an attacker-controlled SMB server. A saved
/// FileZilla site still requires user approval and an explicit connection.
fn safe_local_keyfile_path(path: &str) -> bool {
    if path.is_empty() || path.chars().any(char::is_control) {
        return false;
    }
    if path.starts_with("//") || path.starts_with(r"\\") {
        return false;
    }
    let bytes = path.as_bytes();
    let windows_absolute = bytes.len() >= 3
        && bytes[0].is_ascii_alphabetic()
        && bytes[1] == b':'
        && (bytes[2] == b'\\' || bytes[2] == b'/');
    path.starts_with('/') || path.starts_with("~/") || windows_absolute
}

#[derive(Default)]
struct ServerAccum {
    name: Option<String>,
    host: Option<String>,
    port: Option<u16>,
    protocol: Option<u8>,
    user: Option<String>,
    logon_type: Option<u8>,
    keyfile: Option<String>,
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
            "Logontype" => self.logon_type = value.parse().ok(),
            "Keyfile" => self.keyfile = Some(value.to_string()),
            _ => {}
        }
    }

    fn into_preview(self, folder_stack: &[String]) -> Option<ProfilePreview> {
        let host = self.host?.trim().to_string();
        if host.is_empty() {
            return None;
        }
        // FileZilla ServerProtocol numeric IDs are persisted in sitemanager.xml:
        //   0 FTP (attempts AUTH TLS, may fall back to plaintext)
        //   1 SFTP
        //   3 FTPS (implicit TLS, typically port 990)
        //   4 FTPES (required explicit TLS, typically port 21)
        //   6 INSECURE_FTP (plaintext only)
        // Ghost FTP currently supports required explicit FTPS, not implicit
        // TLS or opportunistic downgrade. Never mis-import implicit TLS as
        // explicit, and never downgrade an opportunistic-TLS site to plaintext.
        let (proto, default_port, security_note): (&str, u16, Option<&str>) =
            match self.protocol.unwrap_or(0) {
                1 => ("sftp", 22, None),
                4 => ("ftps", 21, None),
                0 => (
                    "ftps",
                    21,
                    Some("FileZilla optional TLS upgraded to required explicit FTPS; verify server support"),
                ),
                6 => (
                    "ftp",
                    21,
                    Some("FileZilla insecure FTP: passwords and file data are sent without encryption"),
                ),
                3 => return None, // implicit TLS must not be represented as explicit
                _ => return None, // unsupported protocol
            };

        let mut p = ProfilePreview::new(self.name.unwrap_or_else(|| host.clone()));
        p.protocol = proto.into();
        p.host = host;
        p.port = self.port.filter(|port| *port != 0).unwrap_or(default_port);
        p.username = self.user.unwrap_or_default();
        // FileZilla's LogonType::key is 5. A Keyfile field can remain in
        // an export even after changing the logon mode; only honor it when
        // the explicitly selected SFTP key-file mode is still active.
        let key_warning = if proto == "sftp" && self.logon_type == Some(5) {
            match self.keyfile {
                Some(path) if !path.trim().is_empty() => {
                    let path = path.trim();
                    if !safe_local_keyfile_path(path) {
                        Some("FileZilla key path must be local and absolute; choose a safe SSH private key in Ghost FTP")
                    } else if path.to_ascii_lowercase().ends_with(".ppk") {
                        // russh's private-key loader does not promise support
                        // for PuTTY's proprietary PPK format. Never mark a
                        // PPK path as working key authentication by default.
                        Some("FileZilla PuTTY .ppk key needs conversion to OpenSSH/PEM before connecting")
                    } else {
                        // This transfers only the path string, never key
                        // material, and does not read the key during import.
                        p.identity_file = Some(path.to_owned());
                        None
                    }
                }
                _ => Some("FileZilla key-file login has no key path; select an SSH private key before connecting"),
            }
        } else {
            None
        };
        let breadcrumb = folder_stack
            .iter()
            .filter(|s| !s.is_empty())
            .cloned()
            .collect::<Vec<_>>()
            .join(" / ");
        let origin = if breadcrumb.is_empty() {
            "from FileZilla".to_string()
        } else {
            format!("FileZilla: {breadcrumb}")
        };
        p.note = Some(match (security_note, key_warning) {
            (Some(security), Some(key)) => format!("{origin} — {security} — {key}"),
            (Some(security), None) => format!("{origin} — {security}"),
            (None, Some(key)) => format!("{origin} — {key}"),
            (None, None) => origin,
        });
        // Import into Ghost FTP's existing Sites folder surface, rather than
        // displaying the breadcrumb only as an informational note.
        p.group = (!breadcrumb.is_empty()).then_some(breadcrumb);
        Some(p)
    }
}

#[cfg(test)]
mod tests {
    use super::parse;

    fn site(protocol: u8) -> Vec<super::ProfilePreview> {
        let xml = format!(
            "<FileZilla3><Servers><Server><Name>Test</Name><Host>example.com</Host><Port>0</Port><Protocol>{protocol}</Protocol><User>alice</User><Pass encoding=\"base64\">c2VjcmV0</Pass></Server></Servers></FileZilla3>"
        );
        parse(&xml).expect("valid FileZilla XML")
    }

    #[test]
    fn filezilla_required_explicit_tls_maps_to_real_backend() {
        let sites = site(4);
        assert_eq!(sites.len(), 1);
        assert_eq!(sites[0].protocol, "ftps");
        assert_eq!(sites[0].port, 21);
        assert_eq!(sites[0].username, "alice");
    }

    #[test]
    fn filezilla_opportunistic_tls_never_silently_downgrades_to_plaintext() {
        let sites = site(0);
        assert_eq!(sites.len(), 1);
        assert_eq!(sites[0].protocol, "ftps");
        assert!(sites[0]
            .note
            .as_deref()
            .unwrap_or("")
            .contains("required explicit FTPS"));
    }

    #[test]
    fn filezilla_explicit_plaintext_retains_a_clear_security_warning() {
        let sites = site(6);
        assert_eq!(sites.len(), 1);
        assert_eq!(sites[0].protocol, "ftp");
        assert_eq!(sites[0].port, 21);
        assert!(sites[0]
            .note
            .as_deref()
            .unwrap_or("")
            .contains("without encryption"));
    }

    #[test]
    fn filezilla_implicit_tls_cannot_be_misrepresented_as_explicit_tls() {
        // The current backend's FTPS path negotiates AUTH TLS after plaintext
        // greeting; sending that to an implicit TLS site would be misleading.
        assert!(site(3).is_empty());
    }

    #[test]
    fn unsupported_cloud_protocol_is_not_imported_as_an_ftp_profile() {
        assert!(site(7).is_empty());
    }
    #[test]
    fn real_filezilla_trailing_site_name_and_nested_folders_survive_save() {
        let xml = r#"<FileZilla3><Servers>
            <Folder expanded="1">Production
                <Folder expanded="1">EU
                    <Server><Host>example.org</Host><Port>22</Port>
                        <Protocol>1</Protocol><User>deploy</User>
                        <Pass encoding="base64">c3VwZXItc2VjcmV0</Pass>
                        Website deployment
                    </Server>
                </Folder>
            </Folder>
        </Servers></FileZilla3>"#;
        let sites = parse(xml).expect("FileZilla exported tree");
        assert_eq!(sites.len(), 1);
        let site = &sites[0];
        assert_eq!(site.name, "Website deployment");
        assert_eq!(site.group.as_deref(), Some("Production / EU"));
        assert_eq!(site.protocol, "sftp");
        assert!(site
            .note
            .as_deref()
            .unwrap_or("")
            .contains("Production / EU"));
        assert!(!serde_json::to_string(site)
            .unwrap()
            .contains("super-secret"));
        let stored = sites[0].clone().into_profile();
        assert_eq!(stored.group.as_deref(), Some("Production / EU"));
        assert_eq!(stored.name, "Website deployment");
        match stored.auth {
            crate::profiles::AuthMethod::Password { password } => assert!(password.is_empty()),
            _ => panic!("FileZilla plaintext secrets must not be imported"),
        }
    }

    #[test]
    fn legacy_name_element_remains_authoritative_when_trailing_text_exists() {
        let sites = parse(
            "<FileZilla3><Servers><Server><Name>Explicit name</Name><Host>example.org</Host>             <Protocol>4</Protocol>Trailing label</Server></Servers></FileZilla3>",
        ).unwrap();
        assert_eq!(sites.len(), 1);
        assert_eq!(sites[0].name, "Explicit name");
        assert!(sites[0].group.is_none());
    }

    #[test]
    fn unnamed_folder_does_not_take_a_nested_server_field_as_its_name() {
        let sites = parse(
            "<FileZilla3><Servers><Folder><Server><Host>example.org</Host>             <Name>A site</Name><Protocol>1</Protocol></Server></Folder></Servers></FileZilla3>",
        ).unwrap();
        assert_eq!(sites.len(), 1);
        assert!(sites[0].group.is_none());
        assert_eq!(sites[0].name, "A site");
    }

    #[test]
    fn dtd_and_invalid_xml_entities_are_not_silently_accepted() {
        assert!(parse(
            "<!DOCTYPE FileZilla3 [<!ENTITY password SYSTEM 'file:///private/key'>]>             <FileZilla3><Servers /></FileZilla3>",
        ).is_err());
        assert!(parse(
            "<FileZilla3><Servers><Server><Host>good&amp;bad&unresolved;</Host>             <Protocol>1</Protocol></Server></Servers></FileZilla3>",
        ).is_err());
    }

    #[test]
    fn folder_nesting_and_site_count_are_bounded() {
        let xml = format!(
            "<FileZilla3><Servers>{}<Server><Host>example.org</Host></Server>{}</Servers></FileZilla3>",
            "<Folder>x".repeat(33),
            "</Folder>".repeat(33)
        );
        assert!(parse(&xml).is_err());

        let xml = format!(
            "<FileZilla3><Servers>{}</Servers></FileZilla3>",
            "<Server><Host>example.org</Host><Protocol>1</Protocol></Server>".repeat(2_049)
        );
        assert!(parse(&xml).is_err());
    }

    #[test]
    fn truncated_outer_xml_cannot_return_partial_successful_sites() {
        assert!(parse("<FileZilla3><Servers><Server><Host>example.org</Host></Server>").is_err());
        assert!(
            parse("<FileZilla3><Servers><Server><Host>example.org</Host></Server></Servers>")
                .is_err()
        );
        assert!(
            parse("<FileZilla3><Server><Host>example.org</Host></Server></FileZilla3>").is_err()
        );
    }

    #[test]
    fn empty_saved_site_collection_is_valid() {
        assert!(parse("<FileZilla3><Servers /></FileZilla3>")
            .expect("an empty saved-site collection is valid")
            .is_empty());
    }

    fn sftp_key_site(logon_type: u8, keyfile: &str, protocol: u8) -> Vec<super::ProfilePreview> {
        let xml = format!(
            "<FileZilla3><Servers><Server><Name>Keyed site</Name><Host>sftp.example.org</Host><Protocol>{protocol}</Protocol><Logontype>{logon_type}</Logontype><User>deploy</User><Keyfile>{keyfile}</Keyfile><Pass encoding=\"base64\">Y29weS1tZS1ub3Q=</Pass></Server></Servers></FileZilla3>"
        );
        parse(&xml).expect("valid FileZilla key profile")
    }

    #[test]
    fn filezilla_sftp_key_login_survives_import_without_loading_private_key() {
        // The key path is not required to exist at import time; connecting
        // is the only point where the SSH engine may read key material.
        let sites = sftp_key_site(5, "/home/alice/.ssh/id_ed25519", 1);
        assert_eq!(sites.len(), 1);
        assert_eq!(sites[0].protocol, "sftp");
        assert_eq!(
            sites[0].identity_file.as_deref(),
            Some("/home/alice/.ssh/id_ed25519")
        );
        let profile = sites[0].clone().into_profile();
        match profile.auth {
            crate::profiles::AuthMethod::Key { path, passphrase } => {
                assert_eq!(path, "/home/alice/.ssh/id_ed25519");
                assert!(passphrase.is_none());
            }
            _ => panic!("FileZilla SFTP key login must remain key authentication"),
        }
        assert!(!serde_json::to_string(&sites).unwrap().contains("copy-me-not"));
    }

    #[test]
    fn filezilla_stale_keyfield_must_not_switch_password_login_to_key_login() {
        let sites = sftp_key_site(1, "/home/alice/.ssh/stale_key", 1);
        assert_eq!(sites.len(), 1);
        assert!(sites[0].identity_file.is_none());
        match sites[0].clone().into_profile().auth {
            crate::profiles::AuthMethod::Password { password } => assert!(password.is_empty()),
            _ => panic!("Stale Keyfile must not silently replace password login"),
        }
    }

    #[test]
    fn filezilla_non_sftp_keyfile_is_never_used_as_an_ssh_key() {
        let sites = sftp_key_site(5, "/home/alice/.ssh/id_ed25519", 4);
        assert_eq!(sites.len(), 1);
        assert_eq!(sites[0].protocol, "ftps");
        assert!(sites[0].identity_file.is_none());
    }

    #[test]
    fn filezilla_ppk_and_missing_key_are_visible_but_not_activated() {
        let ppk = sftp_key_site(5, "C:\\Users\\Alice\\keys\\legacy.PPK", 1);
        assert_eq!(ppk.len(), 1);
        assert!(ppk[0].identity_file.is_none());
        assert!(ppk[0].note.as_deref().unwrap_or("").contains("conversion"));
        let missing = sftp_key_site(5, "", 1);
        assert_eq!(missing.len(), 1);
        assert!(missing[0].identity_file.is_none());
        assert!(missing[0]
            .note
            .as_deref()
            .unwrap_or("")
            .contains("no key path"));
    }

    #[test]
    fn oversized_and_truncated_imports_fail_closed() {
        let too_large = " ".repeat(super::MAX_XML_BYTES + 1);
        assert!(parse(&too_large).is_err());
        assert!(
            parse("<FileZilla3><Servers><Folder>Team<Server><Host>example.org</Host>").is_err()
        );
    }
}
