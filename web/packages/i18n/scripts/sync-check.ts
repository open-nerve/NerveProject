/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Usage: tsx scripts/sync-check.ts
// Fails unless src/locales has both en and zh-CN, they have the same namespace files with the same keys,
// and in each of them no key is defined in two namespace files or is also the prefix of another key. All
// namespaces share one key space (src/core/instance.ts makes every namespace a fallback), so such a key
// would be ambiguous. Also fails unless each locale's namespace files match NAMESPACES exactly, and
// unless each locale has at least one namespace file (two empty locales would otherwise agree trivially).

import fs from "node:fs";
import path from "node:path";
import { NAMESPACES } from "../src/constants/namespaces";

const LOCALES_DIR = path.resolve(import.meta.dirname, "../src/locales");
const SOURCE = "en";
const TARGET = "zh-CN";

/** Recursively flatten an object into dot-notation keys. */
function flattenKeys(obj: Record<string, unknown>, prefix = ""): string[] {
  return Object.entries(obj).flatMap(([key, value]) => {
    const full = prefix ? `${prefix}.${key}` : key;
    return value !== null && typeof value === "object" && !Array.isArray(value)
      ? flattenKeys(value as Record<string, unknown>, full)
      : [full];
  });
}

/** Every namespace file of a locale (namespace -> its keys), or undefined when the locale directory is missing. */
function loadLocale(locale: string): Map<string, Set<string>> | undefined {
  const dir = path.join(LOCALES_DIR, locale);
  if (!fs.existsSync(dir)) return undefined;
  const namespaces = new Map<string, Set<string>>();
  for (const file of fs.readdirSync(dir).filter((f) => f.endsWith(".json"))) {
    const data = JSON.parse(fs.readFileSync(path.join(dir, file), "utf-8")) as Record<string, unknown>;
    namespaces.set(path.basename(file, ".json"), new Set(flattenKeys(data)));
  }
  return namespaces;
}

/** Keys of one locale that are defined in two namespace files, or that are also the prefix of another key. */
function findConflicts(locale: string, namespaces: Map<string, Set<string>>): string[] {
  const files = new Map<string, string[]>();
  for (const [namespace, keys] of namespaces) {
    for (const key of keys) files.set(key, [...(files.get(key) ?? []), `${namespace}.json`]);
  }
  const conflicts = new Set<string>();
  for (const [key, where] of files) {
    if (where.length > 1) conflicts.add(`${locale}: ${key} is defined in ${where.join(" and ")}`);
    const parts = key.split(".");
    for (let i = 1; i < parts.length; i++) {
      const prefix = parts.slice(0, i).join(".");
      if (files.has(prefix)) conflicts.add(`${locale}: ${prefix} is a key and also the prefix of other keys`);
    }
  }
  return [...conflicts];
}

/** Namespace files present that NAMESPACES does not list, and namespaces NAMESPACES lists with no file. */
function findNamespaceMismatches(locale: string, namespaces: Map<string, Set<string>>): string[] {
  const onDisk = new Set(namespaces.keys());
  const registered = new Set<string>(NAMESPACES);
  const problems: string[] = [];
  for (const namespace of onDisk) {
    if (!registered.has(namespace))
      problems.push(`${locale}: ${namespace}.json exists but is not registered in NAMESPACES`);
  }
  for (const namespace of registered) {
    if (!onDisk.has(namespace))
      problems.push(`${locale}: ${namespace} is registered in NAMESPACES but ${namespace}.json is missing`);
  }
  return problems;
}

const problems: string[] = [];
const [source, target] = [SOURCE, TARGET].map((locale) => {
  const namespaces = loadLocale(locale);
  if (namespaces) {
    if (namespaces.size === 0) problems.push(`src/locales/${locale} has no namespace files`);
    problems.push(...findConflicts(locale, namespaces));
    problems.push(...findNamespaceMismatches(locale, namespaces));
  } else {
    problems.push(`src/locales/${locale} is missing`);
  }
  return namespaces;
});
if (source && target) {
  for (const namespace of new Set([...source.keys(), ...target.keys()])) {
    const sourceKeys = source.get(namespace);
    const targetKeys = target.get(namespace);
    if (!sourceKeys || !targetKeys) {
      problems.push(`${namespace}.json exists only in ${sourceKeys ? SOURCE : TARGET}`);
      continue;
    }
    for (const key of sourceKeys) {
      if (!targetKeys.has(key)) problems.push(`${namespace}: ${key} is missing in ${TARGET}`);
    }
    for (const key of targetKeys) {
      if (!sourceKeys.has(key)) problems.push(`${namespace}: ${key} is missing in ${SOURCE}`);
    }
  }
}

if (problems.length > 0) {
  console.error(problems.toSorted().join("\n"));
  console.error(
    `${SOURCE} and ${TARGET} must have the same namespaces and keys, each key defined once: fix the above.`
  );
  process.exit(1);
}
console.log(`${SOURCE} and ${TARGET} have the same namespaces and keys, each key defined once.`);
