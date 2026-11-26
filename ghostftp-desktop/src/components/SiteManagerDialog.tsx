// Ghost FTP by Brendigo
import { useEffect, useMemo, useRef, useState } from "react";
import { save as saveDialog } from "@tauri-apps/plugin-dialog";
import {
  Bookmark,
  Cloud,
  CopyPlus,
  Download,
  Edit3,
  Eye,
  EyeOff,
  Link2,
  Plus,
  RadioTower,
  Save,
  Search,
  Server,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import type { ConnectionProfile, Protocol } from "@/lib/types";
import { PROTOCOL_DEFAULT_PORT } from "@/lib/types";
import { duplicateSavedSiteIds, matchingSavedSites } from "@/lib/siteDuplicates";
import { countSiteViews, filterAndSortSites } from "@/lib/siteViewModel";
import { ConfirmModal } from "./ConfirmModal";
import { CloudAccountConnect } from "./CloudAccountConnect";
import { CloudStorageConnect } from "./CloudStorageConnect";
import { SiteManagerEmptyState } from "./SiteManagerEmptyState";
import { FilterChip, SiteSortHeader, type SiteSortField } from "./SiteManagerListControls";
import { SiteRow } from "./SiteManagerSiteRow";
import { ipc } from "@/lib/ipc";
import { toast } from "@/stores/toastStore";
import { toastError } from "@/lib/errors";
import { useDialog } from "@/hooks/useDialog";

interface Props {
  onClose: () => void;
  initialView?: View;
}

type View = "all" | "favorites" | "recent" | "bookmarks" | string;
type Action = "save" | "test" | "connect" | "duplicate" | "delete" | "favorite" | null;


const DIRECT_EDIT_PROTOCOLS = new Set<Protocol>(["sftp", "ftp", "ftps"]);

export function SiteManagerDialog({ onClose, initialView = "all" }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const allProfiles = useConnections((s) => s.profiles);
  const connect = useConnections((s) => s.connect);
  const sessions = useConnections((s) => s.sessions);
  const saveProfile = useConnections((s) => s.saveProfile);
  const duplicateProfile = useConnections((s) => s.duplicateProfile);
  const deleteProfile = useConnections((s) => s.deleteProfile);
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const openDialog = useLayout((s) => s.openDialog);

  const ephemeralIds = useMemo(
    () =>
      new Set(
        sessions
          .filter((session) => session.ephemeral)
          .map((session) => session.profileId)
      ),
    [sessions]
  );
  const profiles = useMemo(
    () => allProfiles.filter((profile) => !ephemeralIds.has(profile.id)),
    [allProfiles, ephemeralIds]
  );

  const duplicateIds = useMemo(() => duplicateSavedSiteIds(profiles), [profiles]);

  const [query, setQuery] = useState("");
  const [sortField, setSortField] = useState<SiteSortField>("name");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [selectedId, setSelectedId] = useState<string | null>(
    profiles[0]?.id ?? null
  );
  const [view, setView] = useState<View>(initialView);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<ConnectionProfile | null>(
    profiles[0] ? { ...profiles[0] } : null
  );
  const [action, setAction] = useState<Action>(null);
  const actionRef = useRef<Action>(null);
  const beginAction = (next: NonNullable<Action>) => {
    if (actionRef.current) return false;
    actionRef.current = next;
    setAction(next);
    return true;
  };
  const endAction = () => {
    actionRef.current = null;
    setAction(null);
  };
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [revealedPasswordId, setRevealedPasswordId] = useState<string | null>(null);

  const [addingMeta, setAddingMeta] = useState<"tag" | "folder" | null>(null);
  const [metaValue, setMetaValue] = useState("");
  const [favoriteBusyId, setFavoriteBusyId] = useState<string | null>(null);
  const [cloudWizard, setCloudWizard] = useState<"oauth" | "storage" | null>(null);

  const closeIfIdle = () => {
    if (actionRef.current) return;
    if (editing) {
      toast.info("Unsaved changes", "Save or cancel changes to this site before leaving.");
      return;
    }
    onClose();
  };
  useDialog(panelRef, { onClose: closeIfIdle, initialFocus: searchRef, trapFocus: false });

  const folders = useMemo(
    () =>
      Array.from(
        new Set(profiles.map((profile) => profile.group).filter(Boolean) as string[])
      ).sort(),
    [profiles]
  );
  const tags = useMemo(
    () => Array.from(new Set(profiles.flatMap((profile) => profile.tags ?? []))).sort(),
    [profiles]
  );

  const connectedIds = useMemo(() => new Set(sessions.map((session) => session.profileId)), [sessions]);
  const counts = useMemo(() => countSiteViews(profiles, duplicateIds, connectedIds), [profiles, duplicateIds, connectedIds]);
  const filtered = useMemo(
    () => filterAndSortSites(profiles, query, view, sortField, sortDirection, duplicateIds, connectedIds),
    [profiles, query, view, sortField, sortDirection, duplicateIds, connectedIds]
  );
  const resetFilters = () => {
    setQuery("");
    setView("all");
    searchRef.current?.focus();
  };

  const toggleSort = (field: SiteSortField) => {
    if (field === sortField) {
      setSortDirection((direction) => (direction === "asc" ? "desc" : "asc"));
      return;
    }
    setSortField(field);
    setSortDirection(field === "lastUsed" ? "desc" : "asc");
  };

  // Filtering cannot replace the details pane's underlying profile while
  // its draft is being edited, even when the active filter hides that profile.
  const selected = editing
    ? profiles.find((profile) => profile.id === draft?.id) ?? null
    : filtered.find((profile) => profile.id === selectedId) ?? filtered[0] ?? null;
  // Pin destructive confirmation to the originally chosen profile ID. A live
  // profile reload or filter change must never redirect deletion to another row.
  const showPassword = draft !== null && selected?.id === draft.id && revealedPasswordId === draft.id;
  const deletionTarget = confirmDeleteId
    ? profiles.find((profile) => profile.id === confirmDeleteId) ?? null
    : null;
  const matchingSites = useMemo(() => draft ? matchingSavedSites(draft, profiles) : [], [draft, profiles]);
  const isConnected = selected ? connectedIds.has(selected.id) : false;
  useEffect(() => {
    setRevealedPasswordId(null);
    setAddingMeta(null);
    setMetaValue("");
  }, [selected?.id]);
  const canDirectEdit = selected
    ? DIRECT_EDIT_PROTOCOLS.has(selected.protocol)
    : false;
  const canDuplicate =
    canDirectEdit && selected?.auth.kind !== "keyref";

  // Keep the details pane synchronized after a profile reload, but never clobber
  // an edit that is currently in progress.
  useEffect(() => {
    if (editing) return;
    if (!selected) {
      setDraft(null);
      return;
    }
    setSelectedId(selected.id);
    setDraft({ ...selected });
  }, [selected, editing]);

  const select = (id: string) => {
    if (id === selectedId) return;
    if (editing || action) {
      toast.info("Finish editing", "Save or cancel changes before selecting a different site.");
      return;
    }
    const profile = profiles.find((item) => item.id === id) ?? null;
    setRevealedPasswordId(null);
    setAddingMeta(null);
    setMetaValue("");
    setSelectedId(id);
    setDraft(profile ? { ...profile } : null);
    setEditing(false);
  };

  const startEdit = () => {
    if (!selected || !canDirectEdit || actionRef.current) return;
    setAddingMeta(null);
    setMetaValue("");
    setDraft({ ...selected });
    setEditing(true);
  };

  const save = async () => {
    if (!draft || selected?.id !== draft.id || !editing || actionRef.current) return;
    const name = draft.name.trim();
    const host = draft.host.trim();
    const username = draft.username.trim();
    if (!name || !host || !username) {
      toast.error("Missing connection details", "Name, host and username are required.");
      return;
    }
    if (!Number.isInteger(draft.port) || draft.port < 1 || draft.port > 65535) {
      toast.error("Invalid port", "Use a TCP port between 1 and 65535.");
      return;
    }

    if (!beginAction("save")) return;
    try {
      await saveProfile({ ...draft, name, host, username });
      setRevealedPasswordId(null);
      setEditing(false);
      toast.success("Site saved", name);
    } catch (error) {
      toastError(error, `Couldn't save ${name}`);
    } finally {
      endAction();
    }
  };

  const toggleBookmark = async () => {
    if (!selected || editing || action || !beginAction("save")) return;
    try {
      await saveProfile({ ...selected, bookmarked: !selected.bookmarked });
      toast.info(
        selected.bookmarked ? "Bookmark removed" : "Bookmarked",
        selected.name
      );
    } catch (error) {
      toastError(error, `Couldn't update ${selected.name}`);
    } finally {
      endAction();
    }
  };

  const remove = async () => {
    if (!deletionTarget) throw new Error("The selected site is no longer available.");
    if (useConnections.getState().sessions.some((session) => session.profileId === deletionTarget.id)) {
      throw new Error("Disconnect this site before deleting it.");
    }
    if (!beginAction("delete")) throw new Error("Another site operation is in progress.");
    const index = profiles.findIndex((profile) => profile.id === deletionTarget.id);
    const next = profiles[index + 1] ?? profiles[index - 1] ?? null;

    try {
      await deleteProfile(deletionTarget.id);
      setSelectedId(next?.id ?? null);
      setDraft(next ? { ...next } : null);
      setEditing(false);
      toast.info("Site deleted", deletionTarget.name);
    } finally {
      // Errors reject to ConfirmModal, which keeps the deletion confirmation
      // open and displays a retryable failure instead of closing prematurely.
      endAction();
    }
  };

  const exportSites = async () => {
    if (profiles.length === 0) {
      toast.info("Nothing to export", "Create or import a site profile first.");
      return;
    }
    try {
      const path = await saveDialog({
        title: "Export Ghost FTP sites",
        defaultPath: "ghostftp-sites.json",
        filters: [{ name: "Ghost FTP site export", extensions: ["json"] }],
      });
      if (!path) return;
      const count = await ipc.exportProfiles(path);
      toast.success("Sites exported", `${count} profile${count === 1 ? "" : "s"} saved securely without credentials.`);
    } catch (error) {
      toastError(error, "Couldn't export sites");
    }
  };

  const connectSelected = async () => {
    if (!selected || action || editing || !beginAction("connect")) return;
    try {
      await connect(selected.id);
    } catch {
      // connectionsStore already surfaces the redacted FTP/FTPS/SFTP error.
      return;
    } finally {
      endAction();
    }
  };

  const testSelected = async () => {
    if (!selected || action || editing || !beginAction("test")) return;
    try {
      await ipc.testProfileConnection(selected.id);
      toast.success(
        "Connection successful",
        `${selected.name} — ${selected.username}@${selected.host}:${selected.port}`
      );
    } catch (error) {
      toastError(error, `Couldn't connect to ${selected.name}`);
    } finally {
      endAction();
    }
  };

  const applyMeta = async () => {
    if (!selected || !addingMeta || editing || action) return;
    const value = metaValue.trim();
    if (!value) return;
    if (!beginAction("save")) return;
    try {
      if (addingMeta === "tag") {
        const nextTags = Array.from(new Set([...(selected.tags ?? []), value]));
        await saveProfile({ ...selected, tags: nextTags });
        setView(`tag:${value}`);
      } else {
        await saveProfile({ ...selected, group: value });
        setView(`folder:${value}`);
      }
      setMetaValue("");
      setAddingMeta(null);
      toast.success(addingMeta === "tag" ? "Tag added" : "Folder assigned", value);
    } catch (error) {
      toastError(error, "Couldn't update site organization");
    } finally {
      endAction();
    }
  };

  const toggleFavorite = async (profile: ConnectionProfile) => {
    if (editing || action || !beginAction("favorite")) return;
    setFavoriteBusyId(profile.id);
    try {
      await saveProfile({ ...profile, favorite: !profile.favorite });
      toast.info(
        profile.favorite ? "Removed from favorites" : "Added to favorites",
        profile.name
      );
    } catch (error) {
      toastError(error, `Couldn't update ${profile.name}`);
    } finally {
      setFavoriteBusyId(null);
      endAction();
    }
  };

  const duplicateSelected = async () => {
    if (!selected || !canDuplicate || action || editing || !beginAction("duplicate")) return;
    try {
      const copy = await duplicateProfile(selected.id);
      setSelectedId(copy.id);
      setDraft({ ...copy });
      setEditing(false);
      toast.success("Site duplicated", copy.name);
    } catch (error) {
      toastError(error, `Couldn't duplicate ${selected.name}`);
    } finally {
      endAction();
    }
  };

  return (
    <div
      className="ghost-workspace-view ghost-standalone-view bg-[#041425]"
      role="region"
      aria-label="Sites"
    >
      <div
        ref={panelRef}
        className="ghost-site-manager flex h-full w-full flex-col overflow-hidden bg-[#061a2d]"
      >
        <div className="ghost-site-manager-hero flex items-center gap-3 border-b border-border px-5">
          <div className="ghost-dialog-icon">
            <Server size={24} />
          </div>
          <div>
            <div className="text-xl font-semibold">Sites</div>
            <div className="text-[12px] text-text-muted">
              Manage saved connections, bookmarks and server profiles.
            </div>
          </div>
          <div className="flex-1" />
          <button
            type="button"
            className="ghost-mini-button"
            disabled={Boolean(action) || editing}
            onClick={() => setCloudWizard("oauth")}
          >
            <Cloud size={15} /> Connect Cloud
          </button>
          {profiles.length > 0 && (
            <>
              <div className="ghost-site-search relative min-w-0">
                <Search
                  size={14}
                  className="pointer-events-none absolute left-3 top-2.5 text-text-dim"
                />
                <input
                  ref={searchRef}
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  className="ghost-ref-input w-full pl-9 pr-9"
                  placeholder="Search sites…"
                  aria-label="Search saved sites"
                />
                {query && <button type="button" className="ghost-site-search-clear" aria-label="Clear site search" onClick={() => { setQuery(""); searchRef.current?.focus(); }}><X size={14} aria-hidden="true"/></button>}
              </div>
              <button
                type="button"
                className="ghost-mini-button"
                disabled={Boolean(action) || editing}
                title={editing ? "Save or cancel changes before leaving this site." : undefined}
                onClick={() => openDialog("import")}
              >
                <Download size={14} /> Import
              </button>
              <button
                type="button"
                className="ghost-mini-button"
                onClick={() => void exportSites()}
              >
                <Upload size={14} /> Export
              </button>
              <button
                type="button"
                className="ghost-primary-button"
                disabled={Boolean(action) || editing}
                title={editing ? "Save or cancel changes before leaving this site." : undefined}
                onClick={() => openNewConnection()}
              >
                <Plus size={15} /> New Site
              </button>
            </>
          )}
        </div>

        {profiles.length > 0 && (
          <div className="ghost-site-filterbar flex shrink-0 items-center gap-2 overflow-x-auto border-b border-border bg-[#051929] px-3 py-2">
          <FilterChip active={view === "all"} label="All" count={counts.all} onClick={() => setView("all")} />
          <FilterChip active={view === "favorites"} label="Favorites" count={counts.favorites} onClick={() => setView("favorites")} />
          <FilterChip active={view === "recent"} label="Recent" count={counts.recent} onClick={() => setView("recent")} />
          <FilterChip active={view === "bookmarks"} label="Bookmarks" count={counts.bookmarks} onClick={() => setView("bookmarks")} />
          <FilterChip active={view === "duplicates"} label="Duplicates" count={counts.duplicates} onClick={() => setView("duplicates")} />
          <FilterChip active={view === "cloud"} label="Cloud" count={counts.cloud} onClick={() => setView("cloud")} />
          <FilterChip active={view === "connected"} label="Connected" count={counts.connected} onClick={() => setView("connected")} />
          <div className="h-6 w-px shrink-0 bg-border"/>
          <label className="ghost-site-filter-select">Tag<select aria-label="Filter by tag" value={view.startsWith("tag:") ? view : ""} onChange={(event) => setView(event.target.value || "all")}><option value="">All tags</option>{tags.map((tag) => <option key={tag} value={`tag:${tag}`}>{tag}</option>)}</select></label>
          <label className="ghost-site-filter-select">Folder<select aria-label="Filter by folder" value={view.startsWith("folder:") ? view : ""} onChange={(event) => setView(event.target.value || "all")}><option value="">All folders</option>{folders.map((folder) => <option key={folder} value={`folder:${folder}`}>{folder}</option>)}</select></label>
          </div>
        )}

        {profiles.length === 0 && !query ? (
          <SiteManagerEmptyState
            disabled={Boolean(action) || editing}
            onNewSite={() => openNewConnection()}
            onConnectCloud={() => setCloudWizard("oauth")}
            onImportSites={() => openDialog("import")}
          />
        ) : (
          <div className="ghost-site-manager-workspace ghost-site-manager-workspace--populated grid min-h-0 flex-1">
          <section
            className="ghost-site-manager-table min-w-0 overflow-auto p-3"
            aria-label="Saved sites"
          >
            <div className="ghost-sites-column-sorts flex flex-wrap items-center gap-3 border-b border-border px-3 py-2 text-[11px] font-semibold text-text-dim">
              <SiteSortHeader label="Name" field="name" activeField={sortField} direction={sortDirection} onSort={toggleSort}/>
              <SiteSortHeader label="Host" field="host" activeField={sortField} direction={sortDirection} onSort={toggleSort}/>
              <SiteSortHeader label="Protocol" field="protocol" activeField={sortField} direction={sortDirection} onSort={toggleSort}/>
              <span className="flex items-center">Tags</span>
              <SiteSortHeader label="Last Used" field="lastUsed" activeField={sortField} direction={sortDirection} onSort={toggleSort}/>
            </div>

            {filtered.length === 0 && (
              <div className="p-10 text-center text-text-muted">
                <p>{query.trim() ? "No saved sites match your search." : "No saved sites in this view."}</p>
                <button type="button" className="ghost-mini-button mx-auto mt-3" onClick={resetFilters}>Clear filters</button>
              </div>
            )}

            {filtered.map((profile) => (
              <SiteRow
                key={profile.id}
                profile={profile}
                active={selected?.id === profile.id}
                favorite={profile.favorite === true}
                favoriteBusy={Boolean(action) || editing || favoriteBusyId === profile.id}
                connected={connectedIds.has(profile.id)}
                onClick={() => select(profile.id)}
                onFavorite={() => void toggleFavorite(profile)}
              />
            ))}
            <div className="ghost-site-table-footer sticky bottom-0 mt-1 flex h-8 items-center border-t border-border bg-[#051929] px-3 text-[10px] text-text-muted">
              <span role="status">{filtered.length} of {profiles.length} sites</span>
              <span className="mx-2 text-text-dim">|</span>
              <span>{selected ? `1 selected (${selected.name})` : "0 selected"}</span>
            </div>
          </section>

          <aside className="ghost-site-manager-details border-l border-border bg-[#071f35] p-4 overflow-y-auto">
            {selected && draft && selected.id === draft.id ? (
              <>
                <div className="ghost-site-details-title mb-3 flex min-h-9 items-center border-b border-border pb-2">
                  <strong className="text-[14px] text-accent">Connection Details</strong>
                  <div className="flex-1" />
                  <button
                    type="button"
                    className="ghost-site-bookmark-button"
                    aria-label={selected.bookmarked ? "Remove bookmark" : "Bookmark site"}
                    aria-pressed={selected.bookmarked === true}
                    title={selected.bookmarked ? "Remove bookmark" : "Bookmark site"}
                    disabled={Boolean(action) || editing}
                    onClick={() => void toggleBookmark()}
                  >
                    <Bookmark size={14} fill={selected.bookmarked ? "currentColor" : "none"} />
                  </button>
                  <button
                    type="button"
                    className="ghost-mini-button"
                    onClick={() => void (editing ? save() : startEdit())}
                    disabled={Boolean(action) || (!editing && !canDirectEdit)}
                    title={
                      canDirectEdit
                        ? undefined
                        : "This connection type is managed by its dedicated connection editor."
                    }
                  >
                    {editing ? <Save size={13} /> : <Edit3 size={13} />}
                    {action === "save" ? "Saving…" : editing ? "Save" : "Edit"}
                  </button>
                  {editing && (
                    <button
                      type="button"
                      className="ghost-mini-button"
                      disabled={Boolean(action)}
                      onClick={() => {
                        setDraft({ ...selected });
                        setEditing(false);
                        setRevealedPasswordId(null);
                      }}
                    >
                      Cancel
                    </button>
                  )}
                </div>

                <div className="mb-4 flex items-center gap-3">
                  <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-accent-strong text-white">
                    <Server size={24} />
                  </div>
                  <div className="min-w-0">
                    <div className="truncate text-[17px] font-semibold">
                      {selected.name}
                    </div>
                    <div className="text-[11px] text-text-muted">
                      {selected.description || selected.group || "Saved connection"}
                    </div>
                  </div>
                </div>

                {matchingSites.length > 0 && (
                  <div role="status" className="mb-3 rounded-md border border-warning/30 bg-warning/10 px-3 py-2 text-[11px] text-text-muted">
                    <strong className="text-warning">Possible duplicate.</strong>{" "}
                    {matchingSites.length} other saved {matchingSites.length === 1 ? "site uses" : "sites use"}
                    {" "}the same protocol, host, port and username. Profiles are kept separate,
                    including their credentials and bookmarks.
                  </div>
                )}

                {!canDirectEdit && (
                  <div className="mb-3 rounded-md border border-accent/25 bg-accent/10 px-3 py-2 text-[11px] text-text-muted">
                    {selected.protocol.toUpperCase()} authorization is account-bound.
                    Test and connect are available here; use its dedicated connection
                    flow to change credentials.
                  </div>
                )}

                <div className="mb-4 grid grid-cols-2 gap-2 border-t border-border pt-4">
                  <div className="rounded-md border border-border-subtle bg-[#051929] p-2.5">
                    <div className="mb-1 text-[10px] uppercase tracking-wide text-text-dim">Folder</div>
                    <div className="flex items-center gap-2"><span className="min-w-0 flex-1 truncate text-[11px]">{selected.group || "None"}</span><button type="button" className="ghost-site-meta-add" aria-label="Assign folder" disabled={Boolean(action) || editing} onClick={() => { setAddingMeta("folder"); setMetaValue(selected.group ?? ""); }}><Edit3 size={12}/></button></div>
                    {addingMeta === "folder" && <div className="ghost-site-meta-editor mt-2"><input autoFocus value={metaValue} placeholder="Folder name" onChange={(event) => setMetaValue(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") void applyMeta(); if (event.key === "Escape") setAddingMeta(null); }}/><button type="button" onClick={() => void applyMeta()} disabled={!metaValue.trim()}>Set</button><button type="button" onClick={() => setAddingMeta(null)}>×</button></div>}
                  </div>
                  <div className="rounded-md border border-border-subtle bg-[#051929] p-2.5">
                    <div className="mb-1 text-[10px] uppercase tracking-wide text-text-dim">Tags</div>
                    <div className="flex items-center gap-2"><span className="min-w-0 flex-1 truncate text-[11px]">{(selected.tags ?? []).join(", ") || "None"}</span><button type="button" className="ghost-site-meta-add" aria-label="Add tag" disabled={Boolean(action) || editing} onClick={() => { setAddingMeta("tag"); setMetaValue(""); }}><Plus size={12}/></button></div>
                    {addingMeta === "tag" && <div className="ghost-site-meta-editor mt-2"><input autoFocus value={metaValue} placeholder="Tag name" onChange={(event) => setMetaValue(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") void applyMeta(); if (event.key === "Escape") setAddingMeta(null); }}/><button type="button" onClick={() => void applyMeta()} disabled={!metaValue.trim()}>Add</button><button type="button" onClick={() => setAddingMeta(null)}>×</button></div>}
                  </div>
                </div>

                <fieldset disabled={Boolean(action)} className="min-w-0 space-y-3 border-t border-border pt-4">
                  <EditField label="Site Name" editing={editing}>
                    <input value={draft.name} readOnly={!editing} onChange={(event) => setDraft({ ...draft, name: event.target.value })} />
                  </EditField>
                  <EditField label="Protocol" editing={editing}>
                    <select
                      value={draft.protocol}
                      disabled={!editing}
                      onChange={(event) => {
                        const protocol = event.target.value as Protocol;
                        setDraft({
                          ...draft,
                          protocol,
                          port: PROTOCOL_DEFAULT_PORT[protocol],
                        });
                      }}
                    >
                      <option value="sftp">SFTP</option>
                      <option value="ftp">FTP</option>
                      <option value="ftps">FTPS</option>
                      {!DIRECT_EDIT_PROTOCOLS.has(draft.protocol) && (
                        <option value={draft.protocol}>
                          {draft.protocol.toUpperCase()}
                        </option>
                      )}
                    </select>
                  </EditField>
                  <EditField label="Host / Address" editing={editing}>
                    <input
                      value={draft.host}
                      readOnly={!editing}
                      onChange={(event) =>
                        setDraft({ ...draft, host: event.target.value })
                      }
                    />
                  </EditField>
                  <EditField label="Port" editing={editing}>
                    <input
                      type="number"
                      min={1}
                      max={65535}
                      value={draft.port}
                      readOnly={!editing}
                      onChange={(event) =>
                        setDraft({ ...draft, port: Number(event.target.value) })
                      }
                    />
                  </EditField>
                  <EditField label="Username" editing={editing}>
                    <input
                      value={draft.username}
                      readOnly={!editing}
                      onChange={(event) =>
                        setDraft({ ...draft, username: event.target.value })
                      }
                    />
                  </EditField>
                  {draft.auth.kind === "password" && (
                    <EditField label="Password" editing={editing}>
                      <div className="relative">
                        <input
                          type={showPassword ? "text" : "password"}
                          value={draft.auth.password}
                          readOnly={!editing}
                          onChange={(event) =>
                            setDraft({
                              ...draft,
                              auth: { kind: "password", password: event.target.value },
                            })
                          }
                          className="pr-9"
                        />
                        <button
                          type="button"
                          className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-text-dim hover:text-white"
                          onClick={() => setRevealedPasswordId(showPassword ? null : draft.id)}
                          aria-pressed={showPassword}
                          aria-label={showPassword ? "Hide password" : "Show password"}
                        >
                          {showPassword ? <EyeOff size={14}/> : <Eye size={14}/>}
                        </button>
                      </div>
                    </EditField>
                  )}
                  <EditField label="Remote Path" editing={editing}>
                    <input
                      value={draft.defaultRemotePath || ""}
                      readOnly={!editing}
                      onChange={(event) =>
                        setDraft({
                          ...draft,
                          defaultRemotePath: event.target.value,
                        })
                      }
                    />
                  </EditField>
                  <EditField label="Description" editing={editing}>
                    <div className="relative">
                      <textarea
                        value={draft.description || ""}
                        readOnly={!editing}
                        maxLength={500}
                        rows={3}
                        className="pb-6"
                        onChange={(event) =>
                          setDraft({
                            ...draft,
                            description: event.target.value,
                          })
                        }
                        placeholder="Optional note about this server"
                      />
                      <span className="pointer-events-none absolute bottom-2 right-2 text-[9.5px] text-text-dim">
                        {(draft.description || "").length}/500
                      </span>
                    </div>
                  </EditField>
                  <EditField label="Encoding" editing={false}>
                    <input value="UTF-8" readOnly />
                  </EditField>
                  <EditField label="Connection Mode" editing={false}>
                    <input
                      value={
                        draft.protocol === "ftp" || draft.protocol === "ftps"
                          ? "Passive"
                          : "Secure"
                      }
                      readOnly
                    />
                  </EditField>
                </fieldset>
                {DIRECT_EDIT_PROTOCOLS.has(selected.protocol) && (
                  <SiteBandwidthLimit key={selected.id} profileId={selected.id} disabled={editing || Boolean(action)} />
                )}

                <div className="mt-5 flex gap-2">
                  <button
                    type="button"
                    disabled={Boolean(action) || editing}
                    className="ghost-primary-button flex-1"
                    onClick={() => void connectSelected()}
                  >
                    <Link2 size={15} />
                    {action === "connect" ? "Connecting…" : isConnected ? "Open session" : "Connect"}
                  </button>
                  <button
                    type="button"
                    className="ghost-mini-button flex-1"
                    disabled={Boolean(action) || editing}
                    onClick={() => void testSelected()}
                  >
                    <RadioTower size={14} />
                    {action === "test" ? "Testing…" : "Test Connection"}
                  </button>
                </div>

                <div className="mt-3 grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    className="ghost-mini-button justify-center"
                    disabled={!canDuplicate || Boolean(action) || editing}
                    title={
                      canDuplicate
                        ? "Create a new saved site with a separate credential entry."
                        : "This connection type cannot be safely duplicated."
                    }
                    onClick={() => void duplicateSelected()}
                  >
                    <CopyPlus size={13} />
                    {action === "duplicate" ? "Duplicating…" : "Duplicate"}
                  </button>
                  <button
                    type="button"
                    className="flex items-center justify-center gap-2 rounded-md border border-danger/30 bg-danger/10 py-2 text-[11px] text-danger hover:bg-danger/20 disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={Boolean(action) || editing || isConnected}
                    title={
                      isConnected
                        ? "Disconnect this site before deleting it."
                        : undefined
                    }
                    onClick={() => setConfirmDeleteId(selected.id)}
                  >
                    <Trash2 size={13} />
                    {action === "delete" ? "Deleting…" : "Delete Site"}
                  </button>
                </div>
              </>
            ) : (
              <div className="grid h-full place-items-center text-text-muted">
                <div className="text-center">
                  <Server size={58} className="mx-auto text-accent" />
                  <div className="mt-2">No site selected</div>
                  <button
                    type="button"
                    className="ghost-primary-button mt-4"
                    disabled={Boolean(action) || editing}
                title={editing ? "Save or cancel changes before leaving this site." : undefined}
                onClick={() => openNewConnection()}
                  >
                    <Plus size={14} /> New Site
                  </button>
                </div>
              </div>
            )}
          </aside>
        </div>
        )}

      </div>

      {cloudWizard === "oauth" && (
        <CloudAccountConnect
          onClose={() => setCloudWizard(null)}
          onSaved={() => setView("cloud")}
          onStorage={() => setCloudWizard("storage")}
        />
      )}
      {cloudWizard === "storage" && (
        <CloudStorageConnect
          onClose={() => setCloudWizard(null)}
          onBack={() => setCloudWizard("oauth")}
          onSaved={() => setView("cloud")}
        />
      )}

      {deletionTarget && (
        <ConfirmModal
          title="Delete saved site?"
          message={`Delete “${deletionTarget.name}” and its saved Ghost FTP credential? This cannot be undone.`}
          confirmLabel="Delete Site"
          destructive
          onClose={() => setConfirmDeleteId(null)}
          onConfirm={remove}
        />
      )}
    </div>
  );
}


function EditField({
  label,
  editing,
  children,
}: {
  label: string;
  editing: boolean;
  children: React.ReactNode;
}) {
  return (
    <label className="block">
      <div className="mb-1 text-[10px] text-text-dim">{label}</div>
      <div className={`ghost-ref-field ${editing ? "" : "opacity-90"}`}>
        {children}
      </div>
    </label>
  );
}


function SiteBandwidthLimit({ profileId, disabled }: { profileId: string; disabled: boolean }) {
  const [saved, setSaved] = useState<number | null>(null);
  const [value, setValue] = useState("");
  const [busy, setBusy] = useState(false);
  const [loadError, setLoadError] = useState(false);
  useEffect(() => {
    let active = true;
    ipc.transferProfileThrottle(profileId).then((rate) => {
      if (!active) return;
      setSaved(rate);
      setValue(String(rate));
    }).catch((error) => {
      if (!active) return;
      setLoadError(true);
      toastError(error, "Couldn't load site bandwidth limit");
    });
    return () => { active = false; };
  }, [profileId]);
  const rate = Number(value);
  const valid = value.trim() !== "" && Number.isSafeInteger(rate) && rate >= 0 && rate <= 1048576;
  const save = async () => {
    if (busy || disabled || saved === null || !valid || rate === saved) return;
    setBusy(true);
    try {
      const confirmed = await ipc.transferProfileThrottle(profileId, rate);
      setSaved(confirmed);
      setValue(String(confirmed));
      toast.success("Site bandwidth limit saved");
    } catch (error) {
      toastError(error, "Couldn't save site bandwidth limit");
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="mt-4 border-t border-border pt-3">
      <label className="block text-[11px] text-text-muted">
        Transfer limit (KiB/s)
        <input
          className="ghost-ref-input mt-1 w-full"
          type="number" min={0} max={1048576} step={1}
          value={value}
          onChange={(event) => setValue(event.target.value)}
          disabled={disabled || busy || saved === null}
        />
      </label>
      <p className="mt-1 text-[10px] text-text-muted">
        0 = no site limit. The global limit still applies.
      </p>
      {loadError && <p role="alert" className="mt-1 text-[11px] text-danger">Site limit unavailable.</p>}
      <button type="button" className="ghost-mini-button mt-2"
        disabled={disabled || busy || saved === null || !valid || rate === saved}
        onClick={() => void save()}>
        {busy ? "Saving…" : "Apply limit"}
      </button>
    </div>
  );
}
