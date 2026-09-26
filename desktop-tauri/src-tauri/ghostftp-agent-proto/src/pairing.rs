//! Pairing-code handling.
//!
//! Pairing binds a short, human-transcribable code to the Noise handshake so a
//! first-time controller can prove it's talking to the machine whose owner read
//! the code aloud — without either side having pinned the other yet. The code is
//! turned into a 32-byte PSK mixed into `Noise_XXpsk3`; an active
//! man-in-the-middle produces a different handshake and the PSK check fails, so
//! it can't complete the handshake even though it can see the (encrypted) bytes.

use rand::Rng;
use sha2::{Digest, Sha256};

/// Domain-separation label so this PSK derivation can't collide with any other
/// use of the code.
const PSK_LABEL: &[u8] = b"ghostftp-agent-pairing-v1";

/// Number of decimal digits in a pairing code.
pub const CODE_LEN: usize = 6;

/// Generate a fresh random `CODE_LEN`-digit pairing code (zero-padded), e.g.
/// `"042137"`. Displayed by the daemon during a pairing window.
pub fn generate_code() -> String {
    let mut rng = rand::thread_rng();
    let n: u32 = rng.gen_range(0..1_000_000);
    format!("{n:0width$}", width = CODE_LEN)
}

/// Derive the 32-byte Noise PSK from a pairing code. Both sides run this over
/// the identical (trimmed) code string; a mismatch means neither the same code
/// nor the same channel, and the handshake aborts.
pub fn psk_from_code(code: &str) -> [u8; 32] {
    let mut hasher = Sha256::new();
    hasher.update(PSK_LABEL);
    hasher.update(code.trim().as_bytes());
    hasher.finalize().into()
}

/// True if `code` is a syntactically valid pairing code (exactly `CODE_LEN`
/// ASCII digits). Cheap guard before attempting a handshake.
pub fn is_valid_code(code: &str) -> bool {
    let c = code.trim();
    c.len() == CODE_LEN && c.bytes().all(|b| b.is_ascii_digit())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn generated_codes_are_valid() {
        for _ in 0..100 {
            let c = generate_code();
            assert!(is_valid_code(&c), "invalid code generated: {c}");
        }
    }

    #[test]
    fn psk_is_deterministic_and_code_sensitive() {
        assert_eq!(psk_from_code("123456"), psk_from_code(" 123456 "));
        assert_ne!(psk_from_code("123456"), psk_from_code("123457"));
    }

    #[test]
    fn validates_shape() {
        assert!(is_valid_code("000000"));
        assert!(!is_valid_code("12345"));
        assert!(!is_valid_code("1234567"));
        assert!(!is_valid_code("12a456"));
    }
}
