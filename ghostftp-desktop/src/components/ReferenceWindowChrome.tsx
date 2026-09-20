import { useEffect, useMemo, useRef, useState } from "react";
import { ChevronDown, Languages, Minus, Settings, Square, X } from "lucide-react";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { GhostWordmark } from "./GhostBrand";
import { useLayout } from "@/stores/layoutStore";
import { getLocale, setLocale } from "@/lib/i18n";
import { openOfficialUrl } from "@/lib/external";
import { PRODUCT_VERSION_BADGE } from "@/lib/release";

type Item = { label: string; run: () => void; disabled?: boolean } | { separator: true };

function windowAction(action: "minimize" | "maximize" | "close") {
  try {
    const win = getCurrentWindow();
    if (action === "minimize") void win.minimize();
    else if (action === "maximize") void win.toggleMaximize();
    else void win.close();
  } catch {
    // Native window controls are unavailable only in source/browser previews.
  }
}

function onTitlebarDoubleClick(event: React.MouseEvent<HTMLDivElement>) {
  if ((event.target as HTMLElement).closest("button,select,input")) return;
  windowAction("maximize");
}

export function ReferenceWindowControls({ onClose }: { onClose?: () => void }) {
  return (
    <div className="ghost-window-controls">
      <button aria-label="Minimize" onClick={() => windowAction("minimize")}><Minus size={14}/></button>
      <button aria-label="Maximize or restore" onClick={() => windowAction("maximize")}><Square size={12}/></button>
      <button
        className="danger"
        aria-label={onClose ? "Close view" : "Close Ghost FTP"}
        onClick={() => onClose ? onClose() : windowAction("close")}
      >
        <X size={15}/>
      </button>
    </div>
  );
}

export function ReferenceWindowTitlebar({
  suffix,
  onClose,
}: {
  suffix?: string;
  onClose?: () => void;
}) {
  return (
    <div className="ghost-standalone-titlebar" onDoubleClick={onTitlebarDoubleClick}>
      <div className="ghost-standalone-brand" data-tauri-drag-region>
        <GhostWordmark compact />
        {suffix ? <span className="ghost-standalone-suffix">— {suffix}</span> : null}
      </div>
      <div className="ghost-window-title-spacer" data-tauri-drag-region />
      <ReferenceWindowControls onClose={onClose} />
    </div>
  );
}

function ReferenceMenuNav({ onClose }: { onClose?: () => void }) {
  const openDialog = useLayout((s) => s.openDialog);
  const openNewConnection = useLayout((s) => s.openNewConnection);
  const [open, setOpen] = useState<string | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  const menus = useMemo<Record<string, Item[]>>(() => ({
    File: [
      { label: "New Connection…", run: () => openNewConnection() },
      { label: "Site Manager…", run: () => openDialog("siteManager") },
      { separator: true },
      { label: "Close view", run: () => onClose ? onClose() : useLayout.getState().closeDialog() },
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
      { label: "Documentation", run: () => openOfficialUrl("/docs/") },
      { label: "Support Center", run: () => openOfficialUrl("/support/") },
      { separator: true },
      { label: "About Ghost FTP", run: () => openDialog("about") },
    ],
  }), [onClose, openDialog, openNewConnection]);

  useEffect(() => {
    if (!open) return;
    const onDown = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(null);
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(null);
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const focusTrigger = (name: string) => requestAnimationFrame(() => {
    rootRef.current?.querySelector<HTMLButtonElement>(`[data-menu-trigger="${name}"]`)?.focus();
  });
  const openFromKeyboard = (name: string) => {
    setOpen(name);
    requestAnimationFrame(() => {
      rootRef.current?.querySelector<HTMLButtonElement>(`[data-menu-anchor="${name}"] [role="menuitem"]:not(:disabled)`)?.focus();
    });
  };
  const moveTop = (name: string, delta: number) => {
    const names = Object.keys(menus);
    const index = names.indexOf(name);
    const next = names[(index + delta + names.length) % names.length];
    if (open) openFromKeyboard(next);
    else focusTrigger(next);
  };
  const onListKeyDown = (event: React.KeyboardEvent<HTMLDivElement>, name: string) => {
    const items = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button[role="menuitem"]:not(:disabled)'));
    const current = items.indexOf(document.activeElement as HTMLButtonElement);
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      const delta = event.key === "ArrowDown" ? 1 : -1;
      items[(Math.max(current, 0) + delta + items.length) % items.length]?.focus();
    } else if (event.key === "Home" || event.key === "End") {
      event.preventDefault();
      (event.key === "Home" ? items[0] : items[items.length - 1])?.focus();
    } else if (event.key === "Escape") {
      event.preventDefault();
      setOpen(null);
      focusTrigger(name);
    } else if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
      event.preventDefault();
      moveTop(name, event.key === "ArrowRight" ? 1 : -1);
    }
  };

  return (
    <div className="ghost-title-menu-wrap" ref={rootRef}>
      <nav className="ghost-menu" aria-label="Application menu">
        {Object.keys(menus).map((name) => (
          <div className="ghost-menu-anchor" data-menu-anchor={name} key={name}>
            <button
              className={open === name ? "active" : ""}
              aria-haspopup="menu"
              aria-expanded={open === name}
              data-menu-trigger={name}
              onKeyDown={(event) => {
                if (event.key === "ArrowDown") {
                  event.preventDefault();
                  openFromKeyboard(name);
                } else if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
                  event.preventDefault();
                  moveTop(name, event.key === "ArrowRight" ? 1 : -1);
                }
              }}
              onPointerEnter={() => { if (open && open !== name) setOpen(name); }}
              onClick={() => setOpen((value) => value === name ? null : name)}
            >
              {name}
            </button>
            {open === name && (
              <div className="ghost-menu-popover" role="menu" onKeyDown={(event) => onListKeyDown(event, name)}>
                {menus[name].map((item, index) =>
                  "separator" in item
                    ? <div className="ghost-menu-separator" key={index}/>
                    : <button role="menuitem" disabled={item.disabled} key={index} onClick={() => { setOpen(null); item.run(); }}>{item.label}</button>
                )}
              </div>
            )}
          </div>
        ))}
      </nav>
    </div>
  );
}

export function ReferenceMenuTitlebar({ onClose }: { onClose?: () => void } = {}) {
  return (
    <div className="ghost-standalone-titlebar ghost-menu-titlebar" onDoubleClick={onTitlebarDoubleClick}>
      <div className="ghost-standalone-brand" data-tauri-drag-region>
        <GhostWordmark compact />
        <span className="ghost-version-badge">{PRODUCT_VERSION_BADGE}</span>
      </div>
      <ReferenceMenuNav onClose={onClose} />
      <div className="ghost-window-title-spacer" data-tauri-drag-region />
      <ReferenceWindowControls onClose={onClose} />
    </div>
  );
}

export function ReferenceActionRow() {
  const openDialog = useLayout((s) => s.openDialog);
  const locale = getLocale();
  return (
    <div className="ghost-standalone-action-row">
      <div className="flex-1" />
      <div className="ghost-title-actions">
        <button className="ghost-mini-button" onClick={() => openDialog("settings")}><Settings size={14}/><span>Settings</span></button>
        <label className="ghost-language-menu">
          <Languages size={14}/>
          <select aria-label="Language" value={locale} onChange={(event) => setLocale(event.target.value as any)}>
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

/** Compatibility export for any older surface still expecting the second-row component. */
export function ReferenceMenuRow() {
  return <ReferenceActionRow />;
}
