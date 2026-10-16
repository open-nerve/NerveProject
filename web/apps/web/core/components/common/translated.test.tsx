/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { Translated } from "./translated";

// A sentence whose placeholders are nodes (M3 closeout): the stand-in for @nerve/i18n fills a key's placeholders as
// ICU does, from two sentences that order them differently.

const sentences: Record<string, string> = {
  remove: "Remove {name} from {workspace}?",
  reversed: "从 {workspace} 移出 {name}？",
};
vi.mock("@nerve/i18n", () => ({
  useTranslation: () => ({
    t: (key: string, params: Record<string, string>) =>
      sentences[key].replace(/\{(\w+)\}/g, (_, name: string) => params[name]),
  }),
}));

const values = { name: <b>Bob</b>, workspace: <i>Acme</i> };

describe("Translated", () => {
  it("puts each placeholder's node where the sentence has it", () => {
    expect(renderToStaticMarkup(<Translated i18nKey="remove" values={values} />)).toBe(
      "Remove <b>Bob</b> from <i>Acme</i>?"
    );
  });

  it("follows the order of the page's language", () => {
    expect(renderToStaticMarkup(<Translated i18nKey="reversed" values={values} />)).toBe(
      "从 <i>Acme</i> 移出 <b>Bob</b>？"
    );
  });
});
