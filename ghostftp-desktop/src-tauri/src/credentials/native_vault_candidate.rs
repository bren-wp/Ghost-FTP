//! Disposable native test adapter. No production runtime selector.
use anyhow::{anyhow, ensure, Result};
use std::cell::RefCell;
use std::ffi::{c_char, c_int, c_void, CString};
use std::ptr::NonNull;

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

fn call(operation: c_int, service: &str, account: &str, input: &str) -> Result<Option<String>> {
    let service = CString::new(service).map_err(|_| anyhow!("invalid vault service"))?;
    let account = CString::new(account).map_err(|_| anyhow!("invalid vault account"))?;
    let input = CString::new(input).map_err(|_| anyhow!("invalid vault input"))?;
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
        let mut output = [0u8; 8191];
        let mut length = 0;
        // SAFETY: valid C strings and bounded writable output for the whole call;
        // the synchronous bridge joins its cancellation worker before returning.
        let status = unsafe {
            ghostftp_vault_call(
                slot.as_ref().expect("session").0.as_ptr(),
                operation,
                service.as_ptr(),
                account.as_ptr(),
                input.as_ptr(),
                output.as_mut_ptr(),
                output.len(),
                &mut length,
                30_000,
            )
        };
        let result = match status {
            0 if length == 0 => Ok(None),
            1 if operation == 0 && length <= output.len() => {
                String::from_utf8(output[..length].to_vec())
                    .map(Some)
                    .map_err(|_| anyhow!("native vault returned invalid UTF-8"))
            }
            -2 => Err(anyhow!("native vault mutation outcome unknown; reconcile before retry")),
            _ => Err(anyhow!("native vault operation refused or interrupted")),
        };
        output.fill(0);
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
    if matches!(observed, Ok(ref value) if value.as_deref() == expected) {
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
