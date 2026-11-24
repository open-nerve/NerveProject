/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { projectOf } from "@/store/project/fake-projects";
import { viewOf } from "@/store/views/fake-views";
import { FIRST_CHOICES, type ViewListChoices, viewsShown } from "./view-list-order";

// The views a project's views list shows (P6a spec 2.3): those of its views whose names hold the search, whatever its
// case, and that pass the filters, by their owners and the dates of their creation, in the order chosen; views the
// order does not tell apart keep nerve's order. Today is 2026-11-24.

const web = projectOf("WEB", "w-acme");
// nerve's order: the newest first
const bugs = viewOf(web, "Bugs", "u-ann", { created_at: "2026-11-20T09:00:00Z", updated_at: "2026-11-21T09:00:00Z" });
const alpha = viewOf(web, "alpha", "u-bob", { created_at: "2026-11-10T09:00:00Z", updated_at: "2026-11-23T09:00:00Z" });
const old = viewOf(web, "Old bugs", "u-ann", {
  created_at: "2026-10-01T09:00:00Z",
  updated_at: "2026-10-01T09:00:00Z",
});
const listed = [bugs, alpha, old];

const shown = (choices: Partial<ViewListChoices>) =>
  viewsShown(listed, { ...FIRST_CHOICES, ...choices }).map((view: View) => view.name);

beforeEach(() => {
  vi.useFakeTimers({ now: new Date("2026-11-24T12:00:00Z") });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("viewsShown", () => {
  it("shows every view as the page opens, the last changed first", () => {
    expect(shown({})).toEqual(["alpha", "Bugs", "Old bugs"]);
  });

  it("shows the views whose names hold the search, whatever its case, its spaces around aside", () => {
    expect(shown({ search: "BUG" })).toEqual(["Bugs", "Old bugs"]);
    expect(shown({ search: " old " })).toEqual(["Old bugs"]);
    expect(shown({ search: "release" })).toEqual([]);
  });

  it("shows the views of the owners filtered by, and those created as each date filter says", () => {
    expect(shown({ filters: { owned_by: ["u-bob"], created_at: [] } })).toEqual(["alpha"]);
    expect(shown({ filters: { owned_by: ["u-bob", "u-ann"], created_at: [] } })).toEqual(["alpha", "Bugs", "Old bugs"]);
    // created over a week ago; and, too, over a month ago
    expect(shown({ filters: { owned_by: [], created_at: ["1_weeks;before;fromnow"] } })).toEqual(["alpha", "Old bugs"]);
    expect(
      shown({ filters: { owned_by: [], created_at: ["1_weeks;before;fromnow", "1_months;before;fromnow"] } })
    ).toEqual(["Old bugs"]);
    expect(shown({ filters: { owned_by: ["u-bob"], created_at: ["1_months;before;fromnow"] } })).toEqual([]);
  });

  it.each([
    { sortKey: "name", sortBy: "asc", names: ["alpha", "Bugs", "Old bugs"] },
    { sortKey: "name", sortBy: "desc", names: ["Old bugs", "Bugs", "alpha"] },
    { sortKey: "created_at", sortBy: "asc", names: ["Old bugs", "alpha", "Bugs"] },
    { sortKey: "created_at", sortBy: "desc", names: ["Bugs", "alpha", "Old bugs"] },
    { sortKey: "updated_at", sortBy: "asc", names: ["Old bugs", "Bugs", "alpha"] },
  ] as const)("orders the views by $sortKey, $sortBy", ({ sortKey, sortBy, names }) => {
    expect(shown({ sortKey, sortBy })).toEqual(names);
  });

  it("keeps nerve's order of the views the order does not tell apart", () => {
    const same = { updated_at: "2026-11-01T09:00:00Z" };
    const views = [viewOf(web, "B", "u-ann", same), viewOf(web, "A", "u-ann", same), viewOf(web, "C", "u-ann", same)];
    expect(viewsShown(views, FIRST_CHOICES).map(({ name }) => name)).toEqual(["B", "A", "C"]);
  });
});
