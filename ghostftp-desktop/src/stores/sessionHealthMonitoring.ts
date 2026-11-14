import { useConnections } from "./connectionsStore";

/**
 * While the native desktop window is running, probe each live FTP/FTPS/SFTP
 * session periodically. An idle FTP socket may be dropped by a router/server:
 * the native health command reconnects that transport once, without removing
 * its session id or the user's selected tab.
 *
 * This intentionally does NOT re-authenticate saved profiles on application
 * startup, nor keep network sockets open after the user exits the application.
 */
export function startSessionHealthMonitoring(): () => void {
  const inFlight = new Set<string>();
  const timer = globalThis.setInterval(() => {
    const { sessions, profiles, checkHealth } = useConnections.getState();
    for (const session of sessions) {
      if (inFlight.has(session.sessionId)) continue;
      const profile = profiles.find((item) => item.id === session.profileId);
      if (!profile || !["ftp", "ftps", "sftp"].includes(profile.protocol)) continue;
      inFlight.add(session.sessionId);
      void checkHealth(session.sessionId)
        .catch(() => false)
        .finally(() => inFlight.delete(session.sessionId));
    }
  }, 60_000);
  return () => globalThis.clearInterval(timer);
}
