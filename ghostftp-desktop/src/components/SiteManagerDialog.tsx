import { useEffect, useMemo, useRef, useState } from "react";
import { save as saveDialog } from "@tauri-apps/plugin-dialog";
import {
  Bookmark,
  ChevronDown,
  ChevronUp,
  CopyPlus,
  Download,
  Edit3,
  Eye,
  EyeOff,
  Folder,
  Link2,
  Plus,
  RadioTower,
  Save,
  Search,
  Server,
  Star,
  Tag,
  Trash2,
  Upload,
} from "lucide-react";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import type { ConnectionProfile, Protocol } from "@/lib/types";
import { PROTOCOL_DEFAULT_PORT } from "@/lib/types";
import { ConfirmModal } from "./ConfirmModal";
import { ipc } from "@/lib/ipc";
import { toast } from "@/stores/toastStore";
import { toastError } from "@/lib/errors";
import { useDialog } from "@/hooks/useDialog";

interface Props {
  onClose: () => void;
  initialView?: View;
}

type View = "all" | "favorites" | "recent" | "bookmarks" | string;
type Action = "save" | "test" | "connect" | "duplicate" | "delete" | null;
type SiteSortField = "name" | "host" | "protocol" | "lastUsed";

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
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [addingMeta, setAddingMeta] = useState<"tag" | "folder" | null>(null);
  const [metaValue, setMetaValue] = useState("");
  const [favoriteBusyId, setFavoriteBusyId] = useState<string | null>(null);

  useDialog(panelRef, { onClose, initialFocus: searchRef, trapFocus: false });

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

  const filtered = useMemo(
    () =>
      profiles
        .filter((profile) => {
          const haystack =
            `${profile.name} ${profile.host} ${profile.protocol} ${profile.group ?? ""} ${profile.description ?? ""} ${(
              profile.tags ?? []
            ).join(" ")}`.toLowerCase();
          if (query && !haystack.includes(query.toLowerCase())) return false;
          if (view === "favorites" && !profile.favorite) return false;
          if (view === "recent" && !profile.lastUsed) return false;
          if (view === "bookmarks" && !profile.bookmarked) return false;
          if (
            view === "cloud" &&
            !["s3", "azure", "gcs", "webdav", "dropbox", "onedrive", "gdrive", "box"].includes(profile.protocol)
          ) return false;
          if (
            view.startsWith("tag:") &&
            !(profile.tags ?? []).includes(view.slice(4))
          ) {
            return false;
          }
          if (
            view.startsWith("folder:") &&
            (profile.group || "") !== view.slice(7)
          ) {
            return false;
          }
          return true;
        })
        .sort((a, b) => {
          let cmp = 0;
          if (sortField === "host") cmp = a.host.localeCompare(b.host);
          else if (sortField === "protocol") cmp = a.protocol.localeCompare(b.protocol);
          else if (sortField === "lastUsed") cmp = (a.lastUsed ?? 0) - (b.lastUsed ?? 0);
          else cmp = a.name.localeCompare(b.name);
          return sortDirection === "asc" ? cmp : -cmp;
        }),
    [profiles, query, view, sortField, sortDirection]
  );

  const toggleSort = (field: SiteSortField) => {
    if (field === sortField) {
      setSortDirection((direction) => (direction === "asc" ? "desc" : "asc"));
      return;
    }
    setSortField(field);
    setSortDirection(field === "lastUsed" ? "desc" : "asc");
  };

  const selected =
    filtered.find((profile) => profile.id === selectedId) ?? filtered[0] ?? null;
  const isConnected = selected
    ? sessions.some((session) => session.profileId === selected.id)
    : false;
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
    const profile = profiles.find((item) => item.id === id) ?? null;
    setSelectedId(id);
    setDraft(profile ? { ...profile } : null);
    setEditing(false);
  };

  const startEdit = () => {
    if (!selected || !canDirectEdit) return;
    setDraft({ ...selected });
    setEditing(true);
  };

  const save = async () => {
    if (!draft || action) return;
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

    setAction("save");
    try {
      await saveProfile({ ...draft, name, host, username });
      setEditing(false);
      toast.success("Site saved", name);
    } catch (error) {
      toastError(error, `Couldn't save ${name}`);
    } finally {
      setAction(null);
    }
  };

  const toggleBookmark = async () => {
    if (!selected || action) return;
    setAction("save");
    try {
      await saveProfile({ ...selected, bookmarked: !selected.bookmarked });
      toast.info(
        selected.bookmarked ? "Bookmark removed" : "Bookmarked",
        selected.name
      );
    } catch (error) {
      toastError(error, `Couldn't update ${selected.name}`);
    } finally {
      setAction(null);
    }
  };

  const remove = async () => {
    if (!selected || action) return;
    const index = profiles.findIndex((profile) => profile.id === selected.id);
    const next = profiles[index + 1] ?? profiles[index - 1] ?? null;

    setAction("delete");
    try {
      await deleteProfile(selected.id);
      setSelectedId(next?.id ?? null);
      setDraft(next ? { ...next } : null);
      setEditing(false);
      toast.info("Site deleted", selected.name);
    } catch (error) {
      toastError(error, `Couldn't delete ${selected.name}`);
    } finally {
      setAction(null);
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
    if (!selected || isConnected || action || editing) return;
    setAction("connect");
    try {
      await connect(selected.id);
    } catch (error) {
      // connectionsStore already surfaces the structured FTP/FTPS/SFTP error.
      console.debug("Site Manager connection failure was surfaced by the connections store", error);
    } finally {
      setAction(null);
    }
  };

  const testSelected = async () => {
    if (!selected || action) return;
    setAction("test");
    try {
      await ipc.testProfileConnection(selected.id);
      toast.success(
        "Connection successful",
        `${selected.name} — ${selected.username}@${selected.host}:${selected.port}`
      );
    } catch (error) {
      toastError(error, `Couldn't connect to ${selected.name}`);
    } finally {
      setAction(null);
    }
  };

  const applyMeta = async () => {
    if (!selected || !addingMeta || action) return;
    const value = metaValue.trim();
    if (!value) return;
    setAction("save");
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
      setAction(null);
    }
  };

  const toggleFavorite = async (profile: ConnectionProfile) => {
    if (favoriteBusyId) return;
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
    }
  };

  const duplicateSelected = async () => {
    if (!selected || !canDuplicate || action) return;
    setAction("duplicate");
    try {
      const copy = await duplicateProfile(selected.id);
      setSelectedId(copy.id);
      setDraft({ ...copy });
      setEditing(false);
      toast.success("Site duplicated", copy.name);
    } catch (error) {
      toastError(error, `Couldn't duplicate ${selected.name}`);
    } finally {
      setAction(null);
    }
  };

  return (
    <div
      className="ghost-workspace-view ghost-standalone-view bg-[#041425]"
      role="region"
      aria-label="Site Manager"
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
            <div className="text-xl font-semibold">Site Manager</div>
            <div className="text-[12px] text-text-muted">
              Manage saved connections, bookmarks and server profiles.
            </div>
          </div>
          <div className="flex-1" />
          <div className="relative">
            <Search
              size={14}
              className="pointer-events-none absolute left-3 top-2.5 text-text-dim"
            />
            <input
              ref={searchRef}
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              className="ghost-ref-input w-64 pl-9"
              placeholder="Search sites…"
              aria-label="Search saved sites"
            />
          </div>
          <button
            type="button"
            className="ghost-mini-button"
            onClick={() => openDialog("import")}
          >
            <Download size={14} /> Import
          </button>
          <button
            type="button"
            className="ghost-mini-button"
            onClick={() => void exportSites()}
            disabled={profiles.length === 0}
          >
            <Upload size={14} /> Export
          </button>
          <button
            type="button"
            className="ghost-primary-button"
            onClick={() => openNewConnection()}
          >
            <Plus size={15} /> New Site
          </button>
        </div>

        <div className="ghost-site-filterbar flex shrink-0 items-center gap-2 overflow-x-auto border-b border-border bg-[#051929] px-3 py-2">
          <FilterChip active={view === "all"} label="All" count={profiles.length} onClick={() => setView("all")} />
          <FilterChip active={view === "favorites"} label="Favorites" count={profiles.filter((profile) => profile.favorite).length} onClick={() => setView("favorites")} />
          <FilterChip active={view === "recent"} label="Recent" count={profiles.filter((profile) => profile.lastUsed).length} onClick={() => setView("recent")} />
          <FilterChip active={view === "bookmarks"} label="Bookmarks" count={profiles.filter((profile) => profile.bookmarked).length} onClick={() => setView("bookmarks")} />
          <FilterChip active={view === "cloud"} label="Cloud" count={profiles.filter((profile) => ["s3", "azure", "gcs", "webdav", "dropbox", "onedrive", "gdrive", "box"].includes(profile.protocol)).length} onClick={() => setView("cloud")} />
          <div className="h-6 w-px shrink-0 bg-border"/>
          <label className="ghost-site-filter-select">Tag<select aria-label="Filter by tag" value={view.startsWith("tag:") ? view : ""} onChange={(event) => setView(event.target.value || "all")}><option value="">All tags</option>{tags.map((tag) => <option key={tag} value={`tag:${tag}`}>{tag}</option>)}</select></label>
          <label className="ghost-site-filter-select">Folder<select aria-label="Filter by folder" value={view.startsWith("folder:") ? view : ""} onChange={(event) => setView(event.target.value || "all")}><option value="">All folders</option>{folders.map((folder) => <option key={folder} value={`folder:${folder}`}>{folder}</option>)}</select></label>
        </div>

        {profiles.length === 0 && !query ? (
          <div className="ghost-sites-empty">
            <div className="ghost-sites-empty-card">
              <Server size={52} className="mx-auto text-accent" />
              <h2 className="mt-4 text-[20px] font-semibold">Add your first site</h2>
              <p className="mx-auto mt-2 max-w-md text-[12px] leading-5 text-text-muted">
                Save an FTP, FTPS, SFTP or cloud connection here so it is easy to find and reconnect later.
              </p>
              <div className="mt-5 flex flex-wrap justify-center gap-2">
                <button
                  type="button"
                  className="ghost-primary-button"
                  onClick={() => openNewConnection()}
                >
                  <Plus size={15} /> New Site
                </button>
                <button
                  type="button"
                  className="ghost-mini-button"
                  onClick={() => openDialog("import")}
                >
                  <Download size={14} /> Import Sites
                </button>
              </div>
            </div>
          </div>
        ) : (
          <div className="ghost-site-manager-workspace grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_356px] gap-0">
          <section
            className="ghost-site-manager-table min-w-0 overflow-auto p-3"
            aria-label="Saved sites"
          >
            <div className="grid grid-cols-[minmax(180px,1.4fr)_minmax(150px,1fr)_90px_120px_110px] border-b border-border px-3 py-2 text-[11px] font-semibold text-text-dim">
              <SiteSortHeader label="Name" field="name" activeField={sortField} direction={sortDirection} onSort={toggleSort}/>
              <SiteSortHeader label="Host" field="host" activeField={sortField} direction={sortDirection} onSort={toggleSort}/>
              <SiteSortHeader label="Protocol" field="protocol" activeField={sortField} direction={sortDirection} onSort={toggleSort}/>
              <span className="flex items-center">Tags</span>
              <SiteSortHeader label="Last Used" field="lastUsed" activeField={sortField} direction={sortDirection} onSort={toggleSort}/>
            </div>

            {filtered.length === 0 && (
              <div className="p-10 text-center text-text-muted">
                {query
                  ? "No saved sites match your search."
                  : "No saved sites in this view."}
              </div>
            )}

            {filtered.map((profile) => (
              <SiteRow
                key={profile.id}
                profile={profile}
                active={selected?.id === profile.id}
                favorite={profile.favorite === true}
                favoriteBusy={favoriteBusyId === profile.id}
                connected={sessions.some(
                  (session) => session.profileId === profile.id
                )}
                onClick={() => select(profile.id)}
                onFavorite={() => void toggleFavorite(profile)}
              />
            ))}
            <div className="ghost-site-table-footer sticky bottom-0 mt-1 flex h-8 items-center border-t border-border bg-[#051929] px-3 text-[10px] text-text-muted">
              <span>{profiles.length} sites total</span>
              <span className="mx-2 text-text-dim">|</span>
              <span>{selected ? `1 selected (${selected.name})` : "0 selected"}</span>
            </div>
          </section>

          <aside className="ghost-site-manager-details border-l border-border bg-[#071f35] p-4 overflow-y-auto">
            {selected && draft ? (
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
                    <div className="flex items-center gap-2"><span className="min-w-0 flex-1 truncate text-[11px]">{selected.group || "None"}</span><button type="button" className="ghost-site-meta-add" disabled={Boolean(action)} onClick={() => { setAddingMeta("folder"); setMetaValue(selected.group ?? ""); }}><Edit3 size={12}/></button></div>
                    {addingMeta === "folder" && <div className="ghost-site-meta-editor mt-2"><input autoFocus value={metaValue} placeholder="Folder name" onChange={(event) => setMetaValue(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") void applyMeta(); if (event.key === "Escape") setAddingMeta(null); }}/><button type="button" onClick={() => void applyMeta()} disabled={!metaValue.trim()}>Set</button><button type="button" onClick={() => setAddingMeta(null)}>×</button></div>}
                  </div>
                  <div className="rounded-md border border-border-subtle bg-[#051929] p-2.5">
                    <div className="mb-1 text-[10px] uppercase tracking-wide text-text-dim">Tags</div>
                    <div className="flex items-center gap-2"><span className="min-w-0 flex-1 truncate text-[11px]">{(selected.tags ?? []).join(", ") || "None"}</span><button type="button" className="ghost-site-meta-add" disabled={Boolean(action)} onClick={() => { setAddingMeta("tag"); setMetaValue(""); }}><Plus size={12}/></button></div>
                    {addingMeta === "tag" && <div className="ghost-site-meta-editor mt-2"><input autoFocus value={metaValue} placeholder="Tag name" onChange={(event) => setMetaValue(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") void applyMeta(); if (event.key === "Escape") setAddingMeta(null); }}/><button type="button" onClick={() => void applyMeta()} disabled={!metaValue.trim()}>Add</button><button type="button" onClick={() => setAddingMeta(null)}>×</button></div>}
                  </div>
                </div>

                <div className="space-y-3 border-t border-border pt-4">
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
                          onClick={() => setShowPassword((value) => !value)}
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
                </div>

                <div className="mt-5 flex gap-2">
                  <button
                    type="button"
                    disabled={isConnected || Boolean(action) || editing}
                    className="ghost-primary-button flex-1"
                    onClick={() => void connectSelected()}
                  >
                    <Link2 size={15} />
                    {isConnected ? "Connected" : action === "connect" ? "Connecting…" : "Connect"}
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
                    onClick={() => setConfirmDelete(true)}
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

      {confirmDelete && selected && (
        <ConfirmModal
          title="Delete saved site?"
          message={`Delete “${selected.name}” and its saved Ghost FTP credential? This cannot be undone.`}
          confirmLabel="Delete Site"
          destructive
          onClose={() => setConfirmDelete(false)}
          onConfirm={() => void remove()}
        />
      )}
    </div>
  );
}


function FilterChip({ active, label, count, onClick }: { active: boolean; label: string; count: number; onClick: () => void }) {
  return <button type="button" className={`ghost-site-filter-chip ${active ? "active" : ""}`} aria-pressed={active} onClick={onClick}><span>{label}</span><b>{count}</b></button>;
}

function SiteSortHeader({
  label,
  field,
  activeField,
  direction,
  onSort,
}: {
  label: string;
  field: SiteSortField;
  activeField: SiteSortField;
  direction: "asc" | "desc";
  onSort: (field: SiteSortField) => void;
}) {
  const active = activeField === field;
  return (
    <button
      type="button"
      className={`flex min-w-0 items-center gap-1 text-left hover:text-white focus-visible:outline-none focus-visible:text-white ${active ? "text-text" : "text-text-dim"}`}
      aria-label={`Sort by ${label}`}
      aria-pressed={active}
      onClick={() => onSort(field)}
    >
      <span className="truncate">{label}</span>
      {active ? direction === "asc" ? <ChevronUp size={12}/> : <ChevronDown size={12}/> : null}
    </button>
  );
}

function SideItem({
  icon,
  label,
  count,
  active = false,
  onClick,
}: {
  icon: React.ReactNode;
  label: string;
  count: number;
  active?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "page" : undefined}
      className={`mb-1 flex w-full items-center gap-2 rounded-md border px-2.5 py-2 text-left text-[12px] ${
        active
          ? "border-accent/35 bg-accent/15 text-white"
          : "border-transparent text-text-muted hover:bg-bg-hover hover:text-white"
      }`}
    >
      <span className="[&>svg]:h-4 [&>svg]:w-4 text-accent">{icon}</span>
      <span className="min-w-0 flex-1 truncate">{label}</span>
      <span className="rounded bg-[#0a2d4b] px-1.5 py-.5 text-[10px]">
        {count}
      </span>
    </button>
  );
}

function SiteRow({
  profile,
  active,
  favorite,
  favoriteBusy,
  connected,
  onClick,
  onFavorite,
}: {
  profile: ConnectionProfile;
  active: boolean;
  favorite: boolean;
  favoriteBusy: boolean;
  connected: boolean;
  onClick: () => void;
  onFavorite: () => void;
}) {
  const tag = profile.tags?.[0] || profile.group || "Sites";
  const activateFromKeyboard = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onClick();
    }
  };

  return (
    <div
      role="button"
      tabIndex={0}
      aria-pressed={active}
      onClick={onClick}
      onKeyDown={activateFromKeyboard}
      className={`grid w-full cursor-pointer grid-cols-[minmax(180px,1.4fr)_minmax(150px,1fr)_90px_120px_110px] items-center border-b border-border-subtle px-3 py-2.5 text-left text-[12px] outline-none focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-accent ${
        active
          ? "bg-accent/15 outline outline-1 outline-accent/60"
          : "hover:bg-bg-hover/70"
      }`}
    >
      <span className="flex min-w-0 items-center gap-2">
        <button
          type="button"
          onClick={(event) => {
            event.stopPropagation();
            onFavorite();
          }}
          onKeyDown={(event) => event.stopPropagation()}
          className="rounded p-0.5 text-text-dim outline-none hover:bg-bg-hover hover:text-warning focus-visible:ring-1 focus-visible:ring-accent"
          disabled={favoriteBusy}
          aria-label={favorite ? "Remove from favorites" : "Add to favorites"}
          aria-pressed={favorite}
        >
          <Star
            size={14}
            className={
              favorite
                ? "shrink-0 fill-warning text-warning"
                : "shrink-0 text-text-dim"
            }
          />
        </button>
        <Server size={14} className="shrink-0 text-accent" />
        <span className="truncate font-medium">{profile.name}</span>
      </span>
      <span className="truncate text-text-muted">{profile.host}</span>
      <span className="text-text-muted">{profile.protocol.toUpperCase()}</span>
      <span>
        <em className={`not-italic rounded-full border px-2 py-1 text-[10px] ${tagToneClass(tag)}`}>
          {tag}
        </em>
      </span>
      <span className={connected ? "text-success" : "text-text-dim"}>
        {connected ? "Connected" : formatLastUsed(profile.lastUsed)}
      </span>
    </div>
  );
}


function tagToneClass(label: string) {
  const normalized = label.trim().toLowerCase();
  if (normalized.includes("production")) return "border-sky-400/55 bg-sky-500/15 text-sky-300";
  if (normalized.includes("staging")) return "border-emerald-400/55 bg-emerald-500/15 text-emerald-300";
  if (normalized.includes("personal")) return "border-pink-400/55 bg-pink-500/15 text-pink-300";
  if (normalized.includes("client")) return "border-orange-400/55 bg-orange-500/15 text-orange-300";
  if (normalized.includes("development") || normalized.includes("dev")) return "border-violet-400/55 bg-violet-500/15 text-violet-300";
  if (normalized.includes("backup")) return "border-rose-400/55 bg-rose-500/15 text-rose-300";
  const palette = [
    "border-sky-400/55 bg-sky-500/15 text-sky-300",
    "border-emerald-400/55 bg-emerald-500/15 text-emerald-300",
    "border-violet-400/55 bg-violet-500/15 text-violet-300",
    "border-orange-400/55 bg-orange-500/15 text-orange-300",
    "border-pink-400/55 bg-pink-500/15 text-pink-300",
    "border-rose-400/55 bg-rose-500/15 text-rose-300",
  ];
  const hash = Array.from(normalized).reduce((sum, char) => sum + char.charCodeAt(0), 0);
  return palette[hash % palette.length];
}

function TagDot({ label }: { label: string }) {
  const tone = tagToneClass(label);
  const background =
    tone.includes("emerald") ? "bg-emerald-400" :
    tone.includes("violet") ? "bg-violet-400" :
    tone.includes("orange") ? "bg-orange-400" :
    tone.includes("pink") ? "bg-pink-400" :
    tone.includes("rose") ? "bg-rose-400" : "bg-sky-400";
  return <span className={`inline-block h-3 w-3 rounded-full shadow-[0_0_8px_currentColor] ${background}`} aria-hidden="true"/>;
}

function formatLastUsed(value?: number) {
  if (!value) return "—";
  const date = new Date(value * 1000);
  const now = new Date();
  return date.toDateString() === now.toDateString()
    ? date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    : date.toLocaleDateString();
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
