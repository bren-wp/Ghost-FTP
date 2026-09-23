import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { AppErrorBoundary } from "./components/AppErrorBoundary";
import { useSettings } from "./stores/settingsStore";
import { applyAccent } from "./lib/accent";
import { startLocalization } from "./lib/i18n";
import "./styles.css";

// English is the primary UI language. Language selection lives in Settings.
startLocalization();


// Rust seeds the main window theme before the frontend paints. Keep this
// fallback so the single application window always starts with a deterministic
// theme even if persisted settings are unavailable during early startup.
if (!document.documentElement.getAttribute("data-theme")) {
  document.documentElement.setAttribute(
    "data-theme",
    useSettings.getState().appTheme
  );
}
// Apply any saved accent override on top of the theme's own accent. (Accent
// isn't part of the pre-paint injection; the store seeds it synchronously.)
applyAccent(useSettings.getState().accentColor || null);

// Keep the html data-theme in sync with the setting store. On an actual theme
// change, add `.theming` so the (otherwise dormant) crossfade transition runs,
// then drop it once the transition is done.
let prevTheme = useSettings.getState().appTheme;
let prevAccent = useSettings.getState().accentColor;
let themingTimer: ReturnType<typeof setTimeout> | undefined;
useSettings.subscribe((s) => {
  if (s.accentColor !== prevAccent) {
    prevAccent = s.accentColor;
    applyAccent(s.accentColor || null);
  }
  if (s.appTheme === prevTheme) return;
  prevTheme = s.appTheme;
  const el = document.documentElement;
  el.classList.add("theming");
  el.setAttribute("data-theme", s.appTheme);
  if (themingTimer) clearTimeout(themingTimer);
  themingTimer = setTimeout(() => el.classList.remove("theming"), 260);
});

// Keep unexpected async errors observable without leaking state or credentials.
window.addEventListener("unhandledrejection", (event) => {
  const reason = event.reason instanceof Error ? event.reason.message : "Unhandled asynchronous error";
  console.error("Ghost FTP async failure:", reason);
  event.preventDefault();
});
window.addEventListener("error", (event) => {
  console.error("Ghost FTP runtime failure:", event.message || "Unexpected runtime error");
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <AppErrorBoundary>
      <App />
    </AppErrorBoundary>
  </React.StrictMode>
);
