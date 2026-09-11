#!/usr/bin/env python3
from pathlib import Path

path = Path("internal/desktop/gui_linux.go")
source = path.read_text(encoding="utf-8")

old_header = '''\tif err := u.x.text(u.width-300, 34, badge+"  "+u.version, color, premiumTheme.Panel); err != nil {
\t\treturn err
\t}
\treturn u.drawButton(u.layout.settings, u.tr("common.settings"), !u.busy, false)
}'''
new_header = '''\tif err := u.x.text(u.width-300, 34, badge+"  "+u.version, color, premiumTheme.Panel); err != nil {
\t\treturn err
\t}
\tif err := u.renderBookmarksHeaderButton(); err != nil {
\t\treturn err
\t}
\treturn u.drawButton(u.layout.settings, u.tr("common.settings"), !u.busy, false)
}'''

old_mouse = '''\t}
\tl := u.layout
\tif u.handleQueuePriorityMouse(x, y) {'''
new_mouse = '''\t}
\tif u.handleBookmarksHeaderMouse(x, y) {
\t\treturn
\t}
\tl := u.layout
\tif u.handleQueuePriorityMouse(x, y) {'''

header_count = source.count(old_header)
mouse_count = source.count(old_mouse)
if header_count == 0 and "u.renderBookmarksHeaderButton()" in source and "u.handleBookmarksHeaderMouse(x, y)" in source:
    print("LINUX_BOOKMARK_WIRING=ALREADY_FIXED")
    raise SystemExit(0)
if header_count != 1 or mouse_count != 1:
    raise SystemExit(f"Unexpected Linux bookmark wiring anchors: header={header_count} mouse={mouse_count}")

source = source.replace(old_header, new_header, 1).replace(old_mouse, new_mouse, 1)
path.write_text(source, encoding="utf-8")
print("LINUX_BOOKMARK_WIRING=FIXED")
