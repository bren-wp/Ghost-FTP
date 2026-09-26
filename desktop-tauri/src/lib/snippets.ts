// Snippet `{{variable}}` templating (Plan 11 Phase 4). Bodies may embed
// placeholders like `deploy {{site}} --branch {{ref}}`; before a snippet is
// inserted into a shell the UI collects a value for each distinct placeholder
// and substitutes it in. Deliberately tiny and dependency-free — this is a
// text substitution, not a template engine.

const PLACEHOLDER = /\{\{\s*([\w.-]+)\s*\}\}/g;

/** Distinct placeholder names in `body`, in order of first appearance. */
export function extractVariables(body: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const m of body.matchAll(PLACEHOLDER)) {
    const name = m[1];
    if (!seen.has(name)) {
      seen.add(name);
      out.push(name);
    }
  }
  return out;
}

/** Replace every `{{name}}` with `values[name]`; an unfilled placeholder is
 *  left verbatim so nothing silently vanishes from the command. */
export function resolveVariables(
  body: string,
  values: Record<string, string>
): string {
  return body.replace(PLACEHOLDER, (whole, name: string) =>
    name in values ? values[name] : whole
  );
}

/** Strip a single trailing newline so an inserted snippet never auto-submits
 *  its final line — the user reviews it at the prompt and presses Enter. */
export function stripTrailingNewline(text: string): string {
  return text.replace(/\r?\n$/, "");
}
