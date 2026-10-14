// Translation keys read (M3 closeout): fails when the web app reads an i18n key that the English locale does not
// define, which the page would show as the key itself. A key read is one the code holds as a literal: a t("…") call's,
// a Translated's i18nKey="…", a constant's i18n_…: "…", and any string of dotted words whose first word is a top-level
// key of the locale ("project_settings.states.groups.backlog"), wherever the code holds it. A key the locale defines
// only longer keys under (a prefix the code completes) counts as defined; a key the code builds from a template is not
// seen. The sources: web/apps/web's app and core, the constants and types packages; tests and their fakes aside. The
// keys are en's: check:sync keeps zh-CN's the same.
// usage: node tools/i18n-keys.mjs   (from anywhere)
import fs from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "..");
const LOCALE = "web/packages/i18n/src/locales/en";
const SOURCES = ["web/apps/web/app", "web/apps/web/core", "web/packages/constants/src", "web/packages/types/src"];

// every key of the locale's files, dotted
const keys = new Set();
function flatten(node, prefix) {
  if (node !== null && typeof node === "object" && !Array.isArray(node)) {
    for (const [key, value] of Object.entries(node)) flatten(value, prefix ? `${prefix}.${key}` : key);
  } else {
    keys.add(prefix);
  }
}
for (const file of fs.readdirSync(path.join(ROOT, LOCALE)).filter((name) => name.endsWith(".json"))) {
  flatten(JSON.parse(fs.readFileSync(path.join(ROOT, LOCALE, file), "utf8")), "");
}
const prefixes = new Set();
for (const key of keys) {
  const parts = key.split(".");
  for (let i = 1; i < parts.length; i++) prefixes.add(parts.slice(0, i).join("."));
}
const tops = [...new Set([...keys].filter((key) => key.includes(".")).map((key) => key.split(".")[0]))];

const KEY = "[a-z0-9_]+(?:\\.[a-z0-9_]+)*";
const READS = [
  new RegExp(`\\bt\\(\\s*"(${KEY})"`, "g"),
  new RegExp(`\\bi18nKey="(${KEY})"`, "g"),
  new RegExp(`\\bi18n_[a-z_]+:\\s*"(${KEY})"`, "g"),
  new RegExp(`"((?:${tops.join("|")})(?:\\.[a-z0-9_]+)+)"`, "g"),
];

const read = new Set();
const undefinedKeys = [];
for (const source of SOURCES) {
  for (const name of fs.readdirSync(path.join(ROOT, source), { recursive: true })) {
    const base = path.basename(name);
    if (!/\.tsx?$/.test(base) || base.includes(".test.") || base.startsWith("fake-")) continue;
    const file = path.join(source, name);
    const text = fs.readFileSync(path.join(ROOT, file), "utf8");
    const found = new Set(READS.flatMap((pattern) => [...text.matchAll(pattern)].map((match) => match[1])));
    for (const key of found) {
      read.add(key);
      if (!keys.has(key) && !prefixes.has(key)) undefinedKeys.push(`${file}: ${key}`);
    }
  }
}

if (undefinedKeys.length > 0) {
  console.error(undefinedKeys.toSorted().join("\n"));
  console.error(
    `i18n-keys: ${undefinedKeys.length} keys read that ${LOCALE} does not define: define them, or read others.`
  );
  process.exit(1);
}
console.log(`i18n-keys: ${read.size} keys read, each defined in ${LOCALE}.`);
