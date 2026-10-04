import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const desktopRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const iconsDir = path.join(desktopRoot, "src-tauri", "icons");

const PNG_SIGNATURE = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
const requiredIcons = [
  { name: "32x32.png", type: "png" },
  { name: "128x128.png", type: "png" },
  { name: "128x128@2x.png", type: "png" },
  { name: "source.png", type: "png" },
  { name: "icon.ico", type: "ico" },
];

function requireProductionIcon({ name, type }) {
  const file = path.join(iconsDir, name);
  if (!existsSync(file)) {
    throw new Error(
      `Required production icon missing: ${name}. Restore the reviewed Ghost FTP brand assets before building.`
    );
  }

  const bytes = readFileSync(file);
  if (type === "png") {
    if (
      bytes.length < PNG_SIGNATURE.length ||
      !bytes.subarray(0, PNG_SIGNATURE.length).equals(PNG_SIGNATURE)
    ) {
      throw new Error(`Invalid production PNG icon: ${name}`);
    }
    return;
  }

  if (
    bytes.length < 6 ||
    bytes[0] !== 0x00 ||
    bytes[1] !== 0x00 ||
    bytes[2] !== 0x01 ||
    bytes[3] !== 0x00
  ) {
    throw new Error(`Invalid production ICO icon: ${name}`);
  }
}

for (const icon of requiredIcons) requireProductionIcon(icon);
console.log("Production desktop icons verified.");

// Tauri codegen validates that frontendDist exists even for cargo-only checks.
// Create only a neutral Ghost FTP bootstrap document; the real Vite build
// replaces it before packaging.
const distDir = path.join(desktopRoot, "dist");
const distIndex = path.join(distDir, "index.html");
if (!existsSync(distIndex)) {
  mkdirSync(distDir, { recursive: true });
  writeFileSync(
    distIndex,
    "<!doctype html><html><head><meta charset=\"utf-8\"><title>Ghost FTP</title></head><body></body></html>\n"
  );
  console.log("Prepared frontendDist bootstrap for Tauri codegen.");
}
