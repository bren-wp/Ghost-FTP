use crate::profiles::{AuthMethod, ConnectionProfile};
use anyhow::{anyhow, Context, Result};
use object_store::aws::AmazonS3Builder;
use object_store::azure::MicrosoftAzureBuilder;
use object_store::gcp::GoogleCloudStorageBuilder;
use object_store::ObjectStore;
use std::sync::Arc;
use uuid::Uuid;

/// A blob/object-store session. The same struct backs every cloud object
/// store (S3, R2, B2, Azure Blob) — only the builder differs, and the
/// resulting Arc<dyn ObjectStore> hides the implementation from everything
/// downstream. Future protocols (GCS, etc.) drop in with one more arm in
/// `object_connect`.
pub struct ObjectSession {
    pub id: String,
    pub profile: ConnectionProfile,
    /// Bucket (S3) or container (Azure). Surfaced for path-bar display.
    pub container: String,
    pub store: Arc<dyn ObjectStore>,
}

pub async fn object_connect(profile: &ConnectionProfile) -> Result<ObjectSession> {
    match profile.protocol.as_str() {
        "s3" => s3_connect(profile).await,
        "azure" => azure_connect(profile).await,
        "gcs" => gcs_connect(profile).await,
        other => Err(anyhow!("not an object-store protocol: {other}")),
    }
}

async fn s3_connect(profile: &ConnectionProfile) -> Result<ObjectSession> {
    let bucket = profile
        .bucket
        .as_ref()
        .ok_or_else(|| anyhow!("S3 profile is missing the bucket name"))?
        .clone();
    let access_key = profile.username.clone();
    let secret_key = match &profile.auth {
        AuthMethod::Password { password } => password.clone(),
        _ => {
            return Err(anyhow!(
                "S3 backend requires an access key (username) and secret (password). \
                 Switch the auth method to Password."
            ))
        }
    };
    let region = profile
        .region
        .clone()
        .unwrap_or_else(|| "us-east-1".to_string());

    let mut builder = AmazonS3Builder::new()
        .with_bucket_name(&bucket)
        .with_access_key_id(&access_key)
        .with_secret_access_key(&secret_key)
        .with_region(&region);

    if let Some(endpoint) = profile.endpoint.as_ref().filter(|s| !s.is_empty()) {
        builder = builder.with_endpoint(endpoint);
        // R2 / B2 / MinIO expect path-style addressing.
        builder = builder.with_virtual_hosted_style_request(false);
    }

    let store = builder
        .build()
        .with_context(|| format!("constructing S3 client for bucket {bucket}"))?;

    Ok(ObjectSession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        container: bucket,
        store: Arc::new(store),
    })
}

async fn azure_connect(profile: &ConnectionProfile) -> Result<ObjectSession> {
    let container = profile
        .bucket
        .as_ref()
        .ok_or_else(|| anyhow!("Azure profile is missing the container name"))?
        .clone();
    // Account is stored either in the dedicated `account` field or, for
    // back-compat with hand-edited profiles, in `username`.
    let account = profile
        .account
        .clone()
        .filter(|s| !s.is_empty())
        .or_else(|| {
            if profile.username.is_empty() {
                None
            } else {
                Some(profile.username.clone())
            }
        })
        .ok_or_else(|| anyhow!("Azure profile is missing the storage account name"))?;
    let access_key = match &profile.auth {
        AuthMethod::Password { password } => password.clone(),
        _ => {
            return Err(anyhow!(
                "Azure backend requires the storage account key (password field). \
                 Switch the auth method to Password."
            ))
        }
    };

    let mut builder = MicrosoftAzureBuilder::new()
        .with_account(&account)
        .with_container_name(&container)
        .with_access_key(&access_key);

    if let Some(endpoint) = profile.endpoint.as_ref().filter(|s| !s.is_empty()) {
        builder = builder.with_endpoint(endpoint.clone());
    }

    let store = builder
        .build()
        .with_context(|| format!("constructing Azure client for {account}/{container}"))?;

    Ok(ObjectSession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        container,
        store: Arc::new(store),
    })
}

/// Native Google Cloud Storage, the same one-builder move that added Azure:
/// `object_store`'s `gcp` feature does the signing. Credentials come from a
/// service-account key — either a path to the downloaded JSON (auth = Key) or the
/// JSON pasted directly (auth = Password). The bucket rides in `profile.bucket`,
/// exactly like an S3 bucket / Azure container.
async fn gcs_connect(profile: &ConnectionProfile) -> Result<ObjectSession> {
    let bucket = profile
        .bucket
        .as_ref()
        .ok_or_else(|| anyhow!("GCS profile is missing the bucket name"))?
        .clone();

    let mut builder = GoogleCloudStorageBuilder::new().with_bucket_name(&bucket);

    match &profile.auth {
        AuthMethod::Key { path, .. } => {
            builder = builder.with_service_account_path(path.clone());
        }
        AuthMethod::Password { password } if !password.trim().is_empty() => {
            builder = builder.with_service_account_key(password.clone());
        }
        _ => {
            return Err(anyhow!(
                "GCS backend needs a service-account key: either a path to the JSON \
                 key file (Private key file auth) or the pasted JSON (Password auth)."
            ))
        }
    }

    let store = builder
        .build()
        .with_context(|| format!("constructing GCS client for bucket {bucket}"))?;

    Ok(ObjectSession {
        id: Uuid::new_v4().to_string(),
        profile: profile.clone(),
        container: bucket,
        store: Arc::new(store),
    })
}
