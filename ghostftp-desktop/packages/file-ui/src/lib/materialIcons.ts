// Material Icon Theme integration — real per-file-type icons (the Photoshop
// mark on a .psd, the Python logo on a .py, etc.), the same set VS Code file
// managers use. MIT licensed (the upstream Material Icon Theme project).
//
// We import the prebuilt manifest for the extension/name → icon-name mapping and
// bundle the SVGs as hashed asset URLs (built into the app — no network). Files
// only; folders and symlinks keep the lucide glyphs so the tree still reads with
// the accent colour. Anything unmatched falls back to the lucide map in
// `fileIcons.tsx`.

import manifest from "material-icon-theme/dist/material-icons.json";

// Vite bundles each matched SVG and hands back its hashed URL. Absolute path
// from the Vite root (repo root, where node_modules lives).
const svgUrls = import.meta.glob(
  "/node_modules/material-icon-theme/icons/*.svg",
  { query: "?url", import: "default", eager: true }
) as Record<string, string>;

type StrMap = Record<string, string>;
const fileNames = (manifest.fileNames ?? {}) as StrMap;
const fileExtensions = (manifest.fileExtensions ?? {}) as StrMap;

function urlForIconName(name: string | undefined): string | undefined {
  if (!name) return undefined;
  return svgUrls[`/node_modules/material-icon-theme/icons/${name}.svg`];
}

/**
 * A bundled Material Icon Theme SVG URL for a file, or `undefined` to fall back
 * to the lucide icon. Matches by exact filename first (e.g. `dockerfile`,
 * `package.json`), then by the longest extension suffix (so `app.d.ts` beats
 * `ts`), mirroring VS Code's resolution.
 */
export function materialIconUrl(name: string): string | undefined {
  const lower = name.toLowerCase();

  const byName = fileNames[lower];
  if (byName) return urlForIconName(byName);

  const parts = lower.split(".");
  for (let i = 1; i < parts.length; i++) {
    const ext = parts.slice(i).join(".");
    const icon = fileExtensions[ext];
    if (icon) return urlForIconName(icon);
  }
  return undefined;
}
