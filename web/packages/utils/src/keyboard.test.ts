/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { isComposing } from "./keyboard";

// A keydown an input method takes is the method's: the browsers give it in two ways.

describe("isComposing", () => {
  it.each([
    { browser: "Chromium's and Firefox's, while the method composes", event: { isComposing: true, keyCode: 27 } },
    { browser: "Safari's, after the composition's end", event: { isComposing: false, keyCode: 229 } },
  ])("takes $browser for the method's", ({ event }) => {
    expect(isComposing(event)).toBe(true);
  });

  it("leaves the page any other keydown", () => {
    expect([
      isComposing({ isComposing: false, keyCode: 27 }),
      isComposing({ isComposing: false, keyCode: 13 }),
    ]).toEqual([false, false]);
  });
});
