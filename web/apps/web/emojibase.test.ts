/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { EMOJIBASE_URL } from "@nerve/propel/emoji-icon-picker";
import { emojibaseFiles, emojibaseUrl } from "./emojibase";

// Where the emoji picker reads its data, and what the build writes there (M3 design 7.7, 9.5): nerve serves
// emojibase-data's files itself, at the version the web app pins, and the picker reads them there, not from a CDN.

describe("the emoji picker's data", () => {
  it("is read from nerve where the build writes it, at the version pinned", () => {
    expect([EMOJIBASE_URL, emojibaseUrl]).toEqual(["/assets/emojibase/15.3.2", "/assets/emojibase/15.3.2"]);
  });

  it("is emojibase-data's en emoji and their groups' names, which frimousse reads", () => {
    const files = emojibaseFiles().map(({ fileName, source }) => ({ fileName, read: JSON.parse(source) }));
    expect(files.map(({ fileName }) => fileName)).toEqual([
      "assets/emojibase/15.3.2/en/data.json",
      "assets/emojibase/15.3.2/en/messages.json",
    ]);
    const [data, messages] = files.map(({ read }) => read);
    expect(data).toEqual(expect.arrayContaining([expect.objectContaining({ emoji: "🚀", label: "rocket" })]));
    expect(messages).toEqual(expect.objectContaining({ groups: expect.any(Array), skinTones: expect.any(Array) }));
  });
});
