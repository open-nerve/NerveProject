/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The command palette's filter, both hosts' (P7c review re-77 M1): the search finds an item by the text it gives, case
// aside: its keywords when it gives them, its value then never searched (only naming the item to cmdk, which keeps a
// list's selection by it), its value otherwise; the no-results row always shows.
import { expect, it } from "vitest";
import { powerKFilter } from "./filter";

const rows: [string, string, string, string[] | undefined, number][] = [
  ["finds an item by its value, case aside", "Open Sprint", "sprint", undefined, 1],
  ["finds an item by its value alone", "Open Sprint", "next", undefined, 0],
  ["finds an item by its value, no keywords given", "Open Sprint", "spr", [], 1],
  ["finds an item by its keywords, case aside", "c-1", "SPR", ["Sprint"], 1],
  ["finds an item by its keywords, never by its value", "c-1", "c-1", ["Sprint"], 0],
  ["always finds the no-results row", "no-results", "anything", undefined, 1],
];

it.each(rows)("the search %s", (_, value, search, keywords, found) => {
  expect(powerKFilter(value, search, keywords)).toBe(found);
});
