/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, expectTypeOf, it } from "vitest";
import type { IntakeFilters } from "./filters";
import { filterKey, listQuery } from "./filters";

// The filters of a project's intake, as the intake page holds them, and what nerve is asked with them (P5a spec 2.2,
// R6, R9, R10; the contract's traps C1–C6): the key of a list is the same for any filters that list the same work
// items, and tells any others apart; the list's query never sends what nerve refuses (an empty or a repeated
// parameter, a date of more than one range, an order not of its six) nor leaves out what nerve reads as every value
// (the statuses).

/** The open tab as the page opens it: the pending, newest first, no other filter. */
const open: IntakeFilters = {
  tab: "open",
  statuses: ["pending"],
  priorities: [],
  assignees: [],
  creators: [],
  labels: [],
  order: "-created_at",
};
/** The closed tab as the page opens it: the three final statuses, newest first. */
const closed: IntakeFilters = { ...open, tab: "closed", statuses: ["accepted", "declined", "duplicate"] };
const withFilters = (fields: Partial<IntakeFilters>): IntakeFilters => ({ ...open, ...fields });

describe("filterKey", () => {
  it("is the same whatever the order of each filter's values, each named once", () => {
    const key = filterKey(
      withFilters({
        statuses: ["pending", "snoozed"],
        priorities: ["high", "urgent"],
        assignees: ["me", "none", "u-bob"],
        creators: ["me", "u-ann"],
        labels: ["l-bug", "none"],
      })
    );
    expect(
      filterKey(
        withFilters({
          statuses: ["snoozed", "pending", "snoozed"],
          priorities: ["urgent", "high", "urgent"],
          assignees: ["u-bob", "none", "me", "none"],
          creators: ["u-ann", "me", "u-ann"],
          labels: ["none", "l-bug", "l-bug"],
        })
      )
    ).toBe(key);
  });

  it("names a range of no end as no range: either lists every day", () => {
    expect(filterKey(withFilters({ created: {}, updated: {} }))).toBe(filterKey(open));
  });

  it("tells every other filter apart", () => {
    const filters = [
      open,
      closed,
      withFilters({ tab: "closed" }),
      withFilters({ statuses: ["snoozed"] }),
      withFilters({ statuses: ["pending", "snoozed"] }),
      withFilters({ priorities: ["urgent"] }),
      withFilters({ priorities: ["none"] }),
      withFilters({ assignees: ["me"] }),
      withFilters({ assignees: ["none"] }),
      withFilters({ creators: ["me"] }),
      withFilters({ labels: ["none"] }),
      withFilters({ labels: ["l-bug"] }),
      withFilters({ created: { from: "2026-11-01", to: "2026-11-01" } }),
      withFilters({ created: { from: "2026-11-01" } }),
      withFilters({ created: { to: "2026-11-01" } }),
      withFilters({ created: { from: "2026-11-01", to: "2026-11-07" } }),
      withFilters({ updated: { from: "2026-11-01" } }),
      withFilters({ order: "created_at" }),
      withFilters({ order: "-updated_at" }),
      withFilters({ order: "-sequence_id" }),
    ];
    expect(new Set(filters.map(filterKey)).size).toBe(filters.length);
  });
});

describe("listQuery", () => {
  it("sends the statuses in either tab, the tab's first ones too, as one array (C1: left out, every status)", () => {
    expect(listQuery(open)).toMatchObject({ status: ["pending"] });
    expect(listQuery(withFilters({ statuses: ["snoozed", "pending"] }))).toMatchObject({
      status: ["pending", "snoozed"],
    });
    expect(listQuery(closed)).toMatchObject({ status: ["accepted", "declined", "duplicate"] });
    expect(listQuery(withFilters({ tab: "closed", statuses: ["duplicate"] }))).toMatchObject({
      status: ["duplicate"],
    });
    // the filters hold one status at least (R9): none would ask nerve for every status
    expectTypeOf<readonly []>().not.toExtend<IntakeFilters["statuses"]>();
    expectTypeOf<readonly ["snoozed"]>().toExtend<IntakeFilters["statuses"]>();
  });

  it("sends no empty array: no priority, assignee, creator or label named, no parameter (C2)", () => {
    const query = listQuery(open);
    expect(Object.keys(query)).not.toContain("priority");
    expect(Object.keys(query)).not.toContain("assignee_id");
    expect(Object.keys(query)).not.toContain("created_by_id");
    expect(Object.keys(query)).not.toContain("label_id");
    expect(Object.keys(query).filter((name) => Array.isArray(query[name as keyof typeof query]))).toEqual(["status"]);
  });

  it("sends each filter's values as one array, in order, each once (C2: one CSV parameter, never repeated)", () => {
    expect(
      listQuery(
        withFilters({
          statuses: ["snoozed", "pending", "snoozed"],
          priorities: ["urgent", "high", "urgent"],
          assignees: ["u-bob", "none", "me"],
          creators: ["u-ann", "me", "me"],
          labels: ["none", "l-bug"],
        })
      )
    ).toEqual({
      status: ["pending", "snoozed"],
      priority: ["high", "urgent"],
      assignee_id: ["me", "none", "u-bob"],
      created_by_id: ["me", "u-ann"],
      label_id: ["l-bug", "none"],
      order_by: "-created_at",
      limit: 50,
    });
  });

  it.each<[string, IntakeFilters["created"], string | undefined]>([
    ["one day, the same at both ends", { from: "2026-11-01", to: "2026-11-01" }, "2026-11-01"],
    ["from one day to another", { from: "2026-11-01", to: "2026-11-07" }, "2026-11-01..2026-11-07"],
    ["from a day on", { from: "2026-11-01" }, "2026-11-01.."],
    ["till a day", { to: "2026-11-07" }, "..2026-11-07"],
    ["no end", {}, undefined],
    ["none", undefined, undefined],
  ])("sends each date as one range of days: %s (C3)", (_, range, days) => {
    const created = listQuery(withFilters({ created: range }));
    const updated = listQuery(withFilters({ updated: range }));
    expect(created.created_at).toBe(days);
    expect(updated.updated_at).toBe(days);
    expect(Object.keys(created).includes("created_at")).toBe(days !== undefined);
    expect(Object.keys(updated).includes("updated_at")).toBe(days !== undefined);
    expect(Object.keys(created)).not.toContain("updated_at");
    expect(Object.keys(updated)).not.toContain("created_at");
  });

  it.each<IntakeFilters["order"]>([
    "created_at",
    "-created_at",
    "updated_at",
    "-updated_at",
    "sequence_id",
    "-sequence_id",
  ])("sends the order %s as one of nerve's six values, as it is (C4)", (order) => {
    const query = listQuery(withFilters({ order }));
    expect(query.order_by).toBe(order);
    expect(Object.keys(query)).not.toContain("sort_by");
  });

  it("asks for pages of 50 (C5, R6), and never sends the tab: the statuses select", () => {
    expect(listQuery(open)).toEqual({ status: ["pending"], order_by: "-created_at", limit: 50 });
    expect(listQuery(closed)).toEqual({
      status: ["accepted", "declined", "duplicate"],
      order_by: "-created_at",
      limit: 50,
    });
  });

  it("sends the cursor only when given, as nerve answered it (C6)", () => {
    expect(Object.keys(listQuery(open))).not.toContain("cursor");
    expect(listQuery(open, "opaque+/=")).toEqual({
      status: ["pending"],
      order_by: "-created_at",
      limit: 50,
      cursor: "opaque+/=",
    });
  });
});
