import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { fileURLToPath } from "node:url";

const host = process.env.TAURI_DEV_HOST;
const rootDir = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig(() => ({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(rootDir, "src"),
      // Consume the workspace package straight from its TS source during dev —
      // edit a component once and both the app and the package see it instantly.
      "@ghostftp/file-ui": path.resolve(
        rootDir,
        "packages/file-ui/src/index.ts"
      ),
    },
  },
  clearScreen: false,
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          const normalized = id.replaceAll("\\", "/");
          if (normalized.includes("/packages/file-ui/src/")) return "file-ui";
          if (normalized.endsWith("/src/lib/i18n.ts")) return "i18n";
          if (normalized.endsWith("/src/lib/brandIconData.ts")) return "brand-icons";
          if (!normalized.includes("/node_modules/")) return undefined;
          if (normalized.includes("@xterm")) return "terminal-vendor";
          if (normalized.includes("material-icon-theme")) return "file-icon-theme";
          if (normalized.includes("@iconify")) return "icon-vendor";
          if (normalized.includes("@tauri-apps")) return "tauri-vendor";
          if (
            normalized.includes("react-dom") ||
            normalized.includes("/react/") ||
            normalized.includes("/react/") ||
            normalized.includes("zustand")
          ) {
            return "ui-vendor";
          }
          return undefined;
        },
      },
    },
  },
  server: {
    port: 1420,
    strictPort: true,
    host: host || false,
    hmr: host
      ? {
          protocol: "ws",
          host,
          port: 1421,
        }
      : undefined,
    watch: {
      ignored: ["**/src-tauri/**"],
    },
  },
}));
