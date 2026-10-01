// Curated Material Icon Theme integration.
//
// Ghost FTP intentionally bundles only the file-type icons that its browser
// maps explicitly. The previous eager glob pulled the complete ~900-icon
// upstream catalog and manifest into the desktop bundle, producing a >1.5 MB
// JavaScript chunk even though unmatched files already have a Lucide fallback.
//
// Keep these imports explicit: Vite emits only the selected SVG assets and the
// lookup table below stays tiny and deterministic. No network access is needed.

import adobeIllustrator from "material-icon-theme/icons/adobe-illustrator.svg?url";
import adobePhotoshop from "material-icon-theme/icons/adobe-photoshop.svg?url";
import audio from "material-icon-theme/icons/audio.svg?url";
import blender from "material-icon-theme/icons/blender.svg?url";
import cIcon from "material-icon-theme/icons/c.svg?url";
import cmake from "material-icon-theme/icons/cmake.svg?url";
import consoleIcon from "material-icon-theme/icons/console.svg?url";
import cpp from "material-icon-theme/icons/cpp.svg?url";
import csharp from "material-icon-theme/icons/csharp.svg?url";
import css from "material-icon-theme/icons/css.svg?url";
import dart from "material-icon-theme/icons/dart.svg?url";
import database from "material-icon-theme/icons/database.svg?url";
import docker from "material-icon-theme/icons/docker.svg?url";
import document from "material-icon-theme/icons/document.svg?url";
import elixir from "material-icon-theme/icons/elixir.svg?url";
import elm from "material-icon-theme/icons/elm.svg?url";
import erlang from "material-icon-theme/icons/erlang.svg?url";
import figma from "material-icon-theme/icons/figma.svg?url";
import font from "material-icon-theme/icons/font.svg?url";
import git from "material-icon-theme/icons/git.svg?url";
import goIcon from "material-icon-theme/icons/go.svg?url";
import gradle from "material-icon-theme/icons/gradle.svg?url";
import haskell from "material-icon-theme/icons/haskell.svg?url";
import html from "material-icon-theme/icons/html.svg?url";
import image from "material-icon-theme/icons/image.svg?url";
import java from "material-icon-theme/icons/java.svg?url";
import javascript from "material-icon-theme/icons/javascript.svg?url";
import json from "material-icon-theme/icons/json.svg?url";
import julia from "material-icon-theme/icons/julia.svg?url";
import kotlin from "material-icon-theme/icons/kotlin.svg?url";
import less from "material-icon-theme/icons/less.svg?url";
import lock from "material-icon-theme/icons/lock.svg?url";
import lua from "material-icon-theme/icons/lua.svg?url";
import makefile from "material-icon-theme/icons/makefile.svg?url";
import markdown from "material-icon-theme/icons/markdown.svg?url";
import nim from "material-icon-theme/icons/nim.svg?url";
import nodejs from "material-icon-theme/icons/nodejs.svg?url";
import npm from "material-icon-theme/icons/npm.svg?url";
import pdf from "material-icon-theme/icons/pdf.svg?url";
import perl from "material-icon-theme/icons/perl.svg?url";
import php from "material-icon-theme/icons/php.svg?url";
import pnpm from "material-icon-theme/icons/pnpm.svg?url";
import powerpoint from "material-icon-theme/icons/powerpoint.svg?url";
import powershell from "material-icon-theme/icons/powershell.svg?url";
import python from "material-icon-theme/icons/python.svg?url";
import react from "material-icon-theme/icons/react.svg?url";
import ruby from "material-icon-theme/icons/ruby.svg?url";
import rust from "material-icon-theme/icons/rust.svg?url";
import sass from "material-icon-theme/icons/sass.svg?url";
import scala from "material-icon-theme/icons/scala.svg?url";
import settings from "material-icon-theme/icons/settings.svg?url";
import sketch from "material-icon-theme/icons/sketch.svg?url";
import svelte from "material-icon-theme/icons/svelte.svg?url";
import swift from "material-icon-theme/icons/swift.svg?url";
import terraform from "material-icon-theme/icons/terraform.svg?url";
import threeD from "material-icon-theme/icons/3d.svg?url";
import typescript from "material-icon-theme/icons/typescript.svg?url";
import video from "material-icon-theme/icons/video.svg?url";
import vue from "material-icon-theme/icons/vue.svg?url";
import word from "material-icon-theme/icons/word.svg?url";
import xml from "material-icon-theme/icons/xml.svg?url";
import yaml from "material-icon-theme/icons/yaml.svg?url";
import yarn from "material-icon-theme/icons/yarn.svg?url";
import zig from "material-icon-theme/icons/zig.svg?url";
import zip from "material-icon-theme/icons/zip.svg?url";

const FILE_NAMES: Record<string, string> = {
  "dockerfile": docker,
  ".dockerignore": docker,
  "package.json": nodejs,
  "package-lock.json": npm,
  "pnpm-lock.yaml": pnpm,
  "yarn.lock": yarn,
  "tsconfig.json": typescript,
  "jsconfig.json": javascript,
  ".gitignore": git,
  ".gitattributes": git,
  ".gitmodules": git,
  ".env": settings,
  "cargo.toml": rust,
  "cargo.lock": rust,
  "go.mod": goIcon,
  "go.sum": goIcon,
  "makefile": makefile,
  "cmakelists.txt": cmake,
  "gradle.properties": gradle,
  "settings.gradle": gradle,
  "settings.gradle.kts": gradle,
  "build.gradle": gradle,
  "build.gradle.kts": gradle,
};

const FILE_EXTENSIONS: Record<string, string> = {};

function add(exts: string[], url: string) {
  for (const ext of exts) FILE_EXTENSIONS[ext] = url;
}

add(["js", "mjs", "cjs"], javascript);
add(["ts"], typescript);
add(["tsx", "jsx"], react);
add(["py", "pyw", "ipynb"], python);
add(["rs"], rust);
add(["go"], goIcon);
add(["rb"], ruby);
add(["php"], php);
add(["java"], java);
add(["kt", "kts"], kotlin);
add(["scala"], scala);
add(["c", "h"], cIcon);
add(["cpp", "cc", "cxx", "hpp", "hxx"], cpp);
add(["cs"], csharp);
add(["swift"], swift);
add(["dart"], dart);
add(["vue"], vue);
add(["svelte"], svelte);
add(["lua"], lua);
add(["ex", "exs"], elixir);
add(["hs"], haskell);
add(["jl"], julia);
add(["pl", "pm"], perl);
add(["nim"], nim);
add(["zig"], zig);
add(["elm"], elm);
add(["erl", "hrl"], erlang);
add(["html", "htm"], html);
add(["css"], css);
add(["scss", "sass"], sass);
add(["less"], less);
add(["json", "json5"], json);
add(["yaml", "yml"], yaml);
add(["xml"], xml);
add(["sql", "sqlite", "db"], database);
add(["sh", "bash", "zsh", "fish", "bat", "cmd"], consoleIcon);
add(["ps1"], powershell);
add(["tf", "tfvars", "hcl"], terraform);
add(["gradle"], gradle);
add(["cmake"], cmake);
add(["make", "mk"], makefile);
add(
  ["png", "jpg", "jpeg", "gif", "svg", "webp", "bmp", "ico", "tiff", "tif", "avif", "heic"],
  image
);
add(["psd", "psb"], adobePhotoshop);
add(["ai", "eps", "ps"], adobeIllustrator);
add(["fig"], figma);
add(["sketch"], sketch);
add(["blend"], blender);
add(["obj", "fbx", "gltf", "glb", "stl", "3ds", "dae", "usdz"], threeD);
add(["mp4", "mov", "mkv", "avi", "webm", "flv", "wmv", "m4v"], video);
add(["mp3", "wav", "flac", "ogg", "m4a", "aac", "opus"], audio);
add(["zip", "tar", "gz", "tgz", "rar", "7z", "bz2", "xz", "zst"], zip);
add(["pdf"], pdf);
add(["doc", "docx", "odt", "rtf"], word);
add(["ppt", "pptx", "odp"], powerpoint);
add(["md", "markdown", "mdx"], markdown);
add(["txt", "log"], document);
add(["ttf", "otf", "woff", "woff2"], font);
add(["lock"], lock);

/**
 * A bundled Material Icon Theme SVG URL for a known file, or `undefined` so
 * the caller can use Ghost FTP's Lucide fallback.
 *
 * Exact file names win, then longest extension suffixes (for example
 * `app.d.ts` tries `d.ts` before `ts`).
 */
export function materialIconUrl(name: string): string | undefined {
  const lower = name.toLowerCase();

  const byName = FILE_NAMES[lower];
  if (byName) return byName;

  const parts = lower.split(".");
  for (let i = 1; i < parts.length; i++) {
    const ext = parts.slice(i).join(".");
    const icon = FILE_EXTENSIONS[ext];
    if (icon) return icon;
  }
  return undefined;
}
