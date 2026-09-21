import { useEffect, useState } from "react";
import { FilePane } from "@ghostftp/file-ui";
import { SyncDialog } from "./SyncDialog";
import { useConnections } from "@/stores/connectionsStore";
import { useTransfers, toTransferItem } from "@/stores/transfersStore";
import { useLayout } from "@/stores/layoutStore";
import { LOCAL_SESSION } from "@/lib/types";
import type { DirEntry } from "@/lib/types";

export function DualPaneBrowser() {
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const profile = profiles.find((p) => p.id === activeProfileId) || null;
  const enqueueUploads = useTransfers((s) => s.enqueueUploads);
  const enqueueDownloads = useTransfers((s) => s.enqueueDownloads);

  const [localPath, setLocalPath] = useState(
    navigator.userAgent.includes("Windows") ? "C:\\" : "/"
  );
  // Each live session keeps its own remote location, so switching connection
  // tabs returns you to where you left off on that server.
  const [remotePaths, setRemotePaths] = useState<Record<string, string>>({});
  const [syncOpen, setSyncOpen] = useState(false);
  const openNewConnection = useLayout((s) => s.openNewConnection);

  const remotePath =
    (activeSessionId && remotePaths[activeSessionId]) ||
    profile?.defaultRemotePath ||
    ".";
  const setRemotePath = (path: string) => {
    if (!activeSessionId) return;
    setRemotePaths((m) => ({ ...m, [activeSessionId]: path }));
  };

  useEffect(() => {
    const openSync = () => {
      if (!activeSessionId) {
        openNewConnection();
        return;
      }
      setSyncOpen(true);
    };
    window.addEventListener("ghostftp:open-sync", openSync);
    return () => window.removeEventListener("ghostftp:open-sync", openSync);
  }, [activeSessionId, openNewConnection]);

  // "Reveal in browser" from the Disk Usage explorer. Local targets go to the
  // local pane; a matching server target updates that session's remote pane.
  const revealTarget = useLayout((s) => s.revealTarget);
  const clearReveal = useLayout((s) => s.clearReveal);
  useEffect(() => {
    if (!revealTarget) return;
    if (revealTarget.sessionId === LOCAL_SESSION) {
      setLocalPath(revealTarget.path);
      clearReveal();
    } else if (revealTarget.sessionId === activeSessionId) {
      setRemotePaths((m) => ({ ...m, [activeSessionId]: revealTarget.path }));
      clearReveal();
    }
  }, [revealTarget, activeSessionId, clearReveal]);

  const uploadAll = (entries: DirEntry[]) => {
    if (!activeSessionId) return;
    enqueueUploads(activeSessionId, entries.map(toTransferItem), remotePath).catch(
      () => {}
    );
  };

  const downloadAll = (entries: DirEntry[]) => {
    if (!activeSessionId) return;
    enqueueDownloads(activeSessionId, entries.map(toTransferItem), localPath).catch(
      () => {}
    );
  };

  return (
    <div className="ghost-dual-browser relative flex h-full flex-1 gap-0 p-0">
      <FilePane
        paneId="local"
        title="Local Files"
        sessionId={LOCAL_SESSION}
        path={localPath}
        onPathChange={setLocalPath}
        onTransfer={uploadAll}
        onDrop={downloadAll}
        transferLabel="Upload"
      />
      <div className="ghost-pane-separator" aria-hidden="true" />
      <FilePane
        paneId="remote"
        title="Remote Server"
        sessionId={activeSessionId}
        path={remotePath}
        onPathChange={setRemotePath}
        onTransfer={downloadAll}
        onDrop={uploadAll}
        transferLabel="Download"
      />

      {syncOpen && activeSessionId && (
        <SyncDialog
          localPath={localPath}
          remotePath={remotePath}
          onClose={() => setSyncOpen(false)}
        />
      )}
    </div>
  );
}
