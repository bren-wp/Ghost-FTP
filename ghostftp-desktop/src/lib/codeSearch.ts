/**
 * Literal (never regex / executable) find and replace for untrusted remote
 * source code. The returned string is only a draft; the existing editor save
 * pipeline remains responsible for confirmation and remote upload.
 */
export interface ReplaceResult {
  content: string;
  count: number;
  cursor: number;
}

function escapeLiteral(value: string): string {
  return value.replace(/[|\\{}()[\]^$+*?.]/g, "\\$&");
}

export function replaceNextLiteral(
  content: string,
  needle: string,
  replacement: string,
  selectionStart: number,
  selectionEnd: number,
): ReplaceResult {
  if (!needle) return { content, count: 0, cursor: selectionEnd };
  const start = Math.max(0, Math.min(content.length, selectionStart));
  const end = Math.max(start, Math.min(content.length, selectionEnd));
  const pattern = new RegExp(escapeLiteral(needle), "giu");
  let first: RegExpMatchArray | undefined;
  let target: RegExpMatchArray | undefined;
  for (const match of content.matchAll(pattern)) {
    first ??= match;
    const index = match.index ?? 0;
    if (index === start && index + match[0].length === end) {
      target = match;
      break;
    }
    if (index >= end) {
      target = match;
      break;
    }
  }
  target ??= first;
  if (!target) return { content, count: 0, cursor: end };
  const at = target.index ?? 0;
  const next = content.slice(0, at) + replacement + content.slice(at + target[0].length);
  return { content: next, count: 1, cursor: at + replacement.length };
}

export function replaceAllLiteral(
  content: string,
  needle: string,
  replacement: string,
): ReplaceResult {
  if (!needle) return { content, count: 0, cursor: 0 };
  const pattern = new RegExp(escapeLiteral(needle), "giu");
  let count = 0;
  const next = content.replace(pattern, () => {
    count++;
    return replacement; // never interpret $&, $1 or backslashes as template syntax
  });
  return { content: next, count, cursor: 0 };
}
