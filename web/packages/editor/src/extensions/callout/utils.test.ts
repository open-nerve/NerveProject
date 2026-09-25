/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
// local imports
import { ECalloutAttributeNames } from "./types";
import { DEFAULT_CALLOUT_BLOCK_ATTRIBUTES, getStoredLogo, updateStoredLogo } from "./utils";

// A new callout takes its logo from local storage, which holds whatever was put there. Only a value of the shape
// the logo selector writes is used; anything else is dropped and the defaults apply, as for text that is not
// JSON. A stored emoji that is not a string used to reach the emoji renderer, which splits it and throws.

const STORAGE_KEY = "editor-calloutComponent-logo";

const DEFAULT_LOGO = {
  [ECalloutAttributeNames.LOGO_IN_USE]: "emoji",
  [ECalloutAttributeNames.EMOJI_UNICODE]: DEFAULT_CALLOUT_BLOCK_ATTRIBUTES[ECalloutAttributeNames.EMOJI_UNICODE],
  [ECalloutAttributeNames.EMOJI_URL]: DEFAULT_CALLOUT_BLOCK_ATTRIBUTES[ECalloutAttributeNames.EMOJI_URL],
};

beforeEach(() => {
  localStorage.clear();
  vi.spyOn(console, "error").mockImplementation(() => undefined);
});

afterEach(() => {
  vi.restoreAllMocks();
});

const expectDropped = (stored: string) => {
  localStorage.setItem(STORAGE_KEY, stored);
  expect(getStoredLogo()).toEqual(DEFAULT_LOGO);
  expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
  expect(console.error).toHaveBeenCalledOnce();
};

describe("the stored callout logo", () => {
  it("gives the stored emoji, with the default image when none is stored", () => {
    updateStoredLogo({ in_use: "emoji", emoji: { value: "128512" } });
    expect(getStoredLogo()).toEqual({
      [ECalloutAttributeNames.LOGO_IN_USE]: "emoji",
      [ECalloutAttributeNames.EMOJI_UNICODE]: "128512",
      [ECalloutAttributeNames.EMOJI_URL]: DEFAULT_CALLOUT_BLOCK_ATTRIBUTES[ECalloutAttributeNames.EMOJI_URL],
    });
  });

  it("gives the stored icon", () => {
    updateStoredLogo({ in_use: "icon", icon: { name: "Star", color: "#6d7b8a" } });
    expect(getStoredLogo()).toEqual({
      [ECalloutAttributeNames.LOGO_IN_USE]: "icon",
      [ECalloutAttributeNames.ICON_NAME]: "Star",
      [ECalloutAttributeNames.ICON_COLOR]: "#6d7b8a",
    });
  });

  it("gives the defaults when nothing is stored", () => {
    expect(getStoredLogo()).toEqual(DEFAULT_LOGO);
    expect(console.error).not.toHaveBeenCalled();
  });

  it.each([
    '{"in_use":"emoji","emoji":{"value":128512}}',
    '{"in_use":"emoji","emoji":{"value":"128512","url":42}}',
    '{"in_use":"emoji","emoji":"128512"}',
    '{"in_use":"icon","icon":{"name":7}}',
    '{"in_use":"icon","icon":{"name":"Star","color":{}}}',
    '{"in_use":"sticker","emoji":{"value":"128512"}}',
  ])("drops a value whose fields have the wrong types: %s", expectDropped);

  it.each(["null", "42", "true", '"128512"'])("drops a value that is not an object: %s", expectDropped);

  it("drops text that is not JSON", () => {
    expectDropped("{not json");
  });
});
