import type { ConnectionProfile } from "./types";

// Compare only a connection endpoint and account name. Authentication secrets,
// remote paths and user-defined aliases are deliberately excluded. This is an
// advisory inventory, never an instruction to merge or delete credentials.
type SiteEndpoint = Pick<ConnectionProfile, "protocol" | "host" | "port" | "username">;

export function savedSiteIdentity(site: SiteEndpoint): string | null {
  const host = site.host.trim().toLowerCase().replace(/\.$/, "");
  const username = site.username.trim();
  if (!host || !username || !Number.isInteger(site.port) || site.port < 1 || site.port > 65535) {
    return null;
  }
  // An encoded tuple avoids collisions between delimiter-bearing hostnames
  // and usernames. SSH/FTP account names remain case-sensitive.
  return JSON.stringify([site.protocol, host, site.port, username]);
}

export function matchingSavedSites(
  candidate: SiteEndpoint & { id?: string },
  profiles: readonly ConnectionProfile[],
): ConnectionProfile[] {
  const identity = savedSiteIdentity(candidate);
  if (identity === null) return [];
  return profiles.filter((site) => site.id !== candidate.id && savedSiteIdentity(site) === identity);
}

export function duplicateSavedSiteIds(profiles: readonly ConnectionProfile[]): Set<string> {
  const seen = new Map<string, string>();
  const duplicates = new Set<string>();
  for (const site of profiles) {
    const identity = savedSiteIdentity(site);
    if (identity === null) continue;
    const previous = seen.get(identity);
    if (previous !== undefined) {
      duplicates.add(previous);
      duplicates.add(site.id);
    } else {
      seen.set(identity, site.id);
    }
  }
  return duplicates;
}
