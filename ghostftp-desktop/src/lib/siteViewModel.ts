// Ghost FTP by Brendigo — read-only saved-site view model.
import type { ConnectionProfile } from "./types";

export type SiteSortField = "name" | "host" | "protocol" | "lastUsed";
export type SiteSortDirection = "asc" | "desc";

const CLOUD_PROTOCOLS = new Set(["s3", "azure", "gcs", "webdav", "dropbox", "onedrive", "gdrive", "box"]);
export function isCloudSite(profile: Pick<ConnectionProfile, "protocol">): boolean {
  return CLOUD_PROTOCOLS.has(profile.protocol);
}

export function validLastUsed(value?: number): number | null {
  return typeof value === "number" && Number.isFinite(value) && value > 0 &&
    Number.isFinite(new Date(value * 1000).getTime()) ? value : null;
}

export function countSiteViews(
  profiles: readonly ConnectionProfile[],
  duplicateIds: ReadonlySet<string>,
  connectedIds: ReadonlySet<string>,
) {
  const counts = { all: profiles.length, favorites: 0, recent: 0, bookmarks: 0, duplicates: 0, cloud: 0, connected: 0 };
  for (const profile of profiles) {
    if (profile.favorite) counts.favorites++;
    if (validLastUsed(profile.lastUsed) !== null) counts.recent++;
    if (profile.bookmarked) counts.bookmarks++;
    if (duplicateIds.has(profile.id)) counts.duplicates++;
    if (isCloudSite(profile)) counts.cloud++;
    if (connectedIds.has(profile.id)) counts.connected++;
  }
  return counts;
}

export function filterAndSortSites(
  profiles: readonly ConnectionProfile[],
  query: string,
  view: string,
  field: SiteSortField,
  direction: SiteSortDirection,
  duplicateIds: ReadonlySet<string>,
  connectedIds: ReadonlySet<string>,
): ConnectionProfile[] {
  const normalized = query.trim().toLowerCase();
  const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });
  const selected = profiles.filter((profile) => {
    if (view === "favorites" && !profile.favorite) return false;
    if (view === "recent" && validLastUsed(profile.lastUsed) === null) return false;
    if (view === "bookmarks" && !profile.bookmarked) return false;
    if (view === "duplicates" && !duplicateIds.has(profile.id)) return false;
    if (view === "cloud" && !isCloudSite(profile)) return false;
    if (view === "connected" && !connectedIds.has(profile.id)) return false;
    if (view.startsWith("tag:") && !(profile.tags ?? []).includes(view.slice(4))) return false;
    if (view.startsWith("folder:") && (profile.group ?? "") !== view.slice(7)) return false;
    // Do not add credentials or authentication material to the search index.
    if (!normalized) return true;
    return `${profile.name} ${profile.host} ${profile.username} ${profile.protocol} ${profile.group ?? ""} ${profile.description ?? ""} ${(profile.tags ?? []).join(" ")}`.toLowerCase().includes(normalized);
  });
  const sign = direction === "asc" ? 1 : -1;
  return selected.sort((a, b) => {
    if (field === "lastUsed") {
      const left = validLastUsed(a.lastUsed);
      const right = validLastUsed(b.lastUsed);
      if (left === null && right === null) return 0;
      if (left === null) return 1;
      if (right === null) return -1;
      return (left - right) * sign;
    }
    return collator.compare(a[field], b[field]) * sign;
  });
}
