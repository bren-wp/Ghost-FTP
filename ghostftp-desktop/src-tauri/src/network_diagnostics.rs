//! Bounded, read-only FTP/FTPS network diagnostics for a saved connection.
//! Probe results reflect actual DNS/TCP/authentication/data operations; they
//! are never inferred from a saved hostname or synthetic success response.
use crate::commands::fs_for_session;
use crate::profiles::ConnectionProfile;
use crate::AppState;
use serde::Serialize;
use std::net::{SocketAddr, TcpStream, ToSocketAddrs};
use std::time::Duration;
use tauri::AppHandle;

const TCP_PROBE_TIMEOUT: Duration = Duration::from_secs(3);
const DNS_PROBE_TIMEOUT: Duration = Duration::from_secs(8);
const MAX_ADDRESSES: usize = 8;

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct FtpNetworkDiagnostic {
    pub dns: &'static str,
    pub control: &'static str,
    pub authentication: &'static str,
    pub data: &'static str,
    pub resolved_ipv4: bool,
    pub resolved_ipv6: bool,
    pub connected_family: Option<&'static str>,
    pub advice: &'static str,
}

impl Default for FtpNetworkDiagnostic {
    fn default() -> Self {
        Self {
            dns: "notRun",
            control: "notRun",
            authentication: "notRun",
            data: "notRun",
            resolved_ipv4: false,
            resolved_ipv6: false,
            connected_family: None,
            advice: "No diagnostics have run.",
        }
    }
}

/// An actual direct TCP handshake, not an FTP login or data-channel proof.
/// The selected saved site is the only permitted target; no public test host
/// or firewall/router setting is ever modified.
fn probe_control(addresses: &[SocketAddr]) -> Option<&'static str> {
    for address in addresses {
        if TcpStream::connect_timeout(address, TCP_PROBE_TIMEOUT).is_ok() {
            return Some(if address.is_ipv4() { "IPv4" } else { "IPv6" });
        }
    }
    None
}

pub async fn diagnose(
    profile: ConnectionProfile,
    app: AppHandle,
    state: &AppState,
) -> FtpNetworkDiagnostic {
    let mut report = FtpNetworkDiagnostic::default();
    let host = profile.host.trim().to_owned();
    if host.is_empty() || profile.port == 0 {
        report.dns = "failed";
        report.advice = "Set a valid saved FTP hostname and port before testing.";
        return report;
    }
    let port = profile.port;
    let resolution = tokio::task::spawn_blocking(move || {
        (host.as_str(), port)
            .to_socket_addrs()
            .map(|addresses| addresses.take(MAX_ADDRESSES).collect::<Vec<_>>())
    });
    let addresses = match tokio::time::timeout(DNS_PROBE_TIMEOUT, resolution).await {
        Ok(Ok(Ok(addresses))) if !addresses.is_empty() => addresses,
        _ => {
            report.dns = "failed";
            report.advice =
                "DNS lookup failed or timed out. Check the saved hostname and DNS settings.";
            return report;
        }
    };

    report.dns = "passed";
    report.resolved_ipv4 = addresses.iter().any(SocketAddr::is_ipv4);
    report.resolved_ipv6 = addresses.iter().any(SocketAddr::is_ipv6);
    report.connected_family = tokio::task::spawn_blocking(move || probe_control(&addresses))
        .await
        .unwrap_or_default();
    if report.connected_family.is_none() {
        report.control = "failed";
        report.advice = "The FTP control TCP port is unreachable. Check server availability, firewall and port.";
        return report;
    }

    report.control = "passed";
    let session_id = match state.sessions.connect(profile, app).await {
        Ok(id) => id,
        Err(_) => {
            report.authentication = "failed";
            report.advice = "Control TCP worked, but FTP/FTPS login or TLS negotiation failed. Check credentials, encryption mode and certificate trust.";
            return report;
        }
    };
    report.authentication = "passed";
    let data_result = match state.sessions.get(&session_id).await {
        Some(session) => fs_for_session(&session).list_dir(".").await,
        None => Err(anyhow::anyhow!("diagnostic session was lost")),
    };
    // Always release the temporary authenticated session, including after a
    // failed LIST; the live Site Manager profile and any open tabs are untouched.
    let _ = state.sessions.disconnect(&session_id).await;
    if data_result.is_err() {
        report.data = "failed";
        report.advice = "FTP/FTPS login succeeded, but passive LIST failed. Check directory permissions, passive data ports and NAT/firewall rules.";
        return report;
    }
    report.data = "passed";
    report.advice = "DNS, control TCP, login/TLS and passive LIST succeeded. Active FTP and external NAT behavior have not been tested.";
    report
}

#[cfg(test)]
mod tests {
    use super::{probe_control, FtpNetworkDiagnostic};
    use std::net::{SocketAddr, TcpListener};

    #[test]
    fn tcp_probe_uses_an_actual_open_loopback_listener() {
        let listener = TcpListener::bind("127.0.0.1:0").expect("bind test listener");
        let address = listener.local_addr().expect("listener address");
        assert_eq!(probe_control(&[address]), Some("IPv4"));
    }

    #[test]
    fn an_empty_probe_cannot_report_control_connectivity() {
        let addresses: [SocketAddr; 0] = [];
        assert_eq!(probe_control(&addresses), None);
    }

    #[test]
    fn unknown_network_state_is_never_reported_as_success() {
        let status = FtpNetworkDiagnostic::default();
        assert_eq!(status.dns, "notRun");
        assert_eq!(status.control, "notRun");
        assert_eq!(status.authentication, "notRun");
        assert_eq!(status.data, "notRun");
    }
}
