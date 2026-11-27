// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A default view's preferences (M7/P6b spec 2.4, R3; design §3.7): its display and properties are kept in the browser,
// by account and page, so that entering the view again shows them; its filters last for one entry only, in memory,
// never in the browser. What the browser gives back is read field by field (the spreadsheet, no grouping, each other
// field its default unless it has the right type); a browser that keeps nothing, or refuses, never fails the page.
import { afterEach, describe, expect, it, vi } from "vitest";
import type { IssueListDisplayFilters } from "@nerve/api-client";
import { DefaultViewPreferences } from "./default-view-preferences";

const ACME = JSON.stringify(["acme", "assigned"]);
const BETA = JSON.stringify(["beta", "assigned"]);
const KEY = "workspace_view_display";
const SHOWN: IssueListDisplayFilters = {
  layout: "spreadsheet",
  group_by: null,
  sub_group_by: null,
  order_by: "-created_at",
  sub_issue: true,
  show_empty_groups: true,
  calendar: { layout: "month", show_weekends: false },
};
const EVERY = {
  assignee: true,
  start_date: true,
  due_date: true,
  labels: true,
  key: true,
  priority: true,
  state: true,
  sub_issue_count: true,
  link: true,
  attachment_count: true,
  created_on: true,
  updated_on: true,
  cycle: true,
  modules: true,
};
const of = (account: string | undefined) => new DefaultViewPreferences(() => account);
afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

describe("a default view's preferences", () => {
  it("are no filters and a new workspace view's display before anything was kept", async () => {
    await expect(of("u-me").get(ACME)).resolves.toEqual({
      filters: {},
      display_filters: SHOWN,
      display_properties: EVERY,
    });
  });

  it("keep the display in the browser, the filters in memory for this entry only", async () => {
    const preferences = of("u-me");
    await preferences.update(ACME, { filters: { priority: ["high"] } });
    const answer = await preferences.update(ACME, {
      display_filters: { ...SHOWN, order_by: "priority", sub_issue: false },
    });
    await preferences.update(ACME, { display_properties: { ...EVERY, labels: false } });
    expect(answer).toMatchObject({ filters: { priority: ["high"] }, display_filters: { order_by: "priority" } });
    expect(window.localStorage.getItem(KEY)).not.toContain("high");
    // the next session's
    await expect(of("u-me").get(ACME)).resolves.toEqual({
      filters: {},
      display_filters: { ...SHOWN, order_by: "priority", sub_issue: false },
      display_properties: { ...EVERY, labels: false },
    });
  });

  it("forget the entry's filters when the page leaves, and keep its display", async () => {
    const preferences = of("u-me");
    await preferences.update(ACME, { filters: { priority: ["high"] } });
    await preferences.update(ACME, { display_filters: { ...SHOWN, order_by: "priority" } });
    preferences.discard(ACME);
    const after = await preferences.get(ACME);
    expect(after.filters).toEqual({});
    expect(after.display_filters.order_by).toBe("priority");
  });

  it("are each account's and each page's own: the same view of two workspaces keeps two displays", async () => {
    await of("u-me").update(ACME, { display_filters: { ...SHOWN, order_by: "priority" } });
    await of("u-me").update(BETA, { display_filters: { ...SHOWN, order_by: "-updated_at" } });
    expect((await of("u-me").get(ACME)).display_filters.order_by).toBe("priority");
    expect((await of("u-me").get(BETA)).display_filters.order_by).toBe("-updated_at");
    expect((await of("u-other").get(ACME)).display_filters).toEqual(SHOWN);
  });

  it("read back the spreadsheet without grouping, and each other field kept only with its type", async () => {
    const stored = {
      display_filters: {
        layout: "list",
        group_by: "state_id",
        sub_group_by: "priority",
        order_by: 5,
        sub_issue: "no",
        show_empty_groups: false,
        calendar: { layout: "year" },
      },
      display_properties: { cycle: false, key: "no", labels: false, extra: true },
    };
    window.localStorage.setItem(KEY, JSON.stringify({ "u-me": { [ACME]: stored } }));
    await expect(of("u-me").get(ACME)).resolves.toEqual({
      filters: {},
      display_filters: { ...SHOWN, show_empty_groups: false },
      display_properties: { ...EVERY, cycle: false, labels: false },
    });
  });

  it.each([
    ["not JSON", "{"],
    ["no record", "[1]"],
    ["an account's entries no record", JSON.stringify({ "u-me": "x" })],
    ["a page's entry no record", JSON.stringify({ "u-me": { [ACME]: 3 } })],
  ])("read back the defaults when the browser holds %s", async (_what, raw) => {
    window.localStorage.setItem(KEY, raw);
    await expect(of("u-me").get(ACME)).resolves.toEqual({
      filters: {},
      display_filters: SHOWN,
      display_properties: EVERY,
    });
  });

  it("never fail when the browser refuses: the defaults read, and a change shown for the session", async () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("denied");
    });
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("denied");
    });
    const preferences = of("u-me");
    await expect(preferences.get(ACME)).resolves.toMatchObject({ display_filters: SHOWN });
    await expect(
      preferences.update(ACME, { display_filters: { ...SHOWN, order_by: "priority" } })
    ).resolves.toMatchObject({ display_filters: { order_by: "priority" } });
    await expect(preferences.get(ACME)).resolves.toMatchObject({ display_filters: { order_by: "priority" } });
  });

  it("neither read nor write the browser without an account", async () => {
    window.localStorage.setItem(KEY, JSON.stringify({ undefined: { [ACME]: { display_filters: { order_by: "x" } } } }));
    const read = vi.spyOn(Storage.prototype, "getItem");
    const write = vi.spyOn(Storage.prototype, "setItem");
    const preferences = of(undefined);
    await expect(preferences.get(ACME)).resolves.toMatchObject({ display_filters: SHOWN });
    await preferences.update(ACME, { display_filters: { ...SHOWN, order_by: "priority" } });
    expect([read.mock.calls, write.mock.calls]).toEqual([[], []]);
  });
});
