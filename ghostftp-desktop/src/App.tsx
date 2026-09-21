import { lazy, Suspense, useEffect } from "react";
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
import { useLayout } from "./stores/layoutStore";
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
import { cn } from "./lib/cn";
import { initTransferScheduler } from "./stores/transferScheduleStore";

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
  const terminalVisible = !!activeSessionId && terminalOpen && supportsTerminal;
  const consoleOpen = useLayout((s) => s.consoleOpen);
  const browserLayout = useSettings((s) => s.browserLayout);
  const dialog = useLayout((s) => s.dialog);
  const closeDialog = useLayout((s) => s.closeDialog);
  const connectionPrefill = useLayout((s) => s.connectionPrefill);
  const standaloneDialog =
    dialog === "settings" ||
    dialog === "siteManager" ||
    dialog === "transferCenter" ||
    dialog === "sync" ||
    dialog === "help" ||
    dialog === "updates" ||
    dialog === "cloudStorage" ||
    dialog === "schedules" ||
    dialog === "activityLogs" ||
    dialog === "about";

  useShortcuts();

  // Native CI visual evidence can request a real application surface through
  // the allow-listed GHOSTFTP_QA_VIEW environment variable injected by Rust.
  // This only selects an existing view; it never seeds servers, transfers,
  // credentials, connection state, or other fake production data.
  useEffect(() => {
    const qaView = (
      globalThis as typeof globalThis & { __GHOSTFTP_QA_VIEW__?: string }
    ).__GHOSTFTP_QA_VIEW__;
    if (!qaView || qaView === "main" || qaView === "properties") return;

    const layout = useLayout.getState();
    if (qaView === "newConnection") {
      layout.openNewConnection();
      return;
    }
    if (
      qaView === "siteManager" ||
      qaView === "settings" ||
      qaView === "transferCenter" ||
      qaView === "about"
    ) {
      layout.openDialog(qaView);
    }
  }, []);

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

  // Keep Transfer Center schedules running even when another workspace is open.
  useEffect(() => {
    const cleanup = initTransferScheduler();
    return cleanup;
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
        {dialog === "cloudStorage" && <SiteManagerDialog onClose={closeDialog} initialView="cloud" />}
        {dialog === "transferCenter" && <TransferCenterDialog onClose={closeDialog} />}
        {dialog === "schedules" && <TransferCenterDialog onClose={closeDialog} initialFocus="scheduler" />}
        {dialog === "activityLogs" && <TransferCenterDialog onClose={closeDialog} initialFocus="log" />}
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
      <div className="ghost-file-manager-body flex min-h-0 flex-1 flex-col overflow-hidden">
        <div className="ghost-file-manager-top flex min-h-0 min-w-0 flex-1 overflow-hidden">
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
          </div>
        </div>
        <TransferQueue />
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
