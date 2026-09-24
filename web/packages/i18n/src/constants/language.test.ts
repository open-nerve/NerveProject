/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { SUPPORTED_LANGUAGES, toSupportedLanguage } from "./language";

describe("toSupportedLanguage", () => {
  it("supports exactly en and zh-CN", () => {
    expect(SUPPORTED_LANGUAGES.map((option) => option.value)).toEqual(["en", "zh-CN"]);
  });

  it.each(["en", "zh-CN"])("keeps %j", (language) => {
    expect(toSupportedLanguage(language)).toBe(language);
  });

  it.each(["fr", "zh-TW", "zh", "EN", "", null, undefined])("turns %j into en", (language) => {
    expect(toSupportedLanguage(language)).toBe("en");
  });
});
