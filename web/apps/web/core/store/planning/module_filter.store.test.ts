/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { reaction } from "mobx";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ModuleFilterStore } from "@/store/planning/module_filter.store";

// What each project's modules list and archive are filtered and searched by (P6a spec 2.4, R4), kept by the
// project's id for the session: every reader and change takes it from its caller (the route's). The store is built
// without the RootStore, so it follows no address; that a retired session's filters do not follow the page's
// RouterStore is store-context.test.ts. The layout and the order are the browser's, kept by the list's pages.

const WEB = "p-web";
const OPS = "p-ops";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("ModuleFilterStore", () => {
  it("keeps each project's filters by its id, those of its list and of its archive apart", () => {
    const store = new ModuleFilterStore();
    store.updateFilters(WEB, "default", { status: ["planned", "in-progress"], lead: ["u-ann"] });
    store.updateFilters(WEB, "archived", { members: ["u-bo"], target_date: ["2026-11-01;before"] });
    store.updateFilters(OPS, "default", { start_date: ["1_weeks;after;fromnow"] });
    expect(store.getFilters(WEB, "default")).toEqual({ status: ["planned", "in-progress"], lead: ["u-ann"] });
    expect(store.getFilters(WEB, "archived")).toEqual({ members: ["u-bo"], target_date: ["2026-11-01;before"] });
    expect(store.getFilters(OPS, "default")).toEqual({ start_date: ["1_weeks;after;fromnow"] });
    // A list nothing filtered yet, of a project filtered or not.
    expect(store.getFilters(OPS, "archived")).toEqual({});
    expect(store.getFilters("p-lab", "default")).toEqual({});
  });

  it("changes the filters given and keeps the others; clearing a list's leaves the other list's and project's", () => {
    const store = new ModuleFilterStore();
    store.updateFilters(WEB, "default", { status: ["planned"] });
    store.updateFilters(WEB, "default", { members: ["u-bo", "u-cy"] });
    store.updateFilters(WEB, "default", { status: ["paused"] });
    expect(store.getFilters(WEB, "default")).toEqual({ status: ["paused"], members: ["u-bo", "u-cy"] });
    store.updateFilters(WEB, "archived", { status: ["completed"] });
    store.updateFilters(OPS, "default", { lead: ["u-ann"] });
    store.clearAllFilters(WEB, "default");
    expect(store.getFilters(WEB, "default")).toEqual({});
    expect(store.getFilters(WEB, "archived")).toEqual({ status: ["completed"] });
    expect(store.getFilters(OPS, "default")).toEqual({ lead: ["u-ann"] });
  });

  it("keeps each project's two searches by its id, apart from its filters", () => {
    const store = new ModuleFilterStore();
    expect(store.getSearchQuery(WEB, "default")).toBe("");
    store.updateSearchQuery(WEB, "default", "auth");
    store.updateSearchQuery(WEB, "archived", "old");
    store.updateSearchQuery(OPS, "default", "ops");
    expect(store.getSearchQuery(WEB, "default")).toBe("auth");
    expect(store.getSearchQuery(WEB, "archived")).toBe("old");
    expect(store.getSearchQuery(OPS, "default")).toBe("ops");
    expect(store.getSearchQuery(OPS, "archived")).toBe("");
    // Clearing a list's filters keeps its search, and a search changes no filter.
    store.updateFilters(WEB, "default", { status: ["planned"] });
    store.clearAllFilters(WEB, "default");
    expect(store.getSearchQuery(WEB, "default")).toBe("auth");
    store.updateSearchQuery(WEB, "default", "");
    expect(store.getSearchQuery(WEB, "default")).toBe("");
    expect(store.getFilters(OPS, "default")).toEqual({});
  });

  it("shows an observer of a project's list each change of it, from the first, and none of another project's", () => {
    const store = new ModuleFilterStore();
    const seen: unknown[] = [];
    const stop = reaction(
      () => [store.getFilters(WEB, "default"), store.getSearchQuery(WEB, "default")],
      (shown) => seen.push(shown)
    );
    store.updateFilters(WEB, "default", { lead: ["u-ann"] });
    store.updateFilters(OPS, "default", { status: ["paused"] });
    store.updateSearchQuery(OPS, "default", "ops");
    store.updateSearchQuery(WEB, "default", "a");
    stop();
    expect(seen).toEqual([
      [{ lead: ["u-ann"] }, ""],
      [{ lead: ["u-ann"] }, "a"],
    ]);
  });

  it("keeps the filters in the session alone: the browser's storage is neither read nor written", () => {
    // The filters name members: kept in the browser, they would reach the next account to sign in there (R4).
    const touched: string[] = [];
    const storage: Storage = {
      length: 0,
      key: (index) => {
        touched.push(`key ${index}`);
        return null;
      },
      getItem: (key) => {
        touched.push(`get ${key}`);
        return null;
      },
      setItem: (key) => {
        touched.push(`set ${key}`);
      },
      removeItem: (key) => {
        touched.push(`remove ${key}`);
      },
      clear: () => {
        touched.push("clear");
      },
    };
    vi.stubGlobal("localStorage", storage);
    vi.stubGlobal("window", { localStorage: storage });
    const store = new ModuleFilterStore();
    store.updateFilters(WEB, "default", { members: ["u-bo"] });
    store.updateFilters(WEB, "archived", { lead: ["u-ann"] });
    store.updateSearchQuery(WEB, "default", "auth");
    store.updateSearchQuery(WEB, "archived", "old");
    store.clearAllFilters(WEB, "archived");
    expect(store.getFilters(WEB, "default")).toEqual({ members: ["u-bo"] });
    expect(store.getSearchQuery(WEB, "archived")).toBe("old");
    expect(touched).toEqual([]);
    // Nor does it keep a layout or an order: those are the browser's (P6a spec 2.4).
    expect(Object.keys(store).filter((key) => /layout|order|display/i.test(key))).toEqual([]);
  });
});
