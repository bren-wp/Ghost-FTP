import { useRef, useState } from "react";
import {
  CheckCircle2,
  ExternalLink,
  FileText,
  Globe2,
  HelpCircle,
  LifeBuoy,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import { GhostMark } from "./GhostBrand";
import { ReferenceWindowTitlebar } from "./ReferenceWindowChrome";
import { useUpdater } from "@/stores/updaterStore";
import { PRODUCT_BUILD, PRODUCT_RELEASE_DATE, PRODUCT_VERSION } from "@/lib/release";
import { useDialog } from "@/hooks/useDialog";
import { openOfficialUrl } from "@/lib/external";

interface Props { onClose: () => void; initialTab?: "about" | "updates" | "help" }

function external(path = "") {
  openOfficialUrl(path);
}

export function AboutDialog({ onClose, initialTab = "about" }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const [tab, setTab] = useState<"about" | "updates" | "help">(initialTab);
  useDialog(panelRef, { onClose });

  return (
    <div className="ghost-workspace-view ghost-standalone-view bg-[#041425]" role="region" aria-label="About Ghost FTP">
      <div ref={panelRef} className="ghost-about flex h-full w-full flex-col overflow-hidden bg-bg-panel">
        <ReferenceWindowTitlebar onClose={onClose} />
        <div className="ghost-about-body flex min-h-0 flex-1">
        <aside className="ghost-about-nav w-[214px] shrink-0 border-r border-border bg-[#061a2d] p-3">
          <AboutNav active={tab === "about"} icon={<Globe2 size={17}/>} label="About" onClick={() => setTab("about")} />
          <AboutNav active={tab === "updates"} icon={<RefreshCw size={17}/>} label="Updates" onClick={() => setTab("updates")} />
          <AboutNav active={tab === "help"} icon={<HelpCircle size={17}/>} label="Help Center" onClick={() => setTab("help")} />
        </aside>

        <main className="ghost-about-content min-w-0 flex-1 overflow-y-auto p-4">
          {tab !== "about" && (
            <div className="mb-3">
              <div className="text-lg font-semibold">
                {tab === "updates" ? "Ghost FTP Updates" : "Ghost FTP Help Center"}
              </div>
              <div className="text-[12px] text-text-muted">Files move forward.</div>
            </div>
          )}

          {tab === "about" && <AboutContent onNavigate={setTab} />}
          {tab === "updates" && <UpdatesContent />}
          {tab === "help" && <HelpContent />}
        </main>
        </div>
      </div>
    </div>
  );
}

function AboutContent({ onNavigate }: { onNavigate: (tab: "about" | "updates" | "help") => void }) {
  return (
    <div className="ghost-about-grid grid grid-cols-[minmax(0,1.55fr)_minmax(280px,.85fr)] gap-4">
      <section className="space-y-4">
        <div className="overflow-hidden rounded-lg border border-border bg-[#071f35]">
          <div className="ghost-about-hero relative flex min-h-52 items-center gap-8 overflow-hidden px-8 py-7">
            <div className="ghost-about-hero-glow absolute inset-0 opacity-30" />
            <div className="relative flex h-28 w-28 items-center justify-center rounded-[2rem] bg-[#0b2b46]/70 shadow-[0_0_55px_rgba(58,181,255,.24)]"><GhostMark size={104}/></div>
            <div className="relative">
              <h2 className="text-[34px] font-semibold tracking-tight">Ghost FTP</h2>
              <p className="mt-1 text-lg text-text-muted">Files Move Forward</p>
            </div>
          </div>
          <div className="grid grid-cols-3 border-t border-border px-6 py-4 text-center">
            <Meta label="Version" value={PRODUCT_VERSION}/>
            <Meta label="Build" value={PRODUCT_BUILD} border/>
            <Meta label="Release Date" value={PRODUCT_RELEASE_DATE} border/>
          </div>
          <UpdateStatusRow />
        </div>

        <div className="rounded-lg border border-border bg-[#071f35] p-5">
          <div className="mb-3 flex items-center gap-2"><FileText className="text-accent" size={22}/><div><div className="text-[16px] font-semibold">What's New</div><div className="text-[12px] text-text-muted">Highlights from the latest release</div></div><div className="flex-1"/><button onClick={() => onNavigate("updates")} className="text-[12px] text-accent hover:underline">View Full Changelog</button></div>
          <div className="rounded-md border border-border-subtle bg-[#051929] p-4">
            <div className="mb-2 font-semibold">Version {PRODUCT_VERSION}</div>
            <ul className="space-y-1.5 text-[12px] text-text-muted">
              <li>• Redesigned Windows and Linux interface based on the Ghost FTP visual system.</li>
              <li>• Faster transfer workflows with a clearer queue, progress and server activity log.</li>
              <li>• English primary interface with complete Croatian language support.</li>
              <li>• Privacy-first defaults, no telemetry and Ghost FTP-only branding.</li>
            </ul>
          </div>
        </div>
      </section>

      <aside className="space-y-4">
        <div className="rounded-lg border border-border bg-[#071f35] p-5">
          <div className="mb-4 flex items-center gap-3"><HelpCircle size={34} className="text-accent"/><div><div className="text-[16px] font-semibold">Get Help</div><div className="text-[12px] text-text-muted">Resources, documentation and support.</div></div></div>
          <LinkRow icon={<Globe2 size={18}/>} title="Visit ghostftp.com" subtitle="Official website" onClick={() => external()}/>
          <LinkRow icon={<FileText size={18}/>} title="Documentation" subtitle="Guides and tutorials" onClick={() => onNavigate("help")}/>
          <LinkRow icon={<LifeBuoy size={18}/>} title="Support Center" subtitle="Get help from our team" onClick={() => onNavigate("help")}/>
          <LinkRow icon={<ShieldCheck size={18}/>} title="Privacy Policy" subtitle="Your privacy matters" onClick={() => external("/privacy")}/>
          <LinkRow icon={<FileText size={18}/>} title="Changelog" subtitle="See what's new" onClick={() => onNavigate("updates")}/>
        </div>
        <div className="rounded-lg border border-border bg-[#071f35] p-5">
          <div className="text-[15px] font-semibold">Platforms & Language</div>
          <div className="mt-4 grid grid-cols-2 gap-3">
            <Platform title="Windows" subtitle="Fully supported" mark="⊞"/>
            <Platform title="Linux" subtitle="Fully supported" mark="◉"/>
          </div>
          <div className="mt-4 border-t border-border pt-4 text-[12px] text-text-muted"><strong className="text-text">Languages</strong><br/>14 interface languages</div>
        </div>
      </aside>
    </div>
  );
}

function UpdatesContent() {
  const status = useUpdater((s) => s.status);
  const offered = useUpdater((s) => s.version);
  const error = useUpdater((s) => s.error);
  const check = useUpdater((s) => s.check);
  const download = useUpdater((s) => s.downloadAndInstall);
  const restart = useUpdater((s) => s.restart);
  const label = status === "checking" ? "Checking for updates…"
    : status === "available" ? `Ghost FTP ${offered ?? "update"} is available.`
    : status === "downloading" ? "Downloading and verifying update…"
    : status === "ready" ? "Update is ready. Restart Ghost FTP to finish."
    : status === "error" ? (error ?? "Update check failed.")
    : "No update check has been run in this view yet.";
  return <div className="rounded-lg border border-border bg-[#071f35] p-6"><div className="mb-4 flex items-center gap-3"><RefreshCw size={30} className="text-accent"/><div><div className="text-[18px] font-semibold">Ghost FTP {PRODUCT_VERSION}</div><div className="text-text-muted">Release candidate · {currentPlatform()}</div></div></div><p className="max-w-2xl text-[13px] leading-6 text-text-muted">Update checks use the official Ghost FTP endpoint at ghostftp.com. Tauri verifies the signed package before installation.</p><p className="mt-3 text-[12px] text-text-muted" aria-live="polite">{label}</p><div className="mt-5 flex gap-2"><button className="ghost-primary-button" disabled={status === "checking" || status === "downloading"} onClick={() => void check(false)}><RefreshCw size={14}/> Check for Updates</button>{status === "available" && <button className="ghost-primary-button" onClick={() => void download()}>Download &amp; install</button>}{status === "ready" && <button className="ghost-primary-button" onClick={() => void restart()}>Restart now</button>}</div></div>;
}

function UpdateStatusRow() {
  const status = useUpdater((s) => s.status);
  const offered = useUpdater((s) => s.version);
  const error = useUpdater((s) => s.error);
  const check = useUpdater((s) => s.check);
  const text = status === "checking" ? "Checking for updates…"
    : status === "available" ? `Ghost FTP ${offered ?? "update"} is available.`
    : status === "downloading" ? "Downloading and verifying update…"
    : status === "ready" ? "Update ready — restart to finish installation."
    : status === "error" ? (error ?? "Update check failed.")
    : "Ready to check the official Ghost FTP update service.";
  const ok = status === "idle";
  return <div className="flex items-center gap-3 border-t border-border px-6 py-4"><CheckCircle2 size={28} className={ok ? "text-success" : "text-accent"}/><div><div className={ok ? "font-semibold text-success" : "font-semibold"}>{text}</div><div className="text-[12px] text-text-muted">Updates are checked only through ghostftp.com.</div></div><div className="flex-1"/><button disabled={status === "checking" || status === "downloading"} onClick={() => void check(false)} className="ghost-primary-button"><RefreshCw size={14}/> Check for Updates</button></div>;
}

function currentPlatform() {
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes("windows")) return "Windows";
  if (ua.includes("linux")) return "Linux";
  return "Desktop";
}

function HelpContent() {
  return <div className="grid grid-cols-2 gap-4"><HelpCard icon={<Globe2/>} title="Website" text="Product information, downloads and platform notes." onClick={() => external()}/><HelpCard icon={<FileText/>} title="Documentation" text="Connection, transfer and troubleshooting guides." onClick={() => external("/docs")}/><HelpCard icon={<LifeBuoy/>} title="Support" text="Get assistance for Ghost FTP on Windows or Linux." onClick={() => external("/support")}/><HelpCard icon={<ShieldCheck/>} title="Privacy" text="Read how Ghost FTP handles credentials and local data." onClick={() => external("/privacy")}/></div>;
}

function AboutNav({ active, icon, label, onClick }: { active: boolean; icon: React.ReactNode; label: string; onClick: () => void }) { return <button onClick={onClick} className={`mb-1 flex w-full items-center gap-3 rounded-md border px-3 py-2.5 text-left ${active ? "border-accent/45 bg-accent/15 text-white" : "border-transparent text-text-muted hover:bg-bg-hover hover:text-white"}`}>{icon}<span>{label}</span></button> }
function Meta({ label, value, border }: { label: string; value: string; border?: boolean }) { return <div className={border ? "border-l border-border" : ""}><div className="text-[11px] text-text-dim">{label}</div><div className="mt-1 font-semibold">{value}</div></div> }
function LinkRow({ icon, title, subtitle, onClick }: { icon: React.ReactNode; title: string; subtitle: string; onClick: () => void }) { return <button onClick={onClick} className="flex w-full items-center gap-3 border-t border-border-subtle py-3 text-left first:border-t-0"><span className="text-accent">{icon}</span><span className="min-w-0 flex-1"><span className="block font-medium text-accent">{title}</span><span className="block text-[11px] text-text-muted">{subtitle}</span></span><ExternalLink size={13} className="text-text-dim"/></button> }
function Platform({ title, subtitle, mark }: { title: string; subtitle: string; mark: string }) { return <div className="flex items-center gap-3 rounded-md border border-border-subtle bg-[#051929] p-3"><div className="text-2xl text-accent">{mark}</div><div><div className="font-semibold">{title}</div><div className="text-[10px] text-text-muted">{subtitle}</div></div></div> }
function HelpCard({ icon, title, text, onClick }: { icon: React.ReactNode; title: string; text: string; onClick: () => void }) { return <button onClick={onClick} className="rounded-lg border border-border bg-[#071f35] p-5 text-left hover:border-accent/50 hover:bg-[#092844]"><div className="mb-3 text-accent">{icon}</div><div className="text-[16px] font-semibold">{title}</div><div className="mt-1 text-[12px] leading-5 text-text-muted">{text}</div></button> }
