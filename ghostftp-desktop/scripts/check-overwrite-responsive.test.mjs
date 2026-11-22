// Ghost FTP by Brendigo
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const source = readFileSync(new URL("../src/components/OverwriteModal.tsx", import.meta.url), "utf8");
const native = readFileSync(new URL("../src-tauri/src/lib.rs", import.meta.url), "utf8");

test("overwrite dialog fits the supported 480px minimum main window", () => {
  assert.ok(native.includes(".min_inner_size(480.0, 600.0)"),
    "responsive acceptance must use the actual native minimum window");
  assert.ok(source.includes("max-w-[92vw]"));
  assert.ok(source.includes("max-[620px]:flex-col-reverse"),
    "the cancel action must have its own safe row at minimum window width");
  assert.ok(source.includes("max-[620px]:items-stretch"));
  assert.ok(source.includes("max-[620px]:grid-cols-3"),
    "the other three actions must stay accessible on one compact row");
  assert.ok(source.includes("max-[620px]:min-w-0"),
    "compact actions must be able to shrink without overflowing");
});

test("very long server filenames cannot expand the overwrite dialog", () => {
  assert.ok(source.includes('className="font-medium text-text break-all" dir="auto"'),
    "untrusted long filenames should wrap and use their own direction");
  assert.ok(source.includes('title={info.destDir}'),
    "the full directory must remain inspectable after visual truncation");
});

test("all four decisions are explicit buttons, preserving safe cancel", () => {
  for (const handler of [
    "onClick={cancel}",
    'onClick={() => pick("skip")}',
    'onClick={() => pick("rename")}',
    'onClick={() => pick("overwrite")}',
  ]) {
    const pos = source.indexOf(handler);
    assert.ok(pos >= 0, "missing overwrite decision: " + handler);
    assert.ok(source.slice(Math.max(0, pos - 85), pos).includes('type="button"'),
      "button must not implicitly submit an enclosing form: " + handler);
  }
  assert.ok(source.includes("key={pending.promptId}"),
    "responsive layout must retain exact conflict prompt identity");
});


test("optional destination-path copy is explicit, accessible, and reports failure", () => {
  assert.ok(source.includes('aria-label="Copy destination directory path"'));
  assert.ok(source.includes('title="Copy destination directory path"'));
  assert.ok(source.includes("navigator.clipboard?.writeText"),
    "clipboard API must be feature-detected before use");
  assert.ok(source.includes("await navigator.clipboard.writeText(info.destDir)"),
    "only the user-selected destination is copied after an explicit click");
  assert.ok(source.includes("onClick={() => { void copyDestination(); }}"),
    "copy must not happen automatically when a prompt opens");
  assert.ok(source.includes('toast.success("Destination path copied")'));
  assert.ok(source.includes('toast.error("Couldn\'t copy destination path")'),
    "clipboard denial must be reported instead of showing false success");
});
