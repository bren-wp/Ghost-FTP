// Generates placeholder PNG icons for the Tauri app so `npm run tauri dev`
// runs out of the box. Replace with real icons via:
//
//     npm run tauri icon src-tauri/icons/source.png
//
// (provide your own source.png — 1024x1024 recommended).

import { existsSync, writeFileSync, mkdirSync } from "node:fs";
import { deflateSync } from "node:zlib";
import path from "node:path";

const ICONS_DIR = path.resolve(
  path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1")),
  "..",
  "src-tauri",
  "icons"
);

// Pure-JS minimal PNG encoder (solid colour RGBA, no deps).
const crcTable = (() => {
  const t = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++)
      c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    t[n] = c >>> 0;
  }
  return t;
})();

function crc32(buf) {
  let crc = 0xffffffff;
  for (let i = 0; i < buf.length; i++) {
    crc = (crc >>> 8) ^ crcTable[(crc ^ buf[i]) & 0xff];
  }
  return (crc ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length);
  const typeB = Buffer.from(type, "ascii");
  const body = Buffer.concat([typeB, data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body));
  return Buffer.concat([len, body, crc]);
}

function makePng(size, [r, g, b, a]) {
  const signature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8; // bit depth
  ihdr[9] = 6; // RGBA
  ihdr[10] = 0;
  ihdr[11] = 0;
  ihdr[12] = 0;

  const rowLen = 1 + size * 4;
  const raw = Buffer.alloc(rowLen * size);
  for (let y = 0; y < size; y++) {
    const off = y * rowLen;
    raw[off] = 0; // filter type: none
    for (let x = 0; x < size; x++) {
      const px = off + 1 + x * 4;
      raw[px] = r;
      raw[px + 1] = g;
      raw[px + 2] = b;
      raw[px + 3] = a;
    }
  }
  const idat = deflateSync(raw);

  return Buffer.concat([
    signature,
    chunk("IHDR", ihdr),
    chunk("IDAT", idat),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

// ICO file embedding a PNG. Modern Windows accepts PNG-in-ICO at any size; this
// is enough for `tauri-build`'s Windows Resource step and for the window icon.
function makeIco(pngBuf, size) {
  const dir = Buffer.alloc(6);
  dir.writeUInt16LE(0, 0); // Reserved
  dir.writeUInt16LE(1, 2); // Type: 1 = icon
  dir.writeUInt16LE(1, 4); // Image count

  const entry = Buffer.alloc(16);
  entry[0] = size >= 256 ? 0 : size; // Width (0 = 256)
  entry[1] = size >= 256 ? 0 : size; // Height
  entry[2] = 0; // Colour palette count
  entry[3] = 0; // Reserved
  entry.writeUInt16LE(1, 4); // Colour planes
  entry.writeUInt16LE(32, 6); // Bits per pixel
  entry.writeUInt32LE(pngBuf.length, 8); // Image size
  entry.writeUInt32LE(6 + 16, 12); // Offset to image data

  return Buffer.concat([dir, entry, pngBuf]);
}

mkdirSync(ICONS_DIR, { recursive: true });

const BLUE = [59, 130, 246, 255]; // accent colour

const pngTargets = [
  { name: "32x32.png", size: 32 },
  { name: "128x128.png", size: 128 },
  { name: "128x128@2x.png", size: 256 },
  { name: "source.png", size: 512 },
];

let wrote = 0;
for (const t of pngTargets) {
  const dest = path.join(ICONS_DIR, t.name);
  if (existsSync(dest)) continue;
  writeFileSync(dest, makePng(t.size, BLUE));
  wrote++;
  console.log(`  wrote ${t.name} (${t.size}x${t.size})`);
}

const icoPath = path.join(ICONS_DIR, "icon.ico");
if (!existsSync(icoPath)) {
  const icoPng = makePng(256, BLUE);
  writeFileSync(icoPath, makeIco(icoPng, 256));
  wrote++;
  console.log("  wrote icon.ico (256x256 PNG-in-ICO)");
}

if (wrote === 0) {
  console.log("Icons already present, skipping.");
} else {
  console.log(
    `Generated ${wrote} placeholder icon(s). For real icons: npm run tauri icon src-tauri/icons/source.png`
  );
}

// Tauri's codegen validates that `frontendDist` exists at compile time, even
// when running `tauri dev` (which actually uses `devUrl`). Drop a placeholder
// so `cargo check` works before the first `npm run build`.
const distDir = path.resolve(
  path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1")),
  "..",
  "dist"
);
const distIndex = path.join(distDir, "index.html");
if (!existsSync(distIndex)) {
  mkdirSync(distDir, { recursive: true });
  writeFileSync(
    distIndex,
    "<!doctype html><title>file-ssh placeholder</title>\n"
  );
  console.log("  wrote dist/index.html placeholder (for Tauri codegen)");
}
