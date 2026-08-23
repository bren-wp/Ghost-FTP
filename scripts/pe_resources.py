#!/usr/bin/env python3
"""Ugrađuje VERSIONINFO, manifest aplikacije i ICO resurse u nepotpisanu PE32/PE32+ izvršnu datoteku.

Koristi samo Python standardnu biblioteku i namijenjen je determinističkim ByFTP buildovima.
Ne potpisuje izvršnu datoteku; Authenticode potpisivanje zaseban je korak izdanja.
"""
from __future__ import annotations

import argparse
import struct
from dataclasses import dataclass, field
from pathlib import Path

RT_ICON = 3
RT_GROUP_ICON = 14
RT_VERSION = 16
RT_MANIFEST = 24
LANG_EN_US = 0x0409
CODEPAGE_UNICODE = 1200
SECTION_CHARS = 0x40000040  # IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ
I386 = 0x014C
AMD64 = 0x8664


def align(value: int, boundary: int) -> int:
    return (value + boundary - 1) // boundary * boundary


def pad4(data: bytearray) -> None:
    data.extend(b"\0" * ((-len(data)) & 3))


def wstr(value: str) -> bytes:
    return (value + "\0").encode("utf-16le")


def make_string_block(key: str, value: str) -> bytes:
    out = bytearray(b"\0" * 6)
    out += wstr(key)
    pad4(out)
    value_bytes = wstr(value)
    out += value_bytes
    struct.pack_into("<HHH", out, 0, len(out), len(value_bytes) // 2, 1)
    return bytes(out)


def make_container_block(key: str, children: list[bytes], value: bytes = b"", value_length: int = 0, value_type: int = 1) -> bytes:
    out = bytearray(b"\0" * 6)
    out += wstr(key)
    pad4(out)
    if value:
        out += value
        pad4(out)
    for child in children:
        out += child
        pad4(out)
    struct.pack_into("<HHH", out, 0, len(out), value_length, value_type)
    return bytes(out)


def make_version_info(version: tuple[int, int, int, int], original_filename: str, role: str) -> bytes:
    major, minor, patch, build = version
    fixed = struct.pack(
        "<13I",
        0xFEEF04BD,
        0x00010000,
        (major << 16) | minor,
        (patch << 16) | build,
        (major << 16) | minor,
        (patch << 16) | build,
        0x0000003F,
        0x00000000,
        0x00040004,
        0x00000001,
        0,
        0,
        0,
    )
    descriptions = {
        "portable": ("ByFTP klijent", "ByFTP"),
        "setup": ("ByFTP instalacijski program", "ByFTPSetup"),
        "uninstaller": ("ByFTP program za uklanjanje", "ByFTPUninstall"),
    }
    description, internal_name = descriptions.get(role, descriptions["portable"])
    strings = [
        ("CompanyName", "ByFTP"),
        ("FileDescription", description),
        ("FileVersion", f"{major}.{minor}.{patch}.{build}"),
        ("InternalName", internal_name),
        ("LegalCopyright", "Copyright © 2026 ByFTP"),
        ("OriginalFilename", original_filename),
        ("ProductName", "ByFTP"),
        ("ProductVersion", f"{major}.{minor}.{patch}.{build}"),
        ("Comments", "Siguran FTP, FTPS i SFTP klijent — ByFTP — github.com/bren-wp/by-ftp"),
    ]
    string_table = make_container_block("040904B0", [make_string_block(k, v) for k, v in strings], value_type=1)
    string_file_info = make_container_block("StringFileInfo", [string_table], value_type=1)
    translation_value = struct.pack("<HH", LANG_EN_US, CODEPAGE_UNICODE)
    var_translation = make_container_block("Translation", [], value=translation_value, value_length=len(translation_value), value_type=0)
    var_file_info = make_container_block("VarFileInfo", [var_translation], value_type=1)
    return make_container_block("VS_VERSION_INFO", [string_file_info, var_file_info], value=fixed, value_length=len(fixed), value_type=0)


def make_manifest(version: tuple[int, int, int, int], role: str, processor_architecture: str) -> bytes:
    major, minor, patch, build = version
    if processor_architecture not in {"amd64", "x86"}:
        raise ValueError(f"Nepodržana Windows arhitektura manifesta: {processor_architecture}")
    identity = {
        "portable": "ByFTP.Client",
        "setup": "ByFTP.Setup",
        "uninstaller": "ByFTP.Uninstaller",
    }.get(role, "ByFTP.Client")
    xml = f'''<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity version="{major}.{minor}.{patch}.{build}" processorArchitecture="{processor_architecture}" name="{identity}" type="win32"/>
  <description>ByFTP klijent tvrtke ByFTP</description>
  <dependency>
    <dependentAssembly>
      <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
    </dependentAssembly>
  </dependency>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security><requestedPrivileges><requestedExecutionLevel level="asInvoker" uiAccess="false"/></requestedPrivileges></security>
  </trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application><supportedOS Id="{{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}}"/></application>
  </compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
      <longPathAware xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">true</longPathAware>
    </windowsSettings>
  </application>
</assembly>
'''
    return xml.encode("utf-8")


def parse_ico(path: Path) -> tuple[list[bytes], bytes]:
    data = path.read_bytes()
    if len(data) < 6:
        raise ValueError("ICO je prekratak")
    reserved, kind, count = struct.unpack_from("<HHH", data, 0)
    if reserved != 0 or kind != 1 or count < 1 or count > 64:
        raise ValueError("Neispravan ICO header")
    images: list[bytes] = []
    group = bytearray(struct.pack("<HHH", 0, 1, count))
    for i in range(count):
        off = 6 + i * 16
        width, height, colors, reserved2, planes, bpp, size, image_off = struct.unpack_from("<BBBBHHII", data, off)
        end = image_off + size
        if end > len(data):
            raise ValueError("ICO zapis izlazi iz datoteke")
        images.append(data[image_off:end])
        group += struct.pack("<BBBBHHIH", width, height, colors, reserved2, planes, bpp, size, i + 1)
    return images, bytes(group)


@dataclass
class Leaf:
    payload: bytes
    data_entry_offset: int = 0
    payload_offset: int = 0


@dataclass
class Node:
    entries: list[tuple[int, "Node | Leaf"]] = field(default_factory=list)
    offset: int = 0


def make_resource_tree(resources: dict[int, list[tuple[int, bytes]]]) -> Node:
    root = Node()
    for resource_type in sorted(resources):
        type_node = Node()
        for resource_id, payload in sorted(resources[resource_type], key=lambda x: x[0]):
            language_node = Node(entries=[(LANG_EN_US, Leaf(payload))])
            type_node.entries.append((resource_id, language_node))
        root.entries.append((resource_type, type_node))
    return root


def walk_nodes(root: Node) -> tuple[list[Node], list[Leaf]]:
    nodes: list[Node] = []
    leaves: list[Leaf] = []

    def visit(node: Node) -> None:
        nodes.append(node)
        for _, child in node.entries:
            if isinstance(child, Node):
                visit(child)
            else:
                leaves.append(child)

    visit(root)
    return nodes, leaves


def build_resource_section(resources: dict[int, list[tuple[int, bytes]]], section_rva: int) -> bytes:
    root = make_resource_tree(resources)
    nodes, leaves = walk_nodes(root)
    cursor = 0
    for node in nodes:
        node.offset = cursor
        cursor += 16 + 8 * len(node.entries)
    cursor = align(cursor, 4)
    for leaf in leaves:
        leaf.data_entry_offset = cursor
        cursor += 16
    cursor = align(cursor, 4)
    for leaf in leaves:
        leaf.payload_offset = cursor
        cursor += len(leaf.payload)
        cursor = align(cursor, 4)
    out = bytearray(cursor)
    for node in nodes:
        struct.pack_into("<IIHHHH", out, node.offset, 0, 0, 0, 0, 0, len(node.entries))
        entry_off = node.offset + 16
        for resource_id, child in node.entries:
            target = (0x80000000 | child.offset) if isinstance(child, Node) else child.data_entry_offset
            struct.pack_into("<II", out, entry_off, resource_id, target)
            entry_off += 8
    for leaf in leaves:
        struct.pack_into("<IIII", out, leaf.data_entry_offset, section_rva + leaf.payload_offset, len(leaf.payload), 0, 0)
        out[leaf.payload_offset:leaf.payload_offset + len(leaf.payload)] = leaf.payload
    return bytes(out)


def patch_pe(exe: Path, ico: Path, version: tuple[int, int, int, int], role: str, original_filename: str) -> None:
    raw = bytearray(exe.read_bytes())
    if raw[:2] != b"MZ":
        raise ValueError("Datoteka nije PE/MZ")
    pe_off = struct.unpack_from("<I", raw, 0x3C)[0]
    if raw[pe_off:pe_off + 4] != b"PE\0\0":
        raise ValueError("PE potpis nije pronađen")
    coff = pe_off + 4
    machine, number_sections, _, _, _, size_opt, _ = struct.unpack_from("<HHIIIHH", raw, coff)
    opt = coff + 20
    magic = struct.unpack_from("<H", raw, opt)[0]
    if machine == AMD64 and magic == 0x20B:
        processor_architecture = "amd64"
        data_directory_offset = 112
    elif machine == I386 and magic == 0x10B:
        processor_architecture = "x86"
        data_directory_offset = 96
    else:
        raise ValueError(f"Nepodržan PE stroj/magic: machine=0x{machine:04x}, magic=0x{magic:04x}")

    section_alignment = struct.unpack_from("<I", raw, opt + 32)[0]
    file_alignment = struct.unpack_from("<I", raw, opt + 36)[0]
    size_headers = struct.unpack_from("<I", raw, opt + 60)[0]
    section_table = opt + size_opt
    new_header_off = section_table + number_sections * 40
    if new_header_off + 40 > size_headers:
        raise ValueError("Nema mjesta za dodatni .rsrc section header")
    max_end_rva = 0
    for i in range(number_sections):
        off = section_table + i * 40
        vsize, vaddr, raw_size, _ = struct.unpack_from("<IIII", raw, off + 8)
        max_end_rva = max(max_end_rva, vaddr + max(vsize, raw_size))
    section_rva = align(max_end_rva, section_alignment)
    icon_images, group_icon = parse_ico(ico)
    resources: dict[int, list[tuple[int, bytes]]] = {
        RT_ICON: [(i + 1, payload) for i, payload in enumerate(icon_images)],
        RT_GROUP_ICON: [(1, group_icon)],
        RT_VERSION: [(1, make_version_info(version, original_filename, role))],
        RT_MANIFEST: [(1, make_manifest(version, role, processor_architecture))],
    }
    rsrc = build_resource_section(resources, section_rva)
    raw_ptr = align(len(raw), file_alignment)
    raw_size = align(len(rsrc), file_alignment)
    if len(raw) < raw_ptr:
        raw.extend(b"\0" * (raw_ptr - len(raw)))
    raw.extend(rsrc)
    raw.extend(b"\0" * (raw_size - len(rsrc)))
    name = b".rsrc\0\0\0"
    header = struct.pack("<8sIIIIIIHHI", name, len(rsrc), section_rva, raw_size, raw_ptr, 0, 0, 0, 0, SECTION_CHARS)
    raw[new_header_off:new_header_off + 40] = header
    struct.pack_into("<H", raw, coff + 2, number_sections + 1)
    initialized_size = struct.unpack_from("<I", raw, opt + 8)[0]
    struct.pack_into("<I", raw, opt + 8, initialized_size + raw_size)
    struct.pack_into("<I", raw, opt + 56, align(section_rva + len(rsrc), section_alignment))
    resource_dd = opt + data_directory_offset + 2 * 8
    struct.pack_into("<II", raw, resource_dd, section_rva, len(rsrc))
    struct.pack_into("<I", raw, opt + 64, 0)
    exe.write_bytes(raw)


def parse_version(text: str) -> tuple[int, int, int, int]:
    parts = [int(p) for p in text.split(".")]
    if len(parts) == 3:
        parts.append(0)
    if len(parts) != 4 or any(p < 0 or p > 65535 for p in parts):
        raise argparse.ArgumentTypeError("verzija mora biti npr. 1.8.0 ili 1.8.0.0")
    return tuple(parts)  # type: ignore[return-value]


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("exe", type=Path)
    ap.add_argument("--ico", required=True, type=Path)
    ap.add_argument("--version", required=True, type=parse_version)
    ap.add_argument("--role", choices=("portable", "setup", "uninstaller"), required=True)
    ap.add_argument("--original-filename", required=True)
    args = ap.parse_args()
    patch_pe(args.exe, args.ico, args.version, args.role, args.original_filename)


if __name__ == "__main__":
    main()
