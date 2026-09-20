import {
  FileText,
  FileCode,
  FileJson,
  FileImage,
  FileVideo,
  FileAudio,
  FileArchive,
  FileSpreadsheet,
  FileCog,
  FileTerminal,
  FileType,
  Folder,
  Link2,
  PenTool,
  Layers,
  Box,
  Figma,
  Frame,
  type LucideIcon,
} from "lucide-react";
import type { DirEntry } from "../types";

export interface FileIconSpec {
  Icon: LucideIcon;
  className: string;
}

const FOLDER: FileIconSpec = { Icon: Folder, className: "text-accent" };
const SYMLINK: FileIconSpec = { Icon: Link2, className: "text-info" };
const DEFAULT_FILE: FileIconSpec = { Icon: FileText, className: "text-text-muted" };

// Extension (lowercase, no dot) → icon + colour. Colours use Tailwind's default
// palette (which sits alongside the theme tokens) so file types read at a glance
// like a real file manager. Class strings are literal so Tailwind keeps them.
const BY_EXT: Record<string, FileIconSpec> = {};
const add = (exts: string[], spec: FileIconSpec) => {
  for (const e of exts) BY_EXT[e] = spec;
};

// Code
add(["js", "mjs", "cjs"], { Icon: FileCode, className: "text-yellow-400" });
add(["ts", "tsx", "jsx"], { Icon: FileCode, className: "text-blue-400" });
add(["py", "pyw", "ipynb"], { Icon: FileCode, className: "text-sky-400" });
add(["rs"], { Icon: FileCode, className: "text-orange-400" });
add(["go"], { Icon: FileCode, className: "text-cyan-400" });
add(["rb"], { Icon: FileCode, className: "text-red-400" });
add(["php"], { Icon: FileCode, className: "text-indigo-400" });
add(
  ["java", "kt", "scala", "c", "cpp", "cc", "h", "hpp", "cs", "swift", "dart", "vue", "svelte", "lua", "r", "ex", "exs"],
  { Icon: FileCode, className: "text-emerald-400" }
);
add(["html", "htm"], { Icon: FileCode, className: "text-orange-400" });
add(["css", "scss", "sass", "less"], { Icon: FileCode, className: "text-blue-400" });
add(["sql"], { Icon: FileCode, className: "text-fuchsia-400" });
add(["hs", "clj", "cljs", "jl", "pl", "pm", "nim", "zig", "elm", "erl"], {
  Icon: FileCode,
  className: "text-teal-400",
});
add(["tf", "tfvars", "hcl"], { Icon: FileCode, className: "text-violet-400" }); // Terraform
add(["gradle", "cmake", "make", "mk"], { Icon: FileCog, className: "text-text-dim" });

// Shell / scripts
add(["sh", "bash", "zsh", "fish", "ps1", "bat", "cmd"], {
  Icon: FileTerminal,
  className: "text-green-400",
});

// Data / config
add(["json", "json5"], { Icon: FileJson, className: "text-amber-400" });
add(["yaml", "yml", "toml", "ini", "conf", "env", "cfg", "properties"], {
  Icon: FileCog,
  className: "text-text-dim",
});
add(["xml"], { Icon: FileCode, className: "text-orange-300" });

// Images
const IMAGE_EXTS = [
  "png", "jpg", "jpeg", "gif", "svg", "webp", "bmp", "ico", "tiff", "tif", "avif", "heic",
];
add(IMAGE_EXTS, { Icon: FileImage, className: "text-green-400" });
const IMAGE_SET = new Set(IMAGE_EXTS);

// Design / creative — distinct glyphs + brand-ish colours so a PSD doesn't look
// like a JPG. Raster editors → Layers; vector/illustration → PenTool.
add(["psd", "psb"], { Icon: Layers, className: "text-blue-400" }); // Photoshop
add(["ai"], { Icon: PenTool, className: "text-orange-400" }); // Illustrator
add(["eps", "ps"], { Icon: PenTool, className: "text-orange-300" });
add(["indd", "idml"], { Icon: FileText, className: "text-rose-400" }); // InDesign
add(["fig"], { Icon: Figma, className: "text-pink-400" }); // Figma
add(["xd"], { Icon: Frame, className: "text-fuchsia-400" }); // Adobe XD
add(["sketch"], { Icon: PenTool, className: "text-yellow-400" });
add(["afphoto"], { Icon: Layers, className: "text-sky-400" }); // Affinity Photo
add(["afdesign"], { Icon: PenTool, className: "text-violet-400" }); // Affinity Designer
add(["cdr"], { Icon: PenTool, className: "text-lime-400" }); // CorelDRAW

// 3D / CAD
add(["blend"], { Icon: Box, className: "text-orange-400" }); // Blender
add(["obj", "fbx", "gltf", "glb", "stl", "3ds", "dae", "usdz"], {
  Icon: Box,
  className: "text-cyan-400",
});

// Video
add(["mp4", "mov", "mkv", "avi", "webm", "flv", "wmv", "m4v"], {
  Icon: FileVideo,
  className: "text-purple-400",
});

// Audio
add(["mp3", "wav", "flac", "ogg", "m4a", "aac", "opus"], {
  Icon: FileAudio,
  className: "text-pink-400",
});

// Archives
add(["zip", "tar", "gz", "tgz", "rar", "7z", "bz2", "xz", "zst"], {
  Icon: FileArchive,
  className: "text-amber-500",
});

// Documents
add(["pdf"], { Icon: FileText, className: "text-red-400" });
add(["doc", "docx", "odt", "rtf"], { Icon: FileText, className: "text-blue-400" });
add(["xls", "xlsx", "csv", "ods"], {
  Icon: FileSpreadsheet,
  className: "text-green-500",
});
add(["ppt", "pptx", "odp"], { Icon: FileText, className: "text-orange-400" });
add(["md", "markdown", "mdx"], { Icon: FileText, className: "text-sky-300" });
add(["txt", "log"], { Icon: FileText, className: "text-text-muted" });

// Binaries / packages
add(
  ["exe", "msi", "dll", "so", "dylib", "bin", "app", "deb", "rpm", "dmg", "appimage"],
  { Icon: FileCog, className: "text-text-dim" }
);

// Fonts
add(["ttf", "otf", "woff", "woff2"], { Icon: FileType, className: "text-rose-300" });

// Lockfiles
add(["lock"], { Icon: FileCog, className: "text-text-dim" });

/** Lowercased extension (no dot) for a name, or "" when there's no extension. */
export function extOf(name: string): string {
  const lower = name.toLowerCase();
  const dot = lower.lastIndexOf(".");
  return dot > 0 ? lower.slice(dot + 1) : "";
}

/** Whether an entry is a raster/vector image we can render a thumbnail for. */
export function isImage(entry: DirEntry): boolean {
  return entry.kind === "file" && IMAGE_SET.has(extOf(entry.name));
}

/** The MIME type for a known image extension, or null. Drives thumbnail decode. */
export function imageMime(name: string): string | null {
  switch (extOf(name)) {
    case "png":
      return "image/png";
    case "jpg":
    case "jpeg":
      return "image/jpeg";
    case "gif":
      return "image/gif";
    case "svg":
      return "image/svg+xml";
    case "webp":
      return "image/webp";
    case "bmp":
      return "image/bmp";
    case "ico":
      return "image/x-icon";
    case "avif":
      return "image/avif";
    case "tif":
    case "tiff":
      return "image/tiff";
    default:
      return null;
  }
}

/** Pick an icon + colour for a directory entry, by kind then file extension. */
export function fileIcon(entry: DirEntry): FileIconSpec {
  if (entry.kind === "directory") return FOLDER;
  if (entry.kind === "symlink") return SYMLINK;
  return BY_EXT[extOf(entry.name)] ?? DEFAULT_FILE;
}
