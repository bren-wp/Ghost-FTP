import { useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import {
  Braces,
  Plus,
  X,
  Search,
  Pencil,
  Trash2,
  TerminalSquare,
  Folder,
} from "lucide-react";
import { useSnippets, newSnippet } from "@/stores/snippetsStore";
import { extractVariables } from "@/lib/snippets";
import { useDialog } from "@/hooks/useDialog";
import { ConfirmModal } from "./ConfirmModal";
import { VariableDialog } from "./VariableDialog";
import type { Snippet } from "@/lib/types";
import { cn } from "@/lib/cn";

/// Snippets subsystem host (Plan 11 Phase 4). Always mounted from App: boots the
/// store, renders the full-screen management panel when open, and renders the
/// insert dialog whenever a snippet insertion needs variables / a multi-line
/// confirm — the dialog is hosted here (not inside the panel) so palette and
/// toolbar inserts work with the panel closed.
export function SnippetsHost() {
  const init = useSnippets((s) => s.init);
  const open = useSnippets((s) => s.open);
  const insertTarget = useSnippets((s) => s.insertTarget);
  const cancelInsert = useSnippets((s) => s.cancelInsert);
  const commitInsert = useSnippets((s) => s.commitInsert);

  useEffect(() => {
    void init();
  }, [init]);

  return (
    <>
      {open && <SnippetsPanel />}
      {insertTarget && (
        <VariableDialog
          snippet={insertTarget}
          onCancel={cancelInsert}
          onInsert={(values) => void commitInsert(insertTarget, values)}
        />
      )}
    </>
  );
}

function SnippetsPanel() {
  const snippets = useSnippets((s) => s.snippets);
  const close = useSnippets((s) => s.closePanel);
  const remove = useSnippets((s) => s.remove);
  const requestInsert = useSnippets((s) => s.requestInsert);

  const [query, setQuery] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [editing, setEditing] = useState<Snippet | "new" | null>(null);
  const [deleting, setDeleting] = useState<Snippet | null>(null);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return snippets;
    return snippets.filter((s) =>
      `${s.name} ${s.body} ${s.folder ?? ""}`.toLowerCase().includes(q)
    );
  }, [snippets, query]);

  // Group by folder for the list; ungrouped last. Preserves the store's
  // most-used-first order within each group.
  const groups = useMemo(() => {
    const map = new Map<string, Snippet[]>();
    for (const s of filtered) {
      const key = s.folder?.trim() || "";
      (map.get(key) ?? map.set(key, []).get(key)!).push(s);
    }
    return [...map.entries()].sort(([a], [b]) => {
      if (a === "") return 1; // ungrouped sinks to the bottom
      if (b === "") return -1;
      return a.localeCompare(b);
    });
  }, [filtered]);

  const selected = useMemo(
    () => snippets.find((s) => s.id === selectedId) ?? null,
    [snippets, selectedId]
  );

  useEffect(() => {
    if (filtered.length === 0) {
      setSelectedId(null);
      return;
    }
    if (!selectedId || !filtered.some((s) => s.id === selectedId)) {
      setSelectedId(filtered[0].id);
    }
  }, [filtered, selectedId]);

  // Esc closes the panel unless an inner dialog is open.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !editing && !deleting) close();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [close, editing, deleting]);

  return createPortal(
    <div className="anim-modal fixed inset-0 z-modal flex flex-col bg-bg">
      {/* Header */}
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-panel px-3 py-2">
        <Braces size={15} className="shrink-0 text-accent" />
        <span className="shrink-0 text-sm font-semibold">Snippets</span>
        <span className="hidden shrink-0 text-xs text-text-dim sm:inline">
          Saved command lines with {"{{variables}}"}, inserted into a live shell
        </span>
        <div className="ml-auto flex items-center gap-2">
          <button
            onClick={() => setEditing("new")}
            className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs hover:bg-bg-subtle"
          >
            <Plus size={13} /> New snippet
          </button>
          <button
            onClick={close}
            className="rounded-md p-1 hover:bg-bg-subtle"
            title="Close (Esc)"
          >
            <X size={16} />
          </button>
        </div>
      </div>

      {/* Body: list + detail */}
      <div className="flex min-h-0 flex-1">
        <div className="flex w-80 shrink-0 flex-col border-r border-border">
          <div className="flex items-center gap-2 border-b border-border px-3 py-2">
            <Search size={13} className="shrink-0 text-text-dim" />
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search snippets…"
              className="w-full bg-transparent text-xs outline-none placeholder:text-text-dim"
            />
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto">
            {snippets.length === 0 ? (
              <div className="p-4 text-xs leading-relaxed text-text-dim">
                No snippets yet. Click <b>New snippet</b> to save a command you
                type often — add <code>{"{{name}}"}</code> placeholders and Ghost FTP
                asks for the values when you insert it.
              </div>
            ) : filtered.length === 0 ? (
              <div className="p-4 text-xs text-text-dim">No matches.</div>
            ) : (
              groups.map(([folder, items]) => (
                <div key={folder || "__ungrouped"}>
                  <div className="flex items-center gap-1.5 px-3 pt-2.5 pb-1 text-[10px] font-semibold uppercase tracking-wider text-text-dim">
                    <Folder size={10} />
                    {folder || "Ungrouped"}
                  </div>
                  {items.map((s) => (
                    <SnippetRow
                      key={s.id}
                      snippet={s}
                      selected={s.id === selectedId}
                      onSelect={() => setSelectedId(s.id)}
                      onInsert={() => requestInsert(s)}
                      onEdit={() => setEditing(s)}
                      onDelete={() => setDeleting(s)}
                    />
                  ))}
                </div>
              ))
            )}
          </div>
        </div>

        <div className="min-w-0 flex-1 overflow-y-auto">
          {selected ? (
            <SnippetDetail
              key={selected.id}
              snippet={selected}
              onInsert={() => requestInsert(selected)}
              onEdit={() => setEditing(selected)}
            />
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-text-dim">
              Select a snippet, or create one.
            </div>
          )}
        </div>
      </div>

      {editing && (
        <SnippetEditor
          snippet={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={(id) => setSelectedId(id)}
        />
      )}
      {deleting && (
        <ConfirmModal
          title={`Delete "${deleting.name || "(unnamed)"}"?`}
          message="This removes the snippet. This can't be undone."
          destructive
          confirmLabel="Delete"
          onClose={() => setDeleting(null)}
          onConfirm={() => {
            void remove(deleting.id);
            setDeleting(null);
          }}
        />
      )}
    </div>,
    document.body
  );
}

function SnippetRow({
  snippet,
  selected,
  onSelect,
  onInsert,
  onEdit,
  onDelete,
}: {
  snippet: Snippet;
  selected: boolean;
  onSelect: () => void;
  onInsert: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const varCount = extractVariables(snippet.body).length;
  return (
    <div
      onClick={onSelect}
      className={cn(
        "group cursor-pointer border-b border-border/50 px-3 py-2",
        selected ? "bg-bg-subtle" : "hover:bg-bg-subtle/50"
      )}
    >
      <div className="flex items-center gap-2">
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium">
          {snippet.name || "(unnamed)"}
        </span>
        {varCount > 0 && (
          <span
            className="shrink-0 rounded bg-accent/10 px-1.5 py-0.5 font-mono text-[10px] text-accent"
            title={`${varCount} variable${varCount === 1 ? "" : "s"}`}
          >
            {varCount} var{varCount === 1 ? "" : "s"}
          </span>
        )}
        <button
          onClick={(e) => {
            e.stopPropagation();
            onInsert();
          }}
          className="shrink-0 rounded p-0.5 text-text-dim opacity-0 hover:text-accent group-hover:opacity-100"
          title="Insert into terminal"
        >
          <TerminalSquare size={13} />
        </button>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onEdit();
          }}
          className="shrink-0 rounded p-0.5 text-text-dim opacity-0 hover:text-text group-hover:opacity-100"
          title="Edit"
        >
          <Pencil size={12} />
        </button>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onDelete();
          }}
          className="shrink-0 rounded p-0.5 text-text-dim opacity-0 hover:text-danger group-hover:opacity-100"
          title="Delete"
        >
          <Trash2 size={12} />
        </button>
      </div>
      <div className="mt-0.5 truncate font-mono text-[11px] text-text-dim">
        {snippet.body.split("\n")[0] || " "}
      </div>
    </div>
  );
}

function SnippetDetail({
  snippet,
  onInsert,
  onEdit,
}: {
  snippet: Snippet;
  onInsert: () => void;
  onEdit: () => void;
}) {
  const vars = extractVariables(snippet.body);
  return (
    <div className="flex h-full flex-col p-5">
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="truncate text-base font-semibold">
            {snippet.name || "(unnamed)"}
          </div>
          <div className="mt-0.5 text-xs text-text-dim">
            {snippet.folder ? `${snippet.folder} · ` : ""}
            used {snippet.useCount} time{snippet.useCount === 1 ? "" : "s"}
          </div>
        </div>
        <button
          onClick={onEdit}
          className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-xs hover:bg-bg-subtle"
        >
          <Pencil size={12} /> Edit
        </button>
        <button
          onClick={onInsert}
          className="btn-accent flex shrink-0 items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium text-white"
        >
          <TerminalSquare size={13} /> Insert
        </button>
      </div>

      <div className="mt-4 text-[10px] font-semibold uppercase tracking-wider text-text-dim">
        Command
      </div>
      <pre className="mt-1 max-h-[40vh] overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-bg-subtle px-3 py-2.5 font-mono text-[13px] text-text">
        {snippet.body || " "}
      </pre>

      {vars.length > 0 && (
        <>
          <div className="mt-4 text-[10px] font-semibold uppercase tracking-wider text-text-dim">
            Variables
          </div>
          <div className="mt-1.5 flex flex-wrap gap-1.5">
            {vars.map((v) => (
              <span
                key={v}
                className="rounded bg-accent/10 px-2 py-0.5 font-mono text-[11px] text-accent"
              >
                {v}
              </span>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function SnippetEditor({
  snippet,
  onClose,
  onSaved,
}: {
  snippet: Snippet | null;
  onClose: () => void;
  onSaved: (id: string) => void;
}) {
  const save = useSnippets((s) => s.save);
  const [draft, setDraft] = useState<Snippet>(() => snippet ?? newSnippet());

  const panelRef = useRef<HTMLDivElement>(null);
  const nameRef = useRef<HTMLInputElement>(null);
  useDialog(panelRef, { onClose, initialFocus: nameRef });

  const vars = extractVariables(draft.body);
  const canSave = draft.name.trim().length > 0 && draft.body.trim().length > 0;

  const commit = () => {
    if (!canSave) return;
    const clean: Snippet = {
      ...draft,
      name: draft.name.trim(),
      folder: draft.folder?.trim() ? draft.folder.trim() : null,
    };
    void save(clean);
    onSaved(clean.id);
    onClose();
  };

  return (
    <div
      className="fixed inset-0 z-palette flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label={snippet ? "Edit snippet" : "New snippet"}
        onClick={(e) => e.stopPropagation()}
        className="anim-modal flex w-[38rem] max-w-[94vw] flex-col rounded-xl border border-border bg-bg-panel p-5 shadow-elev-3"
      >
        <div className="mb-3 text-sm font-semibold">
          {snippet ? "Edit snippet" : "New snippet"}
        </div>

        <div className="flex gap-2.5">
          <label className="flex flex-1 flex-col gap-1">
            <span className="text-[11px] text-text-muted">Name</span>
            <input
              ref={nameRef}
              value={draft.name}
              onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
              placeholder="Restart web server"
              className="rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent"
            />
          </label>
          <label className="flex w-40 flex-col gap-1">
            <span className="text-[11px] text-text-muted">Folder</span>
            <input
              value={draft.folder ?? ""}
              onChange={(e) =>
                setDraft((d) => ({ ...d, folder: e.target.value }))
              }
              placeholder="optional"
              className="rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent"
            />
          </label>
        </div>

        <label className="mt-3 flex flex-col gap-1">
          <span className="text-[11px] text-text-muted">Command</span>
          <textarea
            value={draft.body}
            onChange={(e) => setDraft((d) => ({ ...d, body: e.target.value }))}
            placeholder="wp core update --path=/var/www/{{site}}"
            rows={5}
            spellCheck={false}
            className="resize-y rounded-md border border-border bg-bg-subtle px-2.5 py-2 font-mono text-[13px] outline-none focus:border-accent"
          />
        </label>

        <div className="mt-2 flex items-center gap-2 text-[11px] text-text-dim">
          {vars.length > 0 ? (
            <>
              <span>Variables:</span>
              <div className="flex flex-wrap gap-1">
                {vars.map((v) => (
                  <span
                    key={v}
                    className="rounded bg-accent/10 px-1.5 py-0.5 font-mono text-accent"
                  >
                    {v}
                  </span>
                ))}
              </div>
            </>
          ) : (
            <span>
              Tip: add <code className="text-accent">{"{{name}}"}</code>{" "}
              placeholders to prompt for values on insert.
            </span>
          )}
        </div>

        <div className="mt-4 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded-md border border-border px-3.5 py-1.5 text-sm hover:bg-bg-hover"
          >
            Cancel
          </button>
          <button
            onClick={commit}
            disabled={!canSave}
            className="btn-accent rounded-md px-3.5 py-1.5 text-sm font-medium text-white disabled:opacity-40"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
