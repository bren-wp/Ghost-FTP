import { useEffect, useRef, useState } from "react";
import { FileUp, FolderUp } from "lucide-react";
import { open } from "@tauri-apps/plugin-dialog";
import { downloadDir } from "@tauri-apps/api/path";
import { FilePane } from "@ghostftp/file-ui";
import { ContextMenu, type MenuItem } from "./ContextMenu";
import { useConnections } from "@/stores/connectionsStore";
import { useTransfers, toTransferItem } from "@/stores/transfersStore";
import { useLayout } from "@/stores/layoutStore";
import { useSettings } from "@/stores/settingsStore";
import { toast } from "@/stores/toastStore";
import { LOCAL_SESSION, type DirEntry } from "@/lib/types";
import { toastError } from "@/lib/errors";

// Single-pane file browser: the active server is the main view. Uploads go
// through a native OS picker (Upload button); downloads land in the Downloads
// folder. A "Local" rail bubble swaps the pane to the local filesystem. The
// classic two-pane (local + remote) layout still lives in DualPaneBrowser and is
// chosen via the `browserLayout` setting.
export function FileBrowser() {
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const profile = profiles.find((p) => p.id === activeProfileId) || null;

  const browseLocal = useLayout((s) => s.browseLocal);
  const setBrowseLocal = useLayout((s) => s.setBrowseLocal);
  const revealTarget = useLayout((s) => s.revealTarget);
  const clearReveal = useLayout((s) => s.clearReveal);
  const defaultDownloadFolder = useSettings((s) => s.defaultDownloadFolder);

  const enqueueDownloads = useTransfers((s) => s.enqueueDownloads);
  const enqueueUploads = useTransfers((s) => s.enqueueUploads);
  const activeCount = useTransfers(
    (s) =>
      Object.values(s.byId).filter(
        (t) => t.status === "transferring" || t.status === "queued"
      ).length
  );

  const isWindows = navigator.userAgent.includes("Windows");
  const [localPath, setLocalPath] = useState(isWindows ? "C:\\" : "/");
  const [remotePaths, setRemotePaths] = useState<Record<string, string>>({});

  const serverSid = activeSessionId;
  const serverRemotePath =
    (serverSid && remotePaths[serverSid]) || profile?.defaultRemotePath || ".";

  // Honour a "reveal in browser" request from the Disk Usage explorer: point the
  // pane at the target directory on the matching session, then clear the signal.
  useEffect(() => {
    if (!revealTarget) return;
    if (revealTarget.sessionId === LOCAL_SESSION) {
      setBrowseLocal(true);
      setLocalPath(revealTarget.path);
      clearReveal();
    } else if (revealTarget.sessionId === serverSid) {
      setBrowseLocal(false);
      setRemotePaths((m) => ({ ...m, [serverSid]: revealTarget.path }));
      clearReveal();
    }
    // A reveal for a non-active session is left pending until it becomes active.
  }, [revealTarget, serverSid, setBrowseLocal, clearReveal]);

  // Re-list the directory whenever a batch of transfers drains to zero, so an
  // upload's result shows up without a manual refresh.
  const [reloadToken, setReloadToken] = useState(0);
  const prevActive = useRef(0);
  useEffect(() => {
    if (prevActive.current > 0 && activeCount === 0) {
      setReloadToken((t) => t + 1);
    }
    prevActive.current = activeCount;
  }, [activeCount]);

  // Download destination: the setting if present, else the OS Downloads dir.
  const downloadsRef = useRef<string | null>(null);
  const resolveDest = async (): Promise<string | null> => {
    if (defaultDownloadFolder) return defaultDownloadFolder;
    if (downloadsRef.current == null) {
      try {
        downloadsRef.current = await downloadDir();
      } catch {
        downloadsRef.current = "";
      }
    }
    return downloadsRef.current || null;
  };

  // What the single pane shows: the local filesystem, or the active server.
  const target = browseLocal ? LOCAL_SESSION : serverSid;
  const path = browseLocal ? localPath : serverRemotePath;
  const setPath = (p: string) => {
    if (browseLocal) setLocalPath(p);
    else if (serverSid) setRemotePaths((m) => ({ ...m, [serverSid]: p }));
  };
  const title = browseLocal ? "Local" : profile ? profile.name : "Server";

  // ---- Upload (server view) — native picker into the current remote dir ----
  const [uploadMenu, setUploadMenu] = useState<{ x: number; y: number } | null>(
    null
  );
  const pickAndUpload = async (kind: "files" | "folder") => {
    if (!serverSid) return;
    try {
      const picked = await open({
        multiple: kind === "files",
        directory: kind === "folder",
        title: kind === "folder" ? "Upload a folder" : "Upload files",
      });
      if (!picked) return;
      const paths = Array.isArray(picked) ? picked : [picked];
      const items = paths.map((p) => ({
        path: p,
        kind: (kind === "folder" ? "directory" : "file") as "directory" | "file",
      }));
      await enqueueUploads(serverSid, items, serverRemotePath);
    } catch (error) {
      toastError(error, kind === "folder" ? "Couldn't upload folder" : "Couldn't upload files");
    }
  };
  const onUpload = (e: React.MouseEvent) => {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    setUploadMenu({ x: r.left, y: r.bottom + 4 });
  };
  const uploadMenuItems: MenuItem[] = [
    {
      label: "Upload files…",
      icon: <FileUp size={14} />,
      onClick: () => pickAndUpload("files"),
    },
    {
      label: "Upload folder…",
      icon: <FolderUp size={14} />,
      onClick: () => pickAndUpload("folder"),
    },
  ];

  // ---- Selection transfer ----
  const onTransfer = async (entries: DirEntry[]) => {
    if (browseLocal) {
      // Local view → upload the selection to the active server's current dir.
      if (!serverSid) {
        toast.info("Not connected", "Connect to a server to upload there.");
        return;
      }
      try {
        await enqueueUploads(serverSid, entries.map(toTransferItem), serverRemotePath);
      } catch (error) {
        toastError(error, "Couldn't queue upload");
      }
      return;
    }
    // Server view → download the selection to the Downloads folder.
    if (!serverSid) return;
    const dest = await resolveDest();
    if (!dest) {
      toast.error(
        "No download folder",
        "Set a default download folder in Settings."
      );
      return;
    }
    try {
      await enqueueDownloads(serverSid, entries.map(toTransferItem), dest);
    } catch (error) {
      toastError(error, "Couldn't queue download");
    }
  };

  return (
    <div className="relative flex h-full flex-1">
      <FilePane
        paneId={browseLocal ? "local" : "remote"}
        title={title}
        sessionId={target}
        path={path}
        onPathChange={setPath}
        onTransfer={onTransfer}
        onUpload={browseLocal ? undefined : onUpload}
        transferLabel={browseLocal ? "Upload" : "Download"}
        reloadToken={reloadToken}
      />
      {uploadMenu && (
        <ContextMenu
          x={uploadMenu.x}
          y={uploadMenu.y}
          items={uploadMenuItems}
          onClose={() => setUploadMenu(null)}
        />
      )}
    </div>
  );
}
