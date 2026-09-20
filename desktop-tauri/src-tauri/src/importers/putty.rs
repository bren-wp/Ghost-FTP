use super::ProfilePreview;
use anyhow::Result;

#[cfg(windows)]
pub fn default_path() -> Option<std::path::PathBuf> {
    // PuTTY's sessions live in the registry on Windows, not on disk. We
    // return a synthetic path so the UI has something to show; the parser
    // ignores it and reads the registry directly.
    Some(std::path::PathBuf::from(
        r"HKCU\Software\SimonTatham\PuTTY\Sessions",
    ))
}

#[cfg(not(windows))]
pub fn default_path() -> Option<std::path::PathBuf> {
    dirs::home_dir().map(|h| h.join(".putty").join("sessions"))
}

#[cfg(windows)]
pub fn parse_default() -> Result<Vec<ProfilePreview>> {
    use winreg::enums::HKEY_CURRENT_USER;
    use winreg::RegKey;

    let hkcu = RegKey::predef(HKEY_CURRENT_USER);
    let sessions = match hkcu.open_subkey(r"Software\SimonTatham\PuTTY\Sessions") {
        Ok(k) => k,
        Err(_) => return Ok(Vec::new()), // PuTTY not installed / no sessions
    };

    let mut out = Vec::new();
    for name_result in sessions.enum_keys() {
        let Ok(raw_name) = name_result else { continue };
        // PuTTY URL-encodes spaces in session names (%20 etc.). The display
        // form is what the user typed in PuTTY's session list.
        let display_name = decode_session_name(&raw_name);
        let Ok(session) = sessions.open_subkey(&raw_name) else {
            continue;
        };

        let protocol: String = session.get_value("Protocol").unwrap_or_default();
        let proto = match protocol.to_ascii_lowercase().as_str() {
            "ssh" => "sftp",
            "ftp" => "ftp",
            // raw, telnet, rlogin, serial: out of scope.
            _ => continue,
        };

        let host: String = session.get_value("HostName").unwrap_or_default();
        if host.is_empty() {
            continue;
        }
        // PortNumber is REG_DWORD.
        let port: u32 = session.get_value("PortNumber").unwrap_or(22);
        let user: String = session.get_value("UserName").unwrap_or_default();

        let mut p = ProfilePreview::new(display_name);
        p.protocol = proto.into();
        p.host = host;
        p.port = port.try_into().unwrap_or(22);
        p.username = user;
        p.note = Some("from PuTTY".into());
        out.push(p);
    }
    Ok(out)
}

#[cfg(not(windows))]
pub fn parse_default() -> Result<Vec<ProfilePreview>> {
    // PuTTY on unix stores sessions as files in ~/.putty/sessions/. Each file
    // is a `key=value` text dump similar to the Windows registry layout.
    let Some(dir) = default_path() else {
        return Ok(Vec::new());
    };
    let Ok(entries) = std::fs::read_dir(&dir) else {
        return Ok(Vec::new());
    };
    let mut out = Vec::new();
    for entry in entries.flatten() {
        if !entry.path().is_file() {
            continue;
        }
        let Ok(text) = std::fs::read_to_string(entry.path()) else {
            continue;
        };
        let display_name = decode_session_name(
            entry
                .file_name()
                .to_string_lossy()
                .as_ref(),
        );
        let mut host = None;
        let mut user = None;
        let mut port: u32 = 22;
        let mut protocol = "ssh".to_string();
        for line in text.lines() {
            if let Some((k, v)) = line.split_once('=') {
                match k.trim() {
                    "HostName" => host = Some(v.trim().to_string()),
                    "UserName" => user = Some(v.trim().to_string()),
                    "PortNumber" => port = v.trim().parse().unwrap_or(22),
                    "Protocol" => protocol = v.trim().to_string(),
                    _ => {}
                }
            }
        }
        let proto = match protocol.to_ascii_lowercase().as_str() {
            "ssh" => "sftp",
            "ftp" => "ftp",
            _ => continue,
        };
        let Some(host) = host else { continue };
        let mut p = ProfilePreview::new(display_name);
        p.protocol = proto.into();
        p.host = host;
        p.port = port.try_into().unwrap_or(22);
        p.username = user.unwrap_or_default();
        p.note = Some("from PuTTY".into());
        out.push(p);
    }
    Ok(out)
}

/// PuTTY %-encodes spaces and other characters in session names.
fn decode_session_name(raw: &str) -> String {
    let mut out = String::with_capacity(raw.len());
    let mut chars = raw.chars().peekable();
    while let Some(c) = chars.next() {
        if c == '%' {
            let hex: String = chars.by_ref().take(2).collect();
            if hex.len() == 2 {
                if let Ok(byte) = u8::from_str_radix(&hex, 16) {
                    out.push(byte as char);
                    continue;
                }
            }
            out.push(c);
            out.push_str(&hex);
        } else {
            out.push(c);
        }
    }
    out
}
