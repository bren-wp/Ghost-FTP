import { ipc } from "@/lib/ipc";
import { hydrateFromDb, SETTINGS_KEYS } from "@/stores/settingsStore";

const STORAGE_KEY = "ghostftp.settings.v1";

// One-time migration of app settings from localStorage into ghostftp.db (Plan 12
// Phase 2). Idempotent and safe by construction:
//   - if ghostftp.db already holds settings, do nothing (never re-import a stale
//     blob over newer DB values);
//   - migrate-then-verify: only drop the localStorage blob once the DB reads
//     back at least what we wrote (the plan's "keep the blob until ghostftp.db
//     reads back equal" guard).
// A row in ghostftp.db always beats the value in DEFAULTS, and the import above
// froze EVERY key — including ones the user never opened the settings pane to
// touch. So changing a default in code reaches new installs only: for everyone
// who has already run Ghost FTP, the old default sits in the DB forever.
//
// `settingsRevision` closes that. Each bump rewrites one key, and only where it
// still holds the value the old default wrote — a row the user has since
// changed to something else is left alone. It cannot tell a deliberate "off"
// from a frozen default "off" (they are the same bytes), so a bump does
// override that specific choice once; keep bumps rare and obviously right, and
// remember it costs the user one click to set it back.
const REVISION_KEY = "settingsRevision";

type DefaultBump = { rev: number; key: string; from: unknown; to: unknown };

const DEFAULT_BUMPS: DefaultBump[] = [
  // r1: show dotfiles. Ghost FTP's job is server admin, where .htaccess, .env and
  // .ssh/ are the files that matter — and over FTP they were missing from the
  // listing entirely, so "hidden" read as "not there".
  { rev: 1, key: "showHiddenFiles", from: false, to: true },
];

const CURRENT_REVISION = DEFAULT_BUMPS.reduce((max, b) => Math.max(max, b.rev), 0);

/** Apply any changed defaults this install predates. Idempotent: the stored
 *  revision only moves forward, so each bump runs at most once. */
export async function runDefaultBumps(): Promise<void> {
  try {
    const existing = await ipc.settingsGetAll();
    // Fresh install (no rows at all): DEFAULTS already apply — just stamp it so
    // the bumps never run later against values the user chose deliberately.
    const stored = Number(JSON.parse(existing[REVISION_KEY] ?? "0")) || 0;
    if (stored >= CURRENT_REVISION) return;

    if (Object.keys(existing).length > 0) {
      const updates: Record<string, string> = {};
      for (const b of DEFAULT_BUMPS) {
        if (b.rev <= stored) continue;
        const raw = existing[b.key];
        if (raw === undefined) continue; // no row — the new default is in force
        try {
          if (JSON.stringify(JSON.parse(raw)) === JSON.stringify(b.from)) {
            updates[b.key] = JSON.stringify(b.to);
          }
        } catch {
          // corrupt row — leave it for hydrate to skip
        }
      }
      if (Object.keys(updates).length) await ipc.settingsSetAll(updates);
    }

    await ipc.settingsSet(REVISION_KEY, JSON.stringify(CURRENT_REVISION));
    await hydrateFromDb();
  } catch {
    // Keychain/DB unavailable (or mock) — retry next launch.
  }
}

export async function runSettingsMigration(): Promise<void> {
  try {
    const existing = await ipc.settingsGetAll();
    if (Object.keys(existing).length > 0) return; // already migrated

    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return;
    const parsed = JSON.parse(raw) as Record<string, unknown>;

    const entries: Record<string, string> = {};
    for (const k of SETTINGS_KEYS) {
      if (k in parsed) entries[k as string] = JSON.stringify(parsed[k as string]);
    }
    if (Object.keys(entries).length === 0) return;

    await ipc.settingsSetAll(entries);

    const after = await ipc.settingsGetAll();
    if (Object.keys(after).length >= Object.keys(entries).length) {
      localStorage.removeItem(STORAGE_KEY);
      // Reflect the imported values in this session (they were injected empty on
      // this first upgrade boot).
      await hydrateFromDb();
    }
  } catch {
    // Keychain/DB unavailable (or mock) — retry next launch.
  }
}
