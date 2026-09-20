use super::ProfilePreview;
use anyhow::{Context, Result};
use std::path::PathBuf;

/// Resolve the default location of the user's OpenSSH client config.
pub fn default_path() -> Option<PathBuf> {
    dirs::home_dir().map(|h| h.join(".ssh").join("config"))
}

/// Parse an OpenSSH client config file into one ProfilePreview per Host
/// block. Wildcard / pattern hosts (those containing `*` or `?`) are
/// skipped — they're configuration scaffolding, not actual targets.
///
/// We support a small but sufficient subset of OpenSSH directives:
/// Host, Hostname, User, Port, IdentityFile. Everything else is ignored.
/// Include/Match etc. are deliberately out of scope — most desktop users
/// have flat config files and that's the case we want to cover.
pub fn parse_file(path: &PathBuf) -> Result<Vec<ProfilePreview>> {
    let text = std::fs::read_to_string(path)
        .with_context(|| format!("reading {}", path.display()))?;
    Ok(parse(&text, path.display().to_string()))
}

pub fn parse(text: &str, source_label: String) -> Vec<ProfilePreview> {
    let mut out = Vec::new();
    let mut current: Option<HostBlock> = None;

    for raw_line in text.lines() {
        let line = strip_comment(raw_line).trim();
        if line.is_empty() {
            continue;
        }
        let (key, value) = match split_kv(line) {
            Some(kv) => kv,
            None => continue,
        };
        let key_lc = key.to_ascii_lowercase();

        if key_lc == "host" {
            // Flush the previous block, start a new one for the first
            // non-wildcard alias on this line.
            if let Some(block) = current.take() {
                if let Some(preview) = block.into_preview(&source_label) {
                    out.push(preview);
                }
            }
            let alias = value.split_whitespace().find(|a| !is_pattern(a));
            current = alias.map(|a| HostBlock::new(a.to_string()));
        } else if let Some(block) = current.as_mut() {
            match key_lc.as_str() {
                "hostname" => block.hostname = Some(value.to_string()),
                "user" => block.user = Some(value.to_string()),
                "port" => {
                    block.port = value.parse().ok();
                }
                "identityfile" => {
                    block.identity = Some(expand_tilde(value));
                }
                _ => {}
            }
        }
    }
    if let Some(block) = current {
        if let Some(preview) = block.into_preview(&source_label) {
            out.push(preview);
        }
    }
    out
}

fn split_kv(line: &str) -> Option<(&str, &str)> {
    let bytes = line.as_bytes();
    let key_end = bytes
        .iter()
        .position(|&b| b == b' ' || b == b'\t' || b == b'=')?;
    let key = &line[..key_end];
    let rest = line[key_end..].trim_start_matches(|c: char| c.is_whitespace() || c == '=');
    Some((key, rest))
}

fn strip_comment(line: &str) -> &str {
    match line.find('#') {
        Some(i) => &line[..i],
        None => line,
    }
}

fn is_pattern(host: &str) -> bool {
    host.contains('*') || host.contains('?') || host.contains('!')
}

fn expand_tilde(value: &str) -> String {
    if let Some(stripped) = value.strip_prefix("~/") {
        if let Some(home) = dirs::home_dir() {
            return home.join(stripped).to_string_lossy().into_owned();
        }
    }
    value.to_string()
}

struct HostBlock {
    alias: String,
    hostname: Option<String>,
    user: Option<String>,
    port: Option<u16>,
    identity: Option<String>,
}

impl HostBlock {
    fn new(alias: String) -> Self {
        Self {
            alias,
            hostname: None,
            user: None,
            port: None,
            identity: None,
        }
    }

    fn into_preview(self, source_label: &str) -> Option<ProfilePreview> {
        // A Host block with no Hostname falls back to using the alias as the
        // target — that's how OpenSSH itself treats `ssh prod` where `prod`
        // is both the alias and the resolvable name.
        let host = self.hostname.unwrap_or_else(|| self.alias.clone());
        if host.is_empty() {
            return None;
        }
        let mut p = ProfilePreview::new(self.alias);
        p.protocol = "sftp".into();
        p.host = host;
        p.port = self.port.unwrap_or(22);
        p.username = self.user.unwrap_or_default();
        p.identity_file = self.identity;
        p.note = Some(format!("from {source_label}"));
        Some(p)
    }
}
