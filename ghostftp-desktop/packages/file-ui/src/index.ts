// @ghostftp/file-ui — transport-agnostic React file-browser UI.
//
// Bring your own FileSystemAdapter (SFTP, S3, HTTP, in-memory…) + PaneSettings,
// wrap the tree in <FileUiProvider>, and render <FilePane>. The components never
// import a transport — they talk only to the injected adapter.

export { FilePane } from "./components/FilePane";
export { FileUiProvider, useFileUi } from "./context";
export type {
  FileSystemAdapter,
  PaneSettings,
  FileUiValue,
} from "./context";

// Re-usable presentational pieces, in case a host wants to compose its own UI.
export { ContextMenu, type MenuItem } from "./components/ContextMenu";
export { PromptModal } from "./components/PromptModal";
export { ConfirmModal } from "./components/ConfirmModal";
export { PropertiesModal } from "./components/PropertiesModal";
export { Skeleton, FileListSkeleton } from "./components/Skeleton";
export { EmptyState } from "./components/EmptyState";
export { Thumbnail } from "./components/Thumbnail";

// File-type icons: the default extension→icon map + helpers. Pass a custom
// resolver to <FileUiProvider iconFor={...}> to override.
export {
  fileIcon,
  isImage,
  imageMime,
  extOf,
  type FileIconSpec,
} from "./lib/fileIcons";

// A Material Icon Theme SVG URL for a filename (folders/unknown → undefined).
export { materialIconUrl } from "./lib/materialIcons";

// Remappable file-browser key actions: the default combos + a combo matcher.
export {
  DEFAULT_FILE_BROWSER_KEYBINDINGS,
  comboFromEvent,
  type FileBrowserAction,
} from "./lib/keys";

// Hooks + utils.
export { usePathHistory } from "./hooks/usePathHistory";
export { useDialog } from "./hooks/useDialog";
export { cn } from "./lib/cn";
export { fmtSize, fmtMtime, formatMode } from "./lib/format";

// Types.
export {
  LOCAL_SESSION,
  type FileKind,
  type DirEntry,
  type Capabilities,
  type SessionId,
  type SortField,
  type SortDirection,
  type PaneViewMode,
  type PaneDensity,
  type ArchiveFormat,
} from "./types";
