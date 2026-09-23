/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { describe, expect, it } from "vitest";
// local imports
import type { TExtensions } from "@/types";
import { getSlashCommandFilteredSections } from "./command-items-list";

// The slash menu's general section offers an image right after the code block unless the editor disables images.
// The image used to be pushed in after "code" through a list of "additional options" (M1/P3 writes it in place).
const generalKeys = (disabledExtensions: TExtensions[]) =>
  getSlashCommandFilteredSections({ disabledExtensions })({ query: "" })
    .find((section) => section.key === "general")
    ?.items.map((item) => item.key);

describe("the slash command list", () => {
  it("offers an image right after the code block", () => {
    const keys = generalKeys([]) ?? [];
    expect(keys[keys.indexOf("code") + 1]).toBe("image");
  });

  it("offers no image when the editor disables images", () => {
    expect(generalKeys(["image"])).not.toContain("image");
  });
});
