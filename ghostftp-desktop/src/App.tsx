import { useEffect } from "react";
import { ReferenceSiteSidebar } from "./components/ReferenceSiteSidebar";
import { ReferenceStatusBar } from "./components/ReferenceStatusBar";
import { DualPaneBrowser } from "./components/DualPaneBrowser";
import { FileBrowser } from "./components/FileBrowser";
import { FileUiBridge } from "./components/FileUiBridge";
import { TerminalDock } from "./components/Terminal";
import { HostKeyModal } from "./components/HostKeyModal";
import { AuthPromptModal } from "./components/AuthPromptModal";
import { TitleBar } from "./components/TitleBar";
import { useConnections } from "./stores/connectionsStore";
import { useTransfers } from "./stores/transfersStore";
import { type AppDialog, useLayout } from "./stores/layoutStore";
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
import { Settings } from "./components/Settings";
import { QuickConnectionDialog } from "./components/QuickConnectionDialog";
import { SiteManagerDialog } from "./components/SiteManagerDialog";
import { TransferCenterDialog } from "./components/TransferCenterDialog";
import { GrantDialog } from "./components/GrantDialog";
import { ImportDialog } from "./components/ImportDialog";
import { AboutDialog } from "./components/AboutDialog";
import { useUpdater } from "./stores/updaterStore";
import { WorkspaceErrorBoundary } from "./components/WorkspaceErrorBoundary";

const WORKSPACE_DIALOGS = new Set<AppDialog>([
  "settings",
  "siteManager",
  "transferCenter",
  "sync",
  "help",
  "updates",
  "about",
]);

function workspaceFor(dialog: AppDialog | null, returnDialog: AppDialog | null): AppDialog | null {
  if (dialog && WORKSPACE_DIALOGS.has(dialog)) return dialog;
  if (returnDialog && WORKSPACE_DIALOGS.has(returnDialog)) return returnDialog;
  return null;
}

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
  const returnDialog = useLayout((s) => s.returnDialog);
  const closeDialog = useLayout((s) => s.closeDialog);
  const showFiles = useLayout((s) => s.showFiles);
  const connectionPrefill = useLayout((s) => s.connectionPrefill);
  const workspace = workspaceFor(dialog, returnDialog);
  const fileManager = workspace === null;
  const workspaceLabel =
    workspace === "settings"
      ? "Settings"
      : workspace === "sync"
        ? "Sync & Backup"
        : workspace === "siteManager"
          ? "Sites"
          : workspace === "transferCenter"
            ? "Transfers"
            : workspace === "about" || workspace === "help" || workspace === "updates"
              ? "Help & About"
              : "Files";

  useShortcuts();

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

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    let cancelled = false;
    void useSync
      .getState()
      .init()
      .then((nextCleanup) => {
        if (cancelled) nextCleanup();
        else cleanup = nextCleanup;
      })
      .catch((error) => {
        console.error("Couldn't initialize Sync & Backup", error);
        toast.error("Couldn't initialize Sync & Backup", String(error));
      });

    return () => {
      cancelled = true;
      cleanup?.();
    };
  }, []);

  useEffect(() => {
    void runSettingsMigration()
      .then(runDefaultBumps)
      .then(() => applyTransferEngineSettings())
      .catch((error) => {
        console.error("Couldn't initialize application settings", error);
        toast.error("Couldn't initialize application settings", String(error));
      });
  }, []);

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    let cancelled = false;

    void (async () => {
      await useTransfers.getState().loadInitial();
      const nextCleanup = await useTransfers.getState().initListeners();
      if (cancelled) nextCleanup();
      else cleanup = nextCleanup;
    })().catch((error) => {
      console.error("Couldn't initialize transfer activity", error);
      toast.error("Couldn't initialize transfer activity", String(error));
    });

    return () => {
      cancelled = true;
      cleanup?.();
    };
  }, []);

  useEffect(() => {
    const cleanup = initTransferScheduler();
    return cleanup;
  }, []);

  useEffect(() => {
    const cleanup = initNotifications();
    return cleanup;
  }, []);

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    let cancelled = false;

    void useUpdater.getState().init()
      .then((nextCleanup) => {
        if (cancelled) nextCleanup();
        else cleanup = nextCleanup;
      })
      .catch((error) => {
        // The quiet updater check is non-blocking; an unexpected initialization
        // failure must remain observable without interrupting application launch.
        console.error("Couldn't initialize app updates", error);
      });

    return () => {
      cancelled = true;
      cleanup?.();
    };
  }, []);

  return (
    <div className="ghost-app-shell flex h-full w-full flex-col">
      <TitleBar />
      <DeepLinkListener />

      <div className="ghost-app-body flex min-h-0 flex-1 overflow-hidden">
        <ReferenceSiteSidebar />
        <div className="ghost-content-shell flex min-w-0 flex-1 flex-col overflow-hidden">
          <div className="ghost-main-workspace min-h-0 flex-1 overflow-hidden">
            <WorkspaceErrorBoundary
              label={workspaceLabel}
              resetKey={workspace ?? "files"}
              onReturnToFiles={showFiles}
            >
              {workspace === "settings" && <Settings onClose={closeDialog} />}
              {workspace === "sync" && <Settings onClose={closeDialog} initialSection="sync" />}
              {workspace === "siteManager" && <SiteManagerDialog onClose={closeDialog} />}
              {workspace === "transferCenter" && <TransferCenterDialog onClose={closeDialog} />}
              {workspace === "about" && <AboutDialog onClose={closeDialog} initialTab="about" />}
              {workspace === "help" && <AboutDialog onClose={closeDialog} initialTab="help" />}
              {workspace === "updates" && <AboutDialog onClose={closeDialog} initialTab="updates" />}

              {fileManager && (
                <div className="flex h-full min-h-0 flex-col">
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
              )}
            </WorkspaceErrorBoundary>
          </div>
        </div>
      </div>

      {fileManager && <ReferenceStatusBar />}

      {dialog === "newConnection" && (
        <QuickConnectionDialog
          prefill={connectionPrefill}
          onClose={closeDialog}
          saveByDefault={returnDialog === "siteManager"}
          cancelLabel={returnDialog === "siteManager" ? "Back to Sites" : "Cancel"}
        />
      )}
      {dialog === "import" && <ImportDialog onClose={closeDialog} />}
      {dialog === "grant" && <GrantDialog onClose={closeDialog} />}
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
    </div>
  );
}

function DeepLinkListener() {
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const openGrant = useLayout((s) => s.openGrant);
  useEffect(() => {
    let disposed = false;
    let unlisten: (() => void) | undefined;

    const registration = onDeepLink((dl) => {
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
          useLayout.getState().showFiles();
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
          "Connect it first, then the terminal opens inside the main Ghost FTP window."
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

    void registration
      .then((cleanup) => {
        if (disposed) cleanup();
        else unlisten = cleanup;
      })
      .catch((error) => {
        console.error("Couldn't register Ghost FTP deep-link listener", error);
        toast.error("Couldn't register deep-link handler", String(error));
      });

    return () => {
      disposed = true;
      unlisten?.();
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
