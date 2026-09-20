// Strong random password generator for new connections. Uses the Web Crypto
// CSPRNG (available in the Tauri webview) — never Math.random. Ambiguous glyphs
// (l/I/1, o/O/0) are excluded so a generated password is easy to read back, and
// the symbol set is deliberately shell/paste-safe (no quotes, backslash,
// backtick, `$`, or space) since these are typically pasted straight into a
// server's `passwd`/`adduser`.

const LOWER = "abcdefghijkmnpqrstuvwxyz"; // no l, o
const UPPER = "ABCDEFGHJKLMNPQRSTUVWXYZ"; // no I, O
const DIGITS = "23456789"; // no 0, 1
const SYMBOLS = "!@#%^*_-+=?.";

const ALL = LOWER + UPPER + DIGITS + SYMBOLS;

/** Cryptographically-random integer in [0, max) via rejection sampling (no modulo bias). */
function randInt(max: number): number {
  const limit = Math.floor(0xffffffff / max) * max;
  const buf = new Uint32Array(1);
  let x: number;
  do {
    crypto.getRandomValues(buf);
    x = buf[0];
  } while (x >= limit);
  return x % max;
}

function pick(set: string): string {
  return set[randInt(set.length)];
}

/**
 * Generate a strong password of `length` characters, guaranteed to contain at
 * least one lowercase, uppercase, digit, and symbol.
 */
export function generatePassword(length = 20): string {
  const len = Math.max(8, length);
  const chars = [pick(LOWER), pick(UPPER), pick(DIGITS), pick(SYMBOLS)];
  while (chars.length < len) chars.push(pick(ALL));
  // Fisher–Yates shuffle so the four guaranteed characters aren't always first.
  for (let i = chars.length - 1; i > 0; i--) {
    const j = randInt(i + 1);
    [chars[i], chars[j]] = [chars[j], chars[i]];
  }
  return chars.join("");
}
