// Ghost FTP by Brendigo
use super::*;

async fn parse_wire_request(wire: &[u8]) -> Result<Request> {
    let listener = TcpListener::bind(("127.0.0.1", 0)).await?;
    let addr = listener.local_addr()?;
    let mut client = TcpStream::connect(addr).await?;
    let (mut server, _) = listener.accept().await?;
    client.write_all(wire).await?;
    client.shutdown().await?;
    read_request(&mut server).await
}

#[tokio::test]
async fn bridge_connection_gate_bounds_slow_clients() {
    let slots = Arc::new(Semaphore::new(MAX_BRIDGE_CONNECTIONS));
    let mut permits = Vec::new();
    for _ in 0..MAX_BRIDGE_CONNECTIONS {
        permits.push(slots.clone().try_acquire_owned().unwrap());
    }
    assert!(
        slots.clone().try_acquire_owned().is_err(),
        "a 33rd slow client must not create another request handler"
    );
    drop(permits);
    assert!(slots.try_acquire_owned().is_ok());
}

#[tokio::test]
async fn http_parser_rejects_ambiguous_body_framing() {
    for raw in [
        "POST /exec HTTP/1.1\r\nContent-Length: invalid\r\n\r\n",
        "POST /exec HTTP/1.1\r\nContent-Length: 0\r\nContent-Length: 0\r\n\r\n",
        "POST /exec HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n",
        "POST /exec HTTP/1.1\r\n\r\n",
        "POST /exec HTTP/1.1\r\nContent-Length: 5\r\n\r\n{}",
        "GET /health HTTP/1.1\r\nAuthorization: Bearer a\r\nAuthorization: Bearer b\r\n\r\n",
    ] {
        assert!(parse_wire_request(raw.as_bytes()).await.is_err(), "{raw}");
    }
}

#[tokio::test]
async fn http_parser_rejects_malformed_request_lines_and_headers() {
    for raw in [
        "POST /exec HTTP/1.1 extra\r\nContent-Length: 0\r\n\r\n",
        "POST /exec HTTP/2\r\nContent-Length: 0\r\n\r\n",
        "POST http://127.0.0.1/exec HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
        "POST /exec HTTP/1.1\r\nContent-Length: +2\r\n\r\n{}",
        "POST /exec HTTP/1.1\r\nContent-Length : 2\r\n\r\n{}",
        "POST /exec HTTP/1.1\r\nContent-Length: 0\r\n Invalid-Fold: value\r\n\r\n",
        "POST /exec HTTP/1.1\r\nContent-Length: 0\r\nHeaderWithoutColon\r\n\r\n",
    ] {
        assert!(parse_wire_request(raw.as_bytes()).await.is_err(), "{raw}");
    }
    assert!(parse_wire_request(
        b"POST /exec HTTP/1.1\r\nContent-Length: 0\r\nX-Bad: \xff\r\n\r\n"
    )
    .await
    .is_err());
}

#[tokio::test]
async fn http_parser_tracks_web_origin_and_accepts_valid_local_request() {
    let wire = b"POST /exec HTTP/1.1\r\nAuthorization: Bearer private\r\nContent-Length: 2\r\nOrigin: https://example.invalid\r\n\r\n{}";
    let parsed = parse_wire_request(wire).await.expect("parse request");
    assert_eq!(parsed.body, b"{}");
    assert!(parsed.has_origin);
    let local =
        parse_wire_request(b"GET /health HTTP/1.1\r\nAuthorization: Bearer private\r\n\r\n")
            .await
            .expect("parse native local client");
    assert!(!local.has_origin);
}

#[cfg(unix)]
#[test]
fn discovery_file_token_is_private_on_creation() {
    use std::os::unix::fs::PermissionsExt;
    let path =
        std::env::temp_dir().join(format!("ghostftp-agent-discovery-{}.json", Uuid::new_v4()));
    write_discovery_file(&path, 41235, "test-token").expect("write private discovery");
    let mode = std::fs::metadata(&path)
        .expect("read discovery metadata")
        .permissions()
        .mode();
    assert_eq!(mode & 0o777, 0o600);
    let body: serde_json::Value =
        serde_json::from_slice(&std::fs::read(&path).expect("read file"))
            .expect("parse discovery");
    assert_eq!(body["token"], "test-token");
    std::fs::remove_file(path).expect("cleanup discovery");
}

#[test]
fn http_error_statuses_use_correct_reason_phrases() {
    for (status, reason) in [
        (200, "OK"),
        (400, "Bad Request"),
        (401, "Unauthorized"),
        (403, "Forbidden"),
        (404, "Not Found"),
        (405, "Method Not Allowed"),
        (413, "Content Too Large"),
        (429, "Too Many Requests"),
        (500, "Internal Server Error"),
        (502, "Bad Gateway"),
        (503, "Service Unavailable"),
        (504, "Gateway Timeout"),
    ] {
        assert_eq!(http_reason_phrase(status), reason);
    }
    assert_eq!(http_reason_phrase(599), "Unknown Status");
}

#[test]
fn agent_file_reply_rejects_corruption_oversize_and_false_byte_counts() {
    assert_eq!(decode_agent_file_response("YQ==", 1).unwrap(), b"a");
    assert_eq!(decode_agent_file_response("", 0).unwrap(), b"");
    assert!(decode_agent_file_response("not-valid-base64", 0).is_err());
    assert!(decode_agent_file_response("YQ==", 2).is_err());
    assert!(decode_agent_file_response("A", 0).is_err());
    let oversized = "A".repeat(((MAX_READ_FILE + 2) / 3) * 4 + 1);
    assert!(decode_agent_file_response(&oversized, 0).is_err());
}

#[tokio::test]
async fn revoke_all_access_clears_runtime_profiles_and_approval_policy() {
    let bridge = BridgeState::new();
    *bridge.enabled_master.lock().await = true;
    bridge.enabled.lock().await.insert("session-1".to_string());
    bridge
        .enabled_profiles
        .lock()
        .await
        .insert("profile-1".to_string());
    *bridge.policy.lock().await = ApprovalPolicy {
        allow_all: true,
        auto_read: true,
        auto_safe_exec: true,
        allow_sudo: true,
    };
    let status = bridge
        .revoke_all_access()
        .await
        .expect("revocation succeeds");
    assert!(!status.running);
    assert!(!status.enabled);
    assert!(status.enabled_sessions.is_empty());
    assert!(bridge.enabled_profiles.lock().await.is_empty());
    assert_eq!(*bridge.policy.lock().await, ApprovalPolicy::default());
}

// Plan 10 Phase 1: the CLI base64-encodes a script's raw bytes; the bridge
// decodes them back. Prove that hop is byte-exact for a heredoc + nested
// quotes + trailing-newline script (the content that breaks a command line).
#[test]
fn exec_script_base64_roundtrip_is_byte_exact() {
    use base64::Engine as _;
    let script = "#!/bin/sh\ncat <<'EOF'\nnested 'single' and \"double\" quotes\nEOF\n";
    let b64 = base64::engine::general_purpose::STANDARD.encode(script);
    let decoded = base64::engine::general_purpose::STANDARD
        .decode(b64.as_bytes())
        .unwrap();
    assert_eq!(String::from_utf8(decoded).unwrap(), script);
}

#[test]
fn script_summary_has_header_and_survives_multibyte_truncation() {
    // Short script → header + full body.
    let s = script_approval_summary("script foo.sh", "echo hi\necho bye");
    assert!(
        s.starts_with("Run script foo.sh (16 bytes, 2 lines):"),
        "got: {s}"
    );
    assert!(s.contains("echo bye"));
    // Long script with a multibyte char straddling the 4 KiB cap must not
    // panic on a char-boundary split.
    let big = format!("{}é tail", "a".repeat(4095));
    let s = script_approval_summary("big", &big);
    assert!(s.contains("preview truncated"));
}

#[test]
fn exec_timeout_defaults_and_clamps() {
    // Absent → the 60 s default.
    assert_eq!(exec_timeout_from(None), EXEC_TIMEOUT);
    // In range → taken verbatim.
    assert_eq!(exec_timeout_from(Some(5_000)), Duration::from_secs(5));
    assert_eq!(
        exec_timeout_from(Some(EXEC_TIMEOUT_MS_MAX)),
        Duration::from_millis(EXEC_TIMEOUT_MS_MAX)
    );
    // Below the floor → clamped up (a typo can't insta-kill commands).
    assert_eq!(
        exec_timeout_from(Some(1)),
        Duration::from_millis(EXEC_TIMEOUT_MS_MIN)
    );
    assert_eq!(
        exec_timeout_from(Some(0)),
        Duration::from_millis(EXEC_TIMEOUT_MS_MIN)
    );
    // Above the ceiling → clamped to 15 min.
    assert_eq!(
        exec_timeout_from(Some(3_600_000)),
        Duration::from_millis(EXEC_TIMEOUT_MS_MAX)
    );
}

#[test]
fn human_bytes_scales() {
    assert_eq!(human_bytes(0), "0 B");
    assert_eq!(human_bytes(1023), "1023 B");
    assert_eq!(human_bytes(2 * 1024), "2.0 KB");
    assert_eq!(human_bytes(3 * 1024 * 1024 + 512 * 1024), "3.5 MB");
    assert_eq!(human_bytes(5 * 1024 * 1024 * 1024), "5.00 GB");
}

#[test]
fn upload_dir_policy_defaults_to_rename() {
    assert_eq!(upload_overwrite_policy(false), OverwritePolicy::Rename);
    assert_eq!(upload_overwrite_policy(true), OverwritePolicy::Overwrite);
}

#[test]
fn upload_dir_summary_names_counts_and_mode() {
    let s = upload_dir_summary("C:\\site\\dist", "prod", "/var/www", 12, 3_670_016, false);
    assert_eq!(
        s,
        "Upload directory C:\\site\\dist → prod:/var/www (12 files, 3.5 MB total, overwrite: no)"
    );
    let s = upload_dir_summary("./dist", "prod", "/var/www/", 1, 10, true);
    assert!(s.ends_with("(1 files, 10 B total, overwrite: yes)"));
}

#[test]
fn sync_args_parse_with_defaults() {
    let (d, s) = parse_sync_args("", "").unwrap();
    assert!(matches!(d, SyncDirection::LocalToRemote));
    assert!(matches!(s, SyncStrategy::Additive));
    let (d, s) = parse_sync_args("pull", "mirror").unwrap();
    assert!(matches!(d, SyncDirection::RemoteToLocal));
    assert!(matches!(s, SyncStrategy::Mirror));
    assert!(parse_sync_args("sideways", "").is_err());
    assert!(parse_sync_args("", "destroy").is_err());
}

#[test]
fn sync_summary_names_mirror_deletes() {
    let s = sync_gate_summary(
        SyncDirection::LocalToRemote,
        SyncStrategy::Mirror,
        "./dist",
        "prod",
        "/var/www/app",
        12,
        3_670_016,
        3,
    );
    assert_eq!(
        s,
        "Sync push ./dist ↔ prod:/var/www/app — copy 12 files (3.5 MB), delete 3 files on the destination (mirror)"
    );
}

#[test]
fn sync_summary_promises_no_deletes_for_additive() {
    let s = sync_gate_summary(
        SyncDirection::RemoteToLocal,
        SyncStrategy::Additive,
        "C:\\backup",
        "prod",
        "/etc/nginx",
        2,
        2048,
        0,
    );
    assert_eq!(
        s,
        "Sync pull C:\\backup ↔ prod:/etc/nginx — copy 2 files (2.0 KB), no deletes (additive)"
    );
}

#[tokio::test]
async fn count_local_tree_counts_nested_files() {
    let root = std::env::temp_dir().join(format!("ghostftp-bridge-test-{}", Uuid::new_v4()));
    let sub = root.join("a").join("b");
    std::fs::create_dir_all(&sub).unwrap();
    std::fs::write(root.join("one.txt"), b"12345").unwrap();
    std::fs::write(root.join("a").join("two.txt"), b"123").unwrap();
    std::fs::write(sub.join("three.txt"), b"1").unwrap();

    let (files, bytes) = count_local_tree(&root).await.unwrap();
    assert_eq!(files, 3);
    assert_eq!(bytes, 9);

    std::fs::remove_dir_all(&root).ok();
}

// ---- Skills (Plan 8) ----

fn skill_with_params(params: Vec<SkillParam>) -> Skill {
    Skill {
        params,
        ..Default::default()
    }
}

#[test]
fn param_map_defaults_and_overrides() {
    let skill = skill_with_params(vec![
        SkillParam {
            name: "service".into(),
            required: true,
            ..Default::default()
        },
        SkillParam {
            name: "signal".into(),
            default: Some("HUP".into()),
            ..Default::default()
        },
    ]);
    let mut provided = HashMap::new();
    provided.insert("service".to_string(), "nginx".to_string());
    let map = build_param_map(&skill, &provided).unwrap();
    assert_eq!(map.get("service").unwrap(), "nginx");
    // Declared default fills in when not provided.
    assert_eq!(map.get("signal").unwrap(), "HUP");
    // Provided wins over the declared default.
    provided.insert("signal".to_string(), "TERM".to_string());
    let map = build_param_map(&skill, &provided).unwrap();
    assert_eq!(map.get("signal").unwrap(), "TERM");
}

#[test]
fn param_map_requires_required() {
    let skill = skill_with_params(vec![SkillParam {
        name: "service".into(),
        required: true,
        ..Default::default()
    }]);
    let err = build_param_map(&skill, &HashMap::new()).unwrap_err();
    assert!(err.contains("service"));
}

#[test]
fn substitute_replaces_placeholders() {
    let mut params = HashMap::new();
    params.insert("service".to_string(), "nginx".to_string());
    params.insert("signal".to_string(), "HUP".to_string());
    assert_eq!(
        substitute("systemctl reload ${service} # ${signal}", &params),
        "systemctl reload nginx # HUP"
    );
    // An unresolved placeholder is left verbatim.
    assert_eq!(substitute("echo ${missing}", &params), "echo ${missing}");
}

#[test]
fn migrate_command_seeds_single_step_approved_skill() {
    let cmd = SavedCommand {
        id: "c1".into(),
        name: "disk".into(),
        command: "df -h".into(),
        description: String::new(),
    };
    let skill = migrate_command_to_skill(&cmd);
    assert_eq!(skill.name, "disk");
    assert_eq!(skill.steps.len(), 1);
    assert_eq!(skill.steps[0].command, "df -h");
    assert_eq!(skill.status, SkillStatus::Approved);
    assert_eq!(skill.created_by, "user");
    assert!(!skill.id.is_empty());
}

#[test]
fn skill_tool_name_sanitizes() {
    // Colons/spaces/dots collapse to underscores (MCP names allow no colon).
    assert_eq!(skill_tool_name("Restart Web"), "skill_restart_web");
    assert_eq!(skill_tool_name("rotate.logs"), "skill_rotate_logs");
    assert_eq!(skill_tool_name("  audit  "), "skill_audit");
}

#[test]
fn clip_trims_on_char_boundary() {
    // Short strings pass through untouched.
    assert_eq!(clip("hello", 100), "hello");
    // A multi-byte char at the cut point isn't split.
    let s = "aé"; // 'a' = 1 byte, 'é' = 2 bytes
    let out = clip(s, 2);
    assert!(out.starts_with('a'));
    assert!(out.contains("clipped"));
}

fn one_step_skill(name: &str) -> Skill {
    Skill {
        name: name.into(),
        steps: vec![SkillStep {
            name: String::new(),
            command: "true".into(),
        }],
        ..Default::default()
    }
}

/// The safety crux: a hand-authored skill is born approved, but an
/// AI-authored one is forced to a proposal it can't self-approve; only the
/// (local-UI) approve path flips it to runnable.
#[tokio::test]
async fn skills_store_propose_approve_delete() {
    // No config_path (default) → persist() is a no-op, so this touches no disk.
    let state = BridgeState::default();

    // Hand-authored via the local UI path → approved.
    let list = state.upsert_skill(one_step_skill("restart")).await;
    assert_eq!(list.len(), 1);
    assert_eq!(list[0].status, SkillStatus::Approved);
    let restart_id = list[0].id.clone();
    assert!(!restart_id.is_empty());

    // AI path: even if the definition claims approved/user, it's forced to a
    // proposal authored by "ai" with a fresh id.
    let sneaky = Skill {
        status: SkillStatus::Approved,
        created_by: "user".into(),
        id: "attacker-chosen".into(),
        ..one_step_skill("ai-skill")
    };
    let proposed = state.propose_skill(sneaky).await;
    assert_eq!(proposed.status, SkillStatus::Proposed);
    assert_eq!(proposed.created_by, "ai");
    assert_ne!(proposed.id, "attacker-chosen");

    // The human approve gate flips it to runnable.
    let found = state.find_skill("ai-skill").await.unwrap();
    assert_eq!(found.status, SkillStatus::Proposed);
    let list = state.approve_skill(&found.id).await;
    let ai = list.iter().find(|s| s.name == "ai-skill").unwrap();
    assert_eq!(ai.status, SkillStatus::Approved);

    // Delete removes exactly the targeted skill.
    let list = state.delete_skill(&restart_id).await;
    assert!(list.iter().all(|s| s.id != restart_id));
    assert_eq!(list.len(), 1);
}
