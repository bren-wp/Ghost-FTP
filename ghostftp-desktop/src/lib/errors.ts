import { toast } from "@/stores/toastStore";

// Structured IPC errors (Plan 12 Phase 3). Migrated Rust commands reject with a
// `{ kind, message }` object instead of a bare string, so the UI can pattern-
// match on a stable discriminant and show an actionable toast — never regexing
// English. Un-migrated commands still reject with a string; `kindOf` treats
// those as "other", so both paths render sensibly.

export type ErrorKind =
  | "auth"
  | "permission"
  | "not-found"
  | "network"
  | "timeout"
  | "unsupported"
  | "conflict"
  | "other";

export interface GhostFtpError {
  kind: ErrorKind;
  message: string;
}

const KINDS = new Set<ErrorKind>([
  "auth",
  "permission",
  "not-found",
  "network",
  "timeout",
  "unsupported",
  "conflict",
  "other",
]);

/** Whether `e` is a structured `{ kind, message }` error from a migrated command. */
export function isGhostFtpError(e: unknown): e is GhostFtpError {
  return (
    typeof e === "object" &&
    e !== null &&
    typeof (e as { kind?: unknown }).kind === "string" &&
    KINDS.has((e as GhostFtpError).kind) &&
    typeof (e as { message?: unknown }).message === "string"
  );
}

/** The stable discriminant, or "other" for legacy/unstructured errors. */
export function kindOf(e: unknown): ErrorKind {
  return isGhostFtpError(e) ? e.kind : "other";
}

/** The human-readable message from any error shape. */
export function messageOf(e: unknown): string {
  if (isGhostFtpError(e)) return e.message;
  if (typeof e === "string") return e;
  if (e instanceof Error) return e.message;
  if (e && typeof (e as { message?: unknown }).message === "string") {
    return (e as { message: string }).message;
  }
  return String(e);
}

interface Copy {
  title: string;
  hint?: string;
}

/** Actionable toast copy per kind. `context` (e.g. "Rename failed") overrides
 *  the generic title so the toast names the operation that failed. */
function describe(kind: ErrorKind, context?: string): Copy {
  switch (kind) {
    case "auth":
      return { title: context ?? "Authentication failed", hint: "Reconnect or re-enter your credentials." };
    case "permission":
      return { title: context ?? "Permission denied", hint: "You don't have access to this path." };
    case "not-found":
      return { title: context ?? "Not found", hint: "It may have been moved or deleted." };
    case "network":
      return { title: context ?? "Connection problem", hint: "Check the connection and try again." };
    case "timeout":
      return { title: context ?? "Timed out", hint: "The server took too long to respond." };
    case "unsupported":
      return { title: context ?? "Not supported", hint: "This backend can't do that." };
    case "conflict":
      return { title: context ?? "Already exists", hint: "Choose a different name." };
    default:
      return { title: context ?? "Something went wrong" };
  }
}

/** Show an error toast keyed off the structured `kind`, falling back to a
 *  generic toast for legacy string errors. `context` names the operation. */
export function toastError(e: unknown, context?: string): ErrorKind {
  const kind = kindOf(e);
  const { title, hint } = describe(kind, context);
  const msg = messageOf(e);
  // Show the hint plus the raw message; if they'd duplicate, prefer the message.
  const body = hint ? (msg && msg !== hint ? `${hint} — ${msg}` : hint) : msg;
  toast.error(title, body || undefined);
  return kind;
}
