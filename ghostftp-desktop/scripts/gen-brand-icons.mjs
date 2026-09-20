// Generates src/lib/brandIconData.ts — a curated, OFFLINE subset of Iconify
// brand/protocol logos (Plan 14). We extract only the handful of icons Ghost FTP
// actually references from the full @iconify-json/* sets, so the app bundles a
// few KB of icon data instead of whole 2,000-icon sets, and makes ZERO network
// calls (nothing ever hits api.iconify.design).
//
// Run `npm run gen:brand-icons` after adding an icon to CURATED below. The
// @iconify-json/* packages + @iconify/utils are devDependencies used only here;
// the generated file is committed so normal builds don't need them.
//
// Licences (all permissive — recorded in THIRD_PARTY_LICENSES.md):
//   logos         CC0-1.0     (colour brand marks)
//   simple-icons  CC0-1.0     (monochrome brand marks)
//   mdi           Apache-2.0  (neutral protocol glyphs)

import { getIconData } from "@iconify/utils";
import { writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

import logos from "@iconify-json/logos/icons.json" with { type: "json" };
import simpleIcons from "@iconify-json/simple-icons/icons.json" with { type: "json" };
import mdi from "@iconify-json/mdi/icons.json" with { type: "json" };

// JSON-module default interop (Node exposes the set under `.default`).
const SETS = {
  logos: logos.default ?? logos,
  "simple-icons": simpleIcons.default ?? simpleIcons,
  mdi: mdi.default ?? mdi,
};

// Every brand/protocol icon Ghost FTP renders, keyed by its Iconify `prefix:name`.
// protocolIcon() in src/lib/brandIcons.ts maps a Protocol onto one of these.
const CURATED = [
  // Protocol glyphs
  "mdi:ssh", // sftp / ssh
  "mdi:folder-network-outline", // ftp
  "mdi:folder-network", // ftps
  "mdi:web", // webdav
  "mdi:file-download-outline", // http (read-only source)
  "mdi:lighthouse-on", // ghostftp-agent (the Ghost FTP lighthouse mark)
  // Colour brand marks (render in their own brand colours)
  "logos:aws-s3", // s3
  "logos:microsoft-azure", // azure
  "logos:google-cloud", // gcs
  "logos:dropbox", // dropbox
  "logos:microsoft-onedrive", // onedrive
  "logos:google-drive", // gdrive
  "simple-icons:box", // box
  "simple-icons:shopify", // shopify
  "simple-icons:hubspot", // hubspot
  "simple-icons:dynamics365", // dynamics
  // Vendor marks for the S3 / WebDAV provider preset buttons (Phase 3). Where a
  // provider has no icon (Storj, the generic presets), the button falls back to
  // the neutral lucide glyph — protocolIcon() never maps to these.
  "logos:cloudflare", // s3: Cloudflare R2
  "simple-icons:backblaze", // s3: Backblaze B2
  "simple-icons:wasabi", // s3: Wasabi
  "logos:digital-ocean", // s3: DigitalOcean Spaces
  "simple-icons:minio", // s3: MinIO
  "simple-icons:hetzner", // s3: Hetzner / WebDAV Storage Box
  "simple-icons:scaleway", // s3: Scaleway
  "logos:oracle", // s3: Oracle OCI
  "logos:ibm", // s3: IBM COS
  "logos:supabase-icon", // s3: Supabase
  "simple-icons:nextcloud", // webdav: Nextcloud
  "simple-icons:owncloud", // webdav: ownCloud
  // Personality glyphs for custom connection icons (profile.icon): a small,
  // deliberately fun-but-useful set so a bubble can be a rocket, a database,
  // or a cat instead of the name monogram. All mdi (Apache-2.0).
  "mdi:rocket-launch",
  "mdi:rocket-launch-outline",
  "mdi:database",
  "mdi:database-outline",
  "mdi:cloud",
  "mdi:cloud-outline",
  "mdi:server",
  "mdi:server-network",
  "mdi:star",
  "mdi:star-outline",
  "mdi:heart",
  "mdi:heart-outline",
  "mdi:fire",
  "mdi:lightning-bolt",
  "mdi:lightning-bolt-outline",
  "mdi:earth",
  "mdi:globe-model",
  "mdi:shield",
  "mdi:shield-outline",
  "mdi:home",
  "mdi:home-outline",
  "mdi:office-building",
  "mdi:office-building-outline",
  "mdi:robot",
  "mdi:robot-outline",
  "mdi:alien",
  "mdi:alien-outline",
  "mdi:cat",
  "mdi:dog",
  "mdi:ghost",
  "mdi:ghost-outline",
  "mdi:gamepad-variant",
  "mdi:gamepad-variant-outline",
  "mdi:pizza",
  "mdi:beer-outline",
  "mdi:coffee",
  "mdi:coffee-outline",
  "mdi:cactus",
  "mdi:sail-boat",
  "mdi:airplane",
  "mdi:car",
  "mdi:bicycle",
  "mdi:motorbike",
  "mdi:castle",
  "mdi:wizard-hat",
  "mdi:mushroom",
  "mdi:mushroom-outline",
  "mdi:atom",
  "mdi:brain",
  "mdi:code-tags",
  "mdi:penguin",
  "mdi:fish",
  "mdi:owl",
  "mdi:pokeball",
  "mdi:ninja",
];

// Drop transformation fields that equal the Iconify defaults, to keep the
// generated file tiny and readable.
function trim(data) {
  const out = { body: data.body, width: data.width, height: data.height };
  if (data.left) out.left = data.left;
  if (data.top) out.top = data.top;
  if (data.rotate) out.rotate = data.rotate;
  if (data.hFlip) out.hFlip = true;
  if (data.vFlip) out.vFlip = true;
  return out;
}

const entries = [];
const missing = [];
for (const full of CURATED) {
  const [prefix, name] = full.split(":");
  const set = SETS[prefix];
  const data = set ? getIconData(set, name) : null;
  if (!data) {
    missing.push(full);
    continue;
  }
  entries.push([full, trim(data)]);
}

if (missing.length) {
  console.error("gen-brand-icons: icons not found in the installed sets:");
  for (const m of missing) console.error("  - " + m);
  process.exit(1);
}

const header = `// AUTO-GENERATED by scripts/gen-brand-icons.mjs — DO NOT EDIT BY HAND.
// A curated, offline subset of Iconify brand/protocol logos (Plan 14). Regenerate
// with \`npm run gen:brand-icons\` after editing CURATED in that script.
// Licences: logos & simple-icons = CC0-1.0, mdi = Apache-2.0 (see THIRD_PARTY_LICENSES.md).
import type { IconifyIcon } from "@iconify/react/offline";
`;

const lines = entries.map(
  ([name, data]) => `  ${JSON.stringify(name)}: ${JSON.stringify(data)},`
);
const body = `\nexport const BRAND_ICONS: Record<string, IconifyIcon> = {\n${lines.join(
  "\n"
)}\n};\n`;

const outPath = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "src",
  "lib",
  "brandIconData.ts"
);
writeFileSync(outPath, header + body);

const bytes = Buffer.byteLength(header + body);
console.log(
  `gen-brand-icons: wrote ${entries.length} icons → src/lib/brandIconData.ts (${(
    bytes / 1024
  ).toFixed(1)} KB)`
);
