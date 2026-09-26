//! `ghostftp-agentd` CLI.
//!
//! Usage:
//!   ghostftp-agentd run       [--port N] [--read-only] [--no-mdns]   serve paired controllers
//!   ghostftp-agentd pair      [--port N] [--window MIN] [--json]     serve + open a pairing window
//!   ghostftp-agentd info                                             print this machine's identity + peers
//!   ghostftp-agentd unpair    <fingerprint|all>                      remove a pinned controller
//!   ghostftp-agentd install   [--port N] [--read-only]               run as a background service (autostart)
//!   ghostftp-agentd uninstall                                        remove the background service
//!
//! `pair` is `run` with a pairing window open: already-paired controllers keep
//! working while new ones pair, and nothing needs restarting afterwards.
//! `--json` prints machine-readable events (one JSON object per line) so a
//! hosting panel like ServerKit can read the code and build a `ghostftp://` link.
//! `install` registers a systemd unit (Linux), logon
//! task (Windows) that runs `ghostftp-agentd run` at boot/login.
//!
//! Shared flags: --config-dir <path>

use anyhow::{bail, Context, Result};
use ghostftp_agent_proto::{identity::Identity, pairing};
use ghostftp_agentd::{
    config::{config_path, identity_path},
    discovery::Advertisement,
    ops, server, Config, Daemon,
};
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use std::time::Duration;
use tokio::net::TcpListener;

const DEFAULT_PORT: u16 = 8722;
const DEFAULT_WINDOW_MIN: u64 = 15;

struct Args {
    command: String,
    port: u16,
    config_dir: Option<PathBuf>,
    read_only: bool,
    no_mdns: bool,
    json: bool,
    window_min: u64,
    positional: Vec<String>,
}

fn parse_args() -> Args {
    let mut a = Args {
        command: String::new(),
        port: DEFAULT_PORT,
        config_dir: None,
        read_only: false,
        no_mdns: false,
        json: false,
        window_min: DEFAULT_WINDOW_MIN,
        positional: Vec::new(),
    };
    let mut it = std::env::args().skip(1);
    while let Some(arg) = it.next() {
        match arg.as_str() {
            "--port" => {
                if let Some(v) = it.next() {
                    a.port = v.parse().unwrap_or(DEFAULT_PORT);
                }
            }
            "--config-dir" => a.config_dir = it.next().map(PathBuf::from),
            "--read-only" => a.read_only = true,
            "--no-mdns" => a.no_mdns = true,
            "--json" => a.json = true,
            "--window" => {
                if let Some(v) = it.next() {
                    a.window_min = v.parse().unwrap_or(DEFAULT_WINDOW_MIN).clamp(1, 24 * 60);
                }
            }
            "-h" | "--help" => {
                a.command = "help".into();
            }
            other if other.starts_with('-') => {
                // Unknown flag — ignore rather than mistaking it for the command.
            }
            other => {
                // The first bare token (in any position) is the subcommand; the
                // rest are its positional arguments — so `--config-dir X info`
                // and `info --config-dir X` both run `info`.
                if a.command.is_empty() {
                    a.command = other.to_string();
                } else {
                    a.positional.push(other.to_string());
                }
            }
        }
    }
    if a.command.is_empty() {
        a.command = "run".into();
    }
    a
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "ghostftp_agentd=info".into()),
        )
        .init();

    let args = parse_args();
    let dir = ghostftp_agentd::config_dir(args.config_dir.clone())?;
    let identity = Identity::load_or_create(&identity_path(&dir))?;
    let mut config = Config::load(&config_path(&dir))?;
    if args.read_only {
        config.policy.allow_exec = false;
        config.policy.allow_write = false;
    }

    match args.command.as_str() {
        "run" => run(args, dir, identity, config).await,
        "pair" => pair(args, dir, identity, config).await,
        "info" => info(dir, identity, config),
        "unpair" => unpair(dir, config, &args.positional),
        "install" => ghostftp_agentd::service::install(&service_run_args(&args)),
        "uninstall" => ghostftp_agentd::service::uninstall(),
        "help" => {
            print_help();
            Ok(())
        }
        other => {
            eprintln!("unknown command '{other}'\n");
            print_help();
            bail!("unknown command");
        }
    }
}

/// Reconstruct the `run` flags to bake into a service definition, so the
/// installed service serves with the same options the admin chose (config dir,
/// port, read-only). mDNS is left on by default.
fn service_run_args(args: &Args) -> Vec<String> {
    let mut extra = Vec::new();
    if let Some(dir) = &args.config_dir {
        extra.push("--config-dir".into());
        extra.push(dir.display().to_string());
    }
    if args.port != DEFAULT_PORT {
        extra.push("--port".into());
        extra.push(args.port.to_string());
    }
    if args.read_only {
        extra.push("--read-only".into());
    }
    if args.no_mdns {
        extra.push("--no-mdns".into());
    }
    extra
}

/// Bind the daemon's TCP port, turning the classic AddrInUse into advice
/// instead of a raw OS error.
async fn bind_port(port: u16) -> Result<TcpListener> {
    match TcpListener::bind(("0.0.0.0", port)).await {
        Ok(l) => Ok(l),
        Err(e) if e.kind() == std::io::ErrorKind::AddrInUse => bail!(
            "port {port} is already in use — most likely another ghostftp-agentd is running.\n  \
             Stop it first, or pass --port <N> to use a different port."
        ),
        Err(e) => Err(e).with_context(|| format!("bind 0.0.0.0:{port}")),
    }
}

async fn run(args: Args, dir: PathBuf, identity: Identity, config: Config) -> Result<()> {
    let info = ops::system_info();
    let fingerprint = identity.fingerprint();
    let peers = config.peers.len();
    let daemon = Daemon::new(identity, config, dir);

    let listener = bind_port(args.port).await?;
    let port = listener.local_addr()?.port();

    println!("ghostftp-agentd {} on {} ({})", env!("CARGO_PKG_VERSION"), info.hostname, info.os);
    println!("  identity : {fingerprint}");
    println!("  listening: 0.0.0.0:{port}");
    println!("  peers    : {peers} paired");
    if peers == 0 {
        println!("\n  No controllers paired yet. Run `ghostftp-agentd pair` to add one.");
    }

    let _ad = if args.no_mdns {
        None
    } else {
        match Advertisement::publish(port, &fingerprint, &info.os, false) {
            Ok(ad) => {
                println!("  mDNS     : advertising _ghostftp-agent._tcp");
                Some(ad)
            }
            Err(e) => {
                println!("  mDNS     : unavailable ({e}) — add this machine by IP in Ghost FTP");
                None
            }
        }
    };

    server::serve(listener, daemon).await
}

/// `run` + an open pairing window: serves already-paired controllers while new
/// ones pair with the printed code. The window auto-expires; the daemon keeps
/// serving afterwards, so nothing needs restarting.
async fn pair(args: Args, dir: PathBuf, identity: Identity, config: Config) -> Result<()> {
    let info = ops::system_info();
    let fingerprint = identity.fingerprint();
    let code = pairing::generate_code();
    let window = Duration::from_secs(args.window_min * 60);
    let json = args.json;

    let counter = Arc::new(std::sync::atomic::AtomicUsize::new(0));
    let c2 = counter.clone();
    let daemon = Daemon::new(identity, config, dir).with_on_paired(move |name, key| {
        let n = c2.fetch_add(1, std::sync::atomic::Ordering::SeqCst) + 1;
        if json {
            println!(
                "{}",
                serde_json::json!({ "event": "paired", "name": name, "key": key, "total": n })
            );
        } else {
            println!("  ✓ paired '{name}'  ({key})  [{n} total]");
        }
    });
    daemon.open_pairing(code.clone(), window).await;

    let listener = bind_port(args.port).await?;
    let port = listener.local_addr()?.port();

    if json {
        // One event per line, for panels (ServerKit) driving this headlessly.
        println!(
            "{}",
            serde_json::json!({
                "event": "pairing",
                "code": code,
                "port": port,
                "fingerprint": fingerprint,
                "hostname": info.hostname,
                "os": info.os,
                "windowSecs": window.as_secs(),
            })
        );
    } else {
        println!("\n  Pairing {} ({})", info.hostname, info.os);
        println!("  ┌─────────────────────────┐");
        println!("  │   Pairing code: {code}   │");
        println!("  └─────────────────────────┘");
        println!("  fingerprint: {fingerprint}");
        println!("  port       : {port}");
        println!("\n  In Ghost FTP: New Connection → Ghost FTP Agent → pick this machine → enter the code.");
        println!(
            "  Window open for {} min; paired controllers are served the whole time.\n",
            args.window_min
        );
    }

    let ad = Arc::new(Mutex::new(if args.no_mdns {
        None
    } else {
        Advertisement::publish(port, &fingerprint, &info.os, true).ok()
    }));

    // When the window lapses, say so and drop the advertised pairable flag —
    // serving continues untouched. (Expired windows are also refused lazily,
    // so this task is cosmetic, not load-bearing.)
    {
        let daemon = daemon.clone();
        let ad = ad.clone();
        tokio::spawn(async move {
            tokio::time::sleep(window).await;
            daemon.close_pairing().await;
            if let Some(ad) = ad.lock().unwrap().as_mut() {
                let _ = ad.set_pairable(false);
            }
            if json {
                println!("{}", serde_json::json!({ "event": "windowClosed" }));
            } else {
                println!(
                    "  Pairing window closed — still serving paired controllers. \
                     Run `ghostftp-agentd pair` again for a new code."
                );
            }
        });
    }

    server::serve(listener, daemon).await
}

fn info(dir: PathBuf, identity: Identity, config: Config) -> Result<()> {
    let info = ops::system_info();
    println!("ghostftp-agentd {}", env!("CARGO_PKG_VERSION"));
    println!("  machine    : {} ({}, {})", info.hostname, info.os, info.arch);
    println!("  config dir : {}", dir.display());
    println!("  identity   : {}", identity.fingerprint());
    println!("  public key : {}", identity.public_b64());
    println!(
        "  policy     : exec={}  write={}",
        config.policy.allow_exec, config.policy.allow_write
    );
    println!("  peers      : {}", config.peers.len());
    for p in &config.peers {
        let fp = ghostftp_agent_proto::identity::decode_public(&p.public_key)
            .map(|k| ghostftp_agent_proto::identity::fingerprint_of(&k))
            .unwrap_or_else(|_| "?".into());
        println!("    - {}  {fp}", p.name);
    }
    Ok(())
}

fn unpair(dir: PathBuf, mut config: Config, positional: &[String]) -> Result<()> {
    let Some(target) = positional.first() else {
        bail!("usage: ghostftp-agentd unpair <fingerprint|all>");
    };
    let before = config.peers.len();
    if target == "all" {
        config.peers.clear();
    } else {
        config.peers.retain(|p| {
            let fp = ghostftp_agent_proto::identity::decode_public(&p.public_key)
                .map(|k| ghostftp_agent_proto::identity::fingerprint_of(&k))
                .unwrap_or_default();
            fp != *target && p.public_key != *target
        });
    }
    let removed = before - config.peers.len();
    config.save(&config_path(&dir))?;
    println!("Removed {removed} peer(s). {} remain.", config.peers.len());
    Ok(())
}

fn print_help() {
    println!(
        "ghostftp-agentd {} — control this machine from Ghost FTP over an encrypted, paired link\n\n\
         USAGE:\n\
         \x20 ghostftp-agentd run       [--port N] [--read-only] [--no-mdns]   serve paired controllers\n\
         \x20 ghostftp-agentd pair      [--port N] [--window MIN] [--json]     serve + open a pairing window\n\
         \x20 ghostftp-agentd info                                             show identity + paired peers\n\
         \x20 ghostftp-agentd unpair    <fingerprint|all>                      remove a pinned controller\n\
         \x20 ghostftp-agentd install   [--port N] [--read-only]               run as a service (autostart)\n\
         \x20 ghostftp-agentd uninstall                                        remove the service\n\n\
         `pair` keeps serving already-paired controllers while the window is open\n\
         (default {DEFAULT_WINDOW_MIN} min) and after it closes — no restart needed. `--json` prints\n\
         one machine-readable event per line for hosting panels. `install` sets up\n\
         a systemd unit / LaunchAgent / logon task that runs `ghostftp-agentd run`.\n\n\
         SHARED FLAGS:\n\
         \x20 --config-dir <path>   override the config/identity directory\n\n\
         Default port: {}",
        env!("CARGO_PKG_VERSION"),
        DEFAULT_PORT
    );
}
