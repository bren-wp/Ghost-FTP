import { useEffect, useMemo, useRef, useState } from "react";
import {
  CopyPlus,
  Download,
  Edit3,
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
import { ReferenceActionRow, ReferenceMenuTitlebar } from "./ReferenceWindowChrome";
import { ConfirmModal } from "./ConfirmModal";
import { ipc } from "@/lib/ipc";
import { toast } from "@/stores/toastStore";
import { toastError } from "@/lib/errors";

interface Props {
  onClose: () => void;
}

type View = "all" | "favorites" | "recent" | "bookmarks" | string;
type Action = "save" | "test" | "duplicate" | "delete" | null;

const DIRECT_EDIT_PROTOCOLS = new Set<Protocol>(["sftp", "ftp", "ftps"]);

export function SiteManagerDialog({ onClose }: Props) {
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
  const [selectedId, setSelectedId] = useState<string | null>(
    profiles[0]?.id ?? null
  );
  const [view, setView] = useState<View>("all");
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<ConnectionProfile | null>(
    profiles[0] ? { ...profiles[0] } : null
  );
  const [action, setAction] = useState<Action>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);

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
            `${profile.name} ${profile.host} ${profile.protocol} ${profile.group ?? ""} ${(
              profile.tags ?? []
            ).join(" ")}`.toLowerCase();
          if (query && !haystack.includes(query.toLowerCase())) return false;
          if (view === "favorites" && !profile.favorite) return false;
          if (view === "recent" && !profile.lastUsed) return false;
          if (view === "bookmarks" && !profile.bookmarked) return false;
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
        .sort((a, b) =>
          view === "recent"
            ? (b.lastUsed ?? 0) - (a.lastUsed ?? 0)
            : a.name.localeCompare(b.name)
        ),
    [profiles, query, view]
  );

  const selected =
    profiles.find((profile) => profile.id === selectedId) ?? profiles[0] ?? null;
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

  const exportSites = () => {
    const blob = new Blob([JSON.stringify(profiles, null, 2)], {
      type: "application/json",
    });
    const href = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = href;
    anchor.download = "ghostftp-sites.json";
    anchor.click();
    URL.revokeObjectURL(href);
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
    <section
      className="ghost-app-view ghost-standalone-view bg-[#041425]"
      aria-label="Site Manager"
    >
      <div className="ghost-site-manager flex h-full w-full flex-col overflow-hidden bg-[#061a2d]">
        <ReferenceMenuTitlebar onClose={onClose} />
        <ReferenceActionRow />

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
            onClick={exportSites}
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

        <div className="ghost-site-manager-workspace grid min-h-0 flex-1 grid-cols-[242px_minmax(0,1fr)_356px] gap-0">
          <aside className="ghost-site-manager-nav border-r border-border bg-[#051929] p-3">
            <SideItem
              active={view === "all"}
              icon={<Server />}
              label="All Sites"
              count={profiles.length}
              onClick={() => setView("all")}
            />
            <SideItem
              active={view === "favorites"}
              icon={<Star />}
              label="Favorites"
              count={profiles.filter((profile) => profile.favorite).length}
              onClick={() => setView("favorites")}
            />
            <SideItem
              active={view === "recent"}
              icon={<RadioTower />}
              label="Recent Servers"
              count={profiles.filter((profile) => profile.lastUsed).length}
              onClick={() => setView("recent")}
            />
            <SideItem
              active={view === "bookmarks"}
              icon={<Folder />}
              label="Bookmarks"
              count={profiles.filter((profile) => profile.bookmarked).length}
              onClick={() => setView("bookmarks")}
            />

            <div className="my-3 border-t border-border" />
            <div className="mb-2 flex items-center justify-between px-2 text-[11px] font-semibold text-accent">
              <span>Tags</span>
              <Tag size={13} />
            </div>
            {tags.length === 0 && (
              <div className="px-2 py-2 text-[11px] text-text-dim">No tags yet</div>
            )}
            {tags.map((tag) => (
              <SideItem
                key={tag}
                active={view === `tag:${tag}`}
                icon={<Tag />}
                label={tag}
                count={profiles.filter((profile) => (profile.tags ?? []).includes(tag)).length}
                onClick={() => setView(`tag:${tag}`)}
              />
            ))}

            <div className="my-3 border-t border-border" />
            <div className="mb-2 flex items-center justify-between px-2 text-[11px] font-semibold text-accent">
              <span>Folders</span>
              <Folder size={13} />
            </div>
            <SideItem
              active={view === "all"}
              icon={<Folder />}
              label="My Sites"
              count={profiles.length}
              onClick={() => setView("all")}
            />
            {folders.map((folder) => (
              <SideItem
                key={folder}
                active={view === `folder:${folder}`}
                icon={<Folder />}
                label={folder}
                count={profiles.filter((profile) => (profile.group ?? "") === folder).length}
                onClick={() => setView(`folder:${folder}`)}
              />
            ))}
          </aside>

          <section
            className="ghost-site-manager-table min-w-0 overflow-auto p-3"
            aria-label="Saved sites"
          >
            <div className="grid grid-cols-[minmax(180px,1.4fr)_minmax(150px,1fr)_90px_120px_110px] border-b border-border px-3 py-2 text-[11px] font-semibold text-text-dim">
              <span>Name</span>
              <span>Host</span>
              <span>Protocol</span>
              <span>Tags</span>
              <span>Last Used</span>
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
                connected={sessions.some(
                  (session) => session.profileId === profile.id
                )}
                onClick={() => select(profile.id)}
                onFavorite={() =>
                  void saveProfile({ ...profile, favorite: !profile.favorite })
                }
              />
            ))}
          </section>

          <aside className="ghost-site-manager-details border-l border-border bg-[#071f35] p-4 overflow-y-auto">
            {selected && draft ? (
              <>
                <div className="mb-4 flex items-center gap-3">
                  <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-accent-strong text-white">
                    <Server size={24} />
                  </div>
                  <div className="min-w-0">
                    <div className="truncate text-[17px] font-semibold">
                      {selected.name}
                    </div>
                    <div className="text-[11px] text-text-muted">
                      {selected.group || "Saved connection"}
                    </div>
                  </div>
                  <div className="flex-1" />
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

                {!canDirectEdit && (
                  <div className="mb-3 rounded-md border border-accent/25 bg-accent/10 px-3 py-2 text-[11px] text-text-muted">
                    {selected.protocol.toUpperCase()} authorization is account-bound.
                    Test and connect are available here; use its dedicated connection
                    flow to change credentials.
                  </div>
                )}

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
                  <EditField label="Folder" editing={editing}>
                    <input
                      value={draft.group || ""}
                      readOnly={!editing}
                      onChange={(event) =>
                        setDraft({
                          ...draft,
                          group: event.target.value.trimStart() || undefined,
                        })
                      }
                      placeholder="My Sites"
                    />
                  </EditField>
                  <EditField label="Tags" editing={editing}>
                    <input
                      value={(draft.tags ?? []).join(", ")}
                      readOnly={!editing}
                      onChange={(event) =>
                        setDraft({
                          ...draft,
                          tags: event.target.value
                            .split(",")
                            .map((value) => value.trim())
                            .filter(Boolean),
                        })
                      }
                      placeholder="Production, Client"
                    />
                  </EditField>
                  <label className="flex items-center justify-between gap-3 text-[11px] text-text-muted">
                    <span>Bookmark</span>
                    <input
                      type="checkbox"
                      disabled={!editing}
                      checked={draft.bookmarked === true}
                      onChange={(event) =>
                        setDraft({ ...draft, bookmarked: event.target.checked })
                      }
                    />
                  </label>
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
                    onClick={() => void connect(selected.id)}
                  >
                    <Link2 size={15} />
                    {isConnected ? "Connected" : "Connect"}
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

        <div className="flex h-10 min-h-10 items-center border-t border-border bg-[#051929] px-4 text-[11px] text-text-muted">
          <span className="mr-2 h-2 w-2 rounded-full bg-success" />
          {action === "test"
            ? "Testing connection…"
            : action === "save"
              ? "Saving site…"
              : action === "duplicate"
                ? "Duplicating site…"
                : action === "delete"
                  ? "Deleting site…"
                  : "Ready"}
          <div className="flex-1" />
          <Server size={13} className="mr-1" />
          {profiles.length} sites
        </div>
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
  connected,
  onClick,
  onFavorite,
}: {
  profile: ConnectionProfile;
  active: boolean;
  favorite: boolean;
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
        <em className="not-italic rounded-full border border-accent/30 bg-accent/10 px-2 py-1 text-[10px] text-accent">
          {tag}
        </em>
      </span>
      <span className={connected ? "text-success" : "text-text-dim"}>
        {connected ? "Connected" : formatLastUsed(profile.lastUsed)}
      </span>
    </div>
  );
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
