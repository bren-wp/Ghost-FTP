import { useRef, useState } from "react";
import {
  ChevronRight,
  FileText,
  Globe2,
  HelpCircle,
  LifeBuoy,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import { GhostMark } from "./GhostBrand";
import { useUpdater } from "@/stores/updaterStore";
import { PRODUCT_BUILD, PRODUCT_RELEASE_DATE, PRODUCT_VERSION_DISPLAY } from "@/lib/release";
import { useDialog } from "@/hooks/useDialog";

type AboutTab = "about" | "updates" | "help" | "privacy";
interface Props { onClose: () => void; initialTab?: Exclude<AboutTab, "privacy"> }

export function AboutDialog({ onClose, initialTab = "about" }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const [tab, setTab] = useState<AboutTab>(initialTab);
  useDialog(panelRef, { onClose });

  return (
    <div className="ghost-workspace-view ghost-standalone-view bg-[#041425]" role="region" aria-label="About Ghost FTP">
      <div ref={panelRef} className="ghost-about flex h-full w-full flex-col overflow-hidden bg-bg-panel">
        <div className="ghost-about-body flex min-h-0 flex-1 flex-col">
        <nav className="ghost-about-nav ghost-about-tabs flex shrink-0 items-center gap-1 overflow-x-auto border-b border-border bg-[#061a2d] px-3 py-2" aria-label="Help and About sections">
          <AboutNav active={tab === "about"} icon={<Globe2 size={17}/>} label="About" onClick={() => setTab("about")} />
          <AboutNav active={tab === "updates"} icon={<RefreshCw size={17}/>} label="Updates" onClick={() => setTab("updates")} />
          <AboutNav active={tab === "help"} icon={<HelpCircle size={17}/>} label="Help Center" onClick={() => setTab("help")} />
          <AboutNav active={tab === "privacy"} icon={<ShieldCheck size={17}/>} label="Privacy" onClick={() => setTab("privacy")} />
        </nav>

        <main className="ghost-about-content min-h-0 min-w-0 flex-1 overflow-y-auto p-4">
          {tab !== "about" && (
            <div className="mb-3">
              <div className="text-lg font-semibold">
                {tab === "updates" ? "Ghost FTP Updates" : tab === "privacy" ? "Privacy" : "Ghost FTP Help Center"}
              </div>
              <div className="text-[12px] text-text-muted">Files move forward.</div>
            </div>
          )}

          {tab === "about" && <AboutContent onNavigate={setTab} />}
          {tab === "updates" && <UpdatesContent />}
          {tab === "help" && <HelpContent />}
          {tab === "privacy" && <PrivacyContent />}
        </main>
        </div>
      </div>
    </div>
  );
}

function AboutContent({ onNavigate }: { onNavigate: (tab: AboutTab) => void }) {
  return (
    <div className="ghost-about-grid grid grid-cols-[minmax(0,1.55fr)_minmax(280px,.85fr)] gap-4">
      <section className="space-y-4">
        <div className="overflow-hidden rounded-lg border border-border bg-[#071f35]">
          <div className="ghost-about-hero relative flex min-h-52 items-center gap-8 overflow-hidden px-8 py-7">
            <AboutLandscape />
            <div className="ghost-about-hero-glow absolute inset-0 opacity-30" />
            <div className="relative flex h-28 w-28 items-center justify-center rounded-[2rem] bg-[#0b2b46]/70 shadow-[0_0_55px_rgba(58,181,255,.24)]"><GhostMark size={104}/></div>
            <div className="relative">
              <h2 className="text-[34px] font-semibold tracking-tight">Ghost FTP</h2>
              <p className="mt-1 text-lg text-text-muted">Files Move Forward</p>
            </div>
          </div>
          <div className="grid grid-cols-3 border-t border-border px-6 py-4 text-center">
            <Meta label="Version" value={PRODUCT_VERSION_DISPLAY}/>
            <Meta label="Build" value={PRODUCT_BUILD} border/>
            <Meta label="Release Date" value={PRODUCT_RELEASE_DATE} border/>
          </div>
        </div>

        <div className="rounded-lg border border-border bg-[#071f35] p-5">
          <div className="mb-3 flex items-center gap-2"><FileText className="text-accent" size={22}/><div><div className="text-[16px] font-semibold">What's New</div><div className="text-[12px] text-text-muted">Highlights from the latest release</div></div><div className="flex-1"/><button onClick={() => onNavigate("updates")} className="text-[12px] text-accent hover:underline">View Full Changelog</button></div>
          <div className="rounded-md border border-border-subtle bg-[#051929] p-4">
            <div className="mb-2 font-semibold">Version {PRODUCT_VERSION_DISPLAY}</div>
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
          <LinkRow icon={<LifeBuoy size={18}/>} title="Help Center" subtitle="Guides, troubleshooting and support" onClick={() => onNavigate("help")}/>
          <LinkRow icon={<ShieldCheck size={18}/>} title="Privacy" subtitle="Privacy-first defaults and local data" onClick={() => onNavigate("privacy")}/>
          <LinkRow icon={<FileText size={18}/>} title="Changelog" subtitle="See what's new" onClick={() => onNavigate("updates")}/>
        </div>
        <div className="rounded-lg border border-border bg-[#071f35] p-5">
          <div className="text-[15px] font-semibold">Platforms & Language</div>
          <div className="mt-4 grid grid-cols-2 gap-3">
            <Platform title="Windows" subtitle="Fully supported" icon={<WindowsPlatformMark/>}/>
            <Platform title="Linux" subtitle="Fully supported" icon={<LinuxPlatformMark/>}/>
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
  return <div className="rounded-lg border border-border bg-[#071f35] p-6"><div className="mb-4 flex items-center gap-3"><RefreshCw size={30} className="text-accent"/><div><div className="text-[18px] font-semibold">Ghost FTP {PRODUCT_VERSION_DISPLAY}</div><div className="text-text-muted">Release candidate · {currentPlatform()}</div></div></div><p className="max-w-2xl text-[13px] leading-6 text-text-muted">Update checks use the official Ghost FTP endpoint at ghostftp.com. Installation proceeds only when the package passes Tauri's configured signature verification.</p><p className="mt-3 text-[12px] text-text-muted" aria-live="polite">{label}</p><div className="mt-5 flex gap-2"><button className="ghost-primary-button" disabled={status === "checking" || status === "downloading"} onClick={() => void check(false)}><RefreshCw size={14}/> Check for Updates</button>{status === "available" && <button className="ghost-primary-button" onClick={() => void download()}>Download &amp; install</button>}{status === "ready" && <button className="ghost-primary-button" onClick={() => void restart()}>Restart now</button>}</div></div>;
}

function currentPlatform() {
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes("windows")) return "Windows";
  if (ua.includes("linux")) return "Linux";
  return "Desktop";
}

function HelpContent() {
  return <div className="grid grid-cols-2 gap-4">
    <InfoCard icon={<Globe2/>} title="Connections" text="Create or edit a saved site, then use Test Connection before connecting." />
    <InfoCard icon={<FileText/>} title="Transfers" text="Use the Transfer Center for queue status, retry, scheduling and transfer history." />
    <InfoCard icon={<LifeBuoy/>} title="Troubleshooting" text="Connection and transfer errors stay visible in the app so you can act on the real failure." />
    <InfoCard icon={<RefreshCw/>} title="Updates" text="Open the Updates tab here to check the official Ghost FTP update channel." />
  </div>;
}

function PrivacyContent() {
  return <div className="max-w-3xl space-y-4 text-[13px] leading-6 text-text-muted">
    <div className="rounded-lg border border-border bg-[#071f35] p-5"><div className="mb-2 flex items-center gap-2 text-[16px] font-semibold text-text"><ShieldCheck size={20} className="text-accent"/>Privacy-first by default</div><p>Ghost FTP is designed to keep connection data and application settings under your control. The application does not require analytics or telemetry to transfer files.</p></div>
    <div className="rounded-lg border border-border bg-[#071f35] p-5"><div className="font-semibold text-text">Credentials and connections</div><p className="mt-2">Saved credentials are handled by Ghost FTP's local credential storage. Connections go to the server or provider you configure; the file manager does not need a Ghost FTP relay to perform ordinary FTP, FTPS or SFTP transfers.</p></div>
    <div className="rounded-lg border border-border bg-[#071f35] p-5"><div className="font-semibold text-text">Diagnostics</div><p className="mt-2">Operational errors are shown inside the app so failures are visible instead of silently ignored. Review Settings → Security and Settings → Integrations for privacy-sensitive options.</p></div>
  </div>;
}

function AboutNav({ active, icon, label, onClick }: { active: boolean; icon: React.ReactNode; label: string; onClick: () => void }) { return <button onClick={onClick} className={`flex shrink-0 items-center gap-2 rounded-md border px-3 py-2 text-left ${active ? "border-accent/45 bg-accent/15 text-white" : "border-transparent text-text-muted hover:bg-bg-hover hover:text-white"}`}>{icon}<span>{label}</span></button> }
function Meta({ label, value, border }: { label: string; value: string; border?: boolean }) { return <div className={border ? "border-l border-border" : ""}><div className="text-[11px] text-text-dim">{label}</div><div className="mt-1 font-semibold">{value}</div></div> }
function LinkRow({ icon, title, subtitle, onClick }: { icon: React.ReactNode; title: string; subtitle: string; onClick: () => void }) { return <button onClick={onClick} className="flex w-full items-center gap-3 border-t border-border-subtle py-3 text-left first:border-t-0"><span className="text-accent">{icon}</span><span className="min-w-0 flex-1"><span className="block font-medium text-accent">{title}</span><span className="block text-[11px] text-text-muted">{subtitle}</span></span><ChevronRight size={13} className="text-text-dim"/></button> }
function Platform({ title, subtitle, icon }: { title: string; subtitle: string; icon: React.ReactNode }) { return <div className="flex items-center gap-3 rounded-md border border-border-subtle bg-[#051929] p-3"><div className="grid h-9 w-9 shrink-0 place-items-center text-accent">{icon}</div><div><div className="font-semibold">{title}</div><div className="text-[10px] text-text-muted">{subtitle}</div></div></div> }

function WindowsPlatformMark() {
  return <svg viewBox="0 0 32 32" className="h-8 w-8" aria-hidden="true"><path fill="#6fd2ff" d="M3 5.2 14.2 3.7v10.8H3V5.2Zm13.1-1.8L29 1.7v12.8H16.1V3.4ZM3 16.4h11.2v10.8L3 25.7v-9.3Zm13.1 0H29v12.8l-12.9-1.7V16.4Z"/></svg>;
}

function LinuxPlatformMark() {
  return <svg viewBox="0 0 36 36" className="h-9 w-9" aria-hidden="true"><ellipse cx="18" cy="19" rx="9.3" ry="12" fill="#dff5ff"/><ellipse cx="18" cy="21" rx="6.2" ry="8.2" fill="#17344b"/><circle cx="15.2" cy="12.4" r="1.25" fill="#061522"/><circle cx="20.8" cy="12.4" r="1.25" fill="#061522"/><path d="M15.2 15.4 18 17.2l2.8-1.8L18 14Z" fill="#f5b942"/><path d="M9.5 29.2c2.2-.3 4.1.1 5.6 1.4-2.2 1.8-5 2-7.4.8.3-1 .9-1.7 1.8-2.2Zm17 0c-2.2-.3-4.1.1-5.6 1.4 2.2 1.8 5 2 7.4.8-.3-1-.9-1.7-1.8-2.2Z" fill="#f5b942"/></svg>;
}

function AboutLandscape() {
  return (
    <svg className="ghost-about-landscape absolute inset-0 h-full w-full" viewBox="0 0 1000 300" preserveAspectRatio="xMidYMid slice" aria-hidden="true">
      <defs>
        <linearGradient id="ghost-about-sky" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#082a55"/>
          <stop offset=".58" stopColor="#0b3b6a"/>
          <stop offset="1" stopColor="#071a31"/>
        </linearGradient>
        <linearGradient id="ghost-about-lake" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#0a3157"/>
          <stop offset="1" stopColor="#041421"/>
        </linearGradient>
        <linearGradient id="ghost-about-mountain" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#153f66"/>
          <stop offset="1" stopColor="#071a2d"/>
        </linearGradient>
      </defs>
      <rect width="1000" height="300" fill="url(#ghost-about-sky)"/>
      <g fill="#91d9ff" opacity=".72">
        <circle cx="92" cy="35" r="1.4"/><circle cx="168" cy="58" r="1"/><circle cx="246" cy="28" r="1.4"/>
        <circle cx="352" cy="52" r="1"/><circle cx="468" cy="24" r="1.3"/><circle cx="604" cy="46" r="1"/>
        <circle cx="726" cy="26" r="1.4"/><circle cx="842" cy="54" r="1"/><circle cx="927" cy="31" r="1.2"/>
      </g>
      <path d="M0 191 104 104 171 155 274 59 367 151 447 102 534 167 639 72 733 154 808 109 901 171 1000 94 1000 225 0 225Z" fill="url(#ghost-about-mountain)"/>
      <path d="M0 207 123 154 205 190 318 132 402 198 511 147 594 201 714 139 796 192 896 151 1000 207 1000 236 0 236Z" fill="#06192a"/>
      <rect y="223" width="1000" height="77" fill="url(#ghost-about-lake)"/>
      <path d="M0 240c130-11 222 12 340 0s218-13 330-1 214 10 330-2v63H0Z" fill="#082640" opacity=".78"/>
      <path d="M430 229h140l-34 55h-72Z" fill="#2e8fd0" opacity=".14"/>
    </svg>
  );
}
function InfoCard({ icon, title, text }: { icon: React.ReactNode; title: string; text: string }) { return <div className="rounded-lg border border-border bg-[#071f35] p-5 text-left"><div className="mb-3 text-accent">{icon}</div><div className="text-[16px] font-semibold">{title}</div><div className="mt-1 text-[12px] leading-5 text-text-muted">{text}</div></div> }
