/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import data from "emojibase-data/en/data.json";
import shortcodes from "emojibase-data/en/shortcodes/emojibase.json";
import github from "emojibase-data/en/shortcodes/github.json";

export type EmojiItem = {
  name: string;
  emoji: string;
  shortcodes: string[];
  tags: string[];
  emoticons: string[];
};

// Keep the persisted names and aliases of TipTap's Unicode dataset. Importing its extension just for
// that data also constructs an unused Node and performs Canvas support detection. Use the same pinned
// data as Nerve's emoji picker (M3 §7.7), without an extension, images or capability checks.
const removeVariationSelector = (value: string) => value.replace("\uFE0E", "").replace("\uFE0F", "");

export const emojis: EmojiItem[] = data.map((item) => ({
  name: [github[item.hexcode] ?? shortcodes[item.hexcode]].flat()[0],
  emoji: removeVariationSelector(item.emoji),
  shortcodes: [shortcodes[item.hexcode] ?? []].flat(),
  tags: item.tags ?? [],
  emoticons: [item.emoticon ?? []].flat(),
}));

export function shortcodeToEmoji(shortcode: string, items: EmojiItem[]) {
  return items.find((item) => item.name === shortcode || item.shortcodes.includes(shortcode));
}

export function emojiToShortcode(emoji: string, items: EmojiItem[]) {
  return items.find((item) => item.emoji === removeVariationSelector(emoji))?.shortcodes[0];
}
