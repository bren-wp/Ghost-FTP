import { useEffect, useRef, useState } from "react";
import { ChevronDown, Languages, Minus, Settings, Square, X } from "lucide-react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { GhostWordmark } from "./GhostBrand";
import { useLayout } from "@/stores/layoutStore";
import { getLocale, setLocale } from "@/lib/i18n";

const SITE = "https://ghostftp.com";

type Item = { label: string; run: () => void } | { separator: true };

function windowAction(action: "minimize" | "maximize" | "close") {
  try {
    const win = getCurrentWindow();
    if (action === "minimize") void win.minimize();
    else if (action === "maximize") void win.toggleMaximize();
    else void win.close();
  } catch {
    // Browser/source preview: window controls are native-only.
  }
}

export function ReferenceWindowControls() {
  return (
    <div className="ghost-window-controls">
      <button aria-label="Minimize" onClick={() => windowAction("minimize")}><Minus size={14}/></button>
      <button aria-label="Maximize or restore" onClick={() => windowAction("maximize")}><Square size={12}/></button>
      <button className="danger" aria-label="Close" onClick={() => windowAction("close")}><X size={15}/></button>
    </div>
  );
}

export function ReferenceWindowTitlebar({ suffix }: { suffix?: string }) {
  return (
    <div className="ghost-standalone-titlebar" data-tauri-drag-region onDoubleClick={() => windowAction("maximize")}>
      <div className="ghost-standalone-brand" data-tauri-drag-region>
        <GhostWordmark compact />
        {suffix ? <span className="ghost-standalone-suffix">— {suffix}</span> : null}
      </div>
      <div className="ghost-window-title-spacer" data-tauri-drag-region />
      <ReferenceWindowControls />
    </div>
  );
}

export function ReferenceMenuRow() {
  const openDialog = useLayout((s) => s.openDialog);
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const [open, setOpen] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const locale = getLocale();
  const menus: Record<string, Item[]> = {
    File: [
      { label: "New Connection…", run: () => openNewConnection() },
      { label: "Site Manager…", run: () => openDialog("siteManager") },
      { separator: true },
      { label: "Close view", run: () => useLayout.getState().closeDialog() },
    ],
    Edit: [{ label: "Preferences…", run: () => openDialog("settings") }],
    View: [
      { label: "Transfer Center", run: () => openDialog("transferCenter") },
      { label: "Site Manager", run: () => openDialog("siteManager") },
    ],
    Transfer: [{ label: "Transfer Center", run: () => openDialog("transferCenter") }],
    Server: [{ label: "New Connection…", run: () => openNewConnection() }],
    Bookmarks: [{ label: "Site Manager…", run: () => openDialog("siteManager") }],
    Tools: [{ label: "Preferences…", run: () => openDialog("settings") }],
    Help: [
      { label: "Documentation", run: () => window.open(`${SITE}/docs/`, "_blank", "noopener,noreferrer") },
      { label: "Support Center", run: () => window.open(`${SITE}/support/`, "_blank", "noopener,noreferrer") },
      { separator: true },
      { label: "About Ghost FTP", run: () => openDialog("about") },
    ],
  };

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(null);
    };
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") setOpen(null); };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div className="ghost-standalone-menubar" ref={rootRef}>
      <nav className="ghost-menu" aria-label="Application menu">
        {Object.keys(menus).map((name) => (
          <div className="ghost-menu-anchor" key={name}>
            <button className={open === name ? "active" : ""} onClick={() => setOpen((v) => v === name ? null : name)}>{name}</button>
            {open === name && (
              <div className="ghost-dropdown-menu">
                {menus[name].map((item, i) => "separator" in item ? <div className="ghost-menu-separator" key={i}/> : (
                  <button key={i} onClick={() => { setOpen(null); item.run(); }}>{item.label}</button>
                ))}
              </div>
            )}
          </div>
        ))}
      </nav>
      <div className="ghost-title-actions">
        <button className="ghost-mini-button" onClick={() => openDialog("settings")}><Settings size={14}/><span>Settings</span></button>
        <label className="ghost-language-menu">
          <Languages size={14}/>
          <select aria-label="Language" value={locale} onChange={(e) => setLocale(e.target.value as any)}>
            <option value="en">English (English)</option><option value="hr">Hrvatski (Croatian)</option>
            <option value="de">Deutsch (German)</option><option value="fr">Français (French)</option>
            <option value="es">Español (Spanish)</option><option value="it">Italiano (Italian)</option>
            <option value="pt">Português (Portuguese)</option><option value="nl">Nederlands (Dutch)</option>
            <option value="pl">Polski (Polish)</option><option value="sl">Slovenščina (Slovenian)</option>
            <option value="sr">Srpski (Serbian)</option><option value="bs">Bosanski (Bosnian)</option>
            <option value="mk">Македонски (Macedonian)</option><option value="sq">Shqip (Albanian)</option>
          </select>
          <ChevronDown size={12}/>
        </label>
      </div>
    </div>
  );
}
