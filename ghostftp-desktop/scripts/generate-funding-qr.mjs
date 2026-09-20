#!/usr/bin/env node
/**
 * Generate QR codes for the donation addresses in the README.
 *
 * Why this script exists instead of a web QR generator:
 *
 *   Online QR generators are a known attack surface for payment addresses —
 *   a compromised or malicious one can encode a *different* address than the
 *   one you typed, and the resulting PNG looks completely normal. Donations
 *   then go to the attacker. Generating locally removes that entirely.
 *
 * The addresses below are the single source of truth. They are checksum-
 * validated on every run (base58check for Tron, EIP-55 for Ethereum, bech32
 * for Bitcoin, 32-byte base58 for Solana), so a typo fails loudly instead of
 * silently producing a QR that sends money nowhere.
 *
 * Usage:
 *   npm i qrcode js-sha3          # not repo deps; install ad hoc when regenerating
 *   node scripts/generate-funding-qr.mjs
 *   node scripts/generate-funding-qr.mjs --check   # verify committed PNGs match
 *
 * Output: docs/images/funding/<slug>.png
 *
 * If you change an address here you MUST re-run this script, and the diff
 * should be reviewed by a human — see the project review policy.
 */
import { createHash } from 'node:crypto';
import { mkdirSync, existsSync, readFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const REPO = resolve(__dirname, '..');
const OUT_DIR = join(REPO, 'docs', 'images', 'funding');

/** Single source of truth. Keep in sync with the README Support section. */
export const ADDRESSES = [
  { slug: 'usdt-trc20', label: 'USDT (TRC-20 / Tron)', kind: 'tron',
    address: 'TTiCtqLauF1iSW2YGB3b78KmRxRqoLCgeL' },
  { slug: 'usdt-erc20', label: 'USDT / ETH (ERC-20 / Ethereum)', kind: 'eth',
    address: '0xD13D5355Fa214e8317fea2ff192a065BaeC13527' },
  { slug: 'btc', label: 'Bitcoin', kind: 'btc',
    address: 'bc1qatx67n3qxdvuv3arc9j8aytk34f22g02k9c7vr' },
  { slug: 'sol', label: 'Solana', kind: 'sol',
    address: 'AWXzqtBEgUfteHPQtDegsZ6D5y57M3GGdKPD8rR7h6xu' },
];

// ---------------------------------------------------------------- validation

const B58 = '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz';
const sha256 = (b) => createHash('sha256').update(b).digest();

function b58decode(s) {
  let num = 0n;
  for (const ch of s) {
    const i = B58.indexOf(ch);
    if (i < 0) throw new Error(`invalid base58 char '${ch}'`);
    num = num * 58n + BigInt(i);
  }
  let hex = num.toString(16);
  if (hex.length % 2) hex = '0' + hex;
  let zeros = 0;
  for (const ch of s) { if (ch === '1') zeros++; else break; }
  return Buffer.concat([Buffer.alloc(zeros), Buffer.from(hex, 'hex')]);
}

const B32 = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l';
function bech32Polymod(values) {
  const GEN = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];
  let chk = 1;
  for (const v of values) {
    const b = chk >> 25;
    chk = ((chk & 0x1ffffff) << 5) ^ v;
    for (let i = 0; i < 5; i++) if ((b >> i) & 1) chk ^= GEN[i];
  }
  return chk;
}
function bech32HrpExpand(hrp) {
  const a = [], b = [];
  for (const c of hrp) { a.push(c.charCodeAt(0) >> 5); b.push(c.charCodeAt(0) & 31); }
  return [...a, 0, ...b];
}

function validate({ kind, address }) {
  if (kind === 'tron' || kind === 'btc-legacy') {
    const raw = b58decode(address);
    if (raw.length !== 25) throw new Error(`base58check: ${raw.length} bytes, expected 25`);
    const want = sha256(sha256(raw.subarray(0, 21))).subarray(0, 4);
    if (!raw.subarray(21).equals(want)) throw new Error('base58check checksum mismatch');
    if (kind === 'tron' && raw[0] !== 0x41) throw new Error(`tron version byte 0x${raw[0].toString(16)}`);
    return 'base58check ok';
  }
  if (kind === 'eth') {
    if (!/^0x[0-9a-fA-F]{40}$/.test(address)) throw new Error('not 0x + 40 hex');
    // EIP-55 casing check, only if js-sha3 is available.
    try {
      const { keccak_256 } = require('js-sha3');
      const raw = address.slice(2);
      const h = keccak_256(raw.toLowerCase());
      let out = '0x';
      for (let i = 0; i < raw.length; i++)
        out += parseInt(h[i], 16) >= 8 ? raw[i].toUpperCase() : raw[i].toLowerCase();
      if (out !== address) throw new Error('EIP-55 checksum casing mismatch');
      return 'EIP-55 ok';
    } catch (e) {
      if (String(e.message).includes('EIP-55')) throw e;
      return 'format ok (install js-sha3 for EIP-55 check)';
    }
  }
  if (kind === 'btc') {
    const lower = address.toLowerCase();
    if (address !== lower && address !== address.toUpperCase()) throw new Error('bech32 mixed case');
    const pos = lower.lastIndexOf('1');
    if (pos < 1) throw new Error('bech32 no separator');
    const data = [];
    for (const ch of lower.slice(pos + 1)) {
      const i = B32.indexOf(ch);
      if (i < 0) throw new Error(`invalid bech32 char '${ch}'`);
      data.push(i);
    }
    const pm = bech32Polymod([...bech32HrpExpand(lower.slice(0, pos)), ...data]);
    if (pm !== 1 && pm !== 0x2bc830a3) throw new Error('bech32 checksum mismatch');
    return 'bech32 ok';
  }
  if (kind === 'sol') {
    const raw = b58decode(address);
    if (raw.length !== 32) throw new Error(`solana: ${raw.length} bytes, expected 32`);
    return 'base58/32-byte ok';
  }
  throw new Error(`unknown kind '${kind}'`);
}

// ---------------------------------------------------------------------- main

const createRequire = (await import('node:module')).createRequire;
const require = createRequire(import.meta.url);

const checkOnly = process.argv.includes('--check');
let failed = 0;

console.log('Validating addresses...\n');
for (const entry of ADDRESSES) {
  try {
    console.log(`  PASS  ${entry.label.padEnd(32)} ${validate(entry)}`);
  } catch (e) {
    console.error(`  FAIL  ${entry.label.padEnd(32)} ${e.message}`);
    failed++;
  }
}
if (failed) {
  console.error(`\n${failed} address(es) failed validation — refusing to generate QR codes.`);
  process.exit(1);
}

let QRCode;
try {
  QRCode = require('qrcode');
} catch {
  console.error('\nMissing dep. Run:  npm i qrcode js-sha3');
  process.exit(1);
}

if (!existsSync(OUT_DIR)) mkdirSync(OUT_DIR, { recursive: true });

console.log(`\n${checkOnly ? 'Checking' : 'Writing'} QR codes...\n`);
for (const { slug, address, label } of ADDRESSES) {
  const file = join(OUT_DIR, `${slug}.png`);
  const buf = await QRCode.toBuffer(address, {
    errorCorrectionLevel: 'H', margin: 2, width: 512,
    color: { dark: '#000000ff', light: '#ffffffff' },
  });
  if (checkOnly) {
    if (!existsSync(file)) { console.error(`  MISSING  ${slug}.png`); failed++; continue; }
    const same = readFileSync(file).equals(buf);
    console.log(`  ${same ? 'OK      ' : 'STALE   '}${slug}.png`);
    if (!same) failed++;
  } else {
    const { writeFileSync } = await import('node:fs');
    writeFileSync(file, buf);
    console.log(`  wrote   docs/images/funding/${slug}.png   (${label})`);
  }
}

if (checkOnly && failed) {
  console.error('\nQR codes are out of date. Run without --check to regenerate.');
  process.exit(1);
}
console.log('\nDone.');
