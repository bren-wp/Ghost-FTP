import type { ReactNode } from "react";
import {
  Plug,
  Unplug,
  Download,
  Plus,
  Settings as SettingsIcon,
  Info,
  TerminalSquare,
  ArrowDownUp,
  RefreshCw,
  Keyboard,
  Palette,
  SunMoon,
  Radio,
  Wand2,
  Braces,
  SplitSquareHorizontal,
  SplitSquareVertical,
  Maximize2,
  X,
} from "lucide-react";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import { useSettings, APP_THEMES } from "@/stores/settingsStore";
import { useSkills } from "@/stores/skillsStore";
import { useSnippets } from "@/stores/snippetsStore";
import { useTerminals } from "@/stores/terminalsStore";
import { TERMINAL_CHORDS } from "@/lib/terminalChords";
import { toastError } from "@/lib/errors";

export interface Command {
  id: string;
  title: string;
  group: string;
  icon?: ReactNode;
  combo?: string; // canonical, e.g. "mod+t"
  keywords?: string;
  enabled?: boolean; // default true
  run: () => void;
}

// Single source of truth for app actions. Reads stores directly and returns
// freshly-bound command objects, so the palette, the shortcut hook and the
// cheat-sheet all stay in sync with current state.
export function useCommands(): Command[] {
  const profiles = useConnections((s) => s.profiles);
  const activeSessionId = useConnections((s) => s.activeSessionId);
  const activeProfileId = useConnections((s) => s.activeProfileId);
  const connect = useConnections((s) => s.connect);
  const disconnect = useConnections((s) => s.disconnect);

  const toggleTerminal = useLayout((s) => s.toggleTerminal);
  const terminalOpen = useLayout((s) => s.terminalOpen);
  const toggleConsole = useLayout((s) => s.toggleConsole);
  const openDialog = useLayout((s) => s.openDialog);
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const setShortcutsOpen = useLayout((s) => s.setShortcutsOpen);
  const openSkills = useSkills((s) => s.openPanel);
  const snippets = useSnippets((s) => s.snippets);

  const appTheme = useSettings((s) => s.appTheme);
  const setAppTheme = useSettings((s) => s.setAppTheme);

  const activeProfile = profiles.find((p) => p.id === activeProfileId);
  const supportsTerminal = activeProfile?.protocol === "sftp";

  const cmds: Command[] = [
    {
      id: "new-connection",
      title: "New Connection…",
      group: "File",
      icon: <Plus size={14} />,
      combo: "mod+n",
      run: () => openNewConnection(),
    },
    {
      id: "import",
      title: "Import Connections…",
      group: "File",
      icon: <Download size={14} />,
      run: () => openDialog("import"),
    },
    {
      id: "settings",
      title: "Open Settings",
      group: "General",
      icon: <SettingsIcon size={14} />,
      combo: "mod+,",
      run: () => openDialog("settings"),
    },
    {
      id: "shortcuts",
      title: "Keyboard Shortcuts",
      group: "General",
      icon: <Keyboard size={14} />,
      combo: "mod+/",
      run: () => setShortcutsOpen(true),
    },
    {
      id: "agent-bridge",
      title: "Agent Bridge…",
      group: "General",
      icon: <Radio size={14} />,
      keywords: "agent bridge mcp claude code remote exec",
      run: () => openDialog("agentBridge"),
    },
    {
      id: "fleet-skills",
      title: "Fleet Skills…",
      group: "General",
      icon: <Wand2 size={14} />,
      keywords: "skill skills fleet automation run command multi server mcp agent",
      run: () => openSkills(),
    },
    {
      id: "snippets",
      title: "Snippets…",
      group: "General",
      icon: <Braces size={14} />,
      keywords: "snippet snippets command template variable insert manage terminal",
      run: () => useSnippets.getState().openPanel(),
    },
    {
      id: "agent-console",
      title: "Toggle Agent console (live)",
      group: "General",
      icon: <TerminalSquare size={14} />,
      keywords: "agent console live output stream terminal bridge watch dock",
      run: () => toggleConsole(),
    },
    {
      id: "about",
      title: "About Ghost FTP",
      group: "General",
      icon: <Info size={14} />,
      run: () => openDialog("about"),
    },
    {
      id: "reload",
      title: "Reload Window",
      group: "General",
      icon: <RefreshCw size={14} />,
      combo: "mod+r",
      run: () => window.location.reload(),
    },
    {
      id: "open-transfers",
      title: "Open Transfers",
      group: "View",
      icon: <ArrowDownUp size={14} />,
      combo: "mod+t",
      run: () => openDialog("transferCenter"),
    },
    {
      id: "toggle-terminal",
      title: "Toggle Terminal",
      group: "View",
      icon: <TerminalSquare size={14} />,
      combo: "mod+`",
      enabled: !!activeSessionId && supportsTerminal,
      run: toggleTerminal,
    },
    {
      id: "switch-theme",
      title: "Toggle Light / Dark Theme",
      group: "View",
      icon: <SunMoon size={14} />,
      combo: "mod+shift+t",
      run: () => setAppTheme(appTheme === "light" ? "dark" : "light"),
    },
  ];

  // Terminal split controls — only while a terminal is on screen. They act on
  // the active tab's active pane; the chords are swallowed by xterm (see
  // terminalChords) so they never reach the shell.
  const terminalActive = terminalOpen && supportsTerminal && !!activeSessionId;
  const activeTab = () => {
    const t = useTerminals.getState();
    return t.tabs.find((x) => x.id === t.activeId) ?? null;
  };
  cmds.push(
    {
      id: "term-split-right",
      title: "Terminal: Split right",
      group: "Terminal",
      icon: <SplitSquareHorizontal size={14} />,
      combo: TERMINAL_CHORDS.splitRight,
      enabled: terminalActive,
      run: () => {
        const tab = activeTab();
        if (tab) useTerminals.getState().splitActivePane(tab.id, "row");
      },
    },
    {
      id: "term-split-down",
      title: "Terminal: Split down",
      group: "Terminal",
      icon: <SplitSquareVertical size={14} />,
      combo: TERMINAL_CHORDS.splitDown,
      enabled: terminalActive,
      run: () => {
        const tab = activeTab();
        if (tab) useTerminals.getState().splitActivePane(tab.id, "col");
      },
    },
    {
      id: "term-zoom-pane",
      title: "Terminal: Zoom / unzoom pane",
      group: "Terminal",
      icon: <Maximize2 size={14} />,
      combo: TERMINAL_CHORDS.zoom,
      enabled: terminalActive,
      run: () => {
        const tab = activeTab();
        if (tab) useTerminals.getState().toggleZoom(tab.id);
      },
    },
    {
      id: "term-close-pane",
      title: "Terminal: Close pane",
      group: "Terminal",
      icon: <X size={14} />,
      combo: TERMINAL_CHORDS.closePane,
      enabled: terminalActive,
      run: () => {
        const tab = activeTab();
        if (tab) useTerminals.getState().closePane(tab.id, tab.activePaneId);
      },
    }
  );

  if (activeSessionId) {
    cmds.push({
      id: "disconnect",
      title: "Disconnect",
      group: "Connection",
      icon: <Unplug size={14} />,
      run: () => disconnect(),
    });
  }

  for (const p of profiles) {
    cmds.push({
      id: `connect:${p.id}`,
      title: `Connect: ${p.name}`,
      group: "Connection",
      icon: <Plug size={14} />,
      keywords: `${p.username} ${p.host} ${p.protocol}`,
      enabled: activeProfileId !== p.id,
      run: () => {
        void connect(p.id).catch((error) => toastError(error, "Connection failed"));
      },
    });
  }

  for (const t of APP_THEMES) {
    cmds.push({
      id: `theme:${t.value}`,
      title: `Theme: ${t.label}`,
      group: "Theme",
      icon: <Palette size={14} />,
      enabled: appTheme !== t.value,
      run: () => setAppTheme(t.value),
    });
  }

  // Snippet insert commands — only actionable when a shell is available. The
  // insert targets the focused terminal; the store toasts if there's none.
  for (const s of snippets) {
    cmds.push({
      id: `snippet:${s.id}`,
      title: `Insert snippet: ${s.name}`,
      group: "Snippets",
      icon: <Braces size={14} />,
      keywords: `snippet ${s.folder ?? ""} ${s.body}`,
      enabled: !!activeSessionId && supportsTerminal,
      run: () => useSnippets.getState().requestInsert(s),
    });
  }

  return cmds;
}
