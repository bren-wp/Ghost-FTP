import { Fragment, useEffect, useMemo, useRef, useState } from "react";
import {
  Plus,
  Download,
  Search,
  Radio,
  Plug,
  Unplug,
  TerminalSquare,
  Pencil,
  Trash2,
  Power,
  Shield,
  X,
  HardDrive,
  ChevronsLeft,
  ChevronsRight,
  ChevronDown,
  ChevronRight,
  Folder,
  FolderOpen,
  FolderPlus,
  FolderMinus,
} from "lucide-react";
import { useConnections } from "@/stores/connectionsStore";
import { useBridge } from "@/stores/bridgeStore";
import { useLayout } from "@/stores/layoutStore";
import { useSettings } from "@/stores/settingsStore";
import { ProfileEditor } from "./ProfileEditor";
import { ImportDialog } from "./ImportDialog";
import { ContextMenu, type MenuItem } from "./ContextMenu";
import { ConfirmModal } from "./ConfirmModal";
import { Tooltip } from "./ui/Tooltip";
import { useDialog } from "@/hooks/useDialog";
import { monogram } from "@/lib/format";
import {
  PROTOCOL_DEFAULT_PORT,
  PROTOCOL_LABEL,
  type ConnectionProfile,
  type Protocol,
} from "@/lib/types";
import { cn } from "@/lib/cn";
import { BrandIcon, protocolIcon } from "@/lib/brandIcons";
import { BRAND_ICONS } from "@/lib/brandIconData";
import { toastError } from "@/lib/errors";

type RowState = "focused" | "connected" | "connecting" | "error" | "idle";

// Servers sort by protocol (SFTP first — the primary use), then by name, so the
// rail order stays learnable as connections come and go.
const GROUP_ORDER: Protocol[] = ["sftp", "ftps", "ftp", "s3", "azure", "gcs", "webdav", "http", "dropbox", "onedrive", "gdrive", "box", "shopify", "hubspot", "dynamics", "ghostftp-agent"];

const fallbackColor = "rgb(var(--accent))";

function profileAddress(p: ConnectionProfile): string {
  if (p.protocol === "s3") return `s3://${p.bucket ?? "?"}`;
  if (p.protocol === "azure") return `az://${p.account ?? "?"}/${p.bucket ?? "?"}`;
  if (p.protocol === "gcs") return `gs://${p.bucket ?? "?"}`;
  if (p.protocol === "webdav" || p.protocol === "http") return p.endpoint ?? p.host;
  if (p.protocol === "dropbox") return p.account ? `Dropbox · ${p.account}` : "Dropbox";
  if (p.protocol === "onedrive") return p.account ? `OneDrive · ${p.account}` : "OneDrive";
  if (p.protocol === "gdrive") return p.account ? `Drive · ${p.account}` : "Google Drive";
  if (p.protocol === "box") return p.account ? `Box · ${p.account}` : "Box";
  if (p.protocol === "shopify") return p.host;
  if (p.protocol === "hubspot") return p.host;
  if (p.protocol === "dynamics") return p.host;
  const port =
    p.port !== PROTOCOL_DEFAULT_PORT[p.protocol] ? `:${p.port}` : "";
  // A Ghost FTP Agent connection controls a machine, not a login — no username.
  if (p.protocol === "ghostftp-agent") return `${p.host}${port}`;
  return `${p.username}@${p.host}${port}`;
}

// The single connection navigator — a Discord-style vertical rail of round
// "server" bubbles. Replaces the old ConnectionManager sidebar and the
// ConnectionTabs strip: one bubble per saved server, status dot, hover name,
// click-to-act menu, plus a dedicated Agent Bridge section below the divider.
export function ServerRail() {
  const {
    profiles,
    sessions,
    activeProfileId,
    activeSessionId,
    connecting,
    error,
    loadProfiles,
    connect,
    disconnect,
    deleteProfile,
    setActiveSession,
    saveProfile,
    saveProfiles,
    reorderProfiles,
  } = useConnections();
  const setTerminalOpen = useLayout((s) => s.setTerminalOpen);
  const openDialog = useLayout((s) => s.openDialog);
  const browseLocal = useLayout((s) => s.browseLocal);
  const setBrowseLocal = useLayout((s) => s.setBrowseLocal);
  const browserLayout = useSettings((s) => s.browserLayout);
  // `pinned` is the persisted "keep it open" choice; the rail also flies open on
  // hover (Edge vertical-tabs style) — see `expanded` below.
  const pinned = useSettings((s) => s.railExpanded);
  const setRailExpanded = useSettings((s) => s.setRailExpanded);
  const collapsedGroups = useSettings((s) => s.railCollapsedGroups);
  const toggleRailGroup = useSettings((s) => s.toggleRailGroup);

  const enabledSessions = useBridge((s) => s.status.enabledSessions);
  const bridgeRunning = useBridge((s) => s.status.running);
  const setSessionAccess = useBridge((s) => s.setSessionAccess);
  const startBridge = useBridge((s) => s.start);
  const stopBridge = useBridge((s) => s.stop);

  const [editing, setEditing] = useState<ConnectionProfile | "new" | null>(null);
  const [importing, setImporting] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  // Which profile we last asked to connect — localizes the global
  // connecting/error flags onto the owning bubble (mirrors ConnectionManager).
  const [pendingId, setPendingId] = useState<string | null>(null);
  const [menu, setMenu] = useState<{
    x: number;
    y: number;
    items: MenuItem[];
  } | null>(null);
  const [hovering, setHovering] = useState(false);
  const hoverTimer = useRef<number | null>(null);

  // Drag-and-drop reorder state: the profile being dragged and where it would
  // land if dropped now (between two rows, into a group, at a section's end,
  // or at the very top of the rail).
  const [dragId, setDragId] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<
    | { kind: "row"; id: string; after: boolean }
    | { kind: "group"; name: string }
    | { kind: "sectionEnd"; group: string | undefined }
    | { kind: "top" }
    | null
  >(null);
  // Naming/renaming a group happens through a small prompt dialog.
  const [groupPrompt, setGroupPrompt] = useState<
    | { kind: "assign"; profile: ConnectionProfile }
    | { kind: "rename"; from: string }
    | null
  >(null);
  // Deleting a connection is irreversible (saved credentials are wiped from
  // the OS keychain too) — the context menu arms this instead of deleting
  // outright, and a ConfirmModal does the rest.
  const [pendingDelete, setPendingDelete] = useState<ConnectionProfile | null>(null);

  // Effective expanded state. Open when pinned, while hovering the collapsed
  // rail, while a rail menu / search popover is up (so the flyout doesn't
  // collapse out from under what you're clicking), or mid-drag.
  const expanded = pinned || hovering || !!menu || searchOpen || !!dragId;

  const onRailEnter = () => {
    if (pinned) return;
    if (hoverTimer.current) window.clearTimeout(hoverTimer.current);
    hoverTimer.current = window.setTimeout(() => setHovering(true), 120);
  };
  const onRailLeave = () => {
    if (hoverTimer.current) window.clearTimeout(hoverTimer.current);
    hoverTimer.current = window.setTimeout(() => setHovering(false), 140);
  };
  useEffect(
    () => () => {
      if (hoverTimer.current) window.clearTimeout(hoverTimer.current);
    },
    []
  );

  // Load profiles once, then auto-connect any flagged servers in sequence so
  // host-key prompts queue through the always-mounted HostKeyModal rather than
  // racing. Runs a single time per launch.
  const autoRan = useRef(false);
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      await loadProfiles();
      if (cancelled || autoRan.current) return;
      autoRan.current = true;
      const st = useConnections.getState();
      for (const p of st.profiles) {
        if (!p.autoConnect) continue;
        if (st.sessions.some((s) => s.profileId === p.id)) continue;
        try {
          await st.connect(p.id);
        } catch {
          // connect() already surfaces failures as a toast.
        }
      }
    })().catch((error) => {
      toastError(error, "Couldn't initialize saved sites");
    });
    return () => {
      cancelled = true;
    };
  }, [loadProfiles]);

  // "/" opens the search popover from anywhere (unless already typing).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "/" || e.ctrlKey || e.metaKey || e.altKey) return;
      const t = e.target as HTMLElement | null;
      if (
        t &&
        (t.tagName === "INPUT" ||
          t.tagName === "TEXTAREA" ||
          t.isContentEditable)
      )
        return;
      e.preventDefault();
      setSearchOpen(true);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const connectedIds = useMemo(
    () => new Set(sessions.map((s) => s.profileId)),
    [sessions]
  );
  const sidFor = (id: string) =>
    sessions.find((s) => s.profileId === id)?.sessionId;

  // ProfileIds that currently have a bridge-enabled live session. Keyed via
  // sessionId → profileId so a reconnect (new session UUID) resolves correctly.
  const bridgedIds = useMemo(() => {
    const set = new Set<string>();
    for (const sid of enabledSessions) {
      const sess = sessions.find((s) => s.sessionId === sid);
      if (sess) set.add(sess.profileId);
    }
    return set;
  }, [enabledSessions, sessions]);

  // Manual drag-and-drop order first (sortOrder), then the learnable default
  // (protocol group, name) for profiles that have never been dragged.
  const ordered = useMemo(
    () =>
      [...profiles].sort((a, b) => {
        const sa = a.sortOrder ?? Number.MAX_SAFE_INTEGER;
        const sb = b.sortOrder ?? Number.MAX_SAFE_INTEGER;
        if (sa !== sb) return sa - sb;
        const ga = GROUP_ORDER.indexOf(a.protocol);
        const gb = GROUP_ORDER.indexOf(b.protocol);
        if (ga !== gb) return ga - gb;
        return a.name.localeCompare(b.name);
      }),
    [profiles]
  );

  // Rail sections: ungrouped servers first, then one folder per group name (in
  // order of each group's first member). Groups exist purely as `profile.group`
  // strings — no separate registry to keep in sync.
  const sections = useMemo(() => {
    const ungrouped: ConnectionProfile[] = [];
    const groups = new Map<string, ConnectionProfile[]>();
    for (const p of ordered) {
      if (!p.group) ungrouped.push(p);
      else if (groups.has(p.group)) groups.get(p.group)!.push(p);
      else groups.set(p.group, [p]);
    }
    return { ungrouped, groups: [...groups.entries()] };
  }, [ordered]);
  const groupNames = useMemo(
    () => sections.groups.map(([name]) => name),
    [sections]
  );

  // ---- Drag-and-drop reorder ----

  /** Every server in on-screen order (ungrouped, then each group's members). */
  const flatOrder = () => [
    ...sections.ungrouped,
    ...sections.groups.flatMap(([, items]) => items),
  ];

  const endDrag = () => {
    setDragId(null);
    setDropTarget(null);
  };

  const onRowDragStart = (e: React.DragEvent, p: ConnectionProfile) => {
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", p.id);
    setDragId(p.id);
  };

  const onRowDragOver = (e: React.DragEvent, p: ConnectionProfile) => {
    if (!dragId || dragId === p.id) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    const r = e.currentTarget.getBoundingClientRect();
    const after = e.clientY > r.top + r.height / 2;
    setDropTarget((cur) =>
      cur?.kind === "row" && cur.id === p.id && cur.after === after
        ? cur
        : { kind: "row", id: p.id, after }
    );
  };

  const onGroupDragOver = (e: React.DragEvent, name: string) => {
    if (!dragId) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDropTarget((cur) =>
      cur?.kind === "group" && cur.name === name ? cur : { kind: "group", name }
    );
  };

  // Drop strips between rail sections (top of the rail, end of the ungrouped
  // list, end of a group) — the only way to drop above the first row, below
  // the last, or out of a folder when no ungrouped neighbour exists.
  const onStripDragOver = (
    e: React.DragEvent,
    target: { kind: "top" } | { kind: "sectionEnd"; group: string | undefined }
  ) => {
    if (!dragId) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    setDropTarget((cur) => {
      if (target.kind === "top" && cur?.kind === "top") return cur;
      if (
        target.kind === "sectionEnd" &&
        cur?.kind === "sectionEnd" &&
        cur.group === target.group
      )
        return cur;
      return target;
    });
  };

  // Dropping between rows adopts the neighbour's group; dropping on a group
  // header appends to that group. One ipc round-trip persists both.
  const commitDrop = (e: React.DragEvent) => {
    e.preventDefault();
    const target = dropTarget;
    const id = dragId;
    endDrag();
    if (!id || !target) return;
    const rest = flatOrder().filter((p) => p.id !== id);
    const dragged = profiles.find((p) => p.id === id);
    if (!dragged) return;

    let at: number;
    let group: string | undefined;
    if (target.kind === "row") {
      const ti = rest.findIndex((p) => p.id === target.id);
      if (ti < 0) return;
      group = rest[ti].group;
      at = ti + (target.after ? 1 : 0);
    } else if (target.kind === "top") {
      group = undefined;
      at = 0;
    } else if (target.kind === "sectionEnd" && target.group === undefined) {
      // Ungrouped renders first, so its end is right after the last
      // group-less profile — this is also how a bubble leaves a folder.
      group = undefined;
      at = rest.filter((p) => p.group === undefined).length;
    } else {
      const g = target.kind === "sectionEnd" ? target.group : target.name;
      group = g;
      at = rest.length;
      for (let i = rest.length - 1; i >= 0; i--) {
        if (rest[i].group === g) {
          at = i + 1;
          break;
        }
      }
    }
    const ids = [
      ...rest.slice(0, at).map((p) => p.id),
      id,
      ...rest.slice(at).map((p) => p.id),
    ];
    void reorderProfiles(
      ids,
      dragged.group !== group ? { id, group } : undefined
    );
  };

  const rowState = (p: ConnectionProfile): RowState => {
    const isConnected = connectedIds.has(p.id);
    if (isConnected && p.id === activeProfileId && activeSessionId)
      return "focused";
    if (isConnected) return "connected";
    if (pendingId === p.id && connecting) return "connecting";
    if (pendingId === p.id && error) return "error";
    return "idle";
  };

  const doConnect = (id: string) => {
    setPendingId(id);
    connect(id)
      .then(() => setPendingId((cur) => (cur === id ? null : cur)))
      .catch((error) => toastError(error, "Connection failed"));
  };

  const openShell = async (id: string) => {
    if (!connectedIds.has(id)) setPendingId(id);
    try {
      await connect(id);
      setPendingId((cur) => (cur === id ? null : cur));
      setTerminalOpen(true);
    } catch {
      /* surfaced on the bubble */
    }
  };

  const bubbleMenuItems = (p: ConnectionProfile): MenuItem[] => {
    const sid = sidFor(p.id);
    const connected = connectedIds.has(p.id);
    const sessionBridged = sid ? enabledSessions.includes(sid) : false;
    const items: MenuItem[] = [
      {
        label: connected ? "Switch to" : "Connect",
        icon: <Plug size={14} />,
        onClick: () =>
          connected && sid ? setActiveSession(sid) : doConnect(p.id),
      },
    ];
    if (p.protocol === "sftp") {
      items.push({
        label: connected ? "Open shell" : "Connect & open shell",
        icon: <TerminalSquare size={14} />,
        onClick: () => openShell(p.id),
      });
    }
    items.push({
      label: "Properties / Edit…",
      icon: <Pencil size={14} />,
      onClick: () => setEditing(p),
      separatorAfter: true,
    });
    // Group membership: one flat item per existing group, plus "New group…".
    for (const g of groupNames) {
      if (g === p.group) continue;
      items.push({
        label: `Move to "${g}"`,
        icon: <Folder size={14} />,
        onClick: () => saveProfile({ ...p, group: g }),
      });
    }
    items.push({
      label: "New group…",
      icon: <FolderPlus size={14} />,
      onClick: () => setGroupPrompt({ kind: "assign", profile: p }),
      separatorAfter: !p.group,
    });
    if (p.group) {
      items.push({
        label: "Remove from group",
        icon: <FolderMinus size={14} />,
        onClick: () => saveProfile({ ...p, group: undefined }),
        separatorAfter: true,
      });
    }
    if (sid) {
      items.push({
        label: sessionBridged ? "Disable Agent Bridge" : "Enable Agent Bridge",
        icon: <Radio size={14} />,
        onClick: () => setSessionAccess(sid, !sessionBridged),
      });
    }
    items.push({
      label: p.autoConnect ? "Disable auto-connect" : "Auto-connect on startup",
      icon: <Power size={14} />,
      onClick: () => saveProfile({ ...p, autoConnect: !p.autoConnect }),
    });
    if (connected && sid) {
      items.push({
        label: "Disconnect",
        icon: <Unplug size={14} />,
        onClick: () => disconnect(sid),
        separatorAfter: true,
      });
    }
    items.push({
      label: "Delete",
      icon: <Trash2 size={14} />,
      destructive: true,
      onClick: () => setPendingDelete(p),
    });
    return items;
  };

  // The reconciled click rule: connected → switch; auto-connect → connect;
  // otherwise open the bubble menu (Connect is the first item).
  const onBubbleClick = (e: React.MouseEvent, p: ConnectionProfile) => {
    setBrowseLocal(false);
    const sid = sidFor(p.id);
    if (sid) {
      setActiveSession(sid);
      return;
    }
    if (p.autoConnect) {
      doConnect(p.id);
      return;
    }
    openMenuAt(e, bubbleMenuItems(p));
  };

  const openMenuAt = (e: React.MouseEvent, items: MenuItem[]) => {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    setMenu({ x: r.right + 6, y: r.top, items });
  };

  const groupMenu = (e: React.MouseEvent, name: string) => {
    e.preventDefault();
    const members = sections.groups.find(([g]) => g === name)?.[1] ?? [];
    openMenuAt(e, [
      {
        label: collapsedGroups.includes(name) ? "Expand group" : "Collapse group",
        icon: collapsedGroups.includes(name) ? (
          <ChevronDown size={14} />
        ) : (
          <ChevronRight size={14} />
        ),
        onClick: () => toggleRailGroup(name),
        separatorAfter: true,
      },
      {
        label: "Rename group…",
        icon: <Pencil size={14} />,
        onClick: () => setGroupPrompt({ kind: "rename", from: name }),
      },
      {
        label: "Ungroup servers",
        icon: <FolderMinus size={14} />,
        onClick: () =>
          void saveProfiles(members.map((m) => ({ ...m, group: undefined }))),
      },
    ]);
  };

  const submitGroupPrompt = (name: string) => {
    const prompt = groupPrompt;
    setGroupPrompt(null);
    if (!prompt) return;
    if (prompt.kind === "assign") {
      void saveProfile({ ...prompt.profile, group: name });
    } else {
      const members =
        sections.groups.find(([g]) => g === prompt.from)?.[1] ?? [];
      void saveProfiles(members.map((m) => ({ ...m, group: name })));
      if (collapsedGroups.includes(prompt.from)) toggleRailGroup(prompt.from);
    }
  };

  const pickFromSearch = (p: ConnectionProfile) => {
    const sid = sidFor(p.id);
    if (sid) setActiveSession(sid);
    else doConnect(p.id);
    setSearchOpen(false);
  };

  // ---- Agent Bridge section ----
  const bridgeSessions = useMemo(
    () =>
      enabledSessions
        .map((sid) => {
          const sess = sessions.find((s) => s.sessionId === sid);
          const profile = sess
            ? profiles.find((p) => p.id === sess.profileId)
            : undefined;
          return profile ? { sid, profile } : null;
        })
        .filter((x): x is { sid: string; profile: ConnectionProfile } => !!x),
    [enabledSessions, sessions, profiles]
  );

  const masterBridgeMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    openMenuAt(e, [
      {
        label: "Open bridge panel",
        icon: <Shield size={14} />,
        onClick: () => openDialog("agentBridge"),
        separatorAfter: true,
      },
      bridgeRunning
        ? { label: "Stop bridge", icon: <Power size={14} />, onClick: stopBridge }
        : { label: "Start bridge", icon: <Power size={14} />, onClick: startBridge },
    ]);
  };

  const bridgeSessionMenu = (
    e: React.MouseEvent,
    sid: string
  ) => {
    e.preventDefault();
    openMenuAt(e, [
      {
        label: "Open bridge panel",
        icon: <Shield size={14} />,
        onClick: () => openDialog("agentBridge"),
        separatorAfter: true,
      },
      {
        label: "Disable agent access",
        icon: <X size={14} />,
        destructive: true,
        onClick: () => setSessionAccess(sid, false),
      },
    ]);
  };

  const renderBubble = (p: ConnectionProfile) => (
    <RailBubble
      key={p.id}
      profile={p}
      state={rowState(p)}
      bridged={bridgedIds.has(p.id)}
      expanded={expanded}
      dragging={dragId === p.id}
      dropIndicator={
        dropTarget?.kind === "row" && dropTarget.id === p.id
          ? dropTarget.after
            ? "below"
            : "above"
          : null
      }
      onClick={(e) => onBubbleClick(e, p)}
      onContextMenu={(e) => {
        e.preventDefault();
        openMenuAt(e, bubbleMenuItems(p));
      }}
      onDragStart={(e) => onRowDragStart(e, p)}
      onDragOver={(e) => onRowDragOver(e, p)}
      onDrop={commitDrop}
      onDragEnd={endDrag}
    />
  );

  return (
    // Outer spacer reserves the rail's *layout* width (collapsed unless pinned),
    // so a hover flyout overlays the file panes instead of shoving them aside.
    <div className={cn("relative h-full shrink-0", pinned ? "w-64" : "w-[68px]")}>
      <div
        onMouseEnter={onRailEnter}
        onMouseLeave={onRailLeave}
        className={cn(
          "absolute inset-y-0 left-0 z-dropdown flex flex-col border-r border-border bg-bg transition-[width] duration-150 motion-reduce:transition-none",
          expanded ? "w-64" : "w-[68px]",
          !pinned && expanded && "shadow-elev-3"
        )}
      >
      <div className="flex-1 overflow-y-auto overflow-x-hidden py-2 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        {/* Local files (single-pane layout only) */}
        {browserLayout === "single" && (
          <div className="mb-2 flex flex-col items-center gap-1.5">
            <div
              className={cn(
                "group relative flex w-full items-center",
                expanded ? "justify-start pr-2" : "justify-center"
              )}
            >
              <span
                aria-hidden
                className={cn(
                  "absolute left-0 top-1/2 w-1 -translate-y-1/2 rounded-r-full transition-all duration-150 motion-reduce:transition-none",
                  browseLocal
                    ? "h-7 bg-accent"
                    : "h-0 bg-text group-hover:h-3"
                )}
              />
              <RailRow
                expanded={expanded}
                tooltip="Local files"
                onClick={() => setBrowseLocal(true)}
                ariaLabel="Local files"
                bubble={
                  <span
                    className={cn(
                      "flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl transition-colors",
                      browseLocal
                        ? "glow-accent bg-accent-soft text-accent"
                        : "bg-bg-subtle text-text-muted group-hover:bg-bg-hover group-hover:text-text"
                    )}
                  >
                    <HardDrive size={19} />
                  </span>
                }
                label={<span className="text-[13px] font-medium text-text">Local files</span>}
              />
            </div>
            <span className={cn("h-px rounded-full bg-border", expanded ? "w-full" : "w-7")} />
          </div>
        )}
        {/* Connections */}
        <div className="flex flex-col items-center gap-1.5">
          {ordered.length === 0 ? (
            <div
              className={cn(
                "group flex w-full items-center",
                expanded ? "justify-start px-2" : "justify-center"
              )}
            >
              <RailRow
                expanded={expanded}
                tooltip="Add your first server"
                onClick={() => setEditing("new")}
                ariaLabel="Add a server"
                bubble={
                  <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-dashed border-border text-text-dim transition-colors group-hover:border-accent group-hover:text-accent">
                    <Plus size={20} />
                  </span>
                }
                label={<span className="text-[13px] text-text-dim">Add your first server</span>}
              />
            </div>
          ) : (
            <>
              <DropStrip
                dragging={!!dragId}
                hot={dropTarget?.kind === "top"}
                onDragOver={(e) => onStripDragOver(e, { kind: "top" })}
                onDrop={commitDrop}
              />
              {sections.ungrouped.map(renderBubble)}
              <DropStrip
                dragging={!!dragId}
                hot={dropTarget?.kind === "sectionEnd" && dropTarget.group === undefined}
                onDragOver={(e) => onStripDragOver(e, { kind: "sectionEnd", group: undefined })}
                onDrop={commitDrop}
              />
              {sections.groups.map(([name, items]) => {
                const collapsed = collapsedGroups.includes(name);
                return (
                  <Fragment key={name}>
                    <RailGroupHeader
                      name={name}
                      count={items.length}
                      collapsed={collapsed}
                      expanded={expanded}
                      isDropTarget={
                        dropTarget?.kind === "group" && dropTarget.name === name
                      }
                      onToggle={() => toggleRailGroup(name)}
                      onContextMenu={(e) => groupMenu(e, name)}
                      onDragOver={(e) => onGroupDragOver(e, name)}
                      onDrop={commitDrop}
                    />
                    {!collapsed && items.map(renderBubble)}
                    {!collapsed && (
                      <DropStrip
                        dragging={!!dragId}
                        hot={
                          dropTarget?.kind === "sectionEnd" && dropTarget.group === name
                        }
                        onDragOver={(e) => onStripDragOver(e, { kind: "sectionEnd", group: name })}
                        onDrop={commitDrop}
                      />
                    )}
                  </Fragment>
                );
              })}
            </>
          )}
        </div>

        {/* Divider + Agent Bridge */}
        <div className="mt-3 flex flex-col items-center gap-1.5">
          <span className={cn("mb-1 h-px rounded-full bg-border", expanded ? "w-full" : "w-7")} />
          <div
            className={cn(
              "group relative flex w-full items-center",
              expanded ? "justify-start pr-2" : "justify-center"
            )}
          >
            <RailRow
              expanded={expanded}
              tooltip={`Agent Bridge — ${bridgeRunning ? "running" : "stopped"}`}
              onClick={() => openDialog("agentBridge")}
              onContextMenu={masterBridgeMenu}
              ariaLabel="Agent Bridge"
              bubble={
                <span
                  className={cn(
                    "relative flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-bg-subtle transition-colors group-hover:bg-bg-hover",
                    bridgeRunning ? "text-success" : "text-text-dim group-hover:text-text"
                  )}
                >
                  <Radio size={19} />
                  <StatusDot kind={bridgeRunning ? "connected" : "idle"} />
                </span>
              }
              label={
                <span className="flex flex-col text-left">
                  <span className="text-[13px] font-medium text-text">Agent Bridge</span>
                  <span className="text-[10px] text-text-dim">
                    {bridgeRunning ? "running" : "stopped"}
                  </span>
                </span>
              }
            />
          </div>

          {bridgeSessions.map(({ sid, profile }) => (
            <BridgeBubble
              key={sid}
              profile={profile}
              focused={sid === activeSessionId}
              expanded={expanded}
              onClick={() => setActiveSession(sid)}
              onContextMenu={(e) => bridgeSessionMenu(e, sid)}
            />
          ))}
        </div>
      </div>

      {/* Pinned controls */}
      <div className="relative flex flex-col items-center gap-1 border-t border-border py-2">
        <RailIconButton
          expanded={expanded}
          label="Search servers  ·  /"
          active={searchOpen}
          onClick={() => setSearchOpen((v) => !v)}
        >
          <Search size={17} />
        </RailIconButton>
        <RailIconButton
          expanded={expanded}
          label="New connection"
          onClick={() => setEditing("new")}
        >
          <Plus size={19} />
        </RailIconButton>
        <RailIconButton
          expanded={expanded}
          label="Import from PuTTY, OpenSSH, FileZilla…"
          onClick={() => setImporting(true)}
        >
          <Download size={16} />
        </RailIconButton>
        <RailIconButton
          expanded={expanded}
          label={pinned ? "Collapse rail" : "Keep rail expanded"}
          active={pinned}
          onClick={() => setRailExpanded(!pinned)}
        >
          {pinned ? <ChevronsLeft size={18} /> : <ChevronsRight size={18} />}
        </RailIconButton>

        {searchOpen && (
          <RailSearch
            profiles={ordered}
            connectedIds={connectedIds}
            onPick={pickFromSearch}
            onClose={() => setSearchOpen(false)}
          />
        )}
      </div>

      {menu && (
        <ContextMenu
          x={menu.x}
          y={menu.y}
          items={menu.items}
          onClose={() => setMenu(null)}
        />
      )}
      {editing && (
        <ProfileEditor
          profile={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {importing && <ImportDialog onClose={() => setImporting(false)} />}
      {groupPrompt && (
        <GroupNameDialog
          title={
            groupPrompt.kind === "assign"
              ? `Group for “${groupPrompt.profile.name}”`
              : `Rename “${groupPrompt.from}”`
          }
          initial={groupPrompt.kind === "rename" ? groupPrompt.from : ""}
          existing={groupNames}
          onSubmit={submitGroupPrompt}
          onClose={() => setGroupPrompt(null)}
        />
      )}
      {pendingDelete && (
        <ConfirmModal
          title="Delete connection"
          message={`Delete “${pendingDelete.name}” (${profileAddress(pendingDelete)})? Its saved credentials are removed from the OS keychain too — this can't be undone.`}
          confirmLabel="Delete"
          destructive
          onClose={() => setPendingDelete(null)}
          onConfirm={() => deleteProfile(pendingDelete.id)}
        />
      )}
      </div>
    </div>
  );
}

// Shared rail row: in compact mode it's a tooltip-wrapped icon button; in
// expanded mode a name/label column is shown to the right of the bubble and the
// tooltip only appears when the label is actually clipped (long server names).
// NOTE: the row must be the same height in both modes — the rail flies open on
// hover, and any height change would shift the list under the cursor.
function RailRow({
  expanded,
  tooltip,
  onClick,
  onContextMenu,
  ariaLabel,
  bubble,
  label,
}: {
  expanded: boolean;
  tooltip: React.ReactNode;
  onClick: (e: React.MouseEvent) => void;
  onContextMenu?: (e: React.MouseEvent) => void;
  ariaLabel: string;
  bubble: React.ReactNode;
  label: React.ReactNode;
}) {
  const btnRef = useRef<HTMLButtonElement>(null);
  const [clipped, setClipped] = useState(false);
  // Measured at hover time (the rail may still be mid-expansion on mount).
  const measure = () => {
    const els = btnRef.current?.querySelectorAll<HTMLElement>(".truncate");
    setClipped(
      !!els && [...els].some((el) => el.scrollWidth > el.clientWidth + 1)
    );
  };
  const btn = (
    <button
      ref={btnRef}
      onClick={onClick}
      onContextMenu={onContextMenu}
      onMouseEnter={expanded ? measure : undefined}
      aria-label={ariaLabel}
      className={cn(
        "relative flex items-center rounded-2xl",
        expanded && "w-full gap-2.5 px-2 text-left transition-colors hover:bg-bg-hover"
      )}
    >
      {bubble}
      {expanded && (
        <span className="min-w-0 flex-1 overflow-hidden">{label}</span>
      )}
    </button>
  );
  if (expanded)
    return (
      <Tooltip
        portal
        side="right"
        label={clipped ? tooltip : null}
        className="w-full min-w-0"
      >
        {btn}
      </Tooltip>
    );
  return (
    <Tooltip portal side="right" label={tooltip}>
      {btn}
    </Tooltip>
  );
}

function RailBubble({
  profile: p,
  state,
  bridged,
  expanded,
  dragging,
  dropIndicator,
  onClick,
  onContextMenu,
  onDragStart,
  onDragOver,
  onDrop,
  onDragEnd,
}: {
  profile: ConnectionProfile;
  state: RowState;
  bridged: boolean;
  expanded: boolean;
  dragging: boolean;
  dropIndicator: "above" | "below" | null;
  onClick: (e: React.MouseEvent) => void;
  onContextMenu: (e: React.MouseEvent) => void;
  onDragStart: (e: React.DragEvent) => void;
  onDragOver: (e: React.DragEvent) => void;
  onDrop: (e: React.DragEvent) => void;
  onDragEnd: () => void;
}) {
  const focused = state === "focused";
  const connected = state === "focused" || state === "connected";
  const connecting = state === "connecting";
  const isError = state === "error";
  const plaintext = p.protocol === "ftp";
  const color = p.color || fallbackColor;
  const addr = profileAddress(p);
  // Custom bubble glyph (profile.icon): a bundled Iconify key wins; an emoji /
  // short string renders as text; anything else (or an unknown key) falls back
  // to the name monogram.
  const customIcon = p.icon?.trim();
  const customBrand =
    customIcon && customIcon.includes(":") && BRAND_ICONS[customIcon]
      ? customIcon
      : null;
  const customText = customIcon && !customIcon.includes(":") ? customIcon : null;

  const bubble = (
    <span
      className={cn(
        // Rounded-square tile (not a circle) so it sits as a tidy list item when
        // the rail expands; a touch rounder when active for emphasis.
        "relative flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-bg-subtle text-[13px] font-bold leading-none transition-[border-radius,box-shadow] duration-150 motion-reduce:transition-none",
        (focused || connected) && "rounded-2xl",
        focused && "glow-accent",
        isError && "ring-1 ring-danger",
        plaintext && !isError && "ring-1 ring-warning/50"
      )}
    >
      {customBrand ? (
        <span style={{ color }}>
          <BrandIcon icon={customBrand} size={20} />
        </span>
      ) : (
        <span style={{ color }} className={customText ? "text-[16px]" : undefined}>
          {customText ?? monogram(p.name)}
        </span>
      )}
      {/* Protocol brand mark — a small top-left chip that says "S3 / Azure / SSH
          / a Ghost FTP agent" at a glance. The colour monogram stays the connection's
          primary identity; this only adds recognizability, never replaces it. */}
      <span className="absolute -left-0.5 -top-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-bg text-text-muted ring-1 ring-border">
        <BrandIcon icon={protocolIcon(p.protocol)} size={11} />
      </span>
      <StatusDot
        kind={
          connected
            ? "connected"
            : connecting
              ? "connecting"
              : isError
                ? "error"
                : "idle"
        }
      />
      {bridged && (
        <span className="absolute -right-0.5 -top-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-bg text-success">
          <Shield size={9} />
        </span>
      )}
    </span>
  );

  return (
    <div
      draggable
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDrop={onDrop}
      onDragEnd={onDragEnd}
      className={cn(
        "group relative flex w-full items-center",
        expanded ? "justify-start pr-2" : "justify-center",
        dragging && "opacity-40"
      )}
    >
      {/* Drop-position line while another server is dragged over this row. */}
      {dropIndicator && (
        <span
          aria-hidden
          className={cn(
            "pointer-events-none absolute left-2 right-2 z-10 h-0.5 rounded-full bg-accent",
            dropIndicator === "above" ? "-top-1" : "-bottom-1"
          )}
        />
      )}
      {/* Active / connected pill on the rail's left edge. */}
      <span
        aria-hidden
        className={cn(
          "absolute left-0 top-1/2 w-1 -translate-y-1/2 rounded-r-full transition-all duration-150 motion-reduce:transition-none",
          focused
            ? "h-7 bg-accent"
            : connected
              ? "h-3 bg-text group-hover:h-5"
              : "h-0 bg-text group-hover:h-3"
        )}
      />
      <RailRow
        expanded={expanded}
        onClick={onClick}
        onContextMenu={onContextMenu}
        ariaLabel={p.name}
        tooltip={
          <span className="flex flex-col gap-0.5 text-left">
            <span className="font-semibold text-text">{p.name}</span>
            <span className="font-mono text-[10px] text-text-dim">{addr}</span>
            <span className="text-[10px] text-text-dim">
              {PROTOCOL_LABEL[p.protocol]}
              {bridged ? " · bridged" : ""}
            </span>
          </span>
        }
        bubble={bubble}
        label={
          <span className="flex flex-col">
            <span className="truncate text-[13px] font-medium text-text">
              {p.name}
            </span>
            <span className="truncate font-mono text-[10px] text-text-dim">
              {addr}
            </span>
          </span>
        }
      />
    </div>
  );
}

// Thin drop strip between rail sections (rail top, end of the ungrouped list,
// end of a group). Always rendered so it never shifts rows mid-drag; invisible
// normally, a faint line while a bubble is dragged, accent when it's the
// current drop target. It is what makes "drop above the first row", "drop
// below the last", and "drag out of a folder" possible.
function DropStrip({
  dragging,
  hot,
  onDragOver,
  onDrop,
}: {
  dragging: boolean;
  hot: boolean;
  onDragOver: (e: React.DragEvent) => void;
  onDrop: (e: React.DragEvent) => void;
}) {
  return (
    <div
      aria-hidden
      onDragOver={onDragOver}
      onDrop={onDrop}
      className="flex w-full items-center px-3 py-[3px]"
    >
      <span
        className={cn(
          "h-0.5 w-full rounded-full transition-colors",
          hot ? "bg-accent" : dragging ? "bg-border/40" : "bg-transparent"
        )}
      />
    </div>
  );
}

// A collapsible folder heading in the rail. Same height in compact and
// expanded modes (see RailRow's note) — compact shows a folder glyph, expanded
// shows the name, a count and a disclosure chevron. Also a drop target: drag a
// server onto it to file the server into the group.
function RailGroupHeader({
  name,
  count,
  collapsed,
  expanded,
  isDropTarget,
  onToggle,
  onContextMenu,
  onDragOver,
  onDrop,
}: {
  name: string;
  count: number;
  collapsed: boolean;
  expanded: boolean;
  isDropTarget: boolean;
  onToggle: () => void;
  onContextMenu: (e: React.MouseEvent) => void;
  onDragOver: (e: React.DragEvent) => void;
  onDrop: (e: React.DragEvent) => void;
}) {
  const btn = (
    <button
      onClick={onToggle}
      onContextMenu={onContextMenu}
      aria-label={`${name} group, ${count} server${count === 1 ? "" : "s"}`}
      aria-expanded={!collapsed}
      className={cn(
        "flex h-6 items-center rounded-md text-text-dim transition-colors hover:bg-bg-hover hover:text-text",
        expanded ? "w-full gap-1 px-2 text-left" : "w-9 justify-center",
        isDropTarget && "bg-accent-soft text-accent"
      )}
    >
      {expanded ? (
        <>
          {collapsed ? (
            <ChevronRight size={11} className="shrink-0" />
          ) : (
            <ChevronDown size={11} className="shrink-0" />
          )}
          <span className="min-w-0 flex-1 truncate text-[10px] font-semibold uppercase tracking-wider">
            {name}
          </span>
          <span className="shrink-0 text-[10px] tabular-nums">{count}</span>
        </>
      ) : collapsed ? (
        <Folder size={14} />
      ) : (
        <FolderOpen size={14} />
      )}
    </button>
  );
  return (
    <div
      onDragOver={onDragOver}
      onDrop={onDrop}
      className={cn(
        "mt-1.5 flex w-full items-center",
        expanded ? "justify-start px-1" : "justify-center"
      )}
    >
      {expanded ? (
        btn
      ) : (
        <Tooltip
          portal
          side="right"
          label={`${name} — ${count} server${count === 1 ? "" : "s"}`}
        >
          {btn}
        </Tooltip>
      )}
    </div>
  );
}

// Minimal prompt for naming a group (create via "New group…" or rename). A
// datalist offers the existing names so joining a group is one pick away.
function GroupNameDialog({
  title,
  initial,
  existing,
  onSubmit,
  onClose,
}: {
  title: string;
  initial: string;
  existing: string[];
  onSubmit: (name: string) => void;
  onClose: () => void;
}) {
  const [value, setValue] = useState(initial);
  const panelRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  useDialog(panelRef, { onClose });
  useEffect(() => inputRef.current?.select(), []);
  const name = value.trim();
  const submit = () => name && onSubmit(name);
  return (
    <div className="fixed inset-0 z-modal flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        className="anim-modal w-72 rounded-xl border border-border bg-bg-panel p-4 shadow-elev-3"
      >
        <div className="mb-3 text-sm font-semibold">{title}</div>
        <input
          ref={inputRef}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") submit();
          }}
          list="rail-group-names"
          placeholder="Group name"
          aria-label="Group name"
          className="w-full rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent"
        />
        <datalist id="rail-group-names">
          {existing.map((g) => (
            <option key={g} value={g} />
          ))}
        </datalist>
        <div className="mt-4 flex justify-end gap-2">
          <button
            className="rounded-md border border-border px-3 py-1.5 text-sm hover:bg-bg-hover"
            onClick={onClose}
          >
            Cancel
          </button>
          <button
            className="btn-accent rounded-md px-3 py-1.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
            disabled={!name}
            onClick={submit}
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}

// A bridged session, shown in the Agent Bridge section. Deliberately distinct
// from a connection bubble (squarer, green ring, shield) so it never reads as a
// plain server.
function BridgeBubble({
  profile: p,
  focused,
  expanded,
  onClick,
  onContextMenu,
}: {
  profile: ConnectionProfile;
  focused: boolean;
  expanded: boolean;
  onClick: () => void;
  onContextMenu: (e: React.MouseEvent) => void;
}) {
  const bubble = (
    <span className="relative flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-bg-subtle text-[12px] font-bold leading-none ring-1 ring-success transition-colors group-hover:bg-bg-hover">
      <span className="text-success">{monogram(p.name)}</span>
      <span className="absolute -right-0.5 -top-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-bg text-success">
        <Shield size={9} />
      </span>
    </span>
  );
  return (
    <div
      className={cn(
        "group relative flex w-full items-center",
        expanded ? "justify-start pr-2" : "justify-center"
      )}
    >
      <span
        aria-hidden
        className={cn(
          "absolute left-0 top-1/2 w-1 -translate-y-1/2 rounded-r-full bg-success transition-all duration-150 motion-reduce:transition-none",
          focused ? "h-6" : "h-2 group-hover:h-4"
        )}
      />
      <RailRow
        expanded={expanded}
        onClick={onClick}
        onContextMenu={onContextMenu}
        ariaLabel={`${p.name} (bridged)`}
        tooltip={
          <span className="flex flex-col gap-0.5 text-left">
            <span className="font-semibold text-text">{p.name}</span>
            <span className="text-[10px] text-success">· bridged</span>
          </span>
        }
        bubble={bubble}
        label={
          <span className="flex flex-col">
            <span className="truncate text-[13px] font-medium text-text">
              {p.name}
            </span>
            <span className="text-[10px] text-success">bridged</span>
          </span>
        }
      />
    </div>
  );
}

// The one place color means connection state — mirrors ConnectionManager's Lamp,
// rendered as a corner badge on the bubble.
function StatusDot({
  kind,
}: {
  kind: "connected" | "connecting" | "error" | "idle";
}) {
  return (
    <span className="absolute -bottom-0.5 -right-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-bg">
      <span
        className={cn(
          "h-2 w-2 rounded-full",
          kind === "connected"
            ? "bg-success ring-2 ring-success/25"
            : kind === "connecting"
              ? "bg-accent animate-pulse motion-reduce:animate-none"
              : kind === "error"
                ? "bg-danger"
                : "border border-text-dim"
        )}
      />
    </span>
  );
}

function RailIconButton({
  children,
  label,
  active,
  expanded,
  onClick,
}: {
  children: React.ReactNode;
  label: string;
  active?: boolean;
  expanded?: boolean;
  onClick: () => void;
}) {
  const btn = (
    <button
      onClick={onClick}
      aria-label={label}
      className={cn(
        // h-9 in BOTH modes — hover-expansion must not change row heights (the
        // list would shift under the cursor).
        "flex h-9 items-center rounded-xl transition-colors",
        expanded ? "w-full gap-2.5 px-2 text-left" : "w-9 justify-center",
        active
          ? "bg-accent-soft text-accent"
          : "text-text-muted hover:bg-bg-hover hover:text-text"
      )}
    >
      <span
        className={cn(
          "flex shrink-0 items-center justify-center",
          expanded && "w-9"
        )}
      >
        {children}
      </span>
      {expanded && (
        <span className="min-w-0 flex-1 truncate text-[12px]">{label}</span>
      )}
    </button>
  );
  if (expanded) return btn;
  return (
    <Tooltip portal side="right" label={label}>
      {btn}
    </Tooltip>
  );
}

// Searchable popover over every saved server — the rail's replacement for the
// old sidebar filter. Click a result to focus it (if open) or connect.
function RailSearch({
  profiles,
  connectedIds,
  onPick,
  onClose,
}: {
  profiles: ConnectionProfile[];
  connectedIds: Set<string>;
  onPick: (p: ConnectionProfile) => void;
  onClose: () => void;
}) {
  const [query, setQuery] = useState("");
  const wrapRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    const onDown = (e: MouseEvent) => {
      if (!wrapRef.current?.contains(e.target as Node)) onClose();
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("mousedown", onDown);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("mousedown", onDown);
    };
  }, [onClose]);

  const q = query.trim().toLowerCase();
  const results = q
    ? profiles.filter((p) =>
        [p.name, p.host, p.username, p.bucket, p.account, PROTOCOL_LABEL[p.protocol]]
          .some((s) => !!s && s.toLowerCase().includes(q))
      )
    : profiles;

  return (
    <div
      ref={wrapRef}
      className="anim-modal absolute bottom-2 left-full z-menu ml-2 w-72 overflow-hidden rounded-lg border border-border bg-bg-panel shadow-elev-3"
    >
      <div className="border-b border-border bg-bg-subtle px-2 py-1.5">
        <div className="relative">
          <Search
            size={12}
            className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 text-text-dim"
          />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && results[0]) onPick(results[0]);
            }}
            placeholder="Search servers…"
            aria-label="Search servers"
            className="w-full rounded-md border border-border-subtle bg-bg px-7 py-1 text-xs outline-none focus:border-accent"
          />
          {query && (
            <button
              onClick={() => setQuery("")}
              className="absolute right-1.5 top-1/2 -translate-y-1/2 rounded p-0.5 text-text-dim hover:text-text"
              aria-label="Clear search"
            >
              <X size={11} />
            </button>
          )}
        </div>
      </div>
      <div className="max-h-80 overflow-y-auto py-1">
        {results.length === 0 ? (
          <div className="px-3 py-6 text-center text-xs text-text-dim">
            No servers match “{query}”.
          </div>
        ) : (
          results.map((p) => {
            const online = connectedIds.has(p.id);
            return (
              <button
                key={p.id}
                onClick={() => onPick(p)}
                className="flex w-full items-center gap-2.5 px-3 py-1.5 text-left hover:bg-bg-hover"
              >
                <span
                  className="h-2 w-2 shrink-0 rounded-full"
                  style={{ background: p.color || fallbackColor }}
                />
                {/* Real protocol/brand logo (S3, Azure, SSH, the Ghost FTP
                    lighthouse…) instead of a generic cloud/server glyph. */}
                <span className="flex h-4 w-4 shrink-0 items-center justify-center text-text-dim">
                  <BrandIcon icon={protocolIcon(p.protocol)} size={14} />
                </span>
                <div className="min-w-0 flex-1">
                  <div className="truncate text-[12px] font-medium">{p.name}</div>
                  <div className="truncate font-mono text-[10px] text-text-dim">
                    {profileAddress(p)}
                  </div>
                </div>
                {online && (
                  <span
                    className="h-1.5 w-1.5 shrink-0 rounded-full bg-success"
                    title="Connected"
                  />
                )}
              </button>
            );
          })
        )}
      </div>
    </div>
  );
}
