/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { Favorite, FavoriteUpdate } from "@nerve/api-client";
import { favoriteFolder, favoriteOfTarget } from "@/store/favorites/fake-favorites";
import type { FavoriteDrop } from "@/store/favorites/favorite-place";
import { moveOf, placeAmong } from "@/store/favorites/favorite-place";

// Where a dragged favorite goes (M7/P7 spec 2.1, 2.4, R4; D40): its place among the siblings where it is dropped, in
// their order, the greatest sequence first: above a row, between it and the one above, or 10000 past it at the top;
// below, between it and the one below, or 10000 short of it at the bottom; and the one PATCH a drop sends, with only
// what changes, or none, decided on the list as nerve answered it last.

/** Siblings as the sidebar shows them, by id and sequence. */
const rows = (...sequences: number[]) => sequences.map((sequence, i) => ({ id: `r${i}`, sequence }));

describe("placeAmong", () => {
  it.each([
    // D40: A 30000, B 29000, C 20000; C, dragged below A, goes between A and B (the old 20000 left it after B)
    ["below a row, between it and the one below", rows(30000, 29000), "r0", "below", 29500],
    ["above a row, between it and the one above", rows(30000, 29000), "r1", "above", 29500],
    ["above the first, 10000 past it", rows(30000, 29000), "r0", "above", 40000],
    ["below the last, 10000 short of it", rows(30000, 29000), "r1", "below", 19000],
    ["below a row whose neighbour is 0, between them", rows(10000, 0, -10000), "r0", "below", 5000],
    ["above a row of 0, between it and the one above", rows(10000, 0, -10000), "r1", "above", 5000],
    ["below the last at 0, 10000 short of it", rows(10000, 0), "r1", "below", -10000],
    ["between negatives", rows(-5000, -15000), "r0", "below", -10000],
    ["above the first at 0, 10000 past it", rows(0, -10000), "r0", "above", 10000],
    ["between two of the same sequence, their sequence (the id orders them then)", rows(100, 100), "r0", "below", 100],
    ["the only one, above", rows(500), "r0", "above", 10500],
    ["the only one, below", rows(500), "r0", "below", -9500],
  ] as const)("places %s", (_, siblings, droppedOn, at, place) => {
    expect(placeAmong(siblings, droppedOn, at)).toBe(place);
  });

  it("places nothing by a row that is not among the siblings", () => {
    expect(placeAmong(rows(30000), "gone", "above")).toBeUndefined();
  });
});

const a = favoriteOfTarget("project", "p-a", "p-a", { sequence: 30000 });
const b = favoriteOfTarget("project", "p-b", "p-b", { sequence: 29000 });
const c = favoriteOfTarget("project", "p-c", "p-c", { sequence: 20000 });
const later = favoriteFolder("Later", { sequence: 10000 });
const done = favoriteFolder("Done", { sequence: 5000 });
const x = favoriteOfTarget("cycle", "c-x", "p-a", { parent_id: later.id, sequence: 8000 });
const y = favoriteOfTarget("cycle", "c-y", "p-a", { parent_id: later.id, sequence: 6000 });
/** acme's favorites: A, B, C, the folder Later holding X and Y, the folder Done holding nothing. */
const list: Favorite[] = [a, b, c, later, x, y, done];

describe("moveOf", () => {
  it.each<[string, Favorite, FavoriteDrop, FavoriteUpdate]>([
    // D40's case, with only the sequence: the folder is the same
    ["C below A: between A and B", c, { at: "below", rowId: a.id }, { sequence: 29500 }],
    ["A below the folder Done: last, 10000 short of it", a, { at: "below", rowId: done.id }, { sequence: -5000 }],
    ["C above A: first, 10000 past A", c, { at: "above", rowId: a.id }, { sequence: 40000 }],
    ["X out of Later, below A", x, { at: "below", rowId: a.id }, { parent_id: null, sequence: 29500 }],
    ["B into Later, above Y", b, { at: "above", rowId: y.id }, { parent_id: later.id, sequence: 7000 }],
    ["B into Later, below Y", b, { at: "below", rowId: y.id }, { parent_id: later.id, sequence: -4000 }],
    ["B into Later on its row: its sequence kept", b, { at: "into", folderId: later.id }, { parent_id: later.id }],
    ["X into Done: its sequence kept", x, { at: "into", folderId: done.id }, { parent_id: done.id }],
    ["X out on the section's header: its sequence kept", x, { at: "out" }, { parent_id: null }],
    ["the folder Done above A", done, { at: "above", rowId: a.id }, { sequence: 40000 }],
    ["Y above X, its neighbour in Later", y, { at: "above", rowId: x.id }, { sequence: 18000 }],
  ])("moves %s", (_, dragged, drop, body) => {
    expect(moveOf(list, dragged.id, drop)).toEqual(body);
  });

  it.each<[string, Favorite, FavoriteDrop]>([
    // the same place, by position: C right below B, B right above C; X already in Later; A already at the top
    ["C below B, right below it already", c, { at: "below", rowId: b.id }],
    ["B above C, right above it already", b, { at: "above", rowId: c.id }],
    ["X above Y, right above it already", x, { at: "above", rowId: y.id }],
    ["A onto itself", a, { at: "below", rowId: a.id }],
    ["X into Later, which holds it", x, { at: "into", folderId: later.id }],
    ["A out on the header, at the top already", a, { at: "out" }],
    ["the folder Done out on the header", done, { at: "out" }],
    // a folder holds no folder: not on its row, nor among its favorites
    ["the folder Done into Later", done, { at: "into", folderId: later.id }],
    ["the folder Done below X, in Later", done, { at: "below", rowId: x.id }],
    ["the folder Done above Y, in Later", done, { at: "above", rowId: y.id }],
    // a drop onto what is no folder moves nothing into it
    ["B into A, no folder", b, { at: "into", folderId: a.id }],
  ])("moves nothing: %s", (_, dragged, drop) => {
    expect(moveOf(list, dragged.id, drop)).toBeUndefined();
  });

  it("moves nothing gone from the list: the dragged one, the row, or the folder it is dropped on", () => {
    expect(moveOf([a, b, later], c.id, { at: "below", rowId: a.id })).toBeUndefined();
    expect(moveOf([a, c, later], c.id, { at: "below", rowId: b.id })).toBeUndefined();
    expect(moveOf([a, c, later], c.id, { at: "into", folderId: done.id })).toBeUndefined();
  });

  it("decides the same place by position, not by the sequence a drop would reckon", () => {
    // C right below B: dropped below B again, between B and Later (C left out) reckons 19500, not its 20000
    expect(moveOf(list, c.id, { at: "below", rowId: b.id })).toBeUndefined();
    // B right above C: dropped above C, between A and C (B left out) reckons 25000, not its 29000
    expect(moveOf(list, b.id, { at: "above", rowId: c.id })).toBeUndefined();
    // B dropped below C is a move: between C and Later
    expect(moveOf(list, b.id, { at: "below", rowId: c.id })).toEqual({ sequence: 15000 });
  });
});
