// Ghost FTP by Brendigo
use super::download_staging::DownloadStagingOpenFailure;
use super::*;

#[test]
fn agent_delta_completion_checks_remote_size_and_reused_bytes() {
    assert!(verify_agent_delta_upload_completion(1000, 800, 800, 1000, 1000).is_ok());
    assert!(verify_agent_delta_upload_completion(1000, 800, 799, 1000, 1000).is_err());
    assert!(verify_agent_delta_upload_completion(1000, 800, 800, 999, 1000).is_err());
    assert!(verify_agent_delta_upload_completion(1000, 800, 800, 1000, 999).is_err());
    assert!(verify_agent_delta_upload_completion(0, 0, 0, 0, 0).is_ok());
}

#[test]
fn delta_upload_patch_requires_exact_agent_acknowledgements() {
    assert!(verify_agent_delta_patch_ack(0, 0).is_ok());
    assert!(verify_agent_delta_patch_ack(128 * 1024, 128 * 1024).is_ok());
    assert!(verify_agent_delta_patch_ack(4096, 4095).is_err());
    assert!(verify_agent_delta_patch_ack(4096, 4097).is_err());
}

#[test]
fn delta_patch_length_never_exceeds_requested_range() {
    assert_eq!(checked_agent_delta_chunk_length(8, 8).unwrap(), 8);
    assert_eq!(checked_agent_delta_chunk_length(8, 3).unwrap(), 3);
    assert!(checked_agent_delta_chunk_length(8, 0).is_err());
    assert!(checked_agent_delta_chunk_length(8, 9).is_err());
    assert!(checked_agent_delta_chunk_length(0, 1).is_err());
}

// Inventory thresholds are only referenced by test cases. Keep their
// imports local so the production target passes -D unused-imports.
use super::path_safety::{
    MAX_DIRECTORY_DOWNLOAD_DEPTH, MAX_DIRECTORY_DOWNLOAD_ENTRIES, MAX_DIRECTORY_DOWNLOAD_PATH_BYTES,
};

#[tokio::test]
async fn spawn_registered_makes_handle_visible_before_worker_cleanup() {
    let mgr = Arc::new(TransferManager::new());
    let worker_mgr = Arc::clone(&mgr);
    let id = "immediate-worker".to_string();
    let worker_id = id.clone();
    let (tx, rx) = tokio::sync::oneshot::channel();

    mgr.spawn_registered(id, async move {
        let registered = worker_mgr.tasks.lock().await.remove(&worker_id).is_some();
        let _ = tx.send(registered);
    })
    .await;

    assert_eq!(
        rx.await,
        Ok(true),
        "worker cleanup must observe its registered JoinHandle even when the worker runs immediately"
    );
}

#[tokio::test]
async fn resume_prefix_requires_full_byte_identity_including_same_size_replacement() {
    let root = std::env::temp_dir().join(format!("ghostftp-resume-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let left = root.join("source.bin");
    let right = root.join("remote.bin");
    let content: Vec<u8> = (0..(128 * 1024 + 7)).map(|n| (n % 251) as u8).collect();
    tokio::fs::write(&left, &content).await.unwrap();
    tokio::fs::write(&right, &content).await.unwrap();
    let mut original = tokio::fs::File::open(&left).await.unwrap();
    let mut copy = tokio::fs::File::open(&right).await.unwrap();
    assert!(
        byte_equal_prefix(&mut original, &mut copy, content.len() as u64)
            .await
            .unwrap()
    );

    let mut changed = content.clone();
    changed[65536] ^= 0x40; // Same length, different content after first chunk.
    tokio::fs::write(&right, &changed).await.unwrap();
    let mut original = tokio::fs::File::open(&left).await.unwrap();
    let mut copy = tokio::fs::File::open(&right).await.unwrap();
    assert!(
        !byte_equal_prefix(&mut original, &mut copy, content.len() as u64)
            .await
            .unwrap()
    );

    let mut original = tokio::fs::File::open(&left).await.unwrap();
    let mut copy = tokio::fs::File::open(&right).await.unwrap();
    assert!(byte_equal_prefix(&mut original, &mut copy, 0)
        .await
        .unwrap());
    let mut original = tokio::fs::File::open(&left).await.unwrap();
    let mut copy = tokio::fs::File::open(&right).await.unwrap();
    assert!(byte_equal_prefix(&mut original, &mut copy, 65536)
        .await
        .unwrap());

    tokio::fs::write(&right, &content[..19]).await.unwrap();
    let mut original = tokio::fs::File::open(&left).await.unwrap();
    let mut copy = tokio::fs::File::open(&right).await.unwrap();
    assert!(
        byte_equal_prefix(&mut original, &mut copy, content.len() as u64)
            .await
            .is_err()
    );
    tokio::fs::remove_dir_all(root).await.unwrap();
}

#[tokio::test]
async fn whole_file_download_attempts_use_exclusive_non_clobbering_staging() {
    use tokio::io::AsyncWriteExt;

    let root = std::env::temp_dir().join(format!("ghostftp-exclusive-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let final_path = root.join("original.bin");
    tokio::fs::write(&final_path, b"preserve destination")
        .await
        .unwrap();

    let attempt_one = fresh_download_staging_path(&final_path, "transfer-1");
    let attempt_two = fresh_download_staging_path(&final_path, "transfer-1");
    assert_ne!(attempt_one, attempt_two);
    assert_eq!(attempt_one.parent(), final_path.parent());
    assert_eq!(attempt_two.parent(), final_path.parent());

    let mut file = create_new_download_staging(&attempt_one).await.unwrap();
    file.write_all(b"partial data").await.unwrap();
    file.sync_all().await.unwrap();
    drop(file);

    let collision = create_new_download_staging(&attempt_one).await.unwrap_err();
    assert!(
        collision
            .downcast_ref::<DownloadStagingOpenFailure>()
            .is_some(),
        "failed exclusive creation must be distinguishable from an owned partial transfer"
    );
    assert_eq!(
        tokio::fs::read(&attempt_one).await.unwrap(),
        b"partial data"
    );

    let mut replacement = create_new_download_staging(&attempt_two).await.unwrap();
    replacement.write_all(b"complete data").await.unwrap();
    replacement.sync_all().await.unwrap();
    drop(replacement);
    assert_eq!(
        tokio::fs::read(&final_path).await.unwrap(),
        b"preserve destination"
    );
    assert_eq!(
        tokio::fs::read(&attempt_two).await.unwrap(),
        b"complete data"
    );

    tokio::fs::remove_dir_all(&root).await.unwrap();
}

#[cfg(unix)]
#[tokio::test]
async fn download_staging_create_new_refuses_preplanted_symlink() {
    let root = std::env::temp_dir().join(format!("ghostftp-stage-symlink-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let valuable = root.join("private.txt");
    let staging = root.join("untrusted.part");
    tokio::fs::write(&valuable, b"important contents")
        .await
        .unwrap();
    std::os::unix::fs::symlink(&valuable, &staging).unwrap();

    let error = create_new_download_staging(&staging).await.unwrap_err();
    assert!(error.downcast_ref::<DownloadStagingOpenFailure>().is_some());
    assert_eq!(
        tokio::fs::read(&valuable).await.unwrap(),
        b"important contents"
    );
    assert!(tokio::fs::symlink_metadata(&staging)
        .await
        .unwrap()
        .file_type()
        .is_symlink());
    tokio::fs::remove_dir_all(&root).await.unwrap();
}

#[tokio::test]
async fn every_protocol_download_staging_requires_exact_bytes_and_durable_file() {
    let dir = std::env::temp_dir().join(format!("ghostftp-verify-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&dir).await.unwrap();
    let file = dir.join("transfer.part");
    tokio::fs::write(&file, b"binary\0payload").await.unwrap();
    assert!(verify_staged_download_bytes(&file, 14).await.is_ok());
    assert!(verify_staged_download_bytes(&file, 13).await.is_err());
    assert!(verify_staged_download_bytes(&file, 15).await.is_err());
    tokio::fs::write(&file, b"").await.unwrap();
    assert!(verify_staged_download_bytes(&file, 0).await.is_ok());
    tokio::fs::remove_file(&file).await.unwrap();
    tokio::fs::remove_dir_all(&dir).await.unwrap();
}

#[cfg(any(target_os = "linux", windows))]
#[tokio::test]
async fn original_download_writer_survives_path_substitution() {
    let dir = std::env::temp_dir().join(format!("ghostftp-writer-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&dir).await.unwrap();
    let stage = dir.join("transfer.part");
    let original = dir.join("original.part");
    let target = dir.join("target.bin");
    let guard = DownloadParentGuard::open(&target).unwrap();
    let mut writer = create_new_download_staging(&stage).await.unwrap();
    writer.write_all(b"good").await.unwrap();
    writer.flush().await.unwrap();
    tokio::fs::write(&target, b"keep").await.unwrap();
    tokio::fs::rename(&stage, &original).await.unwrap();
    tokio::fs::write(&stage, b"fake").await.unwrap();

    assert!(verify_download_writer(&writer, 4).await.is_ok());
    assert!(verify_download_writer(&writer, 3).await.is_err());
    assert!(guard
        .promote(&stage, &target, "writer", true, &writer)
        .await
        .is_err());
    assert_eq!(tokio::fs::read(&target).await.unwrap(), b"keep");
    assert_eq!(tokio::fs::read(&original).await.unwrap(), b"good");
    assert_eq!(tokio::fs::read(&stage).await.unwrap(), b"fake");
    drop(writer);
    drop(guard);
    tokio::fs::remove_dir_all(dir).await.unwrap();
}

#[tokio::test]
async fn rename_or_skip_download_never_replaces_racing_destination() {
    let root = std::env::temp_dir().join(format!("ghostftp-noreplace-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let destination = root.join("download.bin");
    let staged = download_staging_path(&destination, "skip");
    tokio::fs::write(&staged, b"downloaded").await.unwrap();

    // Another transfer or local process creates the destination *after*
    // Skip/Rename policy resolution but before this download commits.
    tokio::fs::write(&destination, b"other process")
        .await
        .unwrap();
    assert!(promote_staged_download_noreplace(&staged, &destination)
        .await
        .is_err());
    assert_eq!(
        tokio::fs::read(&destination).await.unwrap(),
        b"other process"
    );
    assert_eq!(tokio::fs::read(&staged).await.unwrap(), b"downloaded");

    tokio::fs::remove_file(&destination).await.unwrap();
    promote_staged_download_noreplace(&staged, &destination)
        .await
        .unwrap();
    assert_eq!(tokio::fs::read(&destination).await.unwrap(), b"downloaded");
    assert!(tokio::fs::symlink_metadata(&staged).await.is_err());
    tokio::fs::remove_dir_all(root).await.unwrap();
}

#[test]
fn persisted_download_policy_defaults_to_no_replace_for_legacy_rows() {
    let legacy = serde_json::json!({
        "kind": "download",
        "profile_id": "p",
        "remote_path": "/remote/file",
        "final_path": "/tmp/file"
    });
    let decoded: PersistedRetryInfo = serde_json::from_value(legacy).unwrap();
    match decoded {
        PersistedRetryInfo::Download { allow_replace, .. } => assert!(!allow_replace),
        PersistedRetryInfo::Upload { .. } => panic!("wrong persisted retry variant"),
    }
}

#[test]
fn legacy_upload_retry_never_grants_remote_overwrite_permission() {
    let legacy = serde_json::json!({
        "kind": "upload",
        "profile_id": "saved-profile",
        "local": "/tmp/local.bin",
        "final_remote": "/remote/protected.bin"
    });
    let decoded: PersistedRetryInfo = serde_json::from_value(legacy).unwrap();
    match decoded {
        PersistedRetryInfo::Upload { allow_replace, .. } => assert!(!allow_replace),
        _ => panic!("expected an upload retry descriptor"),
    }
}

#[tokio::test]
async fn staged_download_promotion_replaces_only_after_complete_temp_exists() {
    let root = std::env::temp_dir().join(format!("ghostftp-stage-test-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let final_path = root.join("file.bin");
    let staging = download_staging_path(&final_path, "t1");
    tokio::fs::write(&final_path, b"old").await.unwrap();
    tokio::fs::write(&staging, b"new-complete").await.unwrap();

    promote_staged_download(&staging, &final_path, "t1")
        .await
        .unwrap();

    assert_eq!(tokio::fs::read(&final_path).await.unwrap(), b"new-complete");
    assert!(!staging.exists());
    assert!(!download_backup_path(&final_path, "t1").exists());
    let _ = tokio::fs::remove_dir_all(&root).await;
}

#[tokio::test]
async fn staged_download_failure_before_promotion_leaves_original_untouched() {
    let root = std::env::temp_dir().join(format!("ghostftp-stage-test-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let final_path = root.join("file.bin");
    let staging = download_staging_path(&final_path, "t2");
    tokio::fs::write(&final_path, b"old-safe").await.unwrap();
    tokio::fs::write(&staging, b"partial").await.unwrap();

    // Simulate a failed/canceled backend before promotion: only the staging
    // file is discarded by the runner; the visible destination is untouched.
    tokio::fs::remove_file(&staging).await.unwrap();

    assert_eq!(tokio::fs::read(&final_path).await.unwrap(), b"old-safe");
    assert!(!staging.exists());
    let _ = tokio::fs::remove_dir_all(&root).await;
}

#[tokio::test]
async fn download_promotion_rejects_existing_directory_and_preserves_contents() {
    let root = std::env::temp_dir().join(format!("ghostftp-dir-audit-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let final_path = root.join("occupied");
    tokio::fs::create_dir(&final_path).await.unwrap();
    tokio::fs::write(final_path.join("important.txt"), b"keep")
        .await
        .unwrap();
    let staging = download_staging_path(&final_path, "dir-id");
    tokio::fs::write(&staging, b"staged").await.unwrap();

    let error = promote_staged_download(&staging, &final_path, "dir-id")
        .await
        .unwrap_err();
    assert!(error.to_string().contains("not a regular file"));
    assert_eq!(
        tokio::fs::read(final_path.join("important.txt"))
            .await
            .unwrap(),
        b"keep"
    );
    assert_eq!(tokio::fs::read(&staging).await.unwrap(), b"staged");
    tokio::fs::remove_dir_all(root).await.unwrap();
}

#[cfg(unix)]
#[tokio::test]
async fn staging_symlink_is_never_promoted_into_download_target() {
    use std::os::unix::fs::symlink;
    let root = std::env::temp_dir().join(format!("ghostftp-stage-link-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let final_path = root.join("download.txt");
    let protected = root.join("protected.txt");
    tokio::fs::write(&protected, b"protected").await.unwrap();
    let staging = download_staging_path(&final_path, "bad-stage");
    symlink(&protected, &staging).unwrap();

    let error = promote_staged_download(&staging, &final_path, "bad-stage")
        .await
        .unwrap_err();
    assert!(error.to_string().contains("not a regular file"));
    assert_eq!(tokio::fs::read(&protected).await.unwrap(), b"protected");
    assert!(!final_path.exists());
    tokio::fs::remove_dir_all(root).await.unwrap();
}

#[tokio::test]
async fn staged_download_paths_are_sibling_scoped_and_transfer_specific() {
    let final_path = PathBuf::from("/tmp/downloads/report.bin");
    assert_eq!(
        download_staging_path(&final_path, "abc"),
        PathBuf::from("/tmp/downloads/.ghostftp-download-abc.part")
    );
    assert_eq!(
        download_backup_path(&final_path, "abc"),
        PathBuf::from("/tmp/downloads/.ghostftp-download-abc.backup")
    );
}

#[tokio::test]
async fn download_promotion_preserves_existing_backup_instead_of_deleting_it() {
    let root = std::env::temp_dir().join(format!("ghostftp-backup-audit-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let final_path = root.join("file.bin");
    let staging = download_staging_path(&final_path, "same-id");
    let backup = download_backup_path(&final_path, "same-id");
    tokio::fs::write(&final_path, b"original").await.unwrap();
    tokio::fs::write(&staging, b"replacement").await.unwrap();
    tokio::fs::write(&backup, b"preserved-backup")
        .await
        .unwrap();

    let error = promote_staged_download(&staging, &final_path, "same-id")
        .await
        .unwrap_err();
    assert!(format!("{error:#}").contains("without overwriting a backup"));
    assert_eq!(tokio::fs::read(&final_path).await.unwrap(), b"original");
    assert_eq!(tokio::fs::read(&backup).await.unwrap(), b"preserved-backup");
    assert_eq!(tokio::fs::read(&staging).await.unwrap(), b"replacement");
    tokio::fs::remove_dir_all(root).await.unwrap();
}

#[cfg(unix)]
#[tokio::test]
async fn dangling_symlinks_are_occupied_and_never_promoted_over() {
    use std::os::unix::fs::symlink;
    let root = std::env::temp_dir().join(format!("ghostftp-link-audit-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let final_path = root.join("file.txt");
    let missing = root.join("missing.txt");
    symlink(&missing, &final_path).unwrap();
    let staging = download_staging_path(&final_path, "link-id");
    tokio::fs::write(&staging, b"new").await.unwrap();

    assert!(local_path_exists_fail_closed(&final_path).await.unwrap());
    assert_ne!(resolve_local_rename(&final_path).unwrap(), final_path);
    let error = promote_staged_download(&staging, &final_path, "link-id")
        .await
        .unwrap_err();
    assert!(error.to_string().contains("not a regular file"));
    assert!(tokio::fs::symlink_metadata(&final_path)
        .await
        .unwrap()
        .file_type()
        .is_symlink());
    assert!(!missing.exists());
    assert_eq!(tokio::fs::read(&staging).await.unwrap(), b"new");
    tokio::fs::remove_dir_all(root).await.unwrap();
}

#[cfg(unix)]
#[test]
fn local_rename_skips_dangling_symlink_candidates() {
    use std::os::unix::fs::symlink;
    let root = std::env::temp_dir().join(format!("ghostftp-rename-links-{}", Uuid::new_v4()));
    std::fs::create_dir_all(&root).unwrap();
    let base = root.join("file.txt");
    std::fs::write(&base, b"existing").unwrap();
    symlink(root.join("missing"), root.join("file_1.txt")).unwrap();
    assert_eq!(
        resolve_local_rename(&base).unwrap(),
        root.join("file_2.txt")
    );
    std::fs::remove_dir_all(root).unwrap();
}

#[test]
fn folder_upload_budget_and_name_guards_are_fail_closed() {
    assert!(validate_directory_upload_budget(100_000, 64, 16 * 1024 * 1024).is_ok());
    assert!(validate_directory_upload_budget(100_001, 0, 0).is_err());
    assert!(validate_directory_upload_budget(1, 65, 0).is_err());
    assert!(validate_directory_upload_budget(1, 0, 16 * 1024 * 1024 + 1).is_err());
    for bad in [
        "",
        ".",
        "..",
        "bad\\name",
        "bad/name",
        "bad\rname",
        "bad\nname",
    ] {
        assert!(!safe_upload_name(bad), "accepted invalid name: {bad:?}");
    }
    assert!(safe_upload_name("račun 2026.txt"));
}

#[test]
fn remote_directory_download_prevents_path_traversal_and_sibling_prefixes() {
    let root = "/remote/home";
    assert_eq!(
        checked_remote_download_relative(root, "/remote/home/photos/day1.jpg").unwrap(),
        PathBuf::from("photos").join("day1.jpg")
    );
    assert_eq!(
        checked_remote_download_relative("C:\\srv\\root", "C:\\srv\\root\\album\\photo.png")
            .unwrap(),
        PathBuf::from("album").join("photo.png")
    );
    for bad in [
        "/remote/home/../secrets",
        "/remote/home/./photo.png",
        "/remote/home//photo.png",
        "/remote/home2/photo.png",
        "/etc/passwd",
        "/remote/home/a:b",
        "/remote/home/dir\\..\\outside",
        "/remote/home",
    ] {
        assert!(
            checked_remote_download_relative(root, bad).is_err(),
            "{bad}"
        );
    }
}

#[test]
fn remote_root_directory_download_keeps_path_validation() {
    assert_eq!(
        checked_remote_download_relative("/", "/docs/report.txt").unwrap(),
        PathBuf::from("docs").join("report.txt")
    );
    assert_eq!(
        checked_remote_download_relative("/", "/file.txt").unwrap(),
        PathBuf::from("file.txt")
    );

    // Root traversal, ambiguous separators and unsafe local names must
    // still fail closed when the selected remote directory is "/".
    for child in [
        "/",
        "file.txt",
        "//etc/passwd",
        "/../secret",
        "/dir/../secret",
        "/dir/./nested.txt",
        "/dir//nested.txt",
        "/dir\\secret.txt",
        "/CON.txt",
    ] {
        assert!(
            checked_remote_download_relative("/", child).is_err(),
            "unsafe remote-root child accepted: {child:?}"
        );
    }
    for root in ["", "//", "///"] {
        assert!(
            checked_remote_download_relative(root, "/file.txt").is_err(),
            "invalid remote root accepted: {root:?}"
        );
    }
}

#[cfg(unix)]
#[tokio::test]
async fn directory_download_rejects_symlinked_local_parents() {
    use std::os::unix::fs::symlink;
    let sandbox = std::env::temp_dir().join(format!("ghostftp-download-links-{}", Uuid::new_v4()));
    let root = sandbox.join("download");
    let outside = sandbox.join("other-user-files");
    tokio::fs::create_dir_all(&root).await.unwrap();
    tokio::fs::create_dir_all(&outside).await.unwrap();
    symlink(&outside, root.join("nested")).unwrap();
    assert!(
        ensure_download_subdirectory(&root, &root.join("nested").join("subdir"))
            .await
            .is_err()
    );
    assert!(tokio::fs::read_dir(&outside)
        .await
        .unwrap()
        .next_entry()
        .await
        .unwrap()
        .is_none());
    tokio::fs::remove_dir_all(sandbox).await.unwrap();
}

#[test]
fn directory_download_10k_inventory_budget_is_bounded() {
    let mut bytes = 0usize;
    for i in 0..10_000 {
        let name = format!("/remote/home/f{i:05}.dat");
        bytes += name.len();
        assert!(checked_remote_download_relative("/remote/home", &name).is_ok());
        validate_directory_download_budget(i + 1, 1, bytes).unwrap();
    }
    assert!(
        validate_directory_download_budget(MAX_DIRECTORY_DOWNLOAD_ENTRIES + 1, 1, bytes).is_err()
    );
    assert!(
        validate_directory_download_budget(1, MAX_DIRECTORY_DOWNLOAD_DEPTH + 1, bytes).is_err()
    );
    assert!(
        validate_directory_download_budget(1, 1, MAX_DIRECTORY_DOWNLOAD_PATH_BYTES + 1).is_err()
    );
}

#[test]
fn cloud_source_length_guards_4gib_and_partial_parts_without_allocations() {
    let huge = 4u64 * 1024 * 1024 * 1024 + 1024;
    assert!(require_exact_upload_length(huge, huge).is_ok());
    assert!(require_exact_upload_length(huge, huge - 1).is_err());
    assert!(require_exact_upload_length(huge, huge + 1).is_err());
    assert!(require_exact_upload_length(8 * 1024 * 1024, 4 * 1024 * 1024).is_err());
    assert_eq!(MAX_BUFFERED_CLOUD_UPLOAD, 32 * 1024 * 1024);
}

#[tokio::test]
async fn in_memory_cloud_adapters_reject_huge_files_without_reading_them() {
    let root = std::env::temp_dir().join(format!("ghostftp-bounded-cloud-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let path = root.join("sparse-large-file");
    let file = tokio::fs::File::create(&path).await.unwrap();
    file.set_len(MAX_BUFFERED_CLOUD_UPLOAD + 1).await.unwrap();
    drop(file);
    assert!(
        read_cloud_upload_bounded(&path, MAX_BUFFERED_CLOUD_UPLOAD + 1)
            .await
            .is_err()
    );
    tokio::fs::remove_dir_all(&root).await.unwrap();
}

#[tokio::test]
async fn bounded_cloud_reader_rejects_changed_files_and_preserves_content() {
    let root = std::env::temp_dir().join(format!("ghostftp-bounded-content-{}", Uuid::new_v4()));
    tokio::fs::create_dir_all(&root).await.unwrap();
    let path = root.join("small.bin");
    tokio::fs::write(&path, b"abcde").await.unwrap();
    assert_eq!(read_cloud_upload_bounded(&path, 5).await.unwrap(), b"abcde");
    assert!(read_cloud_upload_bounded(&path, 6).await.is_err());
    assert!(read_cloud_upload_bounded(&path, 4).await.is_err());
    assert_eq!(tokio::fs::read(&path).await.unwrap(), b"abcde");
    tokio::fs::remove_dir_all(&root).await.unwrap();
}

// ---------- TokenBucket ----------

#[tokio::test(start_paused = true)]
async fn token_bucket_caps_throughput() {
    let bucket = TokenBucket::new();
    bucket.set_rate_kbps(512); // 512 KiB/s
    let start = tokio::time::Instant::now();
    // 1 MiB at 512 KiB/s must take ~2s of (virtual) time, charged in full.
    bucket.acquire(1024 * 1024).await;
    let elapsed = start.elapsed();
    assert!(
        elapsed >= Duration::from_millis(1900),
        "too fast: {elapsed:?}"
    );
    assert!(
        elapsed <= Duration::from_millis(2600),
        "too slow: {elapsed:?}"
    );
}

#[tokio::test(start_paused = true)]
async fn token_bucket_unlimited_is_instant() {
    let bucket = TokenBucket::new(); // rate 0 = unlimited
    let start = tokio::time::Instant::now();
    bucket.acquire(10 * 1024 * 1024).await;
    assert_eq!(start.elapsed(), Duration::ZERO);
}

#[tokio::test(start_paused = true)]
async fn token_bucket_tiny_file_skips_full_window() {
    let bucket = TokenBucket::new();
    bucket.set_rate_kbps(64); // 64 KiB/s
    let start = tokio::time::Instant::now();
    bucket.acquire(1024).await; // 1 KiB → ~16ms, not a whole window
    assert!(start.elapsed() < Duration::from_millis(100));
}

#[tokio::test(start_paused = true)]
async fn token_bucket_enable_does_not_credit_unlimited_idle_time() {
    let bucket = TokenBucket::new();
    tokio::time::sleep(Duration::from_secs(120)).await;
    bucket.set_rate_kbps(1);
    let start = tokio::time::Instant::now();
    bucket.acquire(1024).await;
    assert!(start.elapsed() >= Duration::from_secs(1));
}

#[tokio::test(start_paused = true)]
async fn token_bucket_low_rate_caps_idle_burst_at_one_second() {
    let bucket = TokenBucket::new();
    bucket.set_rate_kbps(1);
    tokio::time::sleep(Duration::from_secs(120)).await;
    let start = tokio::time::Instant::now();
    bucket.acquire(2048).await;
    assert!(start.elapsed() >= Duration::from_secs(1));
    assert!(start.elapsed() <= Duration::from_millis(1250));
}

#[tokio::test(start_paused = true)]
async fn token_bucket_rate_increase_does_not_refill_retroactively() {
    let bucket = TokenBucket::new();
    bucket.set_rate_kbps(1);
    tokio::time::sleep(Duration::from_secs(10)).await;
    bucket.set_rate_kbps(4);
    let start = tokio::time::Instant::now();
    bucket.acquire(4096).await;
    assert!(start.elapsed() >= Duration::from_millis(750));
}

#[tokio::test(start_paused = true)]
async fn token_bucket_rate_decrease_discards_excess_credit() {
    let bucket = TokenBucket::new();
    bucket.set_rate_kbps(64);
    tokio::time::sleep(Duration::from_secs(10)).await;
    bucket.set_rate_kbps(1);
    let start = tokio::time::Instant::now();
    bucket.acquire(2048).await;
    assert!(start.elapsed() >= Duration::from_secs(1));
    assert_eq!(bucket.rate_kbps(), 1);
}

#[tokio::test(start_paused = true)]
async fn ftp_and_async_adapters_debit_the_same_bucket() {
    let bucket = TokenBucket::new();
    bucket.set_rate_kbps(1);
    tokio::time::sleep(Duration::from_secs(1)).await;
    assert_eq!(
        bucket.acquire_ftp(1024, || FtpTransferControl::Continue),
        FtpTransferControl::Continue
    );
    let start = tokio::time::Instant::now();
    bucket.acquire(1024).await;
    assert!(start.elapsed() >= Duration::from_secs(1));
}

#[test]
fn ftp_budget_honors_pause_and_cancel_before_waiting() {
    let bucket = Arc::new(TokenBucket::new());
    bucket.set_rate_kbps(1);
    let canceled = Arc::new(AtomicBool::new(false));
    let pause = PauseGate::new();
    pause.set(true);
    let mut budget = FtpBudget {
        bucket,
        profile: None,
        pause_all: PauseGate::new(),
        pause: Some(pause),
        canceled: Arc::clone(&canceled),
        accounted: 0,
    };
    assert_eq!(budget.control(65536), FtpTransferControl::Pause);
    canceled.store(true, Ordering::Release);
    assert_eq!(budget.control(65536), FtpTransferControl::Cancel);
}

#[test]
fn budgeted_ftp_reader_does_not_report_eof_on_cancellation() {
    let canceled = Arc::new(AtomicBool::new(true));
    let mut reader = BudgetedFtpReader {
        inner: std::io::Cursor::new(b"payload"),
        budget: FtpBudget {
            bucket: Arc::new(TokenBucket::new()),
            profile: None,
            pause_all: PauseGate::new(),
            pause: None,
            canceled,
            accounted: 0,
        },
        stop: FtpTransferControl::Continue,
    };
    let mut buf = [0; 8];
    assert!(std::io::Read::read(&mut reader, &mut buf).is_err());
    assert_eq!(reader.stop, FtpTransferControl::Cancel);
    assert_eq!(reader.inner.position(), 0);
}

#[tokio::test(start_paused = true)]
async fn profile_bandwidth_is_shared_and_persisted() {
    let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
    let mgr = TransferManager::with_db(db.clone());
    mgr.set_profile_throttle_kbps("site-a", 1).unwrap();
    mgr.set_profile_throttle_kbps("site-b", 4).unwrap();
    let a = mgr.profile_bandwidth.lookup("site-a").unwrap();
    let a2 = mgr.profile_bandwidth.lookup("site-a").unwrap();
    assert!(Arc::ptr_eq(&a, &a2));
    assert!(!Arc::ptr_eq(
        &a,
        &mgr.profile_bandwidth.lookup("site-b").unwrap()
    ));
    tokio::time::sleep(Duration::from_secs(1)).await;
    a.acquire(1024).await;
    let start = tokio::time::Instant::now();
    a2.acquire(1024).await;
    assert!(start.elapsed() >= Duration::from_secs(1));
    let restored = TransferManager::with_db(db);
    assert_eq!(restored.profile_throttle_kbps("site-a").unwrap(), 1);
    assert_eq!(restored.profile_throttle_kbps("site-b").unwrap(), 4);
    let held = restored.profile_bandwidth.lookup("site-a").unwrap();
    restored.set_profile_throttle_kbps("site-a", 0).unwrap();
    assert_eq!(held.rate_kbps(), 0);
    assert!(restored.profile_bandwidth.lookup("site-a").is_none());
}

#[test]
fn corrupt_profile_bandwidth_settings_are_not_replaced() {
    let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
    db.settings_set(PROFILE_BANDWIDTH_KEY, "invalid").unwrap();
    let mgr = TransferManager::with_db(db.clone());
    assert!(mgr.profile_throttle_kbps("site").is_err());
    assert!(mgr.set_profile_throttle_kbps("site", 10).is_err());
    assert_eq!(
        db.settings_get(PROFILE_BANDWIDTH_KEY).unwrap().as_deref(),
        Some("invalid")
    );
}

// ---------- PauseGate + checkpoint ----------

#[tokio::test]
async fn pause_gate_parks_until_opened() {
    let gate = PauseGate::new();
    gate.set(true);
    let g2 = gate.clone();
    let handle = tokio::spawn(async move { g2.wait_open().await });
    for _ in 0..10 {
        tokio::task::yield_now().await;
    }
    assert!(!handle.is_finished());
    gate.set(false);
    handle.await.unwrap();
}

#[tokio::test]
async fn checkpoint_parks_then_signals_restart() {
    let mgr = Arc::new(TransferManager::new());
    mgr.pauses
        .lock()
        .await
        .insert("t1".into(), PauseGate::new());
    // Not paused → passes straight through.
    mgr.checkpoint("t1", 128).await.unwrap();

    mgr.pauses.lock().await.get("t1").unwrap().set(true);
    let m2 = Arc::clone(&mgr);
    let handle = tokio::spawn(async move { m2.checkpoint("t1", 128).await });
    // The parked task cannot finish before the gate opens (Ok needs an
    // open gate, Err needs the park loop to break) — so a finished handle
    // here is impossible regardless of scheduling.
    tokio::time::sleep(Duration::from_millis(50)).await;
    assert!(!handle.is_finished());
    mgr.pauses.lock().await.get("t1").unwrap().set(false);
    let err = handle.await.unwrap().unwrap_err();
    assert!(err.downcast_ref::<RestartFromPause>().is_some());
}

fn terminal_race_test_row(id: &str, status: TransferStatus) -> Transfer {
    Transfer {
        id: id.to_string(),
        kind: TransferKind::Upload,
        source: "/tmp/queued-upload.bin".to_string(),
        destination: "/remote/queued-upload.bin".to_string(),
        size: 1024,
        transferred: 0,
        status,
        error: None,
        retry_attempt: None,
        delta: None,
        started_at: 11,
    }
}

#[tokio::test]
async fn canceled_transfer_is_never_resurrected_by_late_completion_or_progress() {
    let mgr = TransferManager::new();
    mgr.insert(terminal_race_test_row(
        "canceled",
        TransferStatus::Transferring,
    ))
    .await;
    let flag = Arc::new(AtomicBool::new(false));
    mgr.cancels
        .lock()
        .await
        .insert("canceled".to_string(), Arc::clone(&flag));
    assert!(mgr.mark_canceled("canceled").await.is_some());
    assert!(flag.load(Ordering::Acquire));
    // Both successful and failed results from a stale worker are ignored.
    assert!(mgr.finish_transfer("canceled", Ok(())).await.is_none());
    assert!(mgr
        .finish_transfer("canceled", Err(anyhow::anyhow!("late network error")))
        .await
        .is_none());
    mgr.update("canceled", |t| {
        t.status = TransferStatus::Transferring;
        t.transferred = 1024;
    })
    .await;
    let row = mgr.snapshot("canceled").await.unwrap();
    assert_eq!(row.status, TransferStatus::Canceled);
    assert_eq!(row.transferred, 0);
    mgr.reset_for_retry("canceled").await.unwrap();
    assert_eq!(
        mgr.snapshot("canceled").await.unwrap().status,
        TransferStatus::Queued
    );
    assert!(mgr.reset_for_retry("canceled").await.is_err());
}

#[tokio::test]
async fn completed_row_and_sqlite_ledger_reject_stale_progress_after_success() {
    let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
    let mgr = TransferManager::with_db(db.clone());
    let id = "completed-ledger";
    mgr.persisted_retry.lock().await.insert(
        id.to_string(),
        PersistedRetryInfo::Upload {
            profile_id: "profile".to_string(),
            local: PathBuf::from("/tmp/queued-upload.bin"),
            final_remote: "/remote/queued-upload.bin".to_string(),
            allow_replace: false,
        },
    );
    mgr.insert(terminal_race_test_row(id, TransferStatus::Transferring))
        .await;
    mgr.update(id, |t| t.transferred = 1024).await;
    let done = mgr.finish_transfer(id, Ok(())).await.unwrap();
    assert_eq!(done.status, TransferStatus::Done);
    mgr.update(id, |t| {
        t.status = TransferStatus::Transferring;
        t.transferred = 1;
        t.error = Some("late callback".to_string());
    })
    .await;
    assert_eq!(mgr.snapshot(id).await.unwrap().status, TransferStatus::Done);
    assert_eq!(mgr.snapshot(id).await.unwrap().transferred, 1024);
    let row = db
        .transfer_ledger_list()
        .unwrap()
        .into_iter()
        .find(|(row_id, _, _)| row_id == id)
        .unwrap();
    let envelope: PersistedTransferEnvelope = serde_json::from_str(&row.1).unwrap();
    assert_eq!(envelope.transfer.status, TransferStatus::Done);
    assert_eq!(envelope.transfer.transferred, 1024);
    assert!(envelope.transfer.error.is_none());
}

#[tokio::test]
async fn concurrent_cancel_and_completion_persist_the_actual_terminal_winner() {
    let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
    let mgr = Arc::new(TransferManager::with_db(db.clone()));
    let id = "competing-terminal";
    mgr.persisted_retry.lock().await.insert(
        id.to_string(),
        PersistedRetryInfo::Upload {
            profile_id: "profile".to_string(),
            local: PathBuf::from("/tmp/source.bin"),
            final_remote: "/remote/source.bin".to_string(),
            allow_replace: false,
        },
    );
    let mut row = terminal_race_test_row(id, TransferStatus::Transferring);
    row.transferred = row.size;
    mgr.insert(row).await;
    let gate = Arc::new(tokio::sync::Barrier::new(3));
    let finisher = {
        let mgr = Arc::clone(&mgr);
        let gate = Arc::clone(&gate);
        tokio::spawn(async move {
            gate.wait().await;
            mgr.finish_transfer(id, Ok(())).await
        })
    };
    let canceler = {
        let mgr = Arc::clone(&mgr);
        let gate = Arc::clone(&gate);
        tokio::spawn(async move {
            gate.wait().await;
            let _ = mgr.mark_canceled(id).await;
        })
    };
    gate.wait().await;
    let _ = finisher.await.unwrap();
    canceler.await.unwrap();

    let memory = mgr.snapshot(id).await.unwrap();
    let entries = db.transfer_ledger_list().unwrap();
    let (_, raw, _) = entries
        .into_iter()
        .find(|(entry_id, _, _)| entry_id == id)
        .unwrap();
    let durable: PersistedTransferEnvelope = serde_json::from_str(&raw).unwrap();
    assert!(matches!(
        memory.status,
        TransferStatus::Done | TransferStatus::Canceled
    ));
    assert_eq!(durable.transfer.status, memory.status);
    assert_eq!(durable.transfer.transferred, memory.transferred);
    assert_eq!(memory.transferred, 1024);
}

#[tokio::test]
async fn concurrent_user_retries_only_allow_one_terminal_to_queued_reset() {
    let mgr = Arc::new(TransferManager::new());
    mgr.insert(terminal_race_test_row("one-retry", TransferStatus::Error))
        .await;
    let gate = Arc::new(tokio::sync::Barrier::new(9));
    let mut workers = Vec::new();
    for _ in 0..8 {
        let mgr = Arc::clone(&mgr);
        let gate = Arc::clone(&gate);
        workers.push(tokio::spawn(async move {
            gate.wait().await;
            mgr.reset_for_retry("one-retry").await.is_ok()
        }));
    }
    gate.wait().await;
    let mut accepted = 0;
    for worker in workers {
        if worker.await.unwrap() {
            accepted += 1;
        }
    }
    assert_eq!(accepted, 1);
    assert_eq!(
        mgr.snapshot("one-retry").await.unwrap().status,
        TransferStatus::Queued
    );
}

#[tokio::test]
async fn incomplete_backend_success_cannot_be_marked_done() {
    let mgr = TransferManager::new();
    mgr.insert(terminal_race_test_row(
        "short",
        TransferStatus::Transferring,
    ))
    .await;
    let failed = mgr.finish_transfer("short", Ok(())).await.unwrap();
    assert_eq!(failed.status, TransferStatus::Error);
    assert!(failed.error.is_some());
    assert!(mgr.finish_transfer("short", Ok(())).await.is_none());
}

// ---------- Durable restart recovery ----------

#[tokio::test]
async fn persisted_failed_transfer_can_reattach_after_restart() {
    let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
    let envelope = PersistedTransferEnvelope {
        schema: TRANSFER_LEDGER_SCHEMA,
        priority: TransferPriority::Normal,
        transfer: Transfer {
            id: "failed-before-restart".to_string(),
            kind: TransferKind::Upload,
            source: "/tmp/source.bin".to_string(),
            destination: "/remote/source.bin".to_string(),
            size: 4096,
            transferred: 2048,
            status: TransferStatus::Error,
            error: Some(
                "connection reset by peer, https://alice:private@host.example/download?x-amz-signature=supersecret".to_string()
            ),
            retry_attempt: Some(2),
            delta: None,
            started_at: 10,
        },
        retry: PersistedRetryInfo::Upload {
            profile_id: "profile-retry".to_string(),
            local: PathBuf::from("/tmp/source.bin"),
            final_remote: "/remote/source.bin".to_string(),
            allow_replace: false,
        },
    };
    db.transfer_ledger_upsert(
        "failed-before-restart",
        &serde_json::to_string(&envelope).unwrap(),
    )
    .unwrap();

    let recovered = TransferManager::with_db(db.clone());
    assert_eq!(
        recovered
            .recovery_profile_id("failed-before-restart")
            .await
            .as_deref(),
        Some("profile-retry")
    );
    let row = recovered.snapshot("failed-before-restart").await.unwrap();
    assert_eq!(row.status, TransferStatus::Error);
    assert_eq!(row.transferred, 2048);
    assert_eq!(
        row.error.as_deref(),
        Some("Network connection was interrupted. Check the server and retry.")
    );
    let history = db.transfer_ledger_list().unwrap();
    let recovered_json = &history[0].1;
    assert!(!recovered_json.contains("supersecret"));
    assert!(!recovered_json.contains("alice:private"));
    assert!(!recovered_json.contains("host.example"));
    assert_eq!(row.transferred, 2048); // recovery progress must be preserved.
}

#[tokio::test]
async fn durable_recovery_keeps_progress_for_display_but_requires_safe_restart() {
    let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
    let id = "recover-me";

    {
        let mgr = TransferManager::with_db(db.clone());
        mgr.persisted_retry.lock().await.insert(
            id.to_string(),
            PersistedRetryInfo::Download {
                profile_id: "profile-1".to_string(),
                remote_path: "/srv/archive.bin".to_string(),
                final_path: PathBuf::from("/tmp/archive.bin"),
                allow_replace: false,
            },
        );
        mgr.insert(Transfer {
            id: id.to_string(),
            kind: TransferKind::Download,
            source: "/srv/archive.bin".to_string(),
            destination: "/tmp/archive.bin".to_string(),
            size: 1024 * 1024,
            transferred: 256 * 1024,
            status: TransferStatus::Transferring,
            error: None,
            retry_attempt: Some(1),
            delta: None,
            started_at: 123,
        })
        .await;

        let rows = db.transfer_ledger_list().unwrap();
        assert_eq!(rows.len(), 1);
        assert_eq!(rows[0].0, id);
        assert!(rows[0].1.contains("profile-1"));
        assert!(!rows[0].1.contains("password"));
        assert!(!rows[0].1.contains("passphrase"));
        assert!(!rows[0].1.contains("token"));
    }

    // A fresh manager simulates the next application process after a crash
    // or update. The row must not pretend to still be actively transferring.
    let recovered = TransferManager::with_db(db);
    let row = recovered.snapshot(id).await.unwrap();
    assert_eq!(row.status, TransferStatus::Error);
    assert_eq!(row.transferred, 256 * 1024);
    assert_eq!(row.retry_attempt, None);
    assert!(row
        .error
        .as_deref()
        .is_some_and(|message| message.contains("interrupted")));
    assert_eq!(
        recovered.recovery_profile_id(id).await.as_deref(),
        Some("profile-1")
    );
    assert!(recovered.recovered_retry.lock().await.contains(id));
    assert!(row
        .error
        .as_deref()
        .is_some_and(|message| message.contains("restart safely from byte zero")));
}

#[tokio::test]
async fn terminal_history_is_pruned_during_runtime_without_dropping_active_rows() {
    let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
    let mgr = TransferManager::with_db(db.clone());

    mgr.persisted_retry.lock().await.insert(
        "active".to_string(),
        PersistedRetryInfo::Upload {
            profile_id: "p".to_string(),
            local: PathBuf::from("/tmp/active"),
            final_remote: "/active".to_string(),
            allow_replace: false,
        },
    );
    mgr.insert(Transfer {
        id: "active".to_string(),
        kind: TransferKind::Upload,
        source: "/tmp/active".to_string(),
        destination: "/active".to_string(),
        size: 10,
        transferred: 1,
        status: TransferStatus::Transferring,
        error: None,
        retry_attempt: None,
        delta: None,
        started_at: 1,
    })
    .await;

    for index in 0..=TRANSFER_LEDGER_KEEP {
        let id = format!("done-{index:04}");
        mgr.persisted_retry.lock().await.insert(
            id.clone(),
            PersistedRetryInfo::Download {
                profile_id: "p".to_string(),
                remote_path: format!("/remote/{index}"),
                final_path: PathBuf::from(format!("/tmp/{index}")),
                allow_replace: false,
            },
        );
        mgr.insert(Transfer {
            id,
            kind: TransferKind::Download,
            source: format!("/remote/{index}"),
            destination: format!("/tmp/{index}"),
            size: 1,
            transferred: 1,
            status: TransferStatus::Done,
            error: None,
            retry_attempt: None,
            delta: None,
            started_at: index as i64 + 10,
        })
        .await;
    }

    let list = mgr.list().await;
    assert!(list.iter().any(|transfer| transfer.id == "active"));
    assert_eq!(
        list.iter()
            .filter(|transfer| {
                matches!(
                    transfer.status,
                    TransferStatus::Done
                        | TransferStatus::Skipped
                        | TransferStatus::Error
                        | TransferStatus::Canceled
                )
            })
            .count(),
        TRANSFER_LEDGER_KEEP
    );
    assert_eq!(
        db.transfer_ledger_list().unwrap().len(),
        TRANSFER_LEDGER_KEEP + 1
    );
}

// ---------- FIFO admission ----------

#[tokio::test]
async fn fifo_skips_paused_and_pause_all_blocks_everyone() {
    let mgr = TransferManager::new();
    {
        let mut w = mgr.waiting.lock().await;
        w.push_back("a".to_string());
        w.push_back("b".to_string());
        w.push_back("c".to_string());
    }
    mgr.pauses.lock().await.insert("a".into(), PauseGate::new());
    mgr.pauses.lock().await.insert("b".into(), PauseGate::new());
    {
        let w = mgr.waiting.lock().await;
        assert!(mgr.is_my_turn(&w, "a").await);
        assert!(!mgr.is_my_turn(&w, "b").await);
    }
    // A paused front row never head-of-line blocks the queue.
    mgr.pauses.lock().await.get("a").unwrap().set(true);
    {
        let w = mgr.waiting.lock().await;
        assert!(!mgr.is_my_turn(&w, "a").await);
        assert!(mgr.is_my_turn(&w, "b").await);
        assert!(!mgr.is_my_turn(&w, "c").await);
    }
    // Pause-all blocks everyone, runnable or not.
    mgr.pause_all.set(true);
    {
        let w = mgr.waiting.lock().await;
        assert!(!mgr.is_my_turn(&w, "b").await);
    }
}

#[tokio::test]
async fn queue_priority_is_ordered_fifo_within_tiers_and_skips_paused_rows() {
    let mgr = TransferManager::new();
    mgr.waiting
        .lock()
        .await
        .extend(["a", "b", "c", "d", "e"].into_iter().map(str::to_string));
    mgr.change_waiting_priority("b", TransferPriority::High)
        .await
        .unwrap();
    mgr.change_waiting_priority("d", TransferPriority::High)
        .await
        .unwrap();
    mgr.change_waiting_priority("a", TransferPriority::Low)
        .await
        .unwrap();
    mgr.change_waiting_priority("b", TransferPriority::High)
        .await
        .unwrap();

    let state = mgr.queue_state().await;
    assert_eq!(state.waiting, ["b", "d", "c", "e", "a"]);
    assert_eq!(state.priorities["b"], TransferPriority::High);
    assert_eq!(state.priorities["c"], TransferPriority::Normal);
    let gate = PauseGate::new();
    gate.set(true);
    mgr.pauses.lock().await.insert("b".into(), gate);
    let slot = mgr.admit("d").await.unwrap();
    assert!(!mgr.queue_state().await.priorities.contains_key("d"));
    assert_eq!(
        mgr.priority_overrides().get("d"),
        Some(&TransferPriority::High)
    );
    drop(slot);

    mgr.change_waiting_priority("a", TransferPriority::Normal)
        .await
        .unwrap();
    assert_eq!(mgr.queue_state().await.waiting, ["c", "e", "a", "b"]);
    assert!(mgr
        .change_waiting_priority("missing", TransferPriority::High)
        .await
        .is_err());
}

#[tokio::test]
async fn queue_priority_survives_sqlite_restart_and_explicit_retry() {
    let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
    let mgr = TransferManager::with_db(db.clone());
    let id = "priority-recovery";
    mgr.persisted_retry.lock().await.insert(
        id.into(),
        PersistedRetryInfo::Upload {
            profile_id: "saved-profile".into(),
            local: PathBuf::from("/tmp/source.bin"),
            final_remote: "/remote/source.bin".into(),
            allow_replace: false,
        },
    );
    mgr.insert(terminal_race_test_row(id, TransferStatus::Queued))
        .await;
    mgr.waiting.lock().await.push_back(id.into());
    mgr.change_waiting_priority(id, TransferPriority::High)
        .await
        .unwrap();
    let slot = mgr.admit(id).await.unwrap();
    drop(slot);
    mgr.finish_transfer(id, Err(anyhow::anyhow!("network timeout")))
        .await
        .unwrap();
    drop(mgr);

    let restored = TransferManager::with_db(db);
    assert_eq!(
        restored.recovery_profile_id(id).await.as_deref(),
        Some("saved-profile")
    );
    assert!(restored.queue_state().await.waiting.is_empty());
    restored.reset_for_retry(id).await.unwrap();
    restored.waiting.lock().await.push_back(id.into());
    assert_eq!(
        restored.queue_state().await.priorities[id],
        TransferPriority::High
    );
    restored
        .change_waiting_priority(id, TransferPriority::Normal)
        .await
        .unwrap();
    assert_eq!(
        restored.queue_state().await.priorities[id],
        TransferPriority::Normal
    );
}

#[tokio::test]
async fn priority_write_without_recovery_descriptor_fails_without_reordering() {
    let db = Arc::new(crate::db::Db::open_in_memory().unwrap());
    let mgr = TransferManager::with_db(db);
    mgr.insert(terminal_race_test_row("a", TransferStatus::Queued))
        .await;
    mgr.waiting.lock().await.extend(["a".into(), "b".into()]);
    assert!(mgr
        .change_waiting_priority("a", TransferPriority::High)
        .await
        .is_err());
    let state = mgr.queue_state().await;
    assert_eq!(state.waiting, ["a", "b"]);
    assert_eq!(state.priorities["a"], TransferPriority::Normal);
}

#[tokio::test]
async fn weighted_admission_serves_low_priority_under_continuous_high_load() {
    let mgr = TransferManager::new();
    mgr.set_concurrency(1);
    mgr.waiting.lock().await.extend(
        [
            "h1", "h2", "h3", "h4", "h5", "h6", "h7", "h8", "n1", "n2", "n3", "n4", "l1", "l2",
        ]
        .into_iter()
        .map(str::to_string),
    );
    for id in ["h1", "h2", "h3", "h4", "h5", "h6", "h7", "h8"] {
        mgr.change_waiting_priority(id, TransferPriority::High)
            .await
            .unwrap();
    }
    for id in ["l1", "l2"] {
        mgr.change_waiting_priority(id, TransferPriority::Low)
            .await
            .unwrap();
    }
    let expected = [
        "h1", "h2", "h3", "h4", "n1", "n2", "l1", "h5", "h6", "h7", "h8", "n3", "n4", "l2",
    ];
    assert_eq!(mgr.queue_state().await.waiting, expected);
    for id in expected {
        assert_eq!(mgr.queue_state().await.waiting[0], id);
        let slot = tokio::time::timeout(Duration::from_secs(2), mgr.admit(id))
            .await
            .expect("projected queue head must be admitted")
            .unwrap();
        drop(slot);
    }
    assert!(mgr.queue_state().await.waiting.is_empty());
}

#[test]
fn weighted_cycle_skips_empty_tiers_and_never_reserves_a_missing_slot() {
    for cursor in 0..QUEUE_TIER_CYCLE.len() {
        assert_eq!(next_queue_tier(cursor, [false; 3]), None);
        for tier in 0..3 {
            let mut available = [false; 3];
            available[tier] = true;
            let (selected, next) = next_queue_tier(cursor, available).unwrap();
            assert_eq!(selected, tier);
            assert!(next < QUEUE_TIER_CYCLE.len());
        }
    }
}

#[tokio::test]
async fn admission_wakes_next_waiter_before_first_slot_is_released() {
    let mgr = TransferManager::new();
    mgr.set_concurrency(2);
    mgr.waiting.lock().await.extend(["a".into(), "b".into()]);

    let mut second = std::pin::pin!(mgr.admit("b"));
    std::future::poll_fn(|cx| {
        assert!(second.as_mut().poll(cx).is_pending());
        std::task::Poll::Ready(())
    })
    .await;

    let first = mgr.admit("a").await.unwrap();
    let second = tokio::time::timeout(Duration::from_secs(2), second)
        .await
        .expect("next FIFO waiter must wake while the first slot is occupied")
        .unwrap();
    assert_eq!(mgr.semaphore.available_permits(), 0);
    assert!(mgr.waiting.lock().await.is_empty());
    drop((first, second));
}

#[tokio::test]
async fn admission_rechecks_reordered_head_while_slots_are_occupied() {
    let mgr = TransferManager::new();
    mgr.set_concurrency(1);
    let active = mgr.reserve_slot().await.unwrap();
    mgr.waiting.lock().await.extend(["a".into(), "b".into()]);
    let mut first = std::pin::pin!(mgr.admit("a"));
    std::future::poll_fn(|cx| {
        assert!(first.as_mut().poll(cx).is_pending());
        std::task::Poll::Ready(())
    })
    .await;

    mgr.waiting.lock().await.swap(0, 1);
    mgr.queue_gen.send_modify(|g| *g += 1);
    std::future::poll_fn(|cx| {
        assert!(first.as_mut().poll(cx).is_pending());
        std::task::Poll::Ready(())
    })
    .await;
    drop(active);

    let second = tokio::time::timeout(Duration::from_secs(2), mgr.admit("b"))
        .await
        .expect("reordered head must not wait behind obsolete semaphore waiter")
        .unwrap();
    assert_eq!(
        mgr.waiting.lock().await.front().map(String::as_str),
        Some("a")
    );
    drop(second);
    assert!(tokio::time::timeout(Duration::from_secs(2), first)
        .await
        .unwrap()
        .is_some());
}

#[tokio::test]
async fn admission_rejects_missing_id_when_every_waiting_row_is_paused() {
    let mgr = TransferManager::new();
    mgr.waiting.lock().await.push_back("a".into());
    let gate = PauseGate::new();
    gate.set(true);
    mgr.pauses.lock().await.insert("a".into(), gate);
    let waiting = mgr.waiting.lock().await;
    assert!(!mgr.is_my_turn(&waiting, "missing").await);
}

#[tokio::test]
async fn concurrency_resize_cancels_stale_shrink_debt() {
    let mgr = TransferManager::new();
    mgr.set_concurrency(4);
    let a = mgr.reserve_slot().await.unwrap();
    let b = mgr.reserve_slot().await.unwrap();
    let c = mgr.reserve_slot().await.unwrap();
    assert_eq!(mgr.semaphore.available_permits(), 1);

    mgr.set_concurrency(1);
    assert_eq!(mgr.semaphore.available_permits(), 0);
    assert_eq!(mgr.concurrency_state.lock().unwrap().pending_shrink, 2);

    // The old spawned acquire_many could wake AFTER this increase and
    // consume new permits permanently, silently reducing throughput.
    mgr.set_concurrency(4);
    assert_eq!(mgr.concurrency_state.lock().unwrap().pending_shrink, 0);
    assert_eq!(mgr.semaphore.available_permits(), 1);
    drop(a);
    drop(b);
    drop(c);
    assert_eq!(mgr.semaphore.available_permits(), 4);
}

#[tokio::test]
async fn concurrency_shrink_retires_running_slots_at_all_supported_limits() {
    for limit in [1usize, 4, 8, 16, 32] {
        let mgr = TransferManager::new();
        mgr.set_concurrency(limit);
        let mut slots = Vec::new();
        for _ in 0..limit {
            slots.push(mgr.reserve_slot().await.unwrap());
        }
        mgr.set_concurrency(1);
        assert_eq!(mgr.semaphore.available_permits(), 0);
        drop(slots);
        assert_eq!(mgr.semaphore.available_permits(), 1, "limit {limit}");
        assert_eq!(mgr.concurrency_state.lock().unwrap().pending_shrink, 0);
        mgr.set_concurrency(limit);
        assert_eq!(mgr.semaphore.available_permits(), limit);
    }
}

#[tokio::test(start_paused = true)]
async fn concurrency_grows_and_shrinks() {
    let mgr = TransferManager::new();
    assert_eq!(mgr.semaphore.available_permits(), DEFAULT_CONCURRENCY);
    mgr.set_concurrency(5);
    assert_eq!(mgr.semaphore.available_permits(), 5);
    mgr.set_concurrency(2);
    for _ in 0..10 {
        tokio::task::yield_now().await;
    }
    assert_eq!(mgr.semaphore.available_permits(), 2);
}

// ---------- Retry classification ----------

#[test]
fn retry_backoff_is_exponential_jittered_and_bounded() {
    for (attempt, floor, ceiling) in [(1, 5, 8), (2, 10, 16), (3, 21, 32), (4, 40, 60)] {
        assert_eq!(retry_delay_seconds(attempt, 0), floor);
        assert_eq!(retry_delay_seconds(attempt, ceiling - floor), ceiling);
        for entropy in 0..128 {
            assert!((floor..=ceiling).contains(&retry_delay_seconds(attempt, entropy)));
        }
    }
    for attempt in [5, 8, u32::MAX] {
        assert_eq!(retry_delay_seconds(attempt, 0), 40);
        assert_eq!(retry_delay_seconds(attempt, 20), 60);
        assert!((40..=60).contains(&retry_delay_seconds(attempt, u64::MAX)));
    }
    assert_eq!(retry_delay_seconds(0, 0), retry_delay_seconds(1, 0));
}

#[test]
fn transient_errors_retry_permanent_ones_dont() {
    assert!(is_transient(&anyhow::anyhow!("connection reset by peer")));
    assert!(is_transient(&anyhow::anyhow!("operation timed out")));
    assert!(!is_transient(&anyhow::anyhow!("authentication failed")));
    assert!(!is_transient(&anyhow::anyhow!(
        "Permission denied (os error 13)"
    )));
    assert!(!is_transient(&anyhow::anyhow!("No such file or directory")));
}

#[test]
fn ambiguous_promotion_failure_must_not_be_automatically_replayed() {
    let network = anyhow::anyhow!("connection reset by peer");
    assert!(is_safe_staged_upload_retry(&network));
    let unknown_commit = anyhow::Error::new(UploadPromotionUnconfirmed(anyhow::anyhow!(
        "connection reset by peer"
    )));
    assert!(!is_safe_staged_upload_retry(&unknown_commit));
    assert_eq!(
        safe_transfer_failure(&unknown_commit),
        "Remote upload completion is uncertain. Verify the destination before retrying."
    );
}

#[test]
fn rate_limited_downloads_retry_but_mutating_cloud_uploads_do_not_replay() {
    for code in [408, 425, 429, 500, 502, 503, 504] {
        let error = anyhow::anyhow!("download failed: HTTP {code}");
        assert!(is_transient_download(&error), "{code}");
        assert!(
            !is_transient(&error),
            "upload must not blindly replay {code}"
        );
    }
    for code in [400, 401, 403, 404, 409, 412] {
        let error = anyhow::anyhow!("download failed: HTTP {code}");
        assert!(!is_transient_download(&error), "{code}");
    }
    assert!(is_transient_download(&anyhow::anyhow!(
        "connection reset by peer"
    )));
}

#[test]
fn directory_upload_metadata_failure_is_not_silently_skipped() {
    let path = PathBuf::from("/source/private.txt");
    let error = require_directory_upload_metadata(
        &path,
        Err(std::io::Error::new(
            std::io::ErrorKind::PermissionDenied,
            "permission denied",
        )),
    )
    .unwrap_err();

    let message = format!("{error:#}");
    assert!(message.contains("stat directory upload entry"));
    assert!(message.contains("private.txt"));
    assert!(message.contains("permission denied"));
    assert!(message.contains("incomplete source tree"));
}

#[test]
fn remote_join_follows_the_destination_style() {
    // POSIX destinations keep forward slashes, with or without a trailing one.
    assert_eq!(join_remote("/var/www", "a.txt"), "/var/www/a.txt");
    assert_eq!(join_remote("/var/www/", "a.txt"), "/var/www/a.txt");
    // A Windows agent target keeps backslashes instead of going mixed.
    assert_eq!(join_remote(r"C:\srv", "a.txt"), r"C:\srv\a.txt");
    assert_eq!(join_remote(r"C:\srv\", "a.txt"), r"C:\srv\a.txt");
    // Forward-slashed Windows paths are already unambiguous — leave them be.
    assert_eq!(join_remote("C:/srv", "a.txt"), "C:/srv/a.txt");
    assert_eq!(join_remote("", "a.txt"), "a.txt");
}

#[test]
fn remote_rename_candidates_respect_hidden_files_and_path_style() {
    assert_eq!(
        remote_rename_candidate("/srv/archive.tar.gz", 1),
        "/srv/archive.tar_1.gz"
    );
    assert_eq!(remote_rename_candidate("/srv/.env", 1), "/srv/.env_1");
    assert_eq!(remote_rename_candidate(".env", 2), ".env_2");
    assert_eq!(
        remote_rename_candidate(r"C:\dir.v1\file.txt", 3),
        r"C:\dir.v1\file_3.txt"
    );
    assert_eq!(
        remote_rename_candidate(r"C:\dir.v1\file", 4),
        r"C:\dir.v1\file_4"
    );
}

#[tokio::test]
async fn remote_rename_returns_original_only_after_proven_absence() {
    let mut probes = Vec::new();
    let resolved = resolve_remote_rename_with_probe("/srv/file.txt", |candidate| {
        probes.push(candidate);
        std::future::ready(Ok::<bool, std::io::Error>(false))
    })
    .await
    .unwrap();

    assert_eq!(resolved, "/srv/file.txt");
    assert_eq!(probes, vec!["/srv/file.txt"]);
}

#[tokio::test]
async fn remote_rename_skips_collisions_until_a_proven_free_candidate() {
    let mut probes = Vec::new();
    let resolved = resolve_remote_rename_with_probe("/srv/file.txt", |candidate| {
        let exists = matches!(
            candidate.as_str(),
            "/srv/file.txt" | "/srv/file_1.txt" | "/srv/file_2.txt"
        );
        probes.push(candidate);
        std::future::ready(Ok::<bool, std::io::Error>(exists))
    })
    .await
    .unwrap();

    assert_eq!(resolved, "/srv/file_3.txt");
    assert_eq!(
        probes,
        vec![
            "/srv/file.txt",
            "/srv/file_1.txt",
            "/srv/file_2.txt",
            "/srv/file_3.txt"
        ]
    );
}

#[tokio::test]
async fn remote_rename_probe_failure_is_fail_closed() {
    let mut probes = 0usize;
    let error = resolve_remote_rename_with_probe("/srv/file.txt", |_candidate| {
        probes += 1;
        std::future::ready(Err::<bool, std::io::Error>(std::io::Error::new(
            std::io::ErrorKind::PermissionDenied,
            "permission denied",
        )))
    })
    .await
    .unwrap_err();

    assert_eq!(probes, 1);
    assert!(
        error
            .to_string()
            .contains("unable to verify remote rename target"),
        "unexpected probe error: {error:#}"
    );
}

#[tokio::test]
async fn remote_rename_candidate_probe_failure_is_fail_closed() {
    let mut probes = 0usize;
    let error = resolve_remote_rename_with_probe("/srv/file.txt", |_candidate| {
        probes += 1;
        let result = if probes == 1 {
            Ok(true)
        } else {
            Err(std::io::Error::new(
                std::io::ErrorKind::ConnectionReset,
                "connection reset",
            ))
        };
        std::future::ready(result)
    })
    .await
    .unwrap_err();

    assert_eq!(probes, 2);
    assert!(
        error
            .to_string()
            .contains("unable to verify remote rename candidate /srv/file_1.txt"),
        "unexpected candidate probe error: {error:#}"
    );
}

#[test]
fn adversarial_remote_filenames_cannot_create_windows_devices_or_ads() {
    for name in [
        "CON",
        "con.txt",
        "NUL.log",
        "LPT1",
        "COM9.txt",
        "COM¹.txt",
        "CONIN$",
        "report.txt:payload",
        "example.",
        "example ",
        "folder\\file",
        "file\nnext",
        "file\u{0001}suffix",
        "photo?.jpg",
        "invoice|2026.pdf",
    ] {
        assert!(
            !safe_local_download_leaf(name),
            "unsafe filename accepted: {name:?}"
        );
        assert!(
            checked_remote_download_relative("/downloads", &format!("/downloads/{name}")).is_err(),
            "unsafe recursive download accepted: {name:?}"
        );
    }
    // A slash separates legitimate nested directory components in a
    // recursive download. It is not an unsafe filename component.
    assert!(!safe_local_download_leaf("sub/path"));
    assert_eq!(
        checked_remote_download_relative("/downloads", "/downloads/sub/path").unwrap(),
        PathBuf::from("sub").join("path")
    );
    for name in [
        "report.txt",
        "račun 2026.pdf",
        ".env",
        "COM10.txt",
        "convention.md",
    ] {
        assert!(
            safe_local_download_leaf(name),
            "valid filename rejected: {name:?}"
        );
    }
}

#[test]
fn local_rename_exhaustion_fails_instead_of_overwriting() {
    let dir = std::env::temp_dir().join(format!(
        "ghostftp-rename-exhaustion-{}-{}",
        std::process::id(),
        Uuid::new_v4().simple()
    ));
    std::fs::create_dir_all(&dir).unwrap();

    let original = dir.join("collision.txt");
    std::fs::write(&original, b"original").unwrap();
    for i in 1..=MAX_RENAME_CANDIDATES {
        std::fs::write(dir.join(format!("collision_{i}.txt")), b"occupied").unwrap();
    }

    let error = resolve_local_rename(&original).unwrap_err();
    assert!(
        error.to_string().contains("no free rename target"),
        "unexpected exhaustion error: {error:#}"
    );
    assert_eq!(std::fs::read(&original).unwrap(), b"original");

    std::fs::remove_dir_all(dir).unwrap();
}

// ---------- Delta sync transfer paths ----------

/// Deterministic pseudo-random bytes (xorshift64*), so tests need no
/// fixtures or extra dev-deps.
fn det_bytes(seed: u64, n: usize) -> Vec<u8> {
    let mut x = seed;
    let mut out = Vec::with_capacity(n);
    while out.len() < n {
        x ^= x >> 12;
        x ^= x << 25;
        x ^= x >> 27;
        out.extend_from_slice(&x.wrapping_mul(0x2545_F491_4F6C_DD1D).to_le_bytes());
    }
    out.truncate(n);
    out
}

fn test_dirs(tag: &str) -> PathBuf {
    static COUNTER: AtomicUsize = AtomicUsize::new(0);
    let dir = std::env::temp_dir().join(format!(
        "ghostftp-transfer-delta-{tag}-{}-{}",
        std::process::id(),
        COUNTER.fetch_add(1, Ordering::SeqCst)
    ));
    std::fs::create_dir_all(dir.join("local")).unwrap();
    std::fs::create_dir_all(dir.join("remote")).unwrap();
    dir
}

/// An AgentSession wired over localhost TCP to an in-process daemon
/// request loop (`ghostftp_agentd::ops::handle`, writes allowed). With
/// `reject_signature`, `Signature` requests error out — simulating a
/// pre-delta-sync daemon for the fallback tests.
async fn delta_test_session(reject_signature: bool) -> Arc<crate::session::AgentSession> {
    use ghostftp_agent_proto::identity::Identity;
    use ghostftp_agent_proto::msg::{Hello, Request, Response, PROTOCOL_VERSION};
    use ghostftp_agent_proto::{Auth, Role, SecureChannel};

    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    let daemon_id = Identity::generate().unwrap();
    let daemon_pub = daemon_id.public_bytes().unwrap();
    tokio::spawn(async move {
        loop {
            let Ok((stream, _)) = listener.accept().await else {
                break;
            };
            let sk = match daemon_id.private_bytes() {
                Ok(sk) => sk,
                Err(_) => break,
            };
            tokio::spawn(async move {
                let jobs = ghostftp_agentd::jobs::JobStore::new();
                let policy = ghostftp_agentd::Policy {
                    allow_exec: false,
                    allow_write: true,
                };
                let mut ch = SecureChannel::establish(
                    stream,
                    Role::Responder,
                    &sk,
                    Auth::Paired {
                        expect_remote: None,
                    },
                )
                .await?;
                let _hello: Hello = ch.recv().await?;
                loop {
                    let req: Request = match ch.recv().await {
                        Ok(r) => r,
                        Err(_) => break,
                    };
                    let resp = if reject_signature && matches!(req, Request::Signature { .. }) {
                        Response::error("unknown op")
                    } else {
                        ghostftp_agentd::ops::handle(req, policy, &jobs).await
                    };
                    if ch.send(&resp).await.is_err() {
                        break;
                    }
                }
                Ok::<(), anyhow::Error>(())
            });
        }
    });

    let ctrl_id = Identity::generate().unwrap();
    let stream = tokio::net::TcpStream::connect(addr).await.unwrap();
    let mut ch = SecureChannel::establish(
        stream,
        Role::Initiator,
        &ctrl_id.private_bytes().unwrap(),
        Auth::Paired {
            expect_remote: Some(daemon_pub),
        },
    )
    .await
    .unwrap();
    ch.send(&Hello {
        protocol_version: PROTOCOL_VERSION,
        client_name: "transfer-test".into(),
    })
    .await
    .unwrap();
    Arc::new(crate::session::AgentSession::for_test(ch))
}

/// A manager with one transfer row registered, so the delta arms' update /
/// checkpoint / emit calls have a row to work on.
async fn delta_test_manager(
    id: &str,
    kind: TransferKind,
    source: &str,
    destination: &str,
    size: u64,
) -> TransferManager {
    let mgr = TransferManager::new();
    mgr.insert(Transfer {
        id: id.to_string(),
        kind,
        source: source.to_string(),
        destination: destination.to_string(),
        size,
        transferred: 0,
        status: TransferStatus::Queued,
        error: None,
        retry_attempt: None,
        delta: None,
        started_at: 0,
    })
    .await;
    mgr
}

// ---------- Delta sync setting switch + cross-backend gate ----------

/// The `deltaSync` setting drives the switch; `GHOSTFTP_DELTA=0` force-off
/// wins over it either way.
#[test]
fn delta_switch_follows_the_setting() {
    let mgr = TransferManager::new();
    assert!(mgr.delta_enabled(), "default on");
    mgr.set_delta_enabled(false);
    assert!(!mgr.delta_enabled());
    mgr.set_delta_enabled(true);
    let env_off = std::env::var("GHOSTFTP_DELTA").ok().as_deref() == Some("0");
    assert_eq!(mgr.delta_enabled(), !env_off);
}

fn delta_gate_profile(protocol: &str) -> crate::profiles::ConnectionProfile {
    crate::profiles::ConnectionProfile {
        id: "t".into(),
        name: "t".into(),
        protocol: protocol.into(),
        host: "127.0.0.1".into(),
        port: 22,
        username: "u".into(),
        auth: crate::profiles::AuthMethod::Agent,
        default_remote_path: None,
        description: None,
        color: None,
        auto_connect: None,
        bucket: None,
        region: None,
        endpoint: None,
        account: None,
        agent_key: None,
        group: None,
        favorite: None,
        bookmarked: None,
        tags: None,
        last_used: None,
        sort_order: None,
        icon: None,
        jump_host: None,
        jump_port: None,
        jump_username: None,
        keep_alive_seconds: None,
        reconnect_attempts: None,
    }
}

/// Cross-backend contract: `supports_delta` is true ONLY for
/// `Session::Agent` — every other backend must take the whole-file path.
/// One table row per variant constructible without a live connection;
/// Ssh/Ftp (live sockets) and the OAuth/API-token cloud sessions (private
/// connect-time state) can't be built in a unit test — for those the
/// exhaustive `matches!` in `supports_delta` is the compile-time
/// guarantee, and dispatch routes them to plain `run_*` arms.
#[tokio::test]
async fn supports_delta_only_for_agent_sessions() {
    use crate::session::http::{HttpAuth, HttpMode};
    use crate::session::webdav::WebdavAuth;
    use crate::session::{HttpSession, ObjectSession, WebdavSession};

    let client = reqwest::Client::new();
    let base = url::Url::parse("https://example.invalid/dav/").unwrap();
    let table: Vec<(&str, Session, bool)> = vec![
        (
            "webdav",
            Session::Webdav(Arc::new(WebdavSession {
                id: "t".into(),
                profile: delta_gate_profile("webdav"),
                client: client.clone(),
                base: base.clone(),
                auth: WebdavAuth::None,
            })),
            false,
        ),
        (
            "http",
            Session::Http(Arc::new(HttpSession {
                id: "t".into(),
                profile: delta_gate_profile("http"),
                client,
                base,
                auth: HttpAuth::None,
                mode: HttpMode::Listing,
            })),
            false,
        ),
        (
            "object",
            Session::Object(Arc::new(ObjectSession {
                id: "t".into(),
                profile: delta_gate_profile("s3"),
                container: "b".into(),
                store: Arc::new(object_store::memory::InMemory::new()),
            })),
            false,
        ),
        (
            "agent",
            Session::Agent(delta_test_session(false).await),
            true,
        ),
    ];
    for (name, session, expected) in &table {
        assert_eq!(supports_delta(session), *expected, "{name} backend");
    }
}

#[tokio::test]
async fn agent_download_resumes_from_committed_offset() {
    let dir = test_dirs("resume-down");
    let local = dir.join("local/file.bin");
    let remote = dir.join("remote/file.bin");
    let remote_s = remote.to_string_lossy().into_owned();
    let content = det_bytes(0xA11CE, 768 * 1024);
    let offset = 192 * 1024;

    std::fs::write(&remote, &content).unwrap();
    std::fs::write(&local, &content[..offset]).unwrap();

    let session = delta_test_session(false).await;
    let mgr = delta_test_manager(
        "resume-down",
        TransferKind::Download,
        &remote_s,
        &local.to_string_lossy(),
        content.len() as u64,
    )
    .await;
    mgr.update("resume-down", |t| t.transferred = offset as u64)
        .await;

    // Drive the same delta-aware entry point production dispatch uses. A
    // non-zero committed offset must bypass delta and continue ReadChunk
    // requests from the verified local prefix.
    mgr.agent_download_with_delta_core("resume-down", &session, &remote_s, &local, None)
        .await
        .unwrap();

    assert_eq!(std::fs::read(&local).unwrap(), content);
    assert_eq!(
        mgr.get("resume-down").await.unwrap().transferred,
        content.len() as u64
    );
    let _ = std::fs::remove_dir_all(&dir);
}

#[tokio::test]
async fn agent_upload_resumes_from_committed_offset() {
    let dir = test_dirs("resume-up");
    let local = dir.join("local/file.bin");
    let remote = dir.join("remote/file.bin");
    let remote_s = remote.to_string_lossy().into_owned();
    let content = det_bytes(0xBEE5, 768 * 1024);
    let offset = 192 * 1024;

    std::fs::write(&local, &content).unwrap();
    std::fs::write(&remote, &content[..offset]).unwrap();

    let session = delta_test_session(false).await;
    let mgr = delta_test_manager(
        "resume-up",
        TransferKind::Upload,
        &local.to_string_lossy(),
        &remote_s,
        content.len() as u64,
    )
    .await;
    mgr.update("resume-up", |t| t.transferred = offset as u64)
        .await;

    // Production dispatch is delta-aware, but an interrupted whole-file
    // upload must keep its exact prefix and continue WriteChunk at offset.
    mgr.agent_upload_with_delta_core("resume-up", &session, &local, &remote_s, None)
        .await
        .unwrap();

    assert_eq!(std::fs::read(&remote).unwrap(), content);
    assert_eq!(
        mgr.get("resume-up").await.unwrap().transferred,
        content.len() as u64
    );
    let _ = std::fs::remove_dir_all(&dir);
}

/// Delta upload: 20 MiB up whole-file, mutate 1 KiB, up again — the second
/// run must reassemble a byte-equal remote file with < 10% of the bytes
/// crossing the wire (the patch is the ONLY WriteChunk traffic).
#[tokio::test]
async fn delta_upload_reuses_unchanged_blocks() {
    let dir = test_dirs("up");
    let local = dir.join("local/file.bin");
    let remote = dir.join("remote/file.bin");
    let remote_s = remote.to_string_lossy().into_owned();
    let size = 20 * 1024 * 1024;
    std::fs::write(&local, det_bytes(0xAAAA, size)).unwrap();

    let session = delta_test_session(false).await;
    // First upload: no remote basis → whole-file.
    let mgr = delta_test_manager(
        "t",
        TransferKind::Upload,
        &local.to_string_lossy(),
        &remote_s,
        size as u64,
    )
    .await;
    mgr.agent_upload_with_delta_core("t", &session, &local, &remote_s, None)
        .await
        .unwrap();
    assert_eq!(
        std::fs::read(&remote).unwrap(),
        std::fs::read(&local).unwrap()
    );
    assert!(
        mgr.get("t").await.unwrap().delta.is_none(),
        "no basis → whole-file"
    );

    // Mutate 1 KiB in the middle and upload again → delta.
    let mut content = std::fs::read(&local).unwrap();
    content[10 * 1024 * 1024..10 * 1024 * 1024 + 1024].copy_from_slice(&det_bytes(0xBBBB, 1024));
    std::fs::write(&local, &content).unwrap();
    mgr.update("t", |t| t.transferred = 0).await;
    mgr.agent_upload_with_delta_core("t", &session, &local, &remote_s, None)
        .await
        .unwrap();

    assert_eq!(std::fs::read(&remote).unwrap(), content);
    let t = mgr.get("t").await.unwrap();
    let stats = t.delta.expect("second upload must run as a delta");
    assert!(
        stats.sent * 10 < size as u64,
        "1 KiB edit sent {} bytes (>= 10% of {size})",
        stats.sent
    );
    assert_eq!(stats.sent + stats.reused, size as u64);
    // No temp litter on either side.
    for side in ["local", "remote"] {
        let litter: Vec<_> = std::fs::read_dir(dir.join(side))
            .unwrap()
            .filter_map(|e| e.ok())
            .filter(|e| {
                let n = e.file_name().to_string_lossy().into_owned();
                n.contains(".ghostftp-patch-")
                    || n.contains(".ghostftp-new-")
                    || n.contains(".ghostftp-delta-")
            })
            .collect();
        assert!(litter.is_empty(), "{side} litter: {litter:?}");
    }
    let _ = std::fs::remove_dir_all(&dir);
}

/// Delta download: same shape as the upload test, other direction.
#[tokio::test]
async fn delta_download_reuses_unchanged_blocks() {
    let dir = test_dirs("down");
    let local = dir.join("local/file.bin");
    let remote = dir.join("remote/file.bin");
    let remote_s = remote.to_string_lossy().into_owned();
    let size = 20 * 1024 * 1024;
    std::fs::write(&remote, det_bytes(0xCCCC, size)).unwrap();

    let session = delta_test_session(false).await;
    // First download: no local basis → whole-file.
    let mgr = delta_test_manager(
        "t",
        TransferKind::Download,
        &remote_s,
        &local.to_string_lossy(),
        size as u64,
    )
    .await;
    mgr.agent_download_with_delta_core("t", &session, &remote_s, &local, None)
        .await
        .unwrap();
    assert_eq!(
        std::fs::read(&local).unwrap(),
        std::fs::read(&remote).unwrap()
    );
    assert!(mgr.get("t").await.unwrap().delta.is_none());

    // Mutate 1 KiB remotely and download again → delta.
    let mut content = std::fs::read(&remote).unwrap();
    content[5 * 1024 * 1024..5 * 1024 * 1024 + 1024].copy_from_slice(&det_bytes(0xDDDD, 1024));
    std::fs::write(&remote, &content).unwrap();
    mgr.update("t", |t| t.transferred = 0).await;
    mgr.agent_download_with_delta_core("t", &session, &remote_s, &local, None)
        .await
        .unwrap();

    assert_eq!(std::fs::read(&local).unwrap(), content);
    let t = mgr.get("t").await.unwrap();
    let stats = t.delta.expect("second download must run as a delta");
    assert!(
        stats.sent * 10 < size as u64,
        "1 KiB edit fetched {} bytes (>= 10% of {size})",
        stats.sent
    );
    assert_eq!(stats.sent + stats.reused, size as u64);
    let _ = std::fs::remove_dir_all(&dir);
}

/// Old daemon (Signature unsupported) ⇒ the delta attempt fails and the
/// wrapper silently completes a correct whole-file copy, both directions.
#[tokio::test]
async fn delta_falls_back_when_daemon_lacks_signature() {
    let dir = test_dirs("old");
    let local = dir.join("local/file.bin");
    let remote = dir.join("remote/file.bin");
    let remote_s = remote.to_string_lossy().into_owned();
    let size = 20 * 1024 * 1024;
    std::fs::write(&remote, det_bytes(0xEEEE, size)).unwrap();
    std::fs::write(&local, std::fs::read(&remote).unwrap()).unwrap();

    let session = delta_test_session(true).await; // Signature → error
                                                  // Upload direction: remote basis exists and file is big enough, so the
                                                  // delta is attempted, fails at Signature, and falls back.
    let mut content = std::fs::read(&local).unwrap();
    content[1024..2048].copy_from_slice(&det_bytes(0xFFFF, 1024));
    std::fs::write(&local, &content).unwrap();
    let mgr = delta_test_manager(
        "u",
        TransferKind::Upload,
        &local.to_string_lossy(),
        &remote_s,
        size as u64,
    )
    .await;
    mgr.agent_upload_with_delta_core("u", &session, &local, &remote_s, None)
        .await
        .unwrap();
    assert_eq!(std::fs::read(&remote).unwrap(), content);
    assert!(
        mgr.get("u").await.unwrap().delta.is_none(),
        "fallback → no delta stats"
    );

    // Download direction.
    content[4096..5120].copy_from_slice(&det_bytes(0x1234, 1024));
    std::fs::write(&remote, &content).unwrap();
    let mgr = delta_test_manager(
        "d",
        TransferKind::Download,
        &remote_s,
        &local.to_string_lossy(),
        size as u64,
    )
    .await;
    mgr.agent_download_with_delta_core("d", &session, &remote_s, &local, None)
        .await
        .unwrap();
    assert_eq!(std::fs::read(&local).unwrap(), content);
    assert!(mgr.get("d").await.unwrap().delta.is_none());
    let _ = std::fs::remove_dir_all(&dir);
}

/// ≥ 60% literal ⇒ the worthwhile heuristic aborts the delta before any
/// patch byte is sent and the wrapper falls back to a correct whole-file
/// copy.
#[tokio::test]
async fn delta_falls_back_when_mostly_changed() {
    let dir = test_dirs("churn");
    let local = dir.join("local/file.bin");
    let remote = dir.join("remote/file.bin");
    let remote_s = remote.to_string_lossy().into_owned();
    let size = 20 * 1024 * 1024;
    std::fs::write(&remote, det_bytes(0x7777, size)).unwrap();
    // Same size, ~completely different content ⇒ ~100% literal.
    std::fs::write(&local, det_bytes(0x8888, size)).unwrap();

    let session = delta_test_session(false).await;
    let mgr = delta_test_manager(
        "u",
        TransferKind::Upload,
        &local.to_string_lossy(),
        &remote_s,
        size as u64,
    )
    .await;
    mgr.agent_upload_with_delta_core("u", &session, &local, &remote_s, None)
        .await
        .unwrap();
    assert_eq!(
        std::fs::read(&remote).unwrap(),
        std::fs::read(&local).unwrap()
    );
    assert!(
        mgr.get("u").await.unwrap().delta.is_none(),
        "≥60% literal → whole-file"
    );

    // Download direction, same setup reversed.
    std::fs::write(&remote, det_bytes(0x9999, size)).unwrap();
    let mgr = delta_test_manager(
        "d",
        TransferKind::Download,
        &remote_s,
        &local.to_string_lossy(),
        size as u64,
    )
    .await;
    mgr.agent_download_with_delta_core("d", &session, &remote_s, &local, None)
        .await
        .unwrap();
    assert_eq!(
        std::fs::read(&local).unwrap(),
        std::fs::read(&remote).unwrap()
    );
    assert!(mgr.get("d").await.unwrap().delta.is_none());
    let _ = std::fs::remove_dir_all(&dir);
}
