import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

const host = process.env.TAURI_DEV_HOST;

export default defineConfig(() => ({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
      // Consume the workspace package straight from its TS source during dev —
      // edit a component once and both the app and the package see it instantly.
      "@ghostftp/file-ui": path.resolve(
        __dirname,
        "packages/file-ui/src/index.ts"
      ),
    },
  },
  clearScreen: false,
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) return undefined;
          if (id.includes("@xterm")) return "terminal-vendor";
          if (id.includes("material-icon-theme")) return "file-icon-theme";
          if (id.includes("@iconify")) return "icon-vendor";
          if (id.includes("@tauri-apps")) return "tauri-vendor";
          if (
            id.includes("react-dom") ||
            id.includes("/react/") ||
            id.includes("\\react\\") ||
            id.includes("zustand")
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
