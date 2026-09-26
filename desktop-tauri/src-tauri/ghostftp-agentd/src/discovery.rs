//! Best-effort mDNS advertising so Ghost FTP can discover this daemon on the LAN
//! without the user typing an IP. Publishes `_ghostftp-agent._tcp` with the
//! machine's hostname, OS, key fingerprint, and whether a pairing window is
//! currently open (`pairable`) in TXT records. Discovery is a convenience:
//! pairing + pinning is what actually authenticates a connection, so a failure
//! here (locked-down network, no multicast) is non-fatal — the user can still
//! add the machine by IP:port.

use anyhow::{Context, Result};
use ghostftp_agent_proto::msg::SERVICE_TYPE;
use mdns_sd::{ServiceDaemon, ServiceInfo};

/// Handle that keeps the mDNS registration alive; drop it to unpublish.
/// Re-registering under the same instance name updates the record, which is
/// how [`set_pairable`](Self::set_pairable) flips the TXT flag live.
pub struct Advertisement {
    daemon: ServiceDaemon,
    fullname: String,
    port: u16,
    fingerprint: String,
    os: String,
}

impl Advertisement {
    /// Announce this daemon on the LAN. `fingerprint` is the identity's short
    /// fingerprint; `os` the platform string; `port` the TCP port we serve on;
    /// `pairable` whether a pairing window is open right now.
    pub fn publish(port: u16, fingerprint: &str, os: &str, pairable: bool) -> Result<Self> {
        let daemon = ServiceDaemon::new().context("start mDNS responder")?;
        let mut ad = Self {
            daemon,
            fullname: String::new(),
            port,
            fingerprint: fingerprint.to_string(),
            os: os.to_string(),
        };
        ad.register(pairable)?;
        Ok(ad)
    }

    /// Flip the advertised `pairable` flag (same instance name → an update,
    /// not a duplicate). Call when a pairing window opens or closes.
    pub fn set_pairable(&mut self, pairable: bool) -> Result<()> {
        self.register(pairable)
    }

    fn register(&mut self, pairable: bool) -> Result<()> {
        let host = crate::agent_name();
        // Instance name is the hostname; keep it stable so re-announcing updates
        // rather than duplicates.
        let instance = sanitize(&host);
        let props = [
            ("fp".to_string(), self.fingerprint.clone()),
            ("os".to_string(), self.os.clone()),
            ("host".to_string(), host.clone()),
            ("v".to_string(), env!("CARGO_PKG_VERSION").to_string()),
            ("pairable".to_string(), if pairable { "1" } else { "0" }.to_string()),
        ];
        // Let mdns-sd resolve our addresses (empty host_ipv4 + auto-addr).
        let service = ServiceInfo::new(
            SERVICE_TYPE,
            &instance,
            &format!("{instance}.local."),
            "",
            self.port,
            &props[..],
        )
        .context("build mDNS service info")?
        .enable_addr_auto();
        self.fullname = service.get_fullname().to_string();
        self.daemon
            .register(service)
            .context("register mDNS service")?;
        Ok(())
    }
}

impl Drop for Advertisement {
    fn drop(&mut self) {
        let _ = self.daemon.unregister(&self.fullname);
    }
}

/// mDNS instance names can't contain dots; replace anything awkward.
fn sanitize(name: &str) -> String {
    name.chars()
        .map(|c| if c.is_ascii_alphanumeric() || c == '-' { c } else { '-' })
        .collect()
}
