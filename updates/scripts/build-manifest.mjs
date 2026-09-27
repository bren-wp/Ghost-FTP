import fs from "node:fs";
import path from "node:path";

const args = Object.fromEntries(process.argv.slice(2).map((part) => {
  const i = part.indexOf("=");
  if (i < 0) throw new Error(`Expected --key=value, received: ${part}`);
  return [part.slice(0, i).replace(/^--/, ""), part.slice(i + 1)];
}));

const required = [
  "version", "notes", "pub-date",
  "windows-url", "windows-signature-file",
  "linux-url", "linux-signature-file", "out"
];
for (const key of required) {
  if (!args[key]) throw new Error(`Missing --${key}=...`);
}

if (!/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(args.version)) {
  throw new Error("version must be SemVer x.y.z");
}
if (Number.isNaN(Date.parse(args["pub-date"]))) {
  throw new Error("pub-date must be an RFC 3339 date/time");
}
const expectedPackagePath = {
  "windows-url": `/updates/package/GhostFTP-Windows-x64-Setup-v${args.version}.exe`,
  "linux-url": `/updates/package/GhostFTP-Linux-x86_64-v${args.version}.AppImage`
};
for (const key of ["windows-url", "linux-url"]) {
  const url = new URL(args[key]);
  if (
    url.protocol !== "https:" ||
    url.hostname !== "ghostftp.com" ||
    url.port !== "" ||
    url.username !== "" ||
    url.password !== "" ||
    url.search !== "" ||
    url.hash !== "" ||
    url.pathname !== expectedPackagePath[key]
  ) {
    throw new Error(`${key} must use the canonical first-party ghostftp.com update package URL`);
  }
}

const signature = (file, label) => {
  const value = fs.readFileSync(file, "utf8").trim();
  if (value.length < 16 || /REPLACE_WITH/i.test(value)) {
    throw new Error(`${label} signature file is missing or unusable: ${file}`);
  }
  return value;
};

const manifest = {
  version: args.version,
  notes: args.notes,
  pub_date: new Date(args["pub-date"]).toISOString(),
  platforms: {
    "windows-x86_64": {
      url: args["windows-url"],
      signature: signature(args["windows-signature-file"], "Windows")
    },
    "linux-x86_64": {
      url: args["linux-url"],
      signature: signature(args["linux-signature-file"], "Linux")
    }
  }
};

fs.mkdirSync(path.dirname(args.out), { recursive: true });
fs.writeFileSync(args.out, JSON.stringify(manifest, null, 2) + "\n", "utf8");
console.log(`Ghost FTP desktop update response written to ${args.out}`);
