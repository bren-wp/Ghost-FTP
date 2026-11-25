//! Disposable native test adapter. No production runtime selector.
use anyhow::{anyhow, ensure, Result};
use std::cell::RefCell;
use std::ffi::{c_char, c_int, c_void, CString};
use std::ptr::NonNull;
use zeroize::Zeroizing;

extern "C" {
    fn ghostftp_vault_session_new() -> *mut c_void;
    fn ghostftp_vault_session_free(session: *mut c_void);
    fn ghostftp_vault_call(
        session: *mut c_void,
        operation: c_int,
        service: *const c_char,
        account: *const c_char,
        input: *const c_char,
        output: *mut u8,
        capacity: usize,
        length: *mut usize,
        timeout_ms: u32,
    ) -> c_int;
}

struct Session(NonNull<c_void>);

impl Drop for Session {
    fn drop(&mut self) {
        // SAFETY: uniquely owned session, never transferred between threads.
        unsafe { ghostftp_vault_session_free(self.0.as_ptr()) }
    }
}

thread_local! {
    static STATE: RefCell<Option<Option<Session>>> = const { RefCell::new(None) };
}

pub(in crate::credentials) fn configure_direct_candidate() {
    if std::env::var("GHOSTFTP_DIRECT_VAULT_TEST").as_deref() == Ok("disposable-fixtures") {
        STATE.with(|state| *state.borrow_mut() = Some(None));
    }
}

pub(in crate::credentials) fn clear_direct_candidate() {
    STATE.with(|state| *state.borrow_mut() = None);
}

pub(super) fn enabled() -> bool {
    STATE.with(|state| state.borrow().is_some())
}

// Validate before allocating so a rejected embedded NUL does not create a
// non-erasing CString error buffer. Reserve room for the terminator once.
fn secret_input(input: &str) -> Result<Zeroizing<Vec<u8>>> {
    ensure!(!input.as_bytes().contains(&0), "invalid vault input");
    ensure!(input.len() <= 8191, "invalid secret length");
    let mut bytes = Zeroizing::new(Vec::with_capacity(input.len() + 1));
    bytes.extend_from_slice(input.as_bytes());
    bytes.push(0);
    Ok(bytes)
}

fn call(operation: c_int, service: &str, account: &str, input: &str) -> Result<Option<String>> {
    let service = CString::new(service).map_err(|_| anyhow!("invalid vault service"))?;
    let account = CString::new(account).map_err(|_| anyhow!("invalid vault account"))?;
    let input = secret_input(input)?;
    STATE.with(|state| {
        let mut state = state.borrow_mut();
        let slot = state
            .as_mut()
            .ok_or_else(|| anyhow!("direct vault candidate disabled"))?;
        if slot.is_none() {
            // SAFETY: factory returns an owned opaque session or null.
            let ptr = unsafe { ghostftp_vault_session_new() };
            *slot =
                Some(Session(NonNull::new(ptr).ok_or_else(|| {
                    anyhow!("native vault session unavailable")
                })?));
        }
        let mut output = Zeroizing::new([0u8; 8191]);
        let mut length = 0;
        // SAFETY: valid C strings and bounded writable output for the whole call;
        // the synchronous bridge joins its cancellation worker before returning.
        let status = unsafe {
            ghostftp_vault_call(
                slot.as_ref().expect("session").0.as_ptr(),
                operation,
                service.as_ptr(),
                account.as_ptr(),
                input.as_ptr().cast::<c_char>(),
                output.as_mut_ptr(),
                output.len(),
                &mut length,
                30_000,
            )
        };
        let result = match status {
            0 if length == 0 => Ok(None),
            1 if operation == 0 && length <= output.len() => {
                std::str::from_utf8(&output[..length])
                    .map(|value| Some(value.to_owned()))
                    .map_err(|_| anyhow!("native vault returned invalid UTF-8"))
            }
            -2 => Err(anyhow!(
                "native vault mutation outcome unknown; reconcile before retry"
            )),
            _ => Err(anyhow!("native vault operation refused or interrupted")),
        };
        // RAII wipes the temporary input and output on every normal/unwinding exit.
        // A successful lookup transfers the returned String to the caller.
        if result.is_err() {
            // Never replay an uncertain mutation or fall back to another store.
            *slot = None;
        }
        result
    })
}

pub(super) fn lookup(service: &str, account: &str) -> Result<Option<String>> {
    call(0, service, account, "")
}

pub(super) fn store(service: &str, account: &str, secret: &str) -> Result<()> {
    ensure!(
        !secret.is_empty() && secret.len() <= 8191,
        "invalid secret length"
    );
    call(1, service, account, secret)?;
    verify_mutation(lookup(service, account), Some(secret))
}

pub(super) fn remove(service: &str, account: &str) -> Result<()> {
    call(2, service, account, "")?;
    verify_mutation(lookup(service, account), None)
}

// A failed post-mutation observation is not proof of rollback. Invalidate the
// session without replaying the operation or consulting a different backend.
fn verify_mutation(observed: Result<Option<String>>, expected: Option<&str>) -> Result<()> {
    let matches_expected = match observed {
        Ok(Some(value)) => {
            let value = Zeroizing::new(value);
            Some(value.as_str()) == expected
        }
        Ok(None) => expected.is_none(),
        Err(_) => false,
    };
    if matches_expected {
        return Ok(());
    }
    STATE.with(|state| {
        if let Some(slot) = state.borrow_mut().as_mut() {
            *slot = None;
        }
    });
    Err(anyhow!(
        "native vault mutation outcome unknown; reconcile before retry"
    ))
}

#[test]
fn secret_input_is_bounded_and_nul_terminated() {
    let bytes = secret_input("fixture-ž").unwrap();
    assert_eq!(bytes.as_slice(), "fixture-ž\0".as_bytes());
    assert!(secret_input("fixture\0suffix").is_err());
    assert!(secret_input(&"x".repeat(8192)).is_err());
    assert_eq!(secret_input(&"x".repeat(8191)).unwrap().len(), 8192);
}

#[test]
fn mutation_readback_mismatch_is_not_success() {
    assert!(verify_mutation(Ok(Some("old".into())), Some("new")).is_err());
    assert!(verify_mutation(Err(anyhow!("unavailable")), None).is_err());
    assert!(verify_mutation(Ok(None), None).is_ok());
}

#[test]
fn candidate_requires_explicit_test_selection() {
    clear_direct_candidate();
    assert!(!enabled());
    assert!(lookup("fixture", "fixture").is_err());
}

// Native acceptance uses an independent vault anchor, not a SQLite-derived
// key or an automatic replay path. All accounts are disposable UUID fixtures.
#[test]
#[ignore = "requires isolated unlocked native Secret Service"]
fn journal_native_vault_anchor_reconciliation() {
    use super::credential_mutation_journal::{Journal, Observation};
    use rand::{rngs::OsRng, RngCore};

    assert_eq!(
        std::env::var("GHOSTFTP_DIRECT_VAULT_TEST").as_deref(),
        Ok("disposable-fixtures")
    );
    configure_direct_candidate();
    let root =
        std::env::temp_dir().join(format!("ghostftp-native-intent-{}", uuid::Uuid::new_v4()));
    std::fs::create_dir(&root).unwrap();
    let service = "com.ghostftp.native-vault.journal.fixture";
    let scope = uuid::Uuid::new_v4().to_string();
    let anchor_account = format!("{scope}-anchor");
    let key_account = format!("{scope}-key");
    let mut key = [0u8; 32];
    OsRng.fill_bytes(&mut key);
    use base64::Engine;
    let encoding = base64::engine::general_purpose::STANDARD;
    store(service, &key_account, &encoding.encode(key)).unwrap();
    store(service, &scope, "old").unwrap();
    let mut journal = Journal::open(&root.join("journal.db"), &scope).unwrap();
    let permit = journal
        .prepare(&key, service, &scope, Some("old"), Some("new"), |anchor| {
            store(service, &anchor_account, &encoding.encode(anchor))?;
            let readback =
                lookup(service, &anchor_account)?.ok_or_else(|| anyhow!("anchor unavailable"))?;
            encoding
                .decode(readback)?
                .try_into()
                .map_err(|_| anyhow!("invalid anchor length"))
        })
        .unwrap();
    permit
        .dispatch(|| {
            store(service, &scope, "new")?;
            Err(anyhow!("synthetic lost acknowledgement"))
        })
        .unwrap_err();
    drop(journal);
    clear_direct_candidate();
    configure_direct_candidate();
    let key: [u8; 32] = encoding
        .decode(lookup(service, &key_account).unwrap().unwrap())
        .unwrap()
        .try_into()
        .unwrap();
    let anchor: [u8; 32] = encoding
        .decode(lookup(service, &anchor_account).unwrap().unwrap())
        .unwrap()
        .try_into()
        .unwrap();
    let journal = Journal::open(&root.join("journal.db"), &scope).unwrap();
    assert_eq!(
        journal.reconcile(&key, &anchor, lookup).unwrap(),
        Observation::DesiredObserved
    );
    drop(journal);
    remove(service, &scope).unwrap();
    remove(service, &anchor_account).unwrap();
    remove(service, &key_account).unwrap();
    clear_direct_candidate();
    std::fs::remove_dir_all(root).unwrap();
}
