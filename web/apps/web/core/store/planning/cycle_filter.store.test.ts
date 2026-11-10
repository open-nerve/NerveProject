/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { reaction } from "mobx";
import { describe, expect, it } from "vitest";
import { CycleFilterStore } from "@/store/planning/cycle_filter.store";

// What each project's cycles list and archive are filtered and searched by (P5a spec 2.3, R4), kept by the project's
// id: every reader and change takes it from its caller (the route's). The store is built without the RootStore, so it
// follows no address; that a retired session's filters do not follow the page's RouterStore is store-context.test.ts.

const WEB = "p-web";
const OPS = "p-ops";

describe("CycleFilterStore", () => {
  it("keeps each project's filters by its id, those of its list and of its archive apart", () => {
    const store = new CycleFilterStore();
    store.updateFilters(WEB, "default", { status: ["current", "upcoming"] });
    store.updateFilters(WEB, "archived", { end_date: ["2026-11-01;before"] });
    store.updateFilters(OPS, "default", { status: ["draft"] });
    expect(store.getFilters(WEB, "default")).toEqual({ status: ["current", "upcoming"] });
    expect(store.getFilters(WEB, "archived")).toEqual({ end_date: ["2026-11-01;before"] });
    expect(store.getFilters(OPS, "default")).toEqual({ status: ["draft"] });
    // A list nothing filtered yet, of a project filtered or not.
    expect(store.getFilters(OPS, "archived")).toEqual({});
    expect(store.getFilters("p-lab", "default")).toEqual({});
  });

  it("changes the filters given and keeps the others; clearing a list's leaves the other list's and project's", () => {
    const store = new CycleFilterStore();
    store.updateFilters(WEB, "default", { status: ["current"] });
    store.updateFilters(WEB, "default", { start_date: ["1_weeks;after;fromnow"] });
    store.updateFilters(WEB, "default", { status: ["draft"] });
    expect(store.getFilters(WEB, "default")).toEqual({ status: ["draft"], start_date: ["1_weeks;after;fromnow"] });
    store.updateFilters(WEB, "archived", { status: ["completed"] });
    store.updateFilters(OPS, "default", { status: ["upcoming"] });
    store.clearAllFilters(WEB, "default");
    expect(store.getFilters(WEB, "default")).toEqual({});
    expect(store.getFilters(WEB, "archived")).toEqual({ status: ["completed"] });
    expect(store.getFilters(OPS, "default")).toEqual({ status: ["upcoming"] });
  });

  it("keeps each project's two searches by its id, apart from its filters", () => {
    const store = new CycleFilterStore();
    expect(store.getSearchQuery(WEB, "default")).toBe("");
    store.updateSearchQuery(WEB, "default", "sprint");
    store.updateSearchQuery(WEB, "archived", "old");
    store.updateSearchQuery(OPS, "default", "ops");
    expect(store.getSearchQuery(WEB, "default")).toBe("sprint");
    expect(store.getSearchQuery(WEB, "archived")).toBe("old");
    expect(store.getSearchQuery(OPS, "default")).toBe("ops");
    expect(store.getSearchQuery(OPS, "archived")).toBe("");
    // Clearing a list's filters keeps its search, and a search changes no filter.
    store.updateFilters(WEB, "default", { status: ["current"] });
    store.clearAllFilters(WEB, "default");
    expect(store.getSearchQuery(WEB, "default")).toBe("sprint");
    store.updateSearchQuery(WEB, "default", "");
    expect(store.getSearchQuery(WEB, "default")).toBe("");
    expect(store.getFilters(OPS, "default")).toEqual({});
  });

  it("shows an observer of a project's list each change of it, from the first, and none of another project's", () => {
    const store = new CycleFilterStore();
    const seen: unknown[] = [];
    const stop = reaction(
      () => [store.getFilters(WEB, "default"), store.getSearchQuery(WEB, "default")],
      (shown) => seen.push(shown)
    );
    store.updateFilters(WEB, "default", { status: ["current"] });
    store.updateFilters(OPS, "default", { status: ["draft"] });
    store.updateSearchQuery(OPS, "default", "ops");
    store.updateSearchQuery(WEB, "default", "s");
    stop();
    expect(seen).toEqual([
      [{ status: ["current"] }, ""],
      [{ status: ["current"] }, "s"],
    ]);
  });
});
