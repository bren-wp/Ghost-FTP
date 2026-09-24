import { useEffect, useId, useRef, useState } from "react";
import { KeyRound, X, Wand2, ShieldCheck } from "lucide-react";
import {
  ipc,
  onAuthPrompt,
  onAuthChanged,
} from "@/lib/ipc";
import { useDialog } from "@/hooks/useDialog";
import { useConnections } from "@/stores/connectionsStore";
import { generatePassword } from "@/lib/password";
import { toast } from "@/stores/toastStore";
import { messageOf } from "@/lib/errors";
import type { AuthPromptEvent } from "@/lib/types";

const lower = (s: string) => s.toLowerCase();
const isNewPwField = (p: string) =>
  lower(p).includes("new") && lower(p).includes("password");
const isRetypeField = (p: string) =>
  ["retype", "again", "confirm", "re-enter", "reenter"].some((w) =>
    lower(p).includes(w)
  );

// Mounted once near the top of the tree. Handles the SSH keyboard-interactive
// auth exchange — most importantly the forced password change a server demands
// for an expired/temp password on first login. After such a change succeeds, it
// offers to update the password saved in the connection profile.
export function AuthPromptModal() {
  const [queue, setQueue] = useState<AuthPromptEvent[]>([]);
  // New passwords the user just set, keyed by profile id, awaiting the
  // backend's `auth://changed` confirmation before we offer to save them.
  const captured = useRef<Map<string, string>>(new Map());
  const [savePrompt, setSavePrompt] = useState<{
    profileId: string;
    password: string;
  } | null>(null);

  useEffect(() => {
    let disposed = false;
    const unsubs: Array<() => void> = [];
    const track = (registration: Promise<() => void>, label: string) => {
      void registration
        .then((cleanup) => {
          if (disposed) cleanup();
          else unsubs.push(cleanup);
        })
        .catch((error) => {
          toast.error(label, messageOf(error));
        });
    };

    track(
      onAuthPrompt((e) => setQueue((q) => [...q, e])),
      "Authentication prompt listener unavailable"
    );
    track(
      onAuthChanged((e) => {
        const pw = captured.current.get(e.profileId);
        if (pw) {
          captured.current.delete(e.profileId);
          setSavePrompt({ profileId: e.profileId, password: pw });
        }
      }),
      "Authentication-change listener unavailable"
    );

    return () => {
      disposed = true;
      unsubs.forEach((cleanup) => cleanup());
    };
  }, []);

  const current = queue[0];

  const submit = async (event: AuthPromptEvent, values: string[]) => {
    // Remember a freshly-typed new password so we can offer to save it once the
    // server confirms the change succeeded (via auth://changed).
    const newIdx = event.prompts.findIndex((p) => isNewPwField(p.prompt));
    if (newIdx >= 0 && values[newIdx]) {
      captured.current.set(event.profileId, values[newIdx]);
    }
    try {
      await ipc.respondToAuthPrompt(event.requestId, values);
      setQueue((q) => q.slice(1));
    } catch (error) {
      toast.error("Couldn't submit authentication response", messageOf(error));
    }
  };

  const cancel = async (event: AuthPromptEvent) => {
    captured.current.delete(event.profileId);
    try {
      await ipc.respondToAuthPrompt(event.requestId, null);
      setQueue((q) => q.slice(1));
    } catch (error) {
      toast.error("Couldn't cancel authentication prompt", messageOf(error));
    }
  };

  if (savePrompt) {
    return (
      <SavePasswordDialog
        profileId={savePrompt.profileId}
        password={savePrompt.password}
        onClose={() => setSavePrompt(null)}
      />
    );
  }

  if (!current) return null;
  return (
    <AuthPromptDialog
      key={current.requestId}
      event={current}
      onSubmit={(values) => submit(current, values)}
      onCancel={() => cancel(current)}
    />
  );
}

function AuthPromptDialog({
  event,
  onSubmit,
  onCancel,
}: {
  event: AuthPromptEvent;
  onSubmit: (values: string[]) => Promise<void>;
  onCancel: () => Promise<void>;
}) {
  const [values, setValues] = useState<string[]>(() =>
    event.prompts.map(() => "")
  );
  const [submitting, setSubmitting] = useState(false);
  const [generating, setGenerating] = useState<number | null>(null);
  const panelRef = useRef<HTMLFormElement>(null);
  const firstInputRef = useRef<HTMLInputElement>(null);
  const titleId = useId();
  const busy = submitting || generating !== null;
  const cancelIfIdle = () => {
    if (!busy) void onCancel();
  };
  // Escape cancels (aborts the connection); the first field takes focus.
  useDialog(panelRef, { onClose: cancelIfIdle, initialFocus: firstInputRef });

  const setAt = (i: number, v: string) =>
    setValues((arr) => arr.map((x, idx) => (idx === i ? v : x)));

  const generateFor = async (i: number) => {
    if (busy) return;
    setGenerating(i);
    const pw = generatePassword();
    setValues((arr) =>
      arr.map((x, idx) => {
        // Fill the new-password field and any retype/confirm field with the same
        // value, so the user doesn't have to type the generated password twice.
        if (idx === i || isRetypeField(event.prompts[idx].prompt)) return pw;
        return x;
      })
    );
    try {
      await navigator.clipboard.writeText(pw);
      toast.success("Password generated", "Copied to clipboard");
    } catch (error) {
      toast.warning(
        "Password generated, but couldn't copy it",
        `The generated password remains in the fields. ${messageOf(error)}`
      );
    } finally {
      setGenerating(null);
    }
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (submitting) return;
    setSubmitting(true);
    try {
      await onSubmit(values);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-secure flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <form
        ref={panelRef}
        onSubmit={(event) => void submit(event)}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="anim-modal w-[30rem] max-w-[92vw] overflow-hidden rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2.5 border-b border-border bg-bg-subtle px-4 py-3">
          <div className="flex h-7 w-7 items-center justify-center rounded-md bg-accent-soft text-accent">
            <KeyRound size={15} />
          </div>
          <div className="min-w-0 flex-1">
            <div id={titleId} className="text-[13px] font-semibold">
              Server authentication
            </div>
            <div className="truncate font-mono text-[11px] text-text-dim">
              {event.host}
            </div>
          </div>
          <button
            type="button"
            onClick={cancelIfIdle}
            disabled={busy}
            className="rounded-md p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:cursor-not-allowed disabled:opacity-50"
            title="Cancel"
          >
            <X size={13} />
          </button>
        </div>

        <div className="px-4 py-4">
          {(event.instructions || event.name) && (
            <p className="mb-3 whitespace-pre-wrap text-[12.5px] leading-relaxed text-text-muted">
              {event.instructions || event.name}
            </p>
          )}

          {event.prompts.map((field, i) => {
            const showGenerate = isNewPwField(field.prompt);
            return (
              <label key={i} className="mb-3 block last:mb-0">
                <div className="mb-1 text-xs text-text-muted">
                  {field.prompt.trim() || (field.echo ? "Response" : "Password")}
                </div>
                <div className="flex items-center gap-2">
                  <input
                    ref={i === 0 ? firstInputRef : undefined}
                    type={field.echo ? "text" : "password"}
                    value={values[i]}
                    onChange={(e) => setAt(i, e.target.value)}
                    autoComplete="off"
                    spellCheck={false}
                    disabled={submitting}
                    className="w-full rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent disabled:cursor-not-allowed disabled:opacity-60"
                  />
                  {showGenerate && (
                    <button
                      type="button"
                      onClick={() => void generateFor(i)}
                      disabled={busy}
                      aria-busy={generating === i}
                      title="Generate a strong password"
                      className="inline-flex shrink-0 items-center gap-1 rounded-md border border-border bg-bg-subtle px-2 py-1.5 text-[11.5px] text-text-muted hover:bg-bg-hover hover:text-text disabled:cursor-wait disabled:opacity-60"
                    >
                      <Wand2 size={12} />
                      {generating === i ? "Generating…" : "Generate"}
                    </button>
                  )}
                </div>
              </label>
            );
          })}
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-border bg-bg-subtle px-4 py-3">
          <button
            type="button"
            onClick={cancelIfIdle}
            disabled={busy}
            className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs font-medium hover:bg-bg-hover disabled:cursor-not-allowed disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={submitting}
            aria-busy={submitting}
            className="btn-accent rounded-md px-3 py-1.5 text-xs font-medium text-white disabled:cursor-wait disabled:opacity-70"
          >
            {submitting ? "Submitting…" : "Submit"}
          </button>
        </div>
      </form>
    </div>
  );
}

function SavePasswordDialog({
  profileId,
  password,
  onClose,
}: {
  profileId: string;
  password: string;
  onClose: () => void;
}) {
  const profiles = useConnections((s) => s.profiles);
  const saveProfile = useConnections((s) => s.saveProfile);
  const profile = profiles.find((p) => p.id === profileId);
  const [saving, setSaving] = useState(false);

  const panelRef = useRef<HTMLDivElement>(null);
  const keepRef = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  const closeIfIdle = () => {
    if (!saving) onClose();
  };
  // Escape = keep the old saved value (the conservative choice).
  useDialog(panelRef, { onClose: closeIfIdle, initialFocus: keepRef });

  const update = async () => {
    if (saving) return;
    setSaving(true);
    if (profile && profile.auth.kind === "password") {
      try {
        await saveProfile({ ...profile, auth: { kind: "password", password } });
        toast.success("Saved password updated", profile.name);
        onClose();
        return;
      } catch (error) {
        toast.error("Couldn't update saved password", messageOf(error));
      }
    }
    setSaving(false);
  };

  return (
    <div className="fixed inset-0 z-secure flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="anim-modal w-[28rem] max-w-[92vw] overflow-hidden rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2.5 border-b border-border bg-bg-subtle px-4 py-3">
          <div className="flex h-7 w-7 items-center justify-center rounded-md bg-accent-soft text-accent">
            <ShieldCheck size={15} />
          </div>
          <div id={titleId} className="text-[13px] font-semibold">
            Password changed
          </div>
        </div>
        <div className="px-4 py-4">
          <p className="text-[12.5px] leading-relaxed text-text-muted">
            You changed the password for{" "}
            <strong>{profile?.name ?? "this connection"}</strong> during login.
            Update the saved password so future connections use the new one?
          </p>
        </div>
        <div className="flex items-center justify-end gap-2 border-t border-border bg-bg-subtle px-4 py-3">
          <button
            ref={keepRef}
            type="button"
            onClick={closeIfIdle}
            disabled={saving}
            className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs font-medium hover:bg-bg-hover disabled:cursor-not-allowed disabled:opacity-50"
          >
            Keep old
          </button>
          <button
            type="button"
            onClick={() => void update()}
            disabled={saving}
            aria-busy={saving}
            className="btn-accent rounded-md px-3 py-1.5 text-xs font-medium text-white disabled:cursor-wait disabled:opacity-70"
          >
            {saving ? "Updating…" : "Update saved password"}
          </button>
        </div>
      </div>
    </div>
  );
}
