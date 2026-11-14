//! Fresh, sibling-staged FTP/FTPS/SFTP uploads. Existing remote files must not
//! be truncated by a failed write before the completed staging file is verified.
//! A server may reject rename-over-existing; we fail closed, never delete the
//! previous version to force promotion.
use crate::remotefs;
use crate::session::Session;
use anyhow::{Context, Result};
use std::sync::Arc;
use uuid::Uuid;

pub(super) fn fresh_remote_upload_staging_path(final_remote: &str, id: &str) -> Result<String> {
    // Do not embed an arbitrary user-supplied id or filename in an FTP command.
    let transfer_id = Uuid::parse_str(id).context("upload transfer id is not a UUID")?;
    let (parent, _) = remotefs::remote_destination_parent_name(final_remote)?;
    let staging_name = format!(
        ".ghostftp-upload-{}-{}.part",
        transfer_id.simple(),
        Uuid::new_v4().simple()
    );
    let staging_path = super::join_remote(parent, &staging_name);
    remotefs::validate_protocol_mutation_path(&staging_path)?;
    anyhow::ensure!(
        staging_path != final_remote,
        "upload staging must differ from destination"
    );
    Ok(staging_path)
}

/// FTP has no atomic exclusive STOR creation. A cryptographically random
/// filename plus strict preflight protects ordinary collision cases; SFTP
/// additionally uses CREATE|EXCLUDE when opening the staging file.
pub(super) async fn preflight_ftp_upload_staging(
    session: &crate::session::FtpSession,
    staging: &str,
) -> Result<()> {
    anyhow::ensure!(
        !remotefs::ftp::path_exists_fail_closed(session, staging).await?,
        "temporary remote upload path is occupied"
    );
    Ok(())
}

/// Promote only a fully verified upload. Non-overwrite policy additionally
/// checks that no competing writer occupied the chosen final name since
/// enqueue. FTP cannot make this last check atomic with RNTO; disclose that.
pub(super) async fn promote_remote_upload(
    session: &Arc<Session>,
    staging: &str,
    final_remote: &str,
    allow_replace: bool,
) -> Result<()> {
    remotefs::validate_protocol_mutation_path(staging)?;
    remotefs::validate_protocol_mutation_path(final_remote)?;
    match &**session {
        Session::Ssh(ssh) => {
            let cell = ssh.ensure_sftp().await?;
            let sftp = cell.lock().await;
            if !allow_replace {
                anyhow::ensure!(
                    !sftp.try_exists(final_remote.to_string()).await?,
                    "SFTP upload destination became occupied before promotion"
                );
            }
            sftp.rename(staging, final_remote)
                .await
                .context("SFTP server could not safely promote verified upload")?;
            Ok(())
        }
        Session::Ftp(ftp) => {
            if !allow_replace {
                anyhow::ensure!(
                    !remotefs::ftp::path_exists_fail_closed(ftp, final_remote).await?,
                    "FTP upload destination became occupied before promotion"
                );
            }
            let source = staging.to_string();
            let target = final_remote.to_string();
            ftp.with_stream(move |stream| {
                stream.rename(&source, &target)
                    .context("FTP server could not safely promote verified upload")?;
                Ok(())
            })
            .await
        }
        _ => anyhow::bail!("staged remote upload requires FTP, FTPS or SFTP"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn staged_upload_uses_an_unpredictable_sibling_not_the_user_target() {
        let id = Uuid::new_v4().to_string();
        let a = fresh_remote_upload_staging_path("/sites/public/file.txt", &id).unwrap();
        let b = fresh_remote_upload_staging_path("/sites/public/file.txt", &id).unwrap();
        assert!(a.starts_with("/sites/public/.ghostftp-upload-"));
        assert!(a.ends_with(".part"));
        assert_ne!(a, b);
        assert_ne!(a, "/sites/public/file.txt");
        assert!(!a.contains("file.txt"));
    }

    #[test]
    fn staging_refuses_unsafe_names_and_traversal_before_remote_io() {
        let id = Uuid::new_v4().to_string();
        for target in [
            "/a/../file",
            "/a/unsafe\\name",
            "/a/control\nname",
            "/",
        ] {
            assert!(fresh_remote_upload_staging_path(target, &id).is_err());
        }
        assert!(fresh_remote_upload_staging_path("/safe/file", "../unsafe").is_err());
    }

    #[test]
    fn concurrent_transfers_cannot_share_the_same_staging_name() {
        let id = Uuid::new_v4().to_string();
        let names: std::collections::HashSet<_> = (0..128)
            .map(|_| fresh_remote_upload_staging_path("shared.dat", &id).unwrap())
            .collect();
        assert_eq!(names.len(), 128);
    }
}
