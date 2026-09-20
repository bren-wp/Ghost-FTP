import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { ReferenceSiteSidebar } from "./components/ReferenceSiteSidebar";
import { ReferenceStatusBar } from "./components/ReferenceStatusBar";
import { DualPaneBrowser } from "./components/DualPaneBrowser";
import { FileBrowser } from "./components/FileBrowser";
import { FileUiBridge } from "./components/FileUiBridge";
import { TerminalDock } from "./components/Terminal";
import { TransferQueue } from "./components/TransferQueue";
import { HostKeyModal } from "./components/HostKeyModal";
import { AuthPromptModal } from "./components/AuthPromptModal";
import { TitleBar } from "./components/TitleBar";
import { useConnections } from "./stores/connectionsStore";
import { useTransfers } from "./stores/transfersStore";
import { useLayout } from "./stores/layoutStore";
import {
  TerminalSquare,
  ArrowDownUp,
  Wifi,
  WifiOff,
  Edit3,
  X,
  Bell,
  CheckCircle2,
  AlertCircle,
  Info,
  AlertTriangle,
  Radio,
  Columns2,
  FolderSync,
} from "lucide-react";
import { useEditor } from "./stores/editorStore";
import { useToasts, type ToastVariant } from "./stores/toastStore";
import { useBridge } from "./stores/bridgeStore";
import { useSync } from "./stores/syncStore";
import { applyTransferEngineSettings, useSettings } from "./stores/settingsStore";
import { onDeepLink } from "./lib/ipc";
import { runDefaultBumps, runSettingsMigration } from "./lib/secretMigration";
import { initNotifications } from "./lib/notifications";
import { toast } from "./stores/toastStore";
import type { DeepLink, Protocol, ConnectionProfile } from "./lib/types";
import { PROTOCOL_DEFAULT_PORT } from "./lib/types";
import { Toaster } from "./components/ui/Toaster";
import { OverwriteDialogHost } from "./components/OverwriteModal";
import { CommandPalette } from "./components/CommandPalette";
import { KeyboardShortcutsDialog } from "./components/KeyboardShortcutsDialog";
import { AgentBridge, AgentBridgeHost } from "./components/AgentBridge";
import { AgentConsoleDock } from "./components/AgentConsole";
import { DiskUsageHost } from "./components/DiskUsage";
import { DirectoryDiffHost } from "./components/DirectoryDiff";
import { FindDuplicatesHost } from "./components/FindDuplicates";
import { FleetSearchHost } from "./components/FleetSearch";
import { SkillsHost } from "./components/SkillsPanel";
import { SnippetsHost } from "./components/SnippetsPanel";
import { useShortcuts } from "./hooks/useShortcuts";
import { relTime } from "./lib/format";
import { cn } from "./lib/cn";

const Settings = lazy(() =>
  import("./components/Settings").then((module) => ({ default: module.Settings }))
);
const QuickConnectionDialog = lazy(() =>
  import("./components/QuickConnectionDialog").then((module) => ({
    default: module.QuickConnectionDialog,
  }))
);
const SiteManagerDialog = lazy(() =>
  import("./components/SiteManagerDialog").then((module) => ({
    default: module.SiteManagerDialog,
  }))
);
const TransferCenterDialog = lazy(() =>
  import("./components/TransferCenterDialog").then((module) => ({
    default: module.TransferCenterDialog,
  }))
);
const GrantDialog = lazy(() =>
  import("./components/GrantDialog").then((module) => ({
    default: module.GrantDialog,
  }))
);
const ImportDialog = lazy(() =>
  import("./components/ImportDialog").then((module) => ({
    default: module.ImportDialog,
  }))
);
const AboutDialog = lazy(() =>
  import("./components/AboutDialog").then((module) => ({
    default: module.AboutDialog,
  }))
);


export default function App() {
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const activeProfile = profiles.find((p) => p.id === activeProfileId);
  const supportsTerminal = activeProfile?.protocol === "sftp";

  const terminalOpen = useLayout((s) => s.terminalOpen);
  const toggleTerminal = useLayout((s) => s.toggleTerminal);
  const terminalVisible = !!activeSessionId && terminalOpen && supportsTerminal;
  const consoleOpen = useLayout((s) => s.consoleOpen);
  const browserLayout = useSettings((s) => s.browserLayout);
  const dialog = useLayout((s) => s.dialog);
  const closeDialog = useLayout((s) => s.closeDialog);
  const connectionPrefill = useLayout((s) => s.connectionPrefill);
  const togglePanel = useTransfers((s) => s.togglePanel);
  const transferPanelOpen = useTransfers((s) => s.panelOpen);
  const activeTransfers = useTransfers((s) =>
    Object.values(s.byId).filter(
      (t) => t.status === "transferring" || t.status === "queued"
    ).length
  );
  const [editsMenuOpen, setEditsMenuOpen] = useState(false);
  const standaloneDialog =
    dialog === "settings" ||
    dialog === "siteManager" ||
    dialog === "transferCenter" ||
    dialog === "sync" ||
    dialog === "help" ||
    dialog === "updates" ||
    dialog === "about";

  useShortcuts();

  // Boot the Folder Sync store: fetch the current pairs and attach the
  // "foldersync://changed" listener so background syncs keep the UI live.
  useEffect(() => {
    let cleanup: (() => void) | undefined;
    let cancelled = false;
    useSync
      .getState()
      .init()
      .then((c) => {
        if (cancelled) c();
        else cleanup = c;
      });
    return () => {
      cancelled = true;
      cleanup?.();
    };
  }, []);

  // One-time migration of app settings from localStorage into ghostftp.db (Plan 12),
  // then any defaults that changed since this install first wrote its rows —
  // in that order, so the bumps apply on top of the imported values.
  useEffect(() => {
    void runSettingsMigration().then(runDefaultBumps).then(() => applyTransferEngineSettings());
  }, []);

  // Desktop notifications (Plan 16 Phase 3): OS toasts for the curated events
  // (transfer batch done/failed, folder-sync error, edit-in-place save failure),
  // gated by the `notifications` setting + window focus.
  useEffect(() => {
    const cleanup = initNotifications();
    return cleanup;
  }, []);

  return (
      <div className="ghost-app-shell flex h-full w-full flex-col">
      {!standaloneDialog && <TitleBar />}
      <DeepLinkListener />
      <Suspense fallback={<DialogLoading workspace={standaloneDialog} />}>
        {dialog === "settings" && <Settings onClose={closeDialog} />}
        {dialog === "sync" && <Settings onClose={closeDialog} initialSection="sync" />}
        {dialog === "newConnection" && (
          <QuickConnectionDialog
            prefill={connectionPrefill}
            onClose={closeDialog}
          />
        )}
        {dialog === "siteManager" && <SiteManagerDialog onClose={closeDialog} />}
        {dialog === "transferCenter" && (
          <TransferCenterDialog onClose={closeDialog} />
        )}
        {dialog === "import" && <ImportDialog onClose={closeDialog} />}
        {dialog === "grant" && <GrantDialog onClose={closeDialog} />}
        {dialog === "about" && <AboutDialog onClose={closeDialog} initialTab="about" />}
        {dialog === "help" && <AboutDialog onClose={closeDialog} initialTab="help" />}
        {dialog === "updates" && <AboutDialog onClose={closeDialog} initialTab="updates" />}
      </Suspense>
      {dialog === "agentBridge" && <AgentBridge onClose={closeDialog} />}
      <HostKeyModal />
      <AuthPromptModal />
      <Toaster />
      <AgentBridgeHost />
      <OverwriteDialogHost />
      <DiskUsageHost />
      <DirectoryDiffHost />
      <FindDuplicatesHost />
      <FleetSearchHost />
      <SkillsHost />
      <SnippetsHost />
      <CommandPalette />
      <KeyboardShortcutsDialog />
      {!standaloneDialog && (
      <>
      <div className="ghost-file-manager-body flex min-h-0 flex-1 overflow-hidden">
        <ReferenceSiteSidebar />
        <div className="ghost-file-manager-right flex min-w-0 flex-1 flex-col">
          <div className="ghost-main-workspace flex min-h-0 flex-1 overflow-hidden">
            <div className="flex min-w-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 overflow-hidden">
                <FileUiBridge>
                  {browserLayout === "dual" ? <DualPaneBrowser /> : <FileBrowser />}
                </FileUiBridge>
              </div>
              {consoleOpen && (
                <div className="h-64 border-t border-border">
                  <AgentConsoleDock />
                </div>
              )}
              {/* The terminal dock stays mounted even when hidden so background
                  shells survive connection-tab switches and toggling it closed. */}
              <div
                className={cn(
                  terminalVisible ? "h-72 border-t border-border" : "h-0 overflow-hidden"
                )}
              >
                <TerminalDock
                  sessionId={supportsTerminal ? activeSessionId : null}
                  visible={terminalVisible}
                />
              </div>
            </div>
          </div>
          <TransferQueue />
        </div>
      </div>
      <ReferenceStatusBar />
      </>
      )}
      </div>
  );
}

function DialogLoading({ workspace = false }: { workspace?: boolean }) {
  return (
    <div
      className={workspace
        ? "ghost-workspace-loading grid min-h-0 flex-1 place-items-center bg-[#041425]"
        : "ghost-transient-overlay fixed inset-0 z-modal grid place-items-center bg-[#041425]/92"}
      role="status"
      aria-live="polite"
    >
      <div className="rounded-lg border border-border bg-bg-panel px-5 py-3 text-sm text-text-muted shadow-elev-3">
        Opening Ghost FTP view…
      </div>
    </div>
  );
}

function StatusBar({
  terminalOpen,
  onToggleTerminal,
  terminalAvailable,
  transferPanelOpen,
  onToggleTransfers,
  activeTransfers,
  editsMenuOpen,
  onToggleEditsMenu,
  onCloseEditsMenu,
}: {
  terminalOpen: boolean;
  onToggleTerminal: () => void;
  terminalAvailable: boolean;
  transferPanelOpen: boolean;
  onToggleTransfers: () => void;
  activeTransfers: number;
  editsMenuOpen: boolean;
  onToggleEditsMenu: () => void;
  onCloseEditsMenu: () => void;
}) {
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const profiles = useConnections((s) => s.profiles);
  const profile = profiles.find((p) => p.id === activeProfileId);
  const connected = !!activeSessionId && !!profile;
  const edits = useEditor((s) => s.edits);
  const stopEditing = useEditor((s) => s.stopEditing);
  const editList = Object.values(edits);

  const history = useToasts((s) => s.history);
  const unread = useToasts((s) => s.unreadCount);
  const markAllRead = useToasts((s) => s.markAllRead);
  const clearHistory = useToasts((s) => s.clearHistory);
  const [notifOpen, setNotifOpen] = useState(false);

  const bridgeRunning = useBridge((s) => s.status.running);
  const consoleRunning = useBridge((s) =>
    s.console.filter((e) => e.status === "running").length
  );
  const syncPairs = useSync((s) => s.pairs);
  const syncRunning = syncPairs.filter((p) => p.running).length;
  const syncActive = syncPairs.filter((p) => p.state === "syncing").length;
  const openDialog = useLayout((s) => s.openDialog);
  const bridgeDialogOpen = useLayout((s) => s.dialog === "agentBridge");
  const consoleOpen = useLayout((s) => s.consoleOpen);
  const toggleConsole = useLayout((s) => s.toggleConsole);
  const browserLayout = useSettings((s) => s.browserLayout);
  const setBrowserLayout = useSettings((s) => s.setBrowserLayout);

  // Dismiss the status-bar popovers on Escape or a click outside, matching how
  // every other overlay closes (the hover-leave behavior stays as a bonus).
  const editsWrapRef = useRef<HTMLDivElement>(null);
  const notifWrapRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!notifOpen && !editsMenuOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      setNotifOpen(false);
      onCloseEditsMenu();
    };
    const onDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (notifOpen && !notifWrapRef.current?.contains(t)) setNotifOpen(false);
      if (editsMenuOpen && !editsWrapRef.current?.contains(t)) onCloseEditsMenu();
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("mousedown", onDown);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("mousedown", onDown);
    };
  }, [notifOpen, editsMenuOpen, onCloseEditsMenu]);

  return (
    <div className="flex h-7 min-w-0 shrink-0 items-center gap-2 overflow-hidden border-t border-border bg-bg-panel px-2 text-[11px]">
      {connected ? (
        <div className="flex min-w-0 items-center gap-1.5 pl-1">
          <div className="relative">
            <div
              className="h-1.5 w-1.5 rounded-full"
              style={{ background: profile.color || "rgb(var(--accent))" }}
            />
          </div>
          <span className="max-w-[18rem] shrink-0 truncate font-medium" title={profile.name}>
            {profile.name}
          </span>
          <span className="min-w-0 truncate font-mono text-text-dim">
            {profile.username}@{profile.host}:{profile.port}
          </span>
          <span className="shrink-0 rounded-sm bg-bg-subtle px-1 text-[9px] font-medium uppercase tracking-wider text-text-dim">
            {profile.protocol}
          </span>
        </div>
      ) : (
        <div className="flex items-center gap-1.5 pl-1 text-text-dim">
          <WifiOff size={11} />
          <span>Not connected</span>
        </div>
      )}
      <div className="flex-1" />
      {editList.length > 0 && (
        <div className="relative" ref={editsWrapRef}>
          <PillButton
            active={editsMenuOpen}
            onClick={onToggleEditsMenu}
            icon={<Edit3 size={11} />}
            badge={editList.length}
          >
            Editing
          </PillButton>
          {editsMenuOpen && (
            <div
              className="anim-modal absolute bottom-7 right-0 z-dropdown w-80 overflow-hidden rounded-md border border-border bg-bg-panel shadow-elev-3"
              onMouseLeave={onCloseEditsMenu}
            >
              <div className="border-b border-border bg-bg-subtle px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-text-muted">
                Live edits — saves upload automatically
              </div>
              {editList.map((e) => {
                const since = e.lastSavedAt
                  ? `${Math.max(1, Math.round((Date.now() - e.lastSavedAt) / 1000))}s ago`
                  : "no saves yet";
                return (
                  <div
                    key={e.editId}
                    className="flex items-start gap-2 border-b border-border-subtle px-3 py-2 last:border-0"
                  >
                    <Edit3
                      size={11}
                      className="mt-0.5 shrink-0 text-accent"
                    />
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-mono text-[11px]">
                        {e.remotePath}
                      </div>
                      {e.lastError ? (
                        <div className="text-[10px] text-danger">
                          {e.lastError}
                        </div>
                      ) : (
                        <div className="text-[10px] text-text-dim">
                          saved {since}
                        </div>
                      )}
                    </div>
                    <button
                      onClick={() => stopEditing(e.editId)}
                      className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-danger"
                      title="Stop editing (closes the watcher)"
                    >
                      <X size={11} />
                    </button>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
      <div className="relative" ref={notifWrapRef}>
        <PillButton
          active={notifOpen}
          onClick={() => {
            const next = !notifOpen;
            setNotifOpen(next);
            if (next) markAllRead();
          }}
          icon={<Bell size={11} />}
          badge={unread > 0 ? unread : undefined}
          title="Notifications"
        />
        {notifOpen && (
          <div
            className="anim-modal absolute bottom-7 right-0 z-dropdown w-80 overflow-hidden rounded-md border border-border bg-bg-panel shadow-elev-3"
            onMouseLeave={() => setNotifOpen(false)}
          >
            <div className="flex items-center justify-between border-b border-border bg-bg-subtle px-3 py-1.5">
              <span className="text-[10px] font-semibold uppercase tracking-wider text-text-muted">
                Notifications
              </span>
              {history.length > 0 && (
                <button
                  onClick={clearHistory}
                  className="text-[10px] text-text-dim hover:text-text"
                >
                  Clear
                </button>
              )}
            </div>
            {history.length === 0 ? (
              <div className="px-3 py-6 text-center text-[11px] text-text-dim">
                No notifications yet
              </div>
            ) : (
              <div className="max-h-72 overflow-y-auto">
                {history.slice(0, 50).map((n) => (
                  <div
                    key={n.id}
                    className="flex items-start gap-2 border-b border-border-subtle px-3 py-2 last:border-0"
                  >
                    <NotifIcon variant={n.variant} />
                    <div className="min-w-0 flex-1">
                      <div className="text-[11px] font-medium">{n.title}</div>
                      {n.message && (
                        <div className="truncate text-[10px] text-text-dim">
                          {n.message}
                        </div>
                      )}
                    </div>
                    <span className="shrink-0 text-[9px] text-text-dim">
                      {relTime(n.createdAt)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
      <PillButton
        onClick={() => openDialog("settings")}
        icon={
          <FolderSync
            size={11}
            className={cn(
              syncActive
                ? "animate-pulse text-warning"
                : syncRunning
                  ? "text-success"
                  : undefined
            )}
          />
        }
        badge={syncPairs.length > 0 ? syncPairs.length : undefined}
        title={
          syncPairs.length === 0
            ? "Folder Sync — keep folders mirrored to your servers"
            : `Folder Sync — ${syncRunning} running${
                syncActive ? `, ${syncActive} syncing` : ""
              } of ${syncPairs.length}`
        }
      >
        Sync
      </PillButton>
      <PillButton
        active={bridgeDialogOpen}
        onClick={() => openDialog("agentBridge")}
        icon={
          <Radio
            size={11}
            className={bridgeRunning ? "text-success" : undefined}
          />
        }
        title={`Agent Bridge${
          bridgeRunning ? " (running)" : ""
        } — let a local AI agent run commands on your servers`}
      >
        Bridge
      </PillButton>
      <PillButton
        active={consoleOpen}
        onClick={toggleConsole}
        icon={<TerminalSquare size={11} />}
        badge={consoleRunning > 0 ? consoleRunning : undefined}
        title="Agent console — live view of what the agent is doing over the bridge"
      >
        Console
      </PillButton>
      <PillButton
        active={browserLayout === "dual"}
        onClick={() =>
          setBrowserLayout(browserLayout === "dual" ? "single" : "dual")
        }
        icon={<Columns2 size={11} />}
        title="Split view — show local files next to the server"
      >
        Split
      </PillButton>
      <PillButton
        active={terminalOpen}
        onClick={onToggleTerminal}
        disabled={!activeSessionId || !terminalAvailable}
        icon={<TerminalSquare size={11} />}
        title={
          !terminalAvailable && activeSessionId
            ? "Terminal is only available for SFTP sessions"
            : undefined
        }
      >
        Terminal
      </PillButton>
      <PillButton
        active={transferPanelOpen}
        onClick={onToggleTransfers}
        icon={<ArrowDownUp size={11} />}
        badge={activeTransfers > 0 ? activeTransfers : undefined}
        title="Transfer queue"
      >
        Transfers
      </PillButton>
      {connected && (
        <div className="flex items-center gap-1 pl-2 pr-1 text-success">
          <Wifi size={11} />
        </div>
      )}
    </div>
  );
}

function PillButton({
  children,
  onClick,
  icon,
  active,
  disabled,
  badge,
  title,
}: {
  children?: React.ReactNode;
  onClick: () => void;
  icon: React.ReactNode;
  active?: boolean;
  disabled?: boolean;
  badge?: number;
  title?: string;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title={title}
      className={cn(
        "flex items-center gap-1.5 rounded-md px-2 py-0.5 transition-colors disabled:opacity-40 disabled:cursor-not-allowed",
        active
          ? "bg-accent-soft text-accent"
          : "text-text-muted hover:bg-bg-hover hover:text-text"
      )}
    >
      {icon}
      {children}
      {badge !== undefined && (
        <span className="ml-0.5 rounded-full bg-accent-strong px-1.5 py-0 text-[9px] font-semibold text-white">
          {badge}
        </span>
      )}
    </button>
  );
}

/// Listens for ghostftp:// deep links (from a hosting panel like a hosting panel) and
/// opens the New Connection editor prefilled. Never auto-connects — the user
/// reviews the target and clicks Connect / Pair, because any web page can fire
/// a protocol handler. `ghostftp://terminal` is the one shortcut: if the named
/// server is ALREADY connected it focuses that session and opens the terminal
/// dock inside the existing Ghost FTP window (no second app window and no new
/// connection is ever made); otherwise it falls back to the editor.
function DeepLinkListener() {
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const openGrant = useLayout((s) => s.openGrant);
  useEffect(() => {
    const un = onDeepLink((dl) => {
      if (dl.action === "grant") {
        // Access grant: never prefill the editor — open the consent dialog,
        // which fetches the manifest and only acts on Accept.
        if (dl.issuer && dl.token) {
          openGrant({ issuer: dl.issuer, token: dl.token, name: dl.name });
        } else {
          toast.error(
            "Invalid grant link",
            "The link is missing its issuer or token."
          );
        }
        return;
      }
      if (dl.action === "terminal") {
        const { profiles, sessions } = useConnections.getState();
        const norm = (v?: string | null) => (v ?? "").trim().toLowerCase();
        const match = profiles.find(
          (p) =>
            (dl.name && norm(p.name) === norm(dl.name)) ||
            (dl.host && norm(p.host) === norm(dl.host))
        );
        const live = match
          ? sessions.find((s) => s.profileId === match.id)
          : undefined;
        if (match && live && match.protocol === "sftp") {
          useConnections.getState().setActiveSession(live.sessionId);
          useLayout.getState().closeDialog();
          useLayout.getState().setTerminalOpen(true);
          toast.info("Terminal ready", match.name);
          return;
        }
        if (match && live) {
          toast.warning(
            "No terminal for this server",
            `${match.name} is a ${match.protocol} connection — terminals need SFTP.`
          );
          return;
        }
        // Not connected: a link never auto-connects, so the best we can do is
        // open the prefilled editor with a single explanatory toast.
        openNewConnection(deepLinkToPrefill(dl));
        toast.warning(
          "Server not connected",
          "Connect it first, then the terminal link can open the in-app shell."
        );
        return;
      }
      const prefill = deepLinkToPrefill(dl);
      openNewConnection(prefill);
      toast.info(
        dl.action === "pair" ? "Pair a machine" : "Open connection",
        prefill.name || prefill.host || undefined
      );
    });
    return () => {
      void un.then((f) => f());
    };
  }, [openNewConnection, openGrant]);
  return null;
}

/// Map a parsed deep link to editor seed values. `pair` links become a
/// ghostftp-agent profile; everything else a server profile of the named protocol.
function deepLinkToPrefill(dl: DeepLink): Partial<ConnectionProfile> {
  const known: Protocol[] = ["sftp", "ftp", "ftps", "s3", "azure", "gcs", "webdav", "http", "dropbox", "onedrive", "gdrive", "box", "shopify", "hubspot", "dynamics", "ghostftp-agent"];
  const protocol: Protocol =
    dl.action === "pair"
      ? "ghostftp-agent"
      : known.includes(dl.protocol as Protocol)
      ? (dl.protocol as Protocol)
      : "sftp";
  return {
    protocol,
    name: dl.name ?? undefined,
    host: dl.host ?? undefined,
    port: dl.port ?? PROTOCOL_DEFAULT_PORT[protocol],
    username: dl.username ?? undefined,
    defaultRemotePath: dl.path ?? undefined,
    bucket: dl.bucket ?? undefined,
    region: dl.region ?? undefined,
    endpoint: dl.endpoint ?? undefined,
    account: dl.account ?? undefined,
  };
}

function NotifIcon({ variant }: { variant: ToastVariant }) {
  const map = {
    info: { Icon: Info, cls: "text-info" },
    success: { Icon: CheckCircle2, cls: "text-success" },
    error: { Icon: AlertCircle, cls: "text-danger" },
    warning: { Icon: AlertTriangle, cls: "text-warning" },
  } as const;
  const { Icon, cls } = map[variant];
  return <Icon size={12} className={cn("mt-0.5 shrink-0", cls)} />;
}
