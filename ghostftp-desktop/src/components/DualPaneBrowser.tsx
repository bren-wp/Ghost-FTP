import { useEffect, useRef, useState } from "react";
import { FilePane } from "@ghostftp/file-ui";
import { open } from "@tauri-apps/plugin-dialog";
import { SyncDialog } from "./SyncDialog";
import { useConnections } from "@/stores/connectionsStore";
import { useTransfers, toTransferItem } from "@/stores/transfersStore";
import { useLayout } from "@/stores/layoutStore";
import { LOCAL_SESSION } from "@/lib/types";
import type { DirEntry } from "@/lib/types";
import { toastError } from "@/lib/errors";

export function DualPaneBrowser() {
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const profile = profiles.find((p) => p.id === activeProfileId) || null;
  const enqueueUploads = useTransfers((s) => s.enqueueUploads);
  const enqueueDownloads = useTransfers((s) => s.enqueueDownloads);
  const activeCount = useTransfers((s) =>
    Object.values(s.byId).filter((transfer) =>
      transfer.status === "transferring" ||
      transfer.status === "queued" ||
      transfer.status === "paused"
    ).length
  );
  const [reloadToken, setReloadToken] = useState(0);
  const previousActiveCount = useRef(0);

  // A transfer batch changes files on disk or on the server. Re-list both
  // real directories when the batch drains, as the single-pane UI already does.
  useEffect(() => {
    if (previousActiveCount.current > 0 && activeCount === 0) {
      setReloadToken((value) => value + 1);
    }
    previousActiveCount.current = activeCount;
  }, [activeCount]);

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

  useEffect(() => {
    const pickFromSharedToolbar = () => {
      if (!activeSessionId) return;
      void (async () => {
        try {
          const picked = await open({ multiple: true, directory: false, title: "Upload files" });
          if (!picked) return;
          const paths = Array.isArray(picked) ? picked : [picked];
          if (paths.length === 0) return;
          await enqueueUploads(
            activeSessionId,
            paths.map((path) => ({ path, kind: "file" as const })),
            remotePath,
          );
        } catch (error) {
          toastError(error, "Couldn't upload files");
        }
      })();
    };
    window.addEventListener("ghostftp:pick-upload", pickFromSharedToolbar);
    return () => window.removeEventListener("ghostftp:pick-upload", pickFromSharedToolbar);
  }, [activeSessionId, remotePath, enqueueUploads]);

  const uploadAll = (entries: DirEntry[]) => {
    if (!activeSessionId) return;
    void enqueueUploads(activeSessionId, entries.map(toTransferItem), remotePath).catch(
      (error) => toastError(error, "Couldn't queue upload")
    );
  };

  const downloadAll = (entries: DirEntry[]) => {
    if (!activeSessionId) return;
    void enqueueDownloads(activeSessionId, entries.map(toTransferItem), localPath).catch(
      (error) => toastError(error, "Couldn't queue download")
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
        reloadToken={reloadToken}
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
        reloadToken={reloadToken}
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
