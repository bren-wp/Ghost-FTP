/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{ts,tsx}",
    "./packages/file-ui/src/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        bg: {
          DEFAULT: "rgb(var(--bg) / <alpha-value>)",
          panel: "rgb(var(--bg-panel) / <alpha-value>)",
          subtle: "rgb(var(--bg-subtle) / <alpha-value>)",
          hover: "rgb(var(--bg-hover) / <alpha-value>)",
        },
        border: {
          DEFAULT: "rgb(var(--border) / <alpha-value>)",
          subtle: "rgb(var(--border-subtle) / <alpha-value>)",
        },
        text: {
          DEFAULT: "rgb(var(--text) / <alpha-value>)",
          muted: "rgb(var(--text-muted) / <alpha-value>)",
          dim: "rgb(var(--text-dim) / <alpha-value>)",
        },
        accent: {
          DEFAULT: "rgb(var(--accent) / <alpha-value>)",
          hover: "rgb(var(--accent-hover) / <alpha-value>)",
          strong: "rgb(var(--accent-strong) / <alpha-value>)",
          soft: "rgb(var(--accent) / 0.15)",
        },
        danger: {
          DEFAULT: "rgb(var(--danger) / <alpha-value>)",
          soft: "rgb(var(--danger) / 0.18)",
        },
        success: {
          DEFAULT: "rgb(var(--success) / <alpha-value>)",
          soft: "rgb(var(--success) / 0.15)",
        },
        warning: {
          DEFAULT: "rgb(var(--warning) / <alpha-value>)",
          soft: "rgb(var(--warning) / 0.15)",
        },
        info: {
          DEFAULT: "rgb(var(--info) / <alpha-value>)",
          soft: "rgb(var(--info) / 0.15)",
        },
      },
      fontFamily: {
        sans: [
          "Inter",
          "BlinkMacSystemFont",
          '"Segoe UI Variable"',
          '"Segoe UI"',
          "Roboto",
          "system-ui",
          "sans-serif",
        ],
        mono: [
          '"JetBrains Mono"',
          '"Fira Code"',
          '"Cascadia Code"',
          "Consolas",
          "monospace",
        ],
      },
      boxShadow: {
        "elev-1": "0 1px 0 0 rgb(var(--shadow) / 0.4)",
        "elev-2":
          "0 2px 4px -1px rgb(var(--shadow) / 0.5), 0 1px 2px -1px rgb(var(--shadow) / 0.4)",
        "elev-3":
          "0 16px 32px -8px rgb(var(--shadow) / 0.5), 0 4px 8px -2px rgb(var(--shadow) / 0.3)",
      },
      ringColor: {
        accent: "rgb(var(--accent) / 0.5)",
      },
      // Semantic stacking order, low → high. Replaces scattered magic z-values.
      // Security gates (host-key, agent approval) sit ABOVE toasts so a toast can
      // never obscure a decision; tooltips are the topmost interactive layer.
      zIndex: {
        sticky: "10", // sticky table/list headers inside a scroll area
        raised: "20", // floating in-content controls (the Sync pill)
        dropdown: "30", // status-bar popovers (notifications, live edits)
        menu: "40", // title-bar menus, right-click context menus
        modal: "50", // standard dialogs + their backdrops
        palette: "60", // command palette, keyboard-shortcuts overlay
        toast: "70", // transient toasts — above dialogs so feedback shows
        secure: "80", // security gates (host-key, agent approval) — above toasts
        tooltip: "90", // topmost interactive layer
      },
    },
  },
  plugins: [],
};
