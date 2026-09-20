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
    const unsubs: Array<() => void> = [];
    onAuthPrompt((e) => setQueue((q) => [...q, e])).then((u) => unsubs.push(u));
    onAuthChanged((e) => {
      const pw = captured.current.get(e.profileId);
      if (pw) {
        captured.current.delete(e.profileId);
        setSavePrompt({ profileId: e.profileId, password: pw });
      }
    }).then((u) => unsubs.push(u));
    return () => unsubs.forEach((u) => u());
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
    } finally {
      setQueue((q) => q.slice(1));
    }
  };

  const cancel = async (event: AuthPromptEvent) => {
    captured.current.delete(event.profileId);
    try {
      await ipc.respondToAuthPrompt(event.requestId, null);
    } finally {
      setQueue((q) => q.slice(1));
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
  onSubmit: (values: string[]) => void;
  onCancel: () => void;
}) {
  const [values, setValues] = useState<string[]>(() =>
    event.prompts.map(() => "")
  );
  const panelRef = useRef<HTMLFormElement>(null);
  const firstInputRef = useRef<HTMLInputElement>(null);
  const titleId = useId();
  // Escape cancels (aborts the connection); the first field takes focus.
  useDialog(panelRef, { onClose: onCancel, initialFocus: firstInputRef });

  const setAt = (i: number, v: string) =>
    setValues((arr) => arr.map((x, idx) => (idx === i ? v : x)));

  const generateFor = async (i: number) => {
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
    } catch {
      // clipboard unavailable; fields are still filled
    }
  };

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(values);
  };

  return (
    <div className="fixed inset-0 z-secure flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <form
        ref={panelRef}
        onSubmit={submit}
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
            onClick={onCancel}
            className="rounded-md p-1 text-text-muted hover:bg-bg-hover hover:text-text"
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
                    className="w-full rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-sm outline-none focus:border-accent"
                  />
                  {showGenerate && (
                    <button
                      type="button"
                      onClick={() => generateFor(i)}
                      title="Generate a strong password"
                      className="inline-flex shrink-0 items-center gap-1 rounded-md border border-border bg-bg-subtle px-2 py-1.5 text-[11.5px] text-text-muted hover:bg-bg-hover hover:text-text"
                    >
                      <Wand2 size={12} />
                      Generate
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
            onClick={onCancel}
            className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs font-medium hover:bg-bg-hover"
          >
            Cancel
          </button>
          <button
            type="submit"
            className="btn-accent rounded-md px-3 py-1.5 text-xs font-medium text-white"
          >
            Submit
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

  const panelRef = useRef<HTMLDivElement>(null);
  const keepRef = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  // Escape = keep the old saved value (the conservative choice).
  useDialog(panelRef, { onClose, initialFocus: keepRef });

  const update = async () => {
    if (profile && profile.auth.kind === "password") {
      try {
        await saveProfile({ ...profile, auth: { kind: "password", password } });
        toast.success("Saved password updated", profile.name);
      } catch (e) {
        toast.error("Couldn't update saved password", String(e));
      }
    }
    onClose();
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
            onClick={onClose}
            className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs font-medium hover:bg-bg-hover"
          >
            Keep old
          </button>
          <button
            onClick={update}
            className="btn-accent rounded-md px-3 py-1.5 text-xs font-medium text-white"
          >
            Update saved password
          </button>
        </div>
      </div>
    </div>
  );
}
