// Plan 14 — recognizable brand/protocol logos via Iconify, bundled OFFLINE.
//
// This is additive: lucide still owns UI chrome and Material Icon Theme still
// owns file-type icons. Iconify fills the one gap neither covers — actual brand
// marks (AWS S3, Azure, an SSH glyph, the Ghost FTP lighthouse) on connections.
//
// Offline by construction: we import the `Icon` component from
// `@iconify/react/offline` (no API/fetch code — nothing can reach
// api.iconify.design) and hand it raw icon DATA from the committed, curated
// `brandIconData.ts`. A string name is never looked up over the network.

import { Icon } from "@iconify/react/offline";
import type { Protocol } from "@/lib/types";
import { cn } from "@/lib/cn";
import { BRAND_ICONS } from "./brandIconData";

// Protocol → Iconify `prefix:name`. Colour `logos:` marks render in their own
// brand palette (AWS orange, Azure blue…); monochrome `mdi:` / `simple-icons:`
// glyphs follow `currentColor`, so they tint to the surrounding UI.
const PROTOCOL_ICON: Record<Protocol, string> = {
  sftp: "mdi:ssh",
  ftp: "mdi:folder-network-outline",
  ftps: "mdi:folder-network",
  s3: "logos:aws-s3",
  azure: "logos:microsoft-azure",
  gcs: "logos:google-cloud",
  webdav: "mdi:web",
  http: "mdi:file-download-outline",
  dropbox: "logos:dropbox",
  onedrive: "logos:microsoft-onedrive",
  gdrive: "logos:google-drive",
  box: "simple-icons:box",
  shopify: "simple-icons:shopify",
  hubspot: "simple-icons:hubspot",
  dynamics: "simple-icons:dynamics365",
  "ghostftp-agent": "mdi:lighthouse-on",
};

/** The Iconify `prefix:name` for a connection protocol's brand/protocol logo. */
export function protocolIcon(protocol: Protocol): string {
  return PROTOCOL_ICON[protocol];
}

/** True for the multi-colour brand marks (they carry their own fill and ignore
 *  `currentColor`) — callers can skip tinting/opacity that only suits a glyph. */
export function isColorBrandIcon(icon: string): boolean {
  return icon.startsWith("logos:");
}

/** A thin, offline wrapper over Iconify's `<Icon>`. Renders the curated icon
 *  DATA (never a network name); returns `null` for an unknown key so callers can
 *  fall back to a monogram / lucide glyph. `size` is a square bounding box: a
 *  non-square brand mark (OneDrive, Azure) is letterboxed inside it via the
 *  SVG's own `preserveAspectRatio`, so it never distorts or overflows the slot. */
export function BrandIcon({
  icon,
  size = 16,
  className,
  title,
}: {
  icon: string;
  size?: number;
  className?: string;
  /** When set, the icon is meaningful (adds a tooltip + drops aria-hidden). */
  title?: string;
}) {
  const data = BRAND_ICONS[icon];
  if (!data) return null;
  return (
    <Icon
      icon={data}
      width={size}
      height={size}
      className={cn("brand-icon", className)}
      role={title ? "img" : undefined}
      aria-label={title}
      aria-hidden={title ? undefined : true}
    />
  );
}
