import { create } from "zustand";
import { ipc } from "@/lib/ipc";
import { toast } from "./toastStore";
import { toastError, messageOf } from "@/lib/errors";
import type { ConnectionProfile, SessionId } from "@/lib/types";
import { useTerminals } from "./terminalsStore";
import { redactSensitiveText } from "@/lib/redact";

function syncBridgeActiveSession(sessionId: SessionId | null): void {
  void ipc.bridgeSetActiveSession(sessionId).catch((error) => {
    console.warn(
      "Couldn't synchronize Agent Bridge active session",
      redactSensitiveText(error, 240)
    );
  });
}

const pendingConnections = new Map<string, Promise<void>>();

function trackConnection(
  key: string,
  set: (partial: Partial<ConnectionsState>) => void,
  operation: () => Promise<void>
): Promise<void> {
  const existing = pendingConnections.get(key);
  if (existing) return existing;

  set({ connecting: true, error: null });
  const promise = operation();
  pendingConnections.set(key, promise);

  const finish = () => {
    if (pendingConnections.get(key) === promise) {
      pendingConnections.delete(key);
    }
    set({ connecting: pendingConnections.size > 0 });
  };
  void promise.then(finish, finish);
  return promise;
}

// One live connection. The backend keeps every session alive in a map, so the
// app can hold several at once; this is the frontend's view of them.
export interface LiveSession {
  sessionId: SessionId;
  profileId: string;
  /** True for Quick Connect sessions that are intentionally never persisted. */
  ephemeral?: boolean;
}

interface ConnectionsState {
  profiles: ConnectionProfile[];
  /** Every live connection, in the order they were opened. */
  sessions: LiveSession[];
  /** The focused session — drives the browser, terminal and status bar. */
  activeSessionId: SessionId | null;
  /** Profile of the focused session; kept in sync with activeSessionId. */
  activeProfileId: string | null;
  connecting: boolean;
  error: string | null;

  loadProfiles: () => Promise<void>;
  saveProfile: (p: ConnectionProfile) => Promise<void>;
  duplicateProfile: (id: string) => Promise<ConnectionProfile>;
  /** Save several profiles with a single reload at the end (group rename etc). */
  saveProfiles: (ps: ConnectionProfile[]) => Promise<void>;
  /** Persist a drag-and-drop rail order (every profile id, display order).
   *  Optionally re-homes the dragged profile into a new group first. */
  reorderProfiles: (
    ids: string[],
    groupChange?: { id: string; group: string | undefined }
  ) => Promise<void>;
  deleteProfile: (id: string) => Promise<void>;
  /** Connect to a profile, or focus its existing session if already open. */
  connect: (profileId: string) => Promise<void>;
  /** Connect with an in-memory profile without writing credentials or profile data. */
  connectTemporary: (profile: ConnectionProfile) => Promise<void>;
  /** Disconnect one session (defaults to the active one). */
  disconnect: (sessionId?: SessionId) => Promise<void>;
  /** Focus an already-open session. */
  setActiveSession: (sessionId: SessionId) => void;
}

export const useConnections = create<ConnectionsState>((set, get) => ({
  profiles: [],
  sessions: [],
  activeSessionId: null,
  activeProfileId: null,
  connecting: false,
  error: null,

  loadProfiles: async () => {
    const persisted = await ipc.listProfiles();
    const state = get();
    const ephemeralIds = new Set(
      state.sessions.filter((session) => session.ephemeral).map((session) => session.profileId)
    );
    const ephemeral = state.profiles.filter((profile) => ephemeralIds.has(profile.id));
    const persistedIds = new Set(persisted.map((profile) => profile.id));
    set({ profiles: [...persisted, ...ephemeral.filter((profile) => !persistedIds.has(profile.id))] });
  },

  saveProfile: async (p) => {
    await ipc.saveProfile(p);
    await get().loadProfiles();
  },

  duplicateProfile: async (id) => {
    const duplicated = await ipc.duplicateProfile(id);
    await get().loadProfiles();
    return duplicated;
  },

  saveProfiles: async (ps) => {
    for (const p of ps) await ipc.saveProfile(p);
    await get().loadProfiles();
  },

  reorderProfiles: async (ids, groupChange) => {
    if (groupChange) {
      const p = get().profiles.find((x) => x.id === groupChange.id);
      if (p) await ipc.saveProfile({ ...p, group: groupChange.group });
    }
    await ipc.reorderProfiles(ids);
    await get().loadProfiles();
  },

  deleteProfile: async (id) => {
    // Delete durable profile data first. If deletion fails, keep any live
    // session usable; if it succeeds, close sessions that no longer have saved
    // metadata. This avoids disconnecting the user for a failed delete.
    await ipc.deleteProfile(id);
    const live = get().sessions.filter((session) => session.profileId === id);
    for (const session of live) {
      await get().disconnect(session.sessionId);
    }
    await get().loadProfiles();
  },

  connect: (profileId) =>
    trackConnection(`saved:${profileId}`, set, async () => {
      // Already connected to this profile? Just focus its tab.
      const existing = get().sessions.find((session) => session.profileId === profileId);
      if (existing) {
        get().setActiveSession(existing.sessionId);
        return;
      }

      const profile = get().profiles.find((item) => item.id === profileId);
      try {
        const sessionId = await ipc.connect(profileId);
        // A concurrent caller for the same profile reuses this exact promise,
        // so this is the only path that can append the resulting live session.
        set((state) => ({
          sessions: state.sessions.some((session) => session.sessionId === sessionId)
            ? state.sessions
            : [...state.sessions, { sessionId, profileId }],
          activeSessionId: sessionId,
          activeProfileId: profileId,
        }));
        syncBridgeActiveSession(sessionId);
        toast.success(
          "Connected",
          profile
            ? `${profile.name} — ${profile.username}@${profile.host}`
            : undefined
        );
        // Refresh persisted profile metadata such as lastUsed without disturbing
        // any active ephemeral Quick Connect profiles.
        void get().loadProfiles().catch((error) =>
          toastError(error, "Connected, but couldn't refresh saved sites")
        );
      } catch (error) {
        set({ error: messageOf(error) });
        toastError(
          error,
          profile ? `Couldn't connect to ${profile.name}` : "Connection failed"
        );
        throw error;
      }
    }),

  connectTemporary: (profile) =>
    trackConnection(`temporary:${profile.id}`, set, async () => {
      const existing = get().sessions.find(
        (session) => session.profileId === profile.id
      );
      if (existing) {
        get().setActiveSession(existing.sessionId);
        return;
      }

      try {
        const sessionId = await ipc.connectEphemeral(profile);
        set((state) => ({
          profiles: state.profiles.some((item) => item.id === profile.id)
            ? state.profiles
            : [...state.profiles, profile],
          sessions: state.sessions.some((session) => session.sessionId === sessionId)
            ? state.sessions
            : [
                ...state.sessions,
                { sessionId, profileId: profile.id, ephemeral: true },
              ],
          activeSessionId: sessionId,
          activeProfileId: profile.id,
        }));
        syncBridgeActiveSession(sessionId);
        toast.success(
          "Connected",
          `${profile.name} — ${profile.username}@${profile.host}`
        );
      } catch (error) {
        set({ error: messageOf(error) });
        toastError(error, `Couldn't connect to ${profile.name}`);
        throw error;
      }
    }),

  disconnect: async (sessionId) => {
    const sid = sessionId ?? get().activeSessionId;
    if (!sid) return;
    const target = get().sessions.find((s) => s.sessionId === sid);
    const profile = get().profiles.find((p) => p.id === target?.profileId);
    try {
      await ipc.disconnect(sid);
    } catch (error) {
      // A dropped/broken transport may make the backend disconnect fail even
      // though the UI still has to forget the dead session. Surface the real
      // backend error, then continue the local cleanup instead of failing
      // silently or leaving a stale Connected state behind.
      toastError(error, profile ? `Couldn't cleanly disconnect from ${profile.name}` : "Disconnect failed");
    }
    useTerminals.getState().dropSessionTabs(sid);
    set((s) => {
      const sessions = s.sessions.filter((x) => x.sessionId !== sid);
      const profiles = target?.ephemeral && !sessions.some((x) => x.profileId === target.profileId)
        ? s.profiles.filter((p) => p.id !== target.profileId)
        : s.profiles;
      // If we closed the focused session, fall back to the most recent one.
      let activeSessionId = s.activeSessionId;
      let activeProfileId = s.activeProfileId;
      if (s.activeSessionId === sid) {
        const next = sessions[sessions.length - 1] ?? null;
        activeSessionId = next?.sessionId ?? null;
        activeProfileId = next?.profileId ?? null;
      }
      return { profiles, sessions, activeSessionId, activeProfileId };
    });
    syncBridgeActiveSession(get().activeSessionId);
    toast.info("Disconnected", profile?.name);
  },

  setActiveSession: (sessionId) => {
    const target = get().sessions.find((s) => s.sessionId === sessionId);
    if (!target) return;
    set({ activeSessionId: sessionId, activeProfileId: target.profileId });
    // Keep the Agent Bridge aware of which connection the user is focused on.
    syncBridgeActiveSession(sessionId);
  },
}));
