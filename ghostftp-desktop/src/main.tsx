import React, { useEffect } from "react";
import { isTauri } from "@tauri-apps/api/core";
import { getCurrentWindow } from "@tauri-apps/api/window";
import ReactDOM from "react-dom/client";
import App from "./App";
import { AppErrorBoundary } from "./components/AppErrorBoundary";
import { useSettings } from "./stores/settingsStore";
import { applyAccent } from "./lib/accent";
import { startLocalization } from "./lib/i18n";
import { redactSensitiveText } from "./lib/redact";
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
  console.error(
    "Ghost FTP async failure:",
    redactSensitiveText(
      event.reason instanceof Error
        ? event.reason.message
        : "Unhandled asynchronous error",
      320
    )
  );
  event.preventDefault();
});
window.addEventListener("error", (event) => {
  console.error(
    "Ghost FTP runtime failure:",
    redactSensitiveText(event.message || "Unexpected runtime error", 320)
  );
});


/**
 * Tauri creates the single main window hidden, avoiding the OS WebView's
 * temporary white/black surface. Reveal it only after React has committed the
 * application shell and its imported stylesheet is available.
 *
 * Do not wait for requestAnimationFrame: an initially hidden native webview
 * may throttle or suppress animation frames, leaving the window invisible.
 */
function RevealNativeWindowAfterMount({ children }: React.PropsWithChildren) {
  useEffect(() => {
    if (!isTauri()) return;
    // React passive effects run after the first DOM commit. The task boundary
    // also allows style/layout initialization before the OS maps the window.
    const reveal = window.setTimeout(() => {
      void getCurrentWindow().show().catch(() => {
        // Never print credentials or runtime error payloads to diagnostics.
        console.error("Ghost FTP could not reveal its main window");
      });
    }, 0);
    // React StrictMode runs the effect twice in development; cancel the first
    // scheduled reveal rather than racing multiple native window operations.
    return () => window.clearTimeout(reveal);
  }, []);
  return <>{children}</>;
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <RevealNativeWindowAfterMount>
      <AppErrorBoundary>
        <App />
      </AppErrorBoundary>
    </RevealNativeWindowAfterMount>
  </React.StrictMode>
);
