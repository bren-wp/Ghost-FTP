import fs from "node:fs";

const args = Object.fromEntries(process.argv.slice(2).map((part) => {
  const i = part.indexOf("=");
  if (i < 0) throw new Error(`Expected --key=value, received: ${part}`);
  return [part.slice(0, i).replace(/^--/, ""), part.slice(i + 1)];
}));

const required = ["channel", "version", "notes", "pub-date", "min-version", "windows-url", "windows-signature", "linux-url", "linux-signature", "out"];
for (const key of required) {
  if (!args[key]) throw new Error(`Missing --${key}=...`);
}
if (!["preview", "stable"].includes(args.channel)) throw new Error("channel must be preview or stable");
for (const key of ["windows-url", "linux-url"]) {
  if (!args[key].startsWith("https://")) throw new Error(`${key} must use HTTPS`);
}
for (const key of ["windows-signature", "linux-signature"]) {
  if (args[key].length < 16) throw new Error(`${key} is not a usable signature`);
}

const manifest = {
  channel: args.channel,
  version: args.version,
  notes: args.notes,
  pub_date: args["pub-date"],
  minimum_supported_version: args["min-version"],
  platforms: {
    "windows-x86_64": { url: args["windows-url"], signature: args["windows-signature"] },
    "linux-x86_64": { url: args["linux-url"], signature: args["linux-signature"] }
  }
};

fs.writeFileSync(args.out, JSON.stringify(manifest, null, 2) + "\n", "utf8");
console.log(`Ghost FTP update manifest written to ${args.out}`);
