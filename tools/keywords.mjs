// Keyword guard (docs/v0/M1-frontend-trim/M1-design.md 7.4): fails when a file hits a rule in
// tools/keywords.json that no exception covers, when an exception no longer matches anything, when
// an exception's hits do not number exactly its "count" (default 1), or when an exception has expired
// (its "until" is at or before the file's top-level "phase").
// The files are those git lists (tracked, plus untracked ones that are not ignored) as they are in the
// working tree. A path rule tests each path; a content rule tests the text of the files its `files`
// pattern selects (binary files are skipped). Patterns are JavaScript regular expressions with explicit
// flags, and every rule is first checked against its own samples. A problem with the rules file, a
// pattern, git or reading a file exits with 2, so an error is never taken for "no hits".
// usage: node tools/keywords.mjs   (from the repository root)
import { spawnSync } from "node:child_process";
import fs from "node:fs";

const RULES_FILE = "tools/keywords.json";

function fail(message) {
  console.error(`keywords: ${message}`);
  process.exit(2);
}

const PHASE_RE = /^M(\d+)(?:\/P(\d+))?$/;

// Parses "M3" or "M1/P4" into a [milestone, phase] pair, the phase number 0 when absent, so two phase
// strings compare lexicographically on that pair.
function parsePhase(spec) {
  const match = PHASE_RE.exec(spec);
  return [Number(match[1]), match[2] ? Number(match[2]) : 0];
}

function comparePhase(a, b) {
  return a[0] - b[0] || a[1] - b[1];
}

function compile(rule, key) {
  const spec = rule[key];
  if (typeof spec?.source !== "string" || typeof spec.flags !== "string") {
    fail(`rule ${rule.id}: "${key}" needs a "source" string and a "flags" string`);
  }
  if (!/^(?!.*(.).*\1)[imsu]*$/.test(spec.flags)) fail(`rule ${rule.id}: the flags of "${key}" may only be i, m, s, u`);
  try {
    return new RegExp(spec.source, spec.flags);
  } catch (error) {
    return fail(`rule ${rule.id}: "${key}" is not a valid regular expression: ${error.message}`);
  }
}

function loadRules() {
  let config;
  try {
    config = JSON.parse(fs.readFileSync(RULES_FILE, "utf8"));
  } catch (error) {
    fail(`cannot read ${RULES_FILE}: ${error.message}`);
  }
  if (!Array.isArray(config.rules) || !Array.isArray(config.exceptions)) {
    fail(`${RULES_FILE} needs "rules" and "exceptions" arrays`);
  }
  if (typeof config.phase !== "string" || !PHASE_RE.test(config.phase)) {
    fail(`${RULES_FILE} needs a top-level "phase" such as "M3" or "M1/P4"`);
  }
  const ids = new Set();
  const rules = config.rules.map((rule) => {
    if (typeof rule.id !== "string" || !/^[a-z0-9-]+$/.test(rule.id) || ids.has(rule.id)) {
      fail(`every rule needs a unique id of lowercase letters, digits and dashes (got ${JSON.stringify(rule.id)})`);
    }
    ids.add(rule.id);
    if (typeof rule.phase !== "string" || typeof rule.why !== "string" || rule.why === "") {
      fail(`rule ${rule.id} needs "phase" and "why"`);
    }
    const isPath = "path" in rule;
    if (isPath === ("files" in rule || "content" in rule)) {
      fail(`rule ${rule.id} needs either "path", or "files" and "content"`);
    }
    const compiled = isPath
      ? { id: rule.id, path: compile(rule, "path") }
      : { id: rule.id, files: compile(rule, "files"), content: compile(rule, "content") };
    const test = isPath ? compiled.path : compiled.content;
    const { hit, miss, files } = rule.samples ?? {};
    if (!Array.isArray(hit) || hit.length === 0 || !Array.isArray(miss) || miss.length === 0) {
      fail(`rule ${rule.id} needs "samples" with at least one "hit" and one "miss"`);
    }
    for (const sample of hit) {
      if (!test.test(sample)) fail(`rule ${rule.id} does not match its hit sample ${JSON.stringify(sample)}`);
    }
    for (const sample of miss) {
      if (test.test(sample)) fail(`rule ${rule.id} matches its miss sample ${JSON.stringify(sample)}`);
    }
    // A content rule's "files" selects what it even reads, so it needs its own hit/miss samples too;
    // otherwise a "files" narrowed down to nothing would still pass every content sample.
    if (!isPath) {
      const { hit: filesHit, miss: filesMiss } = files ?? {};
      if (!Array.isArray(filesHit) || filesHit.length === 0 || !Array.isArray(filesMiss) || filesMiss.length === 0) {
        fail(`rule ${rule.id} needs "samples.files" with at least one "hit" and one "miss"`);
      }
      for (const sample of filesHit) {
        if (!compiled.files.test(sample)) {
          fail(`rule ${rule.id} does not match its files.hit sample ${JSON.stringify(sample)}`);
        }
      }
      for (const sample of filesMiss) {
        if (compiled.files.test(sample)) {
          fail(`rule ${rule.id} matches its files.miss sample ${JSON.stringify(sample)}`);
        }
      }
    }
    return compiled;
  });
  const seenExceptions = new Set();
  for (const e of config.exceptions) {
    if (!ids.has(e.rule) || typeof e.path !== "string" || typeof e.match !== "string") {
      fail(`exception ${JSON.stringify(e)} needs an existing "rule", a "path" and a "match"`);
    }
    if (typeof e.reason !== "string" || e.reason === "" || !PHASE_RE.test(e.until ?? "")) {
      fail(`exception ${JSON.stringify(e)} needs a "reason" and an "until" such as "M3" or "M1/P4"`);
    }
    if (e.count !== undefined && (!Number.isInteger(e.count) || e.count < 1)) {
      fail(`exception ${JSON.stringify(e)} needs "count" to be an integer of at least 1`);
    }
    const exceptionKey = `${e.rule}\u0000${e.path}\u0000${e.match}`;
    if (seenExceptions.has(exceptionKey)) {
      fail(`exception ${JSON.stringify(e)} has the same "rule", "path" and "match" as an earlier exception`);
    }
    seenExceptions.add(exceptionKey);
  }
  return { rules, exceptions: config.exceptions, phase: config.phase };
}

// The text of a file; undefined for a binary file (a NUL byte in the first 8000 bytes, as git decides)
// or when the text is not needed; null for a file deleted in the working tree. git stores a symbolic
// link as the path it points to.
function read(path, needText) {
  try {
    const link = fs.lstatSync(path).isSymbolicLink();
    if (!needText) return undefined;
    const data = link ? Buffer.from(fs.readlinkSync(path)) : fs.readFileSync(path);
    return data.subarray(0, 8000).includes(0) ? undefined : data.toString("utf8");
  } catch (error) {
    if (error.code === "ENOENT") return null;
    return fail(`cannot read ${path}: ${error.message}`);
  }
}

const { rules, exceptions, phase } = loadRules();
const git = spawnSync("git", ["ls-files", "-z", "--cached", "--others", "--exclude-standard"], {
  encoding: "utf8",
  maxBuffer: 256 * 1024 * 1024,
});
if (git.error || git.status !== 0) fail(`git ls-files failed: ${git.error?.message ?? git.stderr.trim()}`);

const hits = [];
for (const path of new Set(git.stdout.split("\0").filter(Boolean))) {
  const contentRules = rules.filter((rule) => rule.files?.test(path));
  const text = read(path, contentRules.length > 0);
  if (text === null) continue;
  for (const rule of rules) if (rule.path?.test(path)) hits.push({ rule: rule.id, path, line: 0, match: path });
  if (text === undefined) continue;
  for (const rule of contentRules) {
    for (const found of text.matchAll(new RegExp(rule.content.source, `${rule.content.flags}g`))) {
      if (found[0] === "") fail(`rule ${rule.id} matches an empty string in ${path}`);
      hits.push({ rule: rule.id, path, line: text.slice(0, found.index).split("\n").length, match: found[0] });
    }
  }
}

const count = (n, noun) => `${n} ${noun}${n === 1 ? "" : "s"}`;
// An exception's key is its (rule, path, match) triple; several hits can share one key (the same match
// found more than once), so hits are grouped by key first and each exception then looks up its own
// group's size. Exceptions themselves cannot share a key: loadRules() rejects duplicates.
const keyOf = (o) => `${o.rule}\u0000${o.path}\u0000${o.match}`;
const hitsByKey = new Map();
for (const hit of hits) {
  const group = hitsByKey.get(keyOf(hit));
  if (group) group.push(hit);
  else hitsByKey.set(keyOf(hit), [hit]);
}

const phasePair = parsePhase(phase);
const covered = new Set();
const stale = [];
const mismatched = [];
const expired = [];
for (const e of exceptions) {
  const hitsForKey = hitsByKey.get(keyOf(e));
  const n = hitsForKey?.length ?? 0;
  if (n === 0) {
    stale.push(e);
    continue;
  }
  covered.add(keyOf(e));
  const m = e.count ?? 1;
  if (n !== m) mismatched.push({ exception: e, n, m, hits: hitsForKey });
  if (comparePhase(parsePhase(e.until), phasePair) <= 0) expired.push(e);
}
const open = hits.filter((hit) => !covered.has(keyOf(hit)));

for (const hit of open) {
  console.error(`${hit.rule}  ${hit.path}${hit.line ? `:${hit.line}` : ""}  ${JSON.stringify(hit.match)}`);
}
for (const { exception: e, n, m, hits: hs } of mismatched) {
  console.error(
    `exception count: ${e.rule}  ${e.path}  ${JSON.stringify(e.match)} covers ${n} hits, "count" says ${m}`
  );
  for (const hit of hs) {
    console.error(`  ${hit.path}${hit.line ? `:${hit.line}` : ""}  ${JSON.stringify(hit.match)}`);
  }
}
for (const e of stale) {
  console.error(`stale exception: ${e.rule}  ${e.path}  ${JSON.stringify(e.match)} matches nothing now; delete it`);
}
for (const e of expired) {
  console.error(
    `expired exception: ${e.rule}  ${e.path}  ${JSON.stringify(e.match)} is until ${JSON.stringify(e.until)}, at or before phase ${JSON.stringify(phase)}; delete it`
  );
}
if (open.length > 0 || stale.length > 0 || mismatched.length > 0 || expired.length > 0) {
  console.error(
    `keywords: ${count(open.length, "hit")} without an exception, ${count(stale.length, "stale exception")}, ${count(mismatched.length, "mismatched exception")}, ${count(expired.length, "expired exception")} (${RULES_FILE}).`
  );
  process.exit(1);
}
console.log(`keywords: ${count(rules.length, "rule")}, ${count(exceptions.length, "exception")}, no hits.`);
