/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for @nerve/i18n that speaks the locales' own texts, for the tests that read what a page says in a
// language: a test file mocks that module with this one,
// vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-locales")), and picks the language by setLanguage. A text's
// placeholders ({name}) take the values it was given, as ICU fills them, which the real @nerve/i18n does not do under
// vitest (it leaves "Remove {name}" as it is); a key no namespace of the language has is its own text.

type Language = "en" | "zh-CN";

const files = import.meta.glob<Record<string, unknown>>("../../../../packages/i18n/src/locales/*/*.json", {
  eager: true,
  import: "default",
});

/** Each language's namespaces, all of which a key is looked up in, as the real one searches them (its fallbackNS). */
const namespaces = (language: Language) =>
  Object.entries(files)
    .filter(([path]) => path.split("/").at(-2) === language)
    .map(([, tree]) => tree);

let language: Language = "en";

/** Sets the language of the texts that useTranslation gives from now on. */
export function setLanguage(next: Language) {
  language = next;
}

/** The text of a key's path in a namespace, if it has one. */
const textOf = (tree: Record<string, unknown>, key: string) => {
  const value = key
    .split(".")
    .reduce<unknown>((node, part) => (node && typeof node === "object" ? Reflect.get(node, part) : undefined), tree);
  return typeof value === "string" ? value : undefined;
};

export function useTranslation() {
  return {
    t: (key: string, values: Record<string, unknown> = {}) => {
      const text = namespaces(language)
        .map((tree) => textOf(tree, key))
        .find((found) => found !== undefined);
      if (text === undefined) return key;
      return text.replace(/\{(\w+)\}/g, (placeholder, name: string) =>
        name in values ? String(values[name]) : placeholder
      );
    },
  };
}
