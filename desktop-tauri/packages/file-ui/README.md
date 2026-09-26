# @ghostftp/file-ui

A transport-agnostic React file-browser UI, extracted from
[Ghost FTP](../../) (File & Agent Remote Operations). It
ships the file **list/grid/details pane** — breadcrumbs, multi-select, keyboard
nav, type-ahead, drag-and-drop, sort, rename/delete/mkdir/chmod, and an optional
edit-in-place action — with **zero knowledge of how files are fetched**.

You supply a `FileSystemAdapter`; the components call that. Back it with SFTP,
S3, an HTTP API, or an in-memory mock — the UI doesn't change.

## Why it's decoupled

The components never import a transport (no Tauri `invoke`, no `fetch`). They
read everything through React context:

- a **`FileSystemAdapter`** — the data/IO operations
- a **`PaneSettings`** object — controlled view state (sort, view mode, density)

Swap the adapter and the same pane drives a completely different backend.

## Usage

```tsx
import { FilePane, FileUiProvider, type FileSystemAdapter } from "@ghostftp/file-ui";

const fs: FileSystemAdapter = {
  listDirectory: (sessionId, path) => myApi.list(sessionId, path),
  capabilities: () => Promise.resolve({
    canChmod: true, canSymlink: true, canRename: true, hasDirectories: true,
  }),
  rename: (s, from, to) => myApi.rename(s, from, to),
  remove: (s, path, recursive) => myApi.remove(s, path, recursive),
  mkdir: (s, path) => myApi.mkdir(s, path),
  chmod: (s, path, mode) => myApi.chmod(s, path, mode),
  // editFile is optional — omit it to hide the "Edit…" action.
};

function Browser() {
  const [path, setPath] = useState("/");
  return (
    <FileUiProvider fs={fs} settings={mySettings}>
      <FilePane
        paneId="remote"
        title="My Server"
        sessionId="session-123"
        path={path}
        onPathChange={setPath}
        onTransfer={(entries) => myApi.download(entries)}
      />
    </FileUiProvider>
  );
}
```

`PaneSettings` is fully controlled — wire it to whatever store you persist with
(Ghost FTP uses zustand). See `src/context.tsx` for the full interface.

## Styling contract (current limitation)

The components use Ghost FTP's **semantic Tailwind classes** and a few global helper
classes. The host app must currently provide these. Specifically:

- **Color tokens** (Tailwind theme extension): `bg`, `bg-panel`, `bg-subtle`,
  `bg-hover`, `border`, `border-subtle`, `text`, `text-muted`, `text-dim`,
  `accent` (+ `accent-strong`, `accent-soft`), `danger` (+ `danger-soft`).
- **z-index scale**: `z-menu`, `z-modal`.
- **Box shadows**: `shadow-elev-3`.
- **Helper classes** (global CSS): `anim-modal`, `anim-fade`, `ghostftp-shimmer`,
  `btn-accent`.
- Your Tailwind `content` globs must include this package's `src` so the JIT
  emits the classes it uses:

  ```js
  content: ["./index.html", "./src/**/*.{ts,tsx}", "./packages/file-ui/src/**/*.{ts,tsx}"]
  ```

> **Roadmap:** make theming portable — ship a tokens stylesheet / CSS-variable
> defaults so the package looks right with no host setup. Tracked as the next
> phase after this extraction spike.

## Building for publish

During in-repo development the host consumes `src/` directly (via a workspace
alias), so there's no build step. To publish to npm you'd add a bundler
(e.g. `tsup`) emitting ESM + `.d.ts` and point `exports` at `dist/`.
