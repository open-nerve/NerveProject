// Translation keys read (M3 closeout): fails when the web app reads an i18n key that the English locale does not
// define, which the page would show as the key itself. The reads it sees are what the code holds as literals:
// - a t("…") call's, a Translated's i18nKey="…" and a constant's i18n_…: "…": each a whole key (a key with keys under
//   it is an object there, not a text);
// - any other string of dotted words whose first word is a top-level key of the locale
//   ("project_settings.states.groups.backlog"), wherever the code holds it: a whole key, or a prefix the code completes.
//   nerve's problem codes ("<module>.<code>": the x-problem-codes of api/dist/openapi.yaml, read as
//   error-messages.test.ts reads them) are no keys, though a module may share a namespace's name ("issue"): such a
//   string is not a read;
// - a template's start before its first ${…} (`greeting.${period}`): a prefix with keys under it.
// A key built otherwise (t(variable), a template that starts with ${…}) is not seen. The sources: web/apps/web's app
// and core, the constants and types packages; tests and their fakes aside. The keys are en's: check:sync keeps
// zh-CN's the same. Like the keyword guard, the tool first checks its reading against its own samples (SAMPLES).
// usage: node tools/i18n-keys.mjs   (from anywhere)
import fs from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "..");
const LOCALE = "web/packages/i18n/src/locales/en";
const CONTRACT = "api/dist/openapi.yaml";
const SOURCES = ["web/apps/web/app", "web/apps/web/core", "web/packages/constants/src", "web/packages/types/src"];

// A small locale and contract, texts that read a key the locale does not define (hit) and texts that do not (miss):
// each way of reading has a sample that fails once the tool reads it otherwise.
const SAMPLES = {
  locale: { common: { save: "Save", toast: { done: "Done", failed: "Failed" } } },
  codes: ["common.not_found"],
  hit: [
    't("common.saved")',
    't("common.toast")',
    '<Translated i18nKey="common.toast" />',
    'i18n_title: "common.toast"',
    'const KEY = "common.toast.gone";',
    "t(`common.toasts.${kind}`)",
  ],
  miss: [
    't("common.save")',
    '<Translated i18nKey="common.toast.done" />',
    'i18n_title: "common.toast.failed"',
    'const GROUP = "common.toast";',
    "t(`common.toast.${kind}`)",
    '  "common.not_found": "errors.unknown",',
  ],
};

function fail(message) {
  console.error(`i18n-keys: ${message}`);
  process.exit(1);
}

/** A locale's keys (dotted), the prefixes with keys under them, and the patterns of its reads. */
function vocabulary(namespaces) {
  const keys = new Set();
  const flatten = (node, prefix) => {
    if (node !== null && typeof node === "object" && !Array.isArray(node)) {
      for (const [key, value] of Object.entries(node)) flatten(value, prefix ? `${prefix}.${key}` : key);
    } else {
      keys.add(prefix);
    }
  };
  for (const namespace of namespaces) flatten(namespace, "");
  const prefixes = new Set();
  for (const key of keys) {
    const parts = key.split(".");
    for (let i = 1; i < parts.length; i++) prefixes.add(parts.slice(0, i).join("."));
  }
  const tops = [...new Set([...keys].filter((key) => key.includes(".")).map((key) => key.split(".")[0]))].join("|");
  const KEY = "[a-z0-9_]+(?:\\.[a-z0-9_]+)*";
  return {
    keys,
    prefixes,
    whole: [
      new RegExp(`\\bt\\(\\s*"(${KEY})"`, "g"),
      new RegExp(`\\bi18nKey="(${KEY})"`, "g"),
      new RegExp(`\\bi18n_[a-z_]+:\\s*"(${KEY})"`, "g"),
    ],
    literal: new RegExp(`"((?:${tops})(?:\\.[a-z0-9_]+)+)"`, "g"),
    template: new RegExp(`\`((?:${tops})(?:\\.[a-z0-9_]+)*)\\.\\$\\{`, "g"),
  };
}

/** The keys (and template prefixes) a text reads, and those of them the vocabulary does not define. */
function reads(text, vocab, codes) {
  const read = new Set();
  const missing = new Set();
  const take = (pattern, defined, skipped = () => false) => {
    for (const [, key] of text.matchAll(pattern)) {
      if (skipped(key)) continue;
      read.add(key);
      if (!defined(key)) missing.add(key);
    }
  };
  for (const pattern of vocab.whole) take(pattern, (key) => vocab.keys.has(key));
  // a problem code is no key
  take(
    vocab.literal,
    (key) => vocab.keys.has(key) || vocab.prefixes.has(key),
    (key) => codes.has(key)
  );
  take(vocab.template, (prefix) => vocab.prefixes.has(prefix));
  return { read, missing };
}

/**
 * The items of every block list under an x-problem-codes: line of the contract, as error-messages.test.ts reads them
 * (its listsUnder; the web workspace's test cannot import this script, which runs as it loads): a change to one reading
 * is made in both.
 */
function problemCodes(lines) {
  const codes = new Set();
  lines.forEach((line, i) => {
    if (!/^\s*x-problem-codes:\s*$/.test(line)) return;
    const indent = line.length - line.trimStart().length;
    for (const next of lines.slice(i + 1)) {
      const item = /^(\s*)- (\S+)$/.exec(next);
      if (!item || item[1].length < indent) break;
      codes.add(item[2]);
    }
  });
  return codes;
}

const sampleVocab = vocabulary([SAMPLES.locale]);
const sampleCodes = new Set(SAMPLES.codes);
const wrong = [
  ...SAMPLES.hit.filter((text) => reads(text, sampleVocab, sampleCodes).missing.size === 0).map((t) => `hit ${t}`),
  ...SAMPLES.miss.filter((text) => reads(text, sampleVocab, sampleCodes).missing.size > 0).map((t) => `miss ${t}`),
];
if (wrong.length > 0) fail(`its reading fails its own samples:\n${wrong.join("\n")}`);

const vocab = vocabulary(
  fs
    .readdirSync(path.join(ROOT, LOCALE))
    .filter((name) => name.endsWith(".json"))
    .map((name) => JSON.parse(fs.readFileSync(path.join(ROOT, LOCALE, name), "utf8")))
);
const codes = problemCodes(fs.readFileSync(path.join(ROOT, CONTRACT), "utf8").split("\n"));
if (codes.size === 0) fail(`read no x-problem-codes from ${CONTRACT}`);

const read = new Set();
const undefinedKeys = [];
for (const source of SOURCES) {
  for (const name of fs.readdirSync(path.join(ROOT, source), { recursive: true })) {
    const base = path.basename(name);
    if (!/\.tsx?$/.test(base) || base.includes(".test.") || base.startsWith("fake-")) continue;
    const file = path.join(source, name);
    const found = reads(fs.readFileSync(path.join(ROOT, file), "utf8"), vocab, codes);
    for (const key of found.read) read.add(key);
    for (const key of found.missing) undefinedKeys.push(`${file}: ${key}`);
  }
}

if (undefinedKeys.length > 0) {
  console.error(undefinedKeys.toSorted().join("\n"));
  fail(`${undefinedKeys.length} keys read that ${LOCALE} does not define: define them, or read others.`);
}
console.log(`i18n-keys: ${read.size} keys read, each defined in ${LOCALE}.`);
