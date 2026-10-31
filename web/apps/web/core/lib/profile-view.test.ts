/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import { parseProfileView } from "./profile-view";

it.each(["assigned", "created", "subscribed"] as const)("preserves the explicit view %s", (value) => {
  expect(parseProfileView(value)).toBe(value);
});
it.each([undefined, "", "constructor", "toString", "__proto__", "unknown", "Assigned"])(
  "rejects non-view %s",
  (value) => {
    expect(parseProfileView(value)).toBeUndefined();
  }
);
