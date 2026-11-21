// Ghost FTP by Brendigo
import { Star } from "lucide-react";
import type { ConnectionProfile } from "@/lib/types";

export function SiteRow({
  profile, active, favorite, favoriteBusy, connected, onClick, onFavorite,
}: {
  profile: ConnectionProfile; active: boolean; favorite: boolean;
  favoriteBusy: boolean; connected: boolean;
  onClick: () => void; onFavorite: () => void;
}) {
  const tag = profile.tags?.[0] || profile.group || "Sites";
  const initials = profile.name.trim().split(/\s+/).slice(0, 2)
    .map((part) => part.charAt(0)).join("").toUpperCase() || "S";
  const subtitle = profile.defaultRemotePath?.trim() || profile.host;
  return (
    <div className={`ghost-site-row ${active ? "is-selected" : ""}`} role="group"
      aria-label={`Saved site: ${profile.name}`}>
      <button type="button" className="ghost-site-row-main"
        onClick={onClick} aria-current={active ? "true" : undefined}
        title={`${profile.name} — ${profile.protocol.toUpperCase()} — ${profile.host}`}>
        <span className="ghost-site-row-avatar" aria-hidden="true">{initials}</span>
        <span className="ghost-site-row-copy">
          <strong className="ghost-site-row-name">{profile.name}</strong>
          <span className="ghost-site-row-subtitle">{profile.protocol.toUpperCase()} · {subtitle}</span>
          <span className="ghost-site-row-extra">
            <em className={`ghost-site-row-tag ${tagToneClass(tag)}`}>{tag}</em>
            <span className={connected ? "text-success" : "text-text-dim"}>
              {connected ? "Connected" : formatLastUsed(profile.lastUsed)}
            </span>
          </span>
        </span>
      </button>
      <button type="button" className="ghost-site-row-favorite"
        onClick={onFavorite} disabled={favoriteBusy} aria-pressed={favorite}
        aria-label={favorite ? `Remove ${profile.name} from favorites` : `Add ${profile.name} to favorites`}
        title={favorite ? "Remove favorite" : "Add to favorites"}>
        <Star size={16} fill={favorite ? "currentColor" : "none"} />
      </button>
    </div>
  );
}

function tagToneClass(label: string) {
  const normalized = label.trim().toLowerCase();
  if (normalized.includes("production")) return "border-sky-400/55 bg-sky-500/15 text-sky-300";
  if (normalized.includes("staging")) return "border-emerald-400/55 bg-emerald-500/15 text-emerald-300";
  if (normalized.includes("personal")) return "border-pink-400/55 bg-pink-500/15 text-pink-300";
  if (normalized.includes("client")) return "border-orange-400/55 bg-orange-500/15 text-orange-300";
  if (normalized.includes("development") || normalized.includes("dev")) return "border-violet-400/55 bg-violet-500/15 text-violet-300";
  if (normalized.includes("backup")) return "border-rose-400/55 bg-rose-500/15 text-rose-300";
  const palette = [
    "border-sky-400/55 bg-sky-500/15 text-sky-300",
    "border-emerald-400/55 bg-emerald-500/15 text-emerald-300",
    "border-violet-400/55 bg-violet-500/15 text-violet-300",
    "border-orange-400/55 bg-orange-500/15 text-orange-300",
    "border-pink-400/55 bg-pink-500/15 text-pink-300",
    "border-rose-400/55 bg-rose-500/15 text-rose-300",
  ];
  const hash = Array.from(normalized).reduce((sum, char) => sum + char.charCodeAt(0), 0);
  return palette[hash % palette.length];
}

function formatLastUsed(value?: number) {
  if (!value) return "—";
  const date = new Date(value * 1000);
  const now = new Date();
  return date.toDateString() === now.toDateString()
    ? date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    : date.toLocaleDateString();
}
