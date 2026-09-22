import { useEffect, useId, useRef, useState } from "react";
import {
  X,
  Copy,
  Check,
  Eye,
  EyeOff,
  Radio,
  ShieldCheck,
  Terminal as TerminalIcon,
  CircleAlert,
  CheckCircle2,
  Ban,
  Power,
  Plus,
  Pencil,
  Trash2,
  RefreshCw,
  Play,
  Wand2,
  ChevronRight,
} from "lucide-react";
import { useBridge } from "@/stores/bridgeStore";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import { useSkills } from "@/stores/skillsStore";
import { useDialog } from "@/hooks/useDialog";
import { ipc } from "@/lib/ipc";
import { toast } from "@/stores/toastStore";
import { ConfirmModal } from "./ConfirmModal";
import { cn } from "@/lib/cn";
import { relTime } from "@/lib/format";
import type {
  BridgeApproval,
  ApprovalPolicy,
  BridgeActivity,
  SavedCommand,
} from "@/lib/types";

// Phrase the approval prompt per operation kind.
const APPROVAL_COPY: Record<string, { title: string; foot: string }> = {
  exec: { title: "Agent wants to run a command", foot: "Runs over your authenticated SSH session." },
  read: { title: "Agent wants to read from the server", foot: "Reads through your authenticated Ghost FTP session." },
  download: { title: "Agent wants to download a file", foot: "Downloads through Ghost FTP's transfer engine." },
  upload: { title: "Agent wants to upload a file", foot: "Uploads through Ghost FTP's transfer engine." },
  upload_dir: { title: "Agent wants to upload a directory", foot: "Uploads the whole tree through Ghost FTP's transfer engine." },
  sync: { title: "Agent wants to sync directories", foot: "Copies through Ghost FTP's transfer engine; mirror mode deletes destination files missing from the source." },
  search: { title: "Agent wants to search the server", foot: "Searches through your authenticated Ghost FTP session." },
  skill: { title: "Agent wants to run a Skill across servers", foot: "Runs the skill's steps on every listed server through your authenticated Ghost FTP sessions." },
};
function approvalCopy(kind: string) {
  return (
    APPROVAL_COPY[kind] ?? {
      title: "Agent wants to run an operation",
      foot: "Runs through your authenticated Ghost FTP session.",
    }
  );
}

// Pull the load-bearing numbers out of a directory-upload / sync summary so
// the modal surfaces them as badges (risk obvious), not just prose. The full
// summary is always shown verbatim above; these only re-emphasize it, so a
// non-matching summary simply renders no badges.
function summaryStats(
  kind: string,
  command: string
): { label: string; danger?: boolean }[] {
  const stats: { label: string; danger?: boolean }[] = [];
  if (kind === "upload_dir") {
    const m = command.match(/\((\d+) files?, (.+?) total, overwrite: (yes|no)\)/);
    if (m) {
      stats.push({ label: `${m[1]} files` }, { label: m[2] });
      if (m[3] === "yes")
        stats.push({ label: "overwrites existing files", danger: true });
    }
  } else if (kind === "sync") {
    if (/dry run — no changes/.test(command)) {
      stats.push({ label: "dry run — nothing changes" });
      return stats;
    }
    const copy = command.match(/copy (\d+) files? \((.+?)\)/);
    if (copy) stats.push({ label: `copies ${copy[1]} files` }, { label: copy[2] });
    const del = command.match(/delete (\d+) files? on the destination \(mirror\)/);
    if (del)
      stats.push({ label: `DELETES ${del[1]} files on the destination`, danger: true });
  } else if (kind === "skill") {
    // "Run skill "X" on N server(s) (…) — M step(s) each"
    const servers = command.match(/on (\d+) servers? \(/);
    if (servers)
      stats.push({ label: `${servers[1]} server${servers[1] === "1" ? "" : "s"}`, danger: true });
    const steps = command.match(/— (\d+) steps? each/);
    if (steps) stats.push({ label: `${steps[1]} step${steps[1] === "1" ? "" : "s"} each` });
  }
  return stats;
}

// Always-mounted: boots the bridge store (status + event listeners) and renders
// the per-command approval modal whenever a request is pending.
export function AgentBridgeHost() {
  const approvals = useBridge((s) => s.approvals);
  const respond = useBridge((s) => s.respond);

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    let cancelled = false;
    useBridge
      .getState()
      .init()
      .then((c) => {
        if (cancelled) c();
        else cleanup = c;
      })
      .catch((error) => {
        toast.error("Couldn't initialize Agent Bridge", String(error));
      });
    return () => {
      cancelled = true;
      cleanup?.();
    };
  }, []);

  const pending = approvals[0];
  if (!pending) return null;
  return (
    <ApprovalModal
      approval={pending}
      count={approvals.length}
      onApprove={() => respond(pending.requestId, "approve")}
      onDeny={() => respond(pending.requestId, "deny")}
    />
  );
}

function ApprovalModal({
  approval,
  count,
  onApprove,
  onDeny,
}: {
  approval: BridgeApproval;
  count: number;
  onApprove: () => void;
  onDeny: () => void;
}) {
  const copy = approvalCopy(approval.kind);
  const stats = summaryStats(approval.kind, approval.command);
  const allowSudo = useBridge((s) => s.status.policy.allowSudo);
  const sudoNote =
    allowSudo && approval.kind === "exec" && /\bsudo\b/.test(approval.command);
  const panelRef = useRef<HTMLDivElement>(null);
  const denyRef = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  // Escape denies (the safe choice); Deny takes initial focus so a stray
  // keypress never auto-approves an agent command.
  useDialog(panelRef, { onClose: onDeny, initialFocus: denyRef });
  return (
    <div className="fixed inset-0 z-secure flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="anim-modal w-[28rem] max-w-[92vw] rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2 border-b border-border px-5 py-3.5">
          <ShieldCheck size={15} className="text-accent" />
          <span id={titleId} className="text-[15px] font-semibold tracking-tight">
            {copy.title}
          </span>
        </div>
        <div className="px-5 py-4">
          <div className="mb-2 text-xs text-text-muted">
            On <span className="font-medium text-text">{approval.sessionName}</span>{" "}
            via the Agent Bridge:
          </div>
          <pre className="selectable max-h-40 max-w-full overflow-auto whitespace-pre-wrap [overflow-wrap:anywhere] rounded-md border border-border bg-bg-subtle px-3 py-2 font-mono text-xs text-text">
            {approval.command}
          </pre>
          {stats.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1.5">
              {stats.map((s) => (
                <span
                  key={s.label}
                  className={cn(
                    "rounded-md border px-2 py-0.5 text-[11px] font-medium",
                    s.danger
                      ? "border-danger/40 bg-danger/10 text-danger"
                      : "border-border bg-bg-subtle text-text-muted"
                  )}
                >
                  {s.label}
                </span>
              ))}
            </div>
          )}
          <div className="mt-3 flex items-center gap-1.5 text-[11px] text-text-dim">
            <TerminalIcon size={11} /> {copy.foot}
          </div>
          {sudoNote && (
            <div className="mt-2 flex items-center gap-1.5 text-[11px] text-accent">
              <ShieldCheck size={11} /> Ghost FTP will answer sudo's prompt with this
              connection's password.
            </div>
          )}
        </div>
        <div className="flex items-center justify-between gap-2 border-t border-border px-5 py-3">
          <span className="text-[11px] text-text-dim">
            {count > 1 ? `${count - 1} more queued` : ""}
          </span>
          <div className="flex gap-2">
            <button
              ref={denyRef}
              onClick={onDeny}
              className="flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-sm text-text-muted hover:bg-bg-hover hover:text-text"
            >
              <Ban size={13} /> Deny
            </button>
            <button
              onClick={onApprove}
              className="btn-accent flex items-center gap-1 rounded-md px-3.5 py-1.5 text-sm font-medium text-white"
            >
              <Check size={14} /> Approve &amp; run
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function savedCommandsBlock(server: string, commands: SavedCommand[]): string {
  const S = server || "<server>";
  if (commands.length === 0) {
    return `## Saved commands

The user can save named, pre-approved commands in Ghost FTP's Agent Bridge panel; once
they do, run them with \`ghostftp-cli agent run "<name>" ${S}\` (no approval prompt).
Check what's available any time with \`ghostftp-cli agent commands\`.

`;
  }
  return `## Saved commands (prefer these)

The user has saved these named, **pre-approved** commands. Prefer running one by
name over composing a raw command — it runs immediately (no approval prompt) and
exactly as the user vetted it. You only supply the name and the connection:

\`\`\`bash
ghostftp-cli agent commands                          # list saved commands
ghostftp-cli agent run "<name>" ${S}                 # run one by name
\`\`\`

Available:
${commands
  .map(
    (c) =>
      `- \`${c.name}\` — \`${c.command}\`${c.description ? `  (${c.description})` : ""}`
  )
  .join("\n")}

`;
}

function skillMdReadOnly(server: string, commands: SavedCommand[]): string {
  const S = server || "<server>";
  const savedBlock = savedCommandsBlock(S, commands);
  return `---
name: ghostftp-helper
description: Help the user with their own Ghost FTP file client — browse, read, search and manage files on computers and servers they are already connected to. Use only for the routine file-management and read-only diagnostic tasks the user asks for.
---

# Helping the user in Ghost FTP (read-only)

Ghost FTP is the user's desktop SSH/SFTP file client (FileZilla + PuTTY). They have
already opened and authenticated a connection to a computer or server **they own
or operate**, and they have turned on the local Agent Bridge so you can help them
through it. Nothing runs unless the user first enables access and, by default,
approves each action in the Ghost FTP UI.

You help through **\`ghostftp-cli agent\`**, which talks to Ghost FTP's local Agent Bridge.
You never handle a URL, port or token — \`ghostftp-cli\` reads them from Ghost FTP's local
discovery file. You only need the **connection name** (here: \`${S}\`).

You are using the **read-only helper** skill. You can browse, read, search and
transfer files, but you do **not** run arbitrary commands. If the user asks for
something that needs a command (for example checking disk usage or service
status), tell them to switch to the admin skill or run it themselves.

## Before you start

Make sure Ghost FTP is running with the **Agent Bridge turned on** (the master switch
at the top of the Bridge panel) and that this connection has been granted access.
Confirm with:

\`\`\`bash
ghostftp-cli agent context           # bridge state, sessions, saved commands
ghostftp-cli agent sessions          # connections you may help with
\`\`\`

If it reports the bridge isn't running, ask the user to turn it on. (\`ghostftp-cli\`
must be on your PATH; if it isn't, the user can give you its full path.)

${savedBlock}## Browse / read / search

Use these for everyday file and diagnostics work. They work on **every protocol**
— SFTP, FTP/FTPS, S3, WebDAV, cloud drives, Ghost FTP Agent — not just SSH:

\`\`\`bash
ghostftp-cli agent ls     ${S} /var/log
ghostftp-cli agent read   ${S} /etc/hostname       # any protocol, capped at 256 KiB
ghostftp-cli agent search ${S} ".log" /var/log     # search <server> <query> [path]
ghostftp-cli agent info   ${S}                      # protocol, host, port, …
\`\`\`

## Transfer files

\`\`\`bash
ghostftp-cli agent download ${S} /etc/hosts         # → the user's Downloads folder
ghostftp-cli agent upload   ${S} ./a.txt /home/user # local file → remote dir (renames on collision)
ghostftp-cli agent upload   ${S} ./a.txt /home/user --overwrite   # replace a file of the same name
ghostftp-cli agent transfer <transferId>            # poll until done/error
\`\`\`

To *inspect* a remote file you don't need \`download\` — \`read\` fetches the text on
any protocol and prints it. Reserve \`download\` for binaries, for files over
256 KiB, and for when the user actually wants a local copy.

**Windows targets:** pass remote paths with forward slashes (\`C:/Users/User/app\`)
or in single quotes. Unquoted \`C:\\Users\\User\` loses its backslashes in bash and
becomes \`C:UsersUser\`, which Windows resolves against the server's working
directory — the transfer succeeds, into the wrong folder.

## Upload or sync a directory

One approval covers the whole tree; the prompt in Ghost FTP shows the exact
file/byte (and delete) counts.

\`\`\`bash
ghostftp-cli agent sync ${S} ./dist /var/www/app --dry-run   # preview — nothing changes
ghostftp-cli agent sync ${S} ./dist /var/www/app             # push new/changed files
ghostftp-cli agent upload-dir ${S} ./dist /var/www/releases  # upload the whole tree
\`\`\`

\`sync --mirror\` also DELETES destination files missing from the source. Never
use it without a \`--dry-run\` first, and only when the user explicitly asked
for an exact mirror.

## Review what you've done

\`\`\`bash
ghostftp-cli agent history --server ${S}            # recent activity, newest first
\`\`\`

Notes:
- Only help with tasks the user explicitly asks for. When in doubt, ask first.
- Each call may pause while the user approves it in Ghost FTP (unless they pre-approved
  that kind). A blocked call means they're being asked to confirm.
- Transfers run in the background and also appear in Ghost FTP's transfer panel; poll
  \`agent transfer <id>\` for the outcome.
`;
}

function skillMdAdmin(server: string, commands: SavedCommand[]): string {
  const S = server || "<server>";
  const savedBlock = savedCommandsBlock(S, commands);
  return `---
name: ghostftp-helper-admin
description: Help the user manage their own Ghost FTP file client and connected servers — browse, read, search, transfer files and run status/diagnostic commands on computers and servers they already connected to. Use only for the routine administration and debugging tasks the user asks for.
---

# Helping the user in Ghost FTP (admin)

Ghost FTP is the user's desktop SSH/SFTP file client (FileZilla + PuTTY). They have
already opened and authenticated a connection to a computer or server **they own
or operate**, and they have turned on the local Agent Bridge so you can help them
through it. Nothing runs unless the user first enables access and, by default,
approves each action in the Ghost FTP UI.

You help through **\`ghostftp-cli agent\`**, which talks to Ghost FTP's local Agent Bridge.
You never handle a URL, port or token — \`ghostftp-cli\` reads them from Ghost FTP's local
discovery file. You only need the **connection name** (here: \`${S}\`).

You are using the **admin helper** skill, which can also run commands on connected
SSH servers. Prefer read-only inspection first, and only run commands the user
explicitly asks for.

## Before you start

Make sure Ghost FTP is running with the **Agent Bridge turned on** (the master switch
at the top of the Bridge panel) and that this connection has been granted access.
Confirm with:

\`\`\`bash
ghostftp-cli agent context           # bridge state, sessions, saved commands
ghostftp-cli agent sessions          # connections you may help with
\`\`\`

If it reports the bridge isn't running, ask the user to turn it on. (\`ghostftp-cli\`
must be on your PATH; if it isn't, the user can give you its full path.)

${savedBlock}## Browse / read / search

Use these for everyday file and diagnostics work. They work on **every protocol**
— SFTP, FTP/FTPS, S3, WebDAV, cloud drives, Ghost FTP Agent — not just SSH:

\`\`\`bash
ghostftp-cli agent ls     ${S} /var/log
ghostftp-cli agent read   ${S} /etc/hostname       # any protocol, capped at 256 KiB
ghostftp-cli agent search ${S} ".log" /var/log     # search <server> <query> [path]
ghostftp-cli agent info   ${S}                      # protocol, host, port, …
\`\`\`

## Change files on the server

You can edit the remote filesystem directly — **never** tell the user to go do it
by hand in Ghost FTP's file manager or their host's control panel:

\`\`\`bash
ghostftp-cli agent write ${S} /var/www/.htaccess --content "…" --overwrite  # replace a file's contents
ghostftp-cli agent rm    ${S} /var/www/shell.php                            # delete a file
ghostftp-cli agent rm    ${S} /var/www/old-cache -r                         # delete a directory + contents
ghostftp-cli agent mv    ${S} /var/www/shell.php /var/www/shell.php.quarantine
ghostftp-cli agent mkdir ${S} /var/www/releases
\`\`\`

**\`write\` replaces; \`upload\` renames.** To change what's already at a path, use
\`write --overwrite\` (or \`upload --overwrite\`). A plain \`upload\` deliberately
side-steps a collision — \`shell.php\` lands as \`shell_1.php\` and the original is
left exactly as it was, which is silently wrong when the intent was to replace it.

Deletes are irreversible. When the file might still be wanted (suspected malware,
a config you're replacing), \`mv\` it aside instead of deleting it.

## Run a status or diagnostic command (SSH or Ghost FTP Agent)

For checking state only — for example disk usage, service status or log tailing:

\`\`\`bash
ghostftp-cli agent exec ${S} "df -h"
ghostftp-cli agent exec ${S} --timeout-ms 300000 "tar czf /tmp/site.tgz /var/www"  # long-running
\`\`\`

Keep commands non-interactive (no prompts/pagers); add \`-y\`, \`| cat\`, etc. The
remote exit code is propagated; stderr is printed to stderr. Avoid mutating
commands unless the user explicitly asks for them.

## Transfer files

\`\`\`bash
ghostftp-cli agent download ${S} /etc/hosts         # → the user's Downloads folder
ghostftp-cli agent upload   ${S} ./a.txt /home/user # local file → remote dir (renames on collision)
ghostftp-cli agent upload   ${S} ./a.txt /home/user --overwrite   # replace a file of the same name
ghostftp-cli agent transfer <transferId>            # poll until done/error
\`\`\`

To *inspect* a remote file you don't need \`download\` at all — \`read\` fetches the
text on any protocol and prints it. Reserve \`download\` for binaries, for files
over 256 KiB, and for when the user actually wants a local copy.

**Windows targets:** pass remote paths with forward slashes (\`C:/Users/User/app\`)
or in single quotes. Unquoted \`C:\\Users\\User\` loses its backslashes in bash and
becomes \`C:UsersUser\`, which Windows resolves against the server's working
directory — the transfer succeeds, into the wrong folder.

## Upload or sync a directory

One approval covers the whole tree; the prompt in Ghost FTP shows the exact
file/byte (and delete) counts.

\`\`\`bash
ghostftp-cli agent sync ${S} ./dist /var/www/app --dry-run   # preview — nothing changes
ghostftp-cli agent sync ${S} ./dist /var/www/app             # push new/changed files
ghostftp-cli agent upload-dir ${S} ./dist /var/www/releases  # upload the whole tree
\`\`\`

\`sync\` walks the whole remote tree before it does anything, so point it at a
specific subdirectory rather than a docroot with a big \`uploads/\` under it — a
huge tree can outrun the request. For a handful of files, \`write\`/\`upload\` per
file is faster and more predictable.

Staged deploy in one line:
\`ghostftp-cli agent upload-dir ${S} ./dist /var/www/releases && ghostftp-cli agent exec ${S} "ln -sfn /var/www/releases/dist /var/www/current"\`

\`sync --mirror\` also DELETES destination files missing from the source. Never
use it without a \`--dry-run\` first, and only when the user explicitly asked
for an exact mirror.

## Review what you've done

\`\`\`bash
ghostftp-cli agent history --server ${S}            # recent activity, newest first
\`\`\`

Notes:
- Only help with tasks the user explicitly asks for. When in doubt, ask first.
- Each call may pause while the user approves it in Ghost FTP (unless they pre-approved
  that kind). A blocked call means they're being asked to confirm.
- Transfers run in the background and also appear in Ghost FTP's transfer panel; poll
  \`agent transfer <id>\` for the outcome.
- \`exec\` output is capped (512 KiB) and times out after 60s by default;
  \`--timeout-ms\` raises that up to 15 minutes for legitimately long commands.
`;
}

function mcpAddCmd(url: string, token: string): string {
  return `claude mcp add --transport http ghostftp ${url}/mcp --header "Authorization: Bearer ${token}"`;
}

// The Agent Bridge control panel (opened as a modal).
export function AgentBridge({ onClose }: { onClose: () => void }) {
  const status = useBridge((s) => s.status);
  const activity = useBridge((s) => s.activity);
  const start = useBridge((s) => s.start);
  const stop = useBridge((s) => s.stop);
  const setEnabled = useBridge((s) => s.setEnabled);
  const setSessionAccess = useBridge((s) => s.setSessionAccess);
  const setPolicy = useBridge((s) => s.setPolicy);
  const refresh = useBridge((s) => s.refresh);
  const savedCommands = useBridge((s) => s.savedCommands);
  const saveCommand = useBridge((s) => s.saveCommand);
  const deleteCommand = useBridge((s) => s.deleteCommand);
  const loadCommands = useBridge((s) => s.loadCommands);
  const refreshActivity = useBridge((s) => s.refreshActivity);
  const clearActivity = useBridge((s) => s.clearActivity);
  const setConsoleOpen = useLayout((s) => s.setConsoleOpen);

  const activeSessionId = useConnections((s) => s.activeSessionId);
  const sessions = useConnections((s) => s.sessions);
  const profiles = useConnections((s) => s.profiles);
  // Every live connection, paired with its profile, for per-session toggles.
  const liveSessions = sessions
    .map((s) => ({ ...s, profile: profiles.find((p) => p.id === s.profileId) }))
    .filter((s) => s.profile);
  const anyGranted = status.enabledSessions.length > 0;
  // The session the copy-paste setup snippet targets: prefer the focused one if
  // it's enabled, otherwise the first enabled session.
  const setupSessionId =
    activeSessionId && status.enabledSessions.includes(activeSessionId)
      ? activeSessionId
      : status.enabledSessions[0] ?? null;
  // The CLI matches servers by their profile NAME, so the skill snippet uses
  // the name (not the session UUID) of the session we're targeting.
  const setupServerName =
    liveSessions.find((s) => s.sessionId === setupSessionId)?.profile?.name ??
    null;
  const policy = status.policy;
  const patchPolicy = (patch: Partial<ApprovalPolicy>) =>
    setPolicy({ ...policy, ...patch });

  const [showToken, setShowToken] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useDialog(panelRef, { onClose });

  useEffect(() => {
    refresh();
    loadCommands();
  }, [refresh, loadCommands]);

  // Resolve a session id -> profile name + color for the history rows.
  const sessionMeta = new Map(
    liveSessions.map((s) => [
      s.sessionId,
      { name: s.profile?.name ?? "", color: s.profile?.color },
    ])
  );

  const canCopySetup =
    status.enabled && status.running && anyGranted && !!setupServerName;

  return (
    <div
      className="fixed inset-0 z-modal flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
        className="anim-modal flex max-h-[88vh] w-[52rem] max-w-[94vw] flex-col rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2 border-b border-border px-5 py-3.5">
          <Radio size={15} className="text-accent" />
          <span id={titleId} className="text-[15px] font-semibold tracking-tight">
            Agent Bridge
          </span>
          {status.running && (
            <span className="flex items-center gap-1 rounded-full bg-success/15 px-1.5 py-0.5 text-[10px] font-medium text-success">
              <span className="h-1.5 w-1.5 rounded-full bg-success" /> running
            </span>
          )}
          <div className="flex-1" />
          <button
            onClick={() => {
              setConsoleOpen(true);
              onClose();
            }}
            className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
          >
            <TerminalIcon size={12} /> Live console
          </button>
          <button
            onClick={onClose}
            className="rounded-md p-1.5 text-text-muted hover:bg-bg-hover hover:text-text"
          >
            <X size={14} />
          </button>
        </div>

        <div className="flex-1 space-y-4 overflow-y-auto px-5 py-4">
          {/* Master switch — default off; off means nothing is exposed. */}
          <Card title="Agent Bridge">
            <div className="flex items-start gap-3">
              <div className="min-w-0 flex-1">
                <div className="text-sm font-medium">
                  {status.enabled
                    ? "On — AI access available"
                    : "Off — nothing exposed"}
                </div>
                <div className="text-[11px] leading-snug text-text-dim">
                  Off means no token and no localhost server exist — nothing is
                  exposed. Turn it on only when you want AI help; even then, each
                  connection stays private until you grant it access below.
                </div>
              </div>
              <Toggle checked={status.enabled} onChange={(v) => setEnabled(v)} />
            </div>
          </Card>

          <p className="text-xs leading-relaxed text-text-muted">
            Let a local AI agent (Claude Code, Cursor…) operate on your connected
            servers through Ghost FTP — run commands, browse, read, search and transfer
            files, with no agent install on the server and no credentials shared.
            Ghost FTP brokers its authenticated session over a localhost endpoint and
            asks you to approve each request (unless you relax that below).
          </p>

          <div
            className={cn(
              // Two-column masonry of cards — each card stays whole, the popup is
              // wider rather than a long single scroll.
              "columns-1 gap-4 md:columns-2 [&>*]:mb-4 [&>*]:break-inside-avoid",
              !status.enabled && "pointer-events-none select-none opacity-50"
            )}
            aria-disabled={!status.enabled}
          >
            {/* Server */}
            <Card title="Local endpoint">
            <div className="flex items-center gap-2">
              <button
                onClick={status.running ? stop : start}
                className={cn(
                  "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium",
                  status.running
                    ? "border border-border text-text-muted hover:bg-bg-hover hover:text-text"
                    : "btn-accent text-white"
                )}
              >
                <Power size={13} />
                {status.running ? "Stop" : "Start bridge"}
              </button>
              {status.running && status.url && (
                <span className="font-mono text-xs text-text-muted">
                  {status.url}
                </span>
              )}
            </div>
            {status.running && (
              <div className="mt-3 space-y-2">
                <Row label="URL">
                  <span className="truncate font-mono text-xs">{status.url}</span>
                  <CopyButton text={status.url ?? ""} />
                </Row>
                <Row label="Token">
                  <span className="truncate font-mono text-xs">
                    {showToken
                      ? status.token
                      : "•".repeat(Math.min(24, status.token?.length ?? 0))}
                  </span>
                  <button
                    onClick={() => setShowToken((v) => !v)}
                    className="rounded p-0.5 text-text-dim hover:text-text"
                    title={showToken ? "Hide" : "Reveal"}
                  >
                    {showToken ? <EyeOff size={12} /> : <Eye size={12} />}
                  </button>
                  <CopyButton text={status.token ?? ""} />
                </Row>
              </div>
            )}
          </Card>

          {/* Access */}
          <Card title="Server access">
            {liveSessions.length === 0 ? (
              <div className="text-xs text-text-dim">
                Connect to a server first, then grant the agent access to it here.
                Each connection gets its own toggle — bridge as many at once as
                you like.
              </div>
            ) : (
              <div className="space-y-1.5">
                {liveSessions.map(({ sessionId, profile }) => {
                  const p = profile!;
                  const isSsh = p.protocol === "sftp";
                  const isAgent = p.protocol === "ghostftp-agent";
                  const canExec = isSsh || isAgent;
                  const isObject = p.protocol === "s3" || p.protocol === "azure" || p.protocol === "gcs";
                  return (
                    <div
                      key={sessionId}
                      className="flex items-center gap-3 rounded-md bg-bg/60 px-2.5 py-2"
                    >
                      <span
                        className="h-2.5 w-2.5 shrink-0 rounded-full"
                        style={{ background: p.color || "rgb(var(--accent))" }}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5 text-sm">
                          <span className="truncate font-medium">{p.name}</span>
                          {sessionId === activeSessionId && (
                            <span className="shrink-0 rounded-sm bg-accent-soft px-1 text-[9px] font-medium uppercase tracking-wider text-accent">
                              active
                            </span>
                          )}
                        </div>
                        <div className="truncate font-mono text-[11px] text-text-dim">
                          {isObject
                            ? `${p.protocol}://${p.bucket ?? "?"}`
                            : isAgent
                              ? p.host
                              : `${p.username}@${p.host}`}
                        </div>
                        {isAgent ? (
                          <div className="mt-0.5 text-[11px] text-text-dim">
                            Full machine — the agent can run commands here.
                          </div>
                        ) : (
                          !canExec && (
                            <div className="mt-0.5 text-[11px] text-text-dim">
                              File ops only — exec needs an SSH or Ghost FTP Agent session.
                            </div>
                          )
                        )}
                      </div>
                      <Toggle
                        checked={status.enabledSessions.includes(sessionId)}
                        onChange={(v) => setSessionAccess(sessionId, v)}
                      />
                    </div>
                  );
                })}
              </div>
            )}
          </Card>

          {/* Auto-approve */}
          <Card title="Auto-approve">
            <div className="mb-2.5 text-xs text-text-muted">
              By default Ghost FTP asks before every agent request. Loosen that here —
              applies to all enabled sessions.
            </div>
            <div className="space-y-2.5">
              <PolicyRow
                label="Allow all — no prompts"
                help="Approve every agent request (commands, reads, transfers) automatically. Most permissive."
                checked={policy.allowAll}
                onChange={(v) => patchPolicy({ allowAll: v })}
                danger
              />
              <PolicyRow
                label="Auto-approve read-only operations"
                help="List directories, read files and search run without asking. Downloads & uploads write to disk, so they still prompt unless Allow all is on."
                checked={policy.allowAll || policy.autoRead}
                disabled={policy.allowAll}
                onChange={(v) => patchPolicy({ autoRead: v })}
              />
              <PolicyRow
                label="Auto-approve safe shell commands"
                help="Read-only commands (ls, cat, df, grep…) run without asking; anything that could change the server still prompts. Best-effort heuristic."
                checked={policy.allowAll || policy.autoSafeExec}
                disabled={policy.allowAll}
                onChange={(v) => patchPolicy({ autoSafeExec: v })}
              />
            </div>
            {policy.allowAll && (
              <div className="mt-2.5 flex items-start gap-1.5 rounded-md border border-warning/30 bg-warning/10 px-2.5 py-1.5 text-[11px] text-warning">
                <CircleAlert size={12} className="mt-0.5 shrink-0" />
                The agent can run anything on enabled sessions without confirmation.
              </div>
            )}
          </Card>

          {/* Sudo — let Ghost FTP answer sudo's prompt with the login password. */}
          <Card title="Privileged commands (sudo)">
            <PolicyRow
              label="Answer sudo prompts with my password"
              help="When a command runs sudo, Ghost FTP types this connection's login password to answer the prompt — over a private pseudo-terminal, and only when sudo actually asks (a passwordless server never receives it). Password-auth connections only; key/agent logins have no password to reuse. The command still needs approval above."
              checked={policy.allowSudo}
              onChange={(v) => patchPolicy({ allowSudo: v })}
            />
            {policy.allowSudo && (
              <div className="mt-2.5 flex items-start gap-1.5 rounded-md border border-warning/30 bg-warning/10 px-2.5 py-1.5 text-[11px] text-warning">
                <CircleAlert size={12} className="mt-0.5 shrink-0" />
                Ghost FTP will send your login password to <span className="font-mono">sudo</span> for
                approved commands. It's never stored in or shown to the agent.
              </div>
            )}
          </Card>

          {/* Setup — MCP is the recommended path; skills are the fallback. */}
          <Card title="Connect your AI agent">
            <div className="mb-2 text-xs text-text-muted">
              <span className="font-medium text-text">Recommended.</span> Add Ghost FTP
              as an MCP server in Claude Code, Cursor or any MCP-compatible agent.
              The agent talks directly to Ghost FTP over a local endpoint — no big skill
              text to paste, and it auto-discovers the tools.
            </div>
            <div className="flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate rounded bg-bg px-2 py-1.5 font-mono text-[10px] text-text-muted">
                {canCopySetup
                  ? mcpAddCmd(status.url ?? "", status.token ?? "")
                  : "claude mcp add --transport http ghostftp …"}
              </code>
              <CopyButton
                disabled={!canCopySetup}
                text={
                  canCopySetup
                    ? mcpAddCmd(status.url ?? "", status.token ?? "")
                    : ""
                }
              />
            </div>
            <button
              disabled={!canCopySetup}
              onClick={async () => {
                try {
                  const msg = await ipc.bridgeRegisterMcp(
                    status.url ?? "",
                    status.token ?? ""
                  );
                  toast.success("MCP registered", msg);
                } catch (e) {
                  toast.error("Couldn't register MCP", String(e));
                }
              }}
              className="mt-2 flex w-full items-center justify-center gap-1 rounded-md border border-border bg-bg px-2 py-1.5 text-[11px] font-medium text-text hover:bg-bg-hover disabled:opacity-40"
            >
              <Plus size={12} /> Add to Claude Code automatically
            </button>

            <details className="mt-4 text-[11px] text-text-dim">
              <summary className="cursor-pointer select-none hover:text-text">
                Fallback: paste a skill (no MCP support)
              </summary>
              <div className="mt-2 space-y-2">
                <div className="text-xs text-text-muted">
                  If your agent doesn't support MCP, paste one of these skills
                  instead. The read-only skill is safest and least likely to be
                  flagged; the admin skill also allows diagnostic commands on SSH
                  servers.
                </div>
                <div className="flex flex-col gap-2 sm:flex-row">
                  <CopyButton
                    wide
                    disabled={!canCopySetup}
                    label="Copy read-only skill"
                    text={
                      canCopySetup
                        ? skillMdReadOnly(setupServerName ?? "", savedCommands)
                        : ""
                    }
                  />
                  <CopyButton
                    wide
                    disabled={!canCopySetup}
                    label="Copy admin skill"
                    text={
                      canCopySetup
                        ? skillMdAdmin(setupServerName ?? "", savedCommands)
                        : ""
                    }
                  />
                </div>
              </div>
            </details>
            {!canCopySetup && (
              <div className="mt-1.5 text-[11px] text-text-dim">
                Start the bridge and grant access to a session to enable this.
              </div>
            )}
          </Card>

          {/* Saved commands — pre-approved, run by name. */}
          <SavedCommandsCard
            commands={savedCommands}
            onSave={saveCommand}
            onDelete={deleteCommand}
          />

          {/* Skills — multi-step, fleet-targetable automations (Plan 8). */}
          <Card title="Fleet Skills">
            <div className="flex items-start gap-2">
              <div className="min-w-0 flex-1 text-[11px] leading-snug text-text-dim">
                Named, parameterized, multi-step workflows the AI can compose and
                run across many servers. Approve AI proposals, author your own,
                dry-run, then run — with a per-target result summary.
              </div>
              <button
                onClick={() => {
                  onClose();
                  useSkills.getState().openPanel();
                }}
                className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] hover:bg-bg-hover"
              >
                <Wand2 size={12} /> Open Skills
                <ChevronRight size={12} />
              </button>
            </div>
          </Card>

          {/* History — what the agent has done (in-memory, last 200). */}
          <HistoryCard
            activity={activity}
            sessionMeta={sessionMeta}
            onRefresh={refreshActivity}
            onClear={clearActivity}
          />
          </div>
        </div>

        <div className="border-t border-border px-5 py-3 text-[11px] text-text-dim">
          <ShieldCheck size={11} className="mr-1 inline" />
          {!status.enabled ? (
            "Agent Bridge is off — no token or localhost server exists."
          ) : (
            <>
              Bound to 127.0.0.1 only · token required · per-session opt-in ·{" "}
              {policy.allowAll
                ? "auto-approving all requests"
                : policy.autoRead || policy.autoSafeExec
                  ? "auto-approving some requests"
                  : "you approve every request"}
              .
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-border-subtle bg-bg-subtle/40 p-3">
      <div className="mb-2 text-[11px] font-semibold uppercase tracking-[0.08em] text-text-muted">
        {title}
      </div>
      {children}
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-2">
      <span className="w-12 shrink-0 text-[11px] text-text-dim">{label}</span>
      <div className="flex min-w-0 flex-1 items-center gap-2 rounded bg-bg px-2 py-1">
        {children}
      </div>
    </div>
  );
}

function CopyButton({
  text,
  label,
  wide,
  disabled,
}: {
  text: string;
  label?: string;
  wide?: boolean;
  disabled?: boolean;
}) {
  const [done, setDone] = useState(false);
  return (
    <button
      disabled={disabled}
      onClick={() => {
        void navigator.clipboard.writeText(text)
          .then(() => {
            setDone(true);
            setTimeout(() => setDone(false), 1200);
          })
          .catch((error) => toast.error("Couldn't copy to clipboard", String(error)));
      }}
      className={cn(
        "flex items-center gap-1 rounded border border-border px-1.5 py-0.5 text-[10px] text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40",
        wide && "px-2.5 py-1.5 text-xs"
      )}
    >
      {done ? <Check size={11} /> : <Copy size={11} />}
      {label ?? (done ? "Copied" : "Copy")}
    </button>
  );
}

function Toggle({
  checked,
  onChange,
  disabled,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative h-5 w-9 shrink-0 rounded-full border transition-colors",
        disabled && "cursor-not-allowed opacity-50",
        checked
          ? "border-accent bg-accent"
          : "border-border bg-bg-subtle hover:border-text-dim"
      )}
    >
      <span
        className={cn(
          "absolute left-0.5 top-0.5 h-4 w-4 rounded-full bg-white shadow-elev-1 transition-transform",
          checked ? "translate-x-4" : "translate-x-0"
        )}
      />
    </button>
  );
}

function PolicyRow({
  label,
  help,
  checked,
  onChange,
  disabled,
  danger,
}: {
  label: string;
  help: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
  danger?: boolean;
}) {
  return (
    <div className="flex items-start gap-3">
      <div className="min-w-0 flex-1">
        <div className={cn("text-sm", danger && "font-medium text-warning")}>
          {label}
        </div>
        <div className="text-[11px] leading-snug text-text-dim">{help}</div>
      </div>
      <Toggle checked={checked} onChange={onChange} disabled={disabled} />
    </div>
  );
}

function ActivityIcon({ kind, ok }: { kind: string; ok: boolean }) {
  if (kind === "denied")
    return <Ban size={12} className="mt-0.5 shrink-0 text-text-dim" />;
  if (kind === "error" || !ok)
    return <CircleAlert size={12} className="mt-0.5 shrink-0 text-danger" />;
  return <CheckCircle2 size={12} className="mt-0.5 shrink-0 text-success" />;
}

function SavedCommandsCard({
  commands,
  onSave,
  onDelete,
}: {
  commands: SavedCommand[];
  onSave: (c: SavedCommand) => void;
  onDelete: (id: string) => void;
}) {
  const [editing, setEditing] = useState<SavedCommand | "new" | null>(null);
  const [deleting, setDeleting] = useState<SavedCommand | null>(null);
  return (
    <Card title="Saved commands">
      <div className="mb-2 flex items-start gap-2">
        <div className="min-w-0 flex-1 text-[11px] leading-snug text-text-dim">
          Named, <span className="text-warning">pre-approved</span> commands the
          agent runs by name with no prompt — it can't change the command, only run
          it. (<span className="font-mono">ghostftp-cli agent run "&lt;name&gt;"</span>)
        </div>
        <button
          onClick={() => setEditing("new")}
          className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
        >
          <Plus size={12} /> Add
        </button>
      </div>
      {commands.length === 0 ? (
        <div className="text-xs text-text-dim">
          No saved commands yet. Add one the agent can run by name.
        </div>
      ) : (
        <div className="space-y-1">
          {commands.map((c) => (
            <div
              key={c.id}
              className="group flex items-center gap-2 rounded-md bg-bg px-2 py-1.5"
            >
              <Play size={12} className="shrink-0 text-success" />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="truncate text-[12px] font-medium">{c.name}</span>
                  {c.description && (
                    <span className="truncate text-[10px] text-text-dim">
                      {c.description}
                    </span>
                  )}
                </div>
                <div className="truncate font-mono text-[10px] text-text-muted">
                  {c.command}
                </div>
              </div>
              <button
                onClick={() => setEditing(c)}
                title="Edit"
                className="shrink-0 rounded p-1 text-text-dim opacity-0 hover:bg-bg-hover hover:text-text group-hover:opacity-100"
              >
                <Pencil size={12} />
              </button>
              <button
                onClick={() => setDeleting(c)}
                title="Delete"
                className="shrink-0 rounded p-1 text-text-dim opacity-0 hover:bg-bg-hover hover:text-danger group-hover:opacity-100"
              >
                <Trash2 size={12} />
              </button>
            </div>
          ))}
        </div>
      )}
      {editing && (
        <SavedCommandEditor
          command={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
          onSave={(c) => {
            onSave(c);
            setEditing(null);
          }}
        />
      )}
      {deleting && (
        <ConfirmModal
          title={`Delete "${deleting.name}"?`}
          message="The agent will no longer be able to run this command by name."
          destructive
          confirmLabel="Delete"
          onClose={() => setDeleting(null)}
          onConfirm={() => {
            onDelete(deleting.id);
            setDeleting(null);
          }}
        />
      )}
    </Card>
  );
}

function SavedCommandEditor({
  command,
  onClose,
  onSave,
}: {
  command: SavedCommand | null;
  onClose: () => void;
  onSave: (c: SavedCommand) => void;
}) {
  const [name, setName] = useState(command?.name ?? "");
  const [cmd, setCmd] = useState(command?.command ?? "");
  const [description, setDescription] = useState(command?.description ?? "");
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useDialog(panelRef, { onClose });
  const canSave = name.trim() !== "" && cmd.trim() !== "";
  const submit = () => {
    if (!canSave) return;
    onSave({
      id: command?.id ?? "",
      name: name.trim(),
      command: cmd.trim(),
      description: description.trim(),
    });
  };
  const inputCls =
    "w-full rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent";
  return (
    <div
      className="fixed inset-0 z-palette flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        onClick={(e) => e.stopPropagation()}
        className="anim-modal w-[30rem] max-w-[92vw] rounded-xl border border-border bg-bg-panel p-5 shadow-elev-3"
      >
        <div id={titleId} className="mb-3 text-sm font-semibold">
          {command ? "Edit saved command" : "New saved command"}
        </div>
        <label className="mb-3 block">
          <div className="mb-1 text-xs text-text-muted">Name</div>
          <input
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="restart-web"
            className={inputCls}
          />
        </label>
        <label className="mb-3 block">
          <div className="mb-1 text-xs text-text-muted">Command</div>
          <textarea
            value={cmd}
            onChange={(e) => setCmd(e.target.value)}
            placeholder="sudo systemctl restart nginx"
            rows={3}
            className={cn(inputCls, "resize-y font-mono text-[13px]")}
          />
        </label>
        <label className="mb-2 block">
          <div className="mb-1 text-xs text-text-muted">Description (optional)</div>
          <input
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Restart the web server"
            className={inputCls}
          />
        </label>
        <div className="mb-4 flex items-start gap-1.5 rounded-md border border-warning/30 bg-warning/10 px-2.5 py-1.5 text-[11px] text-warning">
          <CircleAlert size={12} className="mt-0.5 shrink-0" />
          The agent runs this exact command with no approval prompt.
        </div>
        <div className="flex items-center justify-end gap-2">
          <button
            onClick={onClose}
            className="rounded-md border border-border px-3.5 py-1.5 text-sm hover:bg-bg-hover"
          >
            Cancel
          </button>
          <button
            onClick={submit}
            disabled={!canSave}
            className="btn-accent rounded-md px-3.5 py-1.5 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}

function HistoryCard({
  activity,
  sessionMeta,
  onRefresh,
  onClear,
}: {
  activity: BridgeActivity[];
  sessionMeta: Map<string, { name: string; color?: string }>;
  onRefresh: () => void;
  onClear: () => void;
}) {
  return (
    <Card title="History">
      <div className="mb-2 flex items-center gap-2">
        <div className="min-w-0 flex-1 text-[11px] text-text-dim">
          What the agent has done — newest first (kept in memory, last 200).
        </div>
        <button
          onClick={onRefresh}
          title="Refresh"
          className="shrink-0 rounded p-1 text-text-dim hover:bg-bg-hover hover:text-text"
        >
          <RefreshCw size={12} />
        </button>
        <button
          onClick={onClear}
          disabled={activity.length === 0}
          title="Clear history"
          className="shrink-0 rounded p-1 text-text-dim hover:bg-bg-hover hover:text-danger disabled:opacity-40"
        >
          <Trash2 size={12} />
        </button>
      </div>
      {activity.length === 0 ? (
        <div className="text-xs text-text-dim">
          Nothing yet. Approved commands, reads, transfers and denials show up here.
        </div>
      ) : (
        <div className="max-h-48 space-y-1 overflow-y-auto">
          {activity.slice(0, 100).map((a) => {
            const meta = sessionMeta.get(a.sessionId);
            return (
              <div key={a.id} className="flex items-start gap-2 text-[11px]">
                <ActivityIcon kind={a.kind} ok={a.ok} />
                {meta && meta.name && (
                  <span className="flex shrink-0 items-center gap-1 text-text-dim">
                    <span
                      className="h-1.5 w-1.5 rounded-full"
                      style={{ background: meta.color || "rgb(var(--accent))" }}
                    />
                    <span className="max-w-[6rem] truncate">{meta.name}</span>
                  </span>
                )}
                <span className="min-w-0 flex-1 truncate font-mono text-text-muted">
                  {a.detail}
                </span>
                <span className="shrink-0 text-text-dim">{relTime(a.at)}</span>
              </div>
            );
          })}
        </div>
      )}
    </Card>
  );
}
