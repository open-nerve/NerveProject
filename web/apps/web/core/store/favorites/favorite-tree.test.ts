/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { Favorite } from "@nerve/api-client";
import { favoriteFolder, favoriteOfTarget } from "@/store/favorites/fake-favorites";
import { favoriteTree, siblingsOf } from "@/store/favorites/favorite-tree";

// The sidebar's tree of one's favorites (M7/P7 spec 2.1): the top, and each folder's favorites under it, each in the
// order nerve lists them, the greatest sequence first, then by id, both descending, whatever the list's order.

const web = favoriteOfTarget("project", "p-web", "p-web", { sequence: 50000 });
const ops = favoriteOfTarget("project", "p-ops", "p-ops", { sequence: 50000 });
const sprint = favoriteOfTarget("cycle", "c-sprint", "p-web", { sequence: 20000 });
const later = favoriteFolder("Later", { sequence: 40000 });
const auth = favoriteOfTarget("module", "m-auth", "p-web", { parent_id: later.id, sequence: 10000 });
const bugs = favoriteOfTarget("view", "v-bugs", "p-web", { parent_id: later.id, sequence: 30000 });
const triage = favoriteOfTarget("view", "v-triage", null, { parent_id: later.id, sequence: 30000 });
/** One of a folder the list does not hold (deleted, its row left): nerve never lists it. */
const orphan = favoriteOfTarget("cycle", "c-orphan", "p-web", { parent_id: "f-folder-gone", sequence: 90000 });

/** The tree's ids: each at the top, with its folder's favorites after it, as [id, [ids]]. */
const idsOf = (list: Favorite[]) =>
  favoriteTree(list).map(({ favorite, children }) => [favorite.id, children.map((child) => child.id)]);

describe("favoriteTree", () => {
  it("lists the top by sequence, the greatest first, then by id, both descending, whatever the list's order", () => {
    const expected = [
      [web.id, []],
      [ops.id, []],
      [later.id, [triage.id, bugs.id, auth.id]],
      [sprint.id, []],
    ];
    expect(idsOf([sprint, ops, later, web, auth, bugs, triage])).toEqual(expected);
    expect(idsOf([triage, auth, web, sprint, bugs, later, ops])).toEqual(expected);
  });

  it("puts a folder's favorites under it, in their order, and none of them at the top", () => {
    const [, folder] = favoriteTree([web, later, auth, bugs]);
    expect(folder?.favorite).toBe(later);
    expect(folder?.children).toEqual([bugs, auth]);
    expect(favoriteTree([web, later, auth, bugs]).map((branch) => branch.favorite)).toEqual([web, later]);
    // a folder holding nothing has no favorites under it
    expect(favoriteTree([later])).toEqual([{ favorite: later, children: [] }]);
  });

  it("shows no favorite whose folder the list does not hold, at the top or under another", () => {
    expect(idsOf([web, orphan, later, auth])).toEqual([
      [web.id, []],
      [later.id, [auth.id]],
    ]);
  });

  it("gives the favorites of the top, or of a folder, in their order, its siblings where one is dropped", () => {
    const list = [sprint, auth, web, triage, later, bugs, ops];
    expect(siblingsOf(list, null)).toEqual([web, ops, later, sprint]);
    expect(siblingsOf(list, later.id)).toEqual([triage, bugs, auth]);
  });
});
