/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, expectTypeOf, it } from "vitest";
import { intakeIssueOf } from "./fake-intake";
import type { IntakeFilters } from "./filters";
import { filterKey, listQuery, matches } from "./filters";

// The filters of a project's intake, as the intake page holds them, and what nerve is asked with them (P5a spec 2.2,
// R6, R9, R10; the contract's traps C1–C6): the key of a list is the same for any filters that list the same work
// items, and tells any others apart; the list's query never sends what nerve refuses (an empty or a repeated
// parameter, a date of more than one range, an order not of its six) nor leaves out what nerve reads as every value
// (the statuses). Whether a list of the filters lists a work item is what nerve would answer: its status at the moment,
// each filter's values ORed, the filters ANDed, me the caller, none no value, each day a UTC day.

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

describe("matches", () => {
  const NOW = new Date("2026-11-10T09:00:00Z");
  const ME = "u-me";
  /** Whether a list of the filters lists the row at NOW, for u-me. */
  const lists = (fields: Partial<IntakeFilters>, row = intakeIssueOf("one")) =>
    matches(row, withFilters(fields), NOW, ME);
  const snoozedTill = (till: string) =>
    intakeIssueOf("dozing", {
      intake: { status: "snoozed", snoozed_till: till, duplicate_to_id: null },
    });

  it("lists by the status at now: a snooze whose time has come is pending, one whose time has not snoozed (C7)", () => {
    expect(lists({ statuses: ["pending"] })).toBe(true);
    expect(lists({ statuses: ["snoozed"] })).toBe(false);
    const due = snoozedTill("2026-11-10T09:00:00Z");
    expect(lists({ statuses: ["pending"] }, due)).toBe(true);
    expect(lists({ statuses: ["snoozed"] }, due)).toBe(false);
    const dozing = snoozedTill("2026-11-11T00:00:00Z");
    expect(lists({ statuses: ["pending"] }, dozing)).toBe(false);
    expect(lists({ statuses: ["pending", "snoozed"] }, dozing)).toBe(true);
    const accepted = intakeIssueOf("two", {
      intake: { status: "accepted", snoozed_till: null, duplicate_to_id: null },
    });
    expect(matches(accepted, closed, NOW, ME)).toBe(true);
    expect(matches(accepted, open, NOW, ME)).toBe(false);
  });

  it("lists by priority: any of those named, every one for none named", () => {
    const high = intakeIssueOf("one", { priority: "high" });
    expect(lists({}, high)).toBe(true);
    expect(lists({ priorities: ["urgent", "high"] }, high)).toBe(true);
    expect(lists({ priorities: ["urgent"] }, high)).toBe(false);
    // none is a priority, as nerve takes it
    expect(lists({ priorities: ["none"] })).toBe(true);
    expect(lists({ priorities: ["none"] }, high)).toBe(false);
  });

  it("lists by assignee: any of those named, me the caller, none a work item of no assignee", () => {
    const theirs = intakeIssueOf("one", { assignee_ids: ["u-ann", "u-bob"] });
    const mine = intakeIssueOf("two", { assignee_ids: [ME] });
    const nobodys = intakeIssueOf("three");
    expect(lists({}, theirs)).toBe(true);
    expect(lists({ assignees: ["u-bob"] }, theirs)).toBe(true);
    expect(lists({ assignees: ["u-cid"] }, theirs)).toBe(false);
    expect(lists({ assignees: ["me"] }, mine)).toBe(true);
    expect(lists({ assignees: ["me"] }, theirs)).toBe(false);
    expect(lists({ assignees: ["none"] }, nobodys)).toBe(true);
    expect(lists({ assignees: ["none"] }, mine)).toBe(false);
    expect(lists({ assignees: ["none", "u-ann"] }, theirs)).toBe(true);
    expect(lists({ assignees: ["none", "me"] }, theirs)).toBe(false);
    // the caller not known: me names no one
    expect(matches(mine, withFilters({ assignees: ["me"] }), NOW, undefined)).toBe(false);
  });

  it("lists by creator: any of those named, me the caller", () => {
    const annS = intakeIssueOf("one", { created_by_id: "u-ann" });
    const mine = intakeIssueOf("two", { created_by_id: ME });
    expect(lists({ creators: ["u-ann"] }, annS)).toBe(true);
    expect(lists({ creators: ["u-bob", "u-ann"] }, annS)).toBe(true);
    expect(lists({ creators: ["u-bob"] }, annS)).toBe(false);
    expect(lists({ creators: ["me"] }, mine)).toBe(true);
    expect(lists({ creators: ["me"] }, annS)).toBe(false);
    expect(lists({ creators: ["me"] }, intakeIssueOf("three", { created_by_id: null }))).toBe(false);
  });

  it("lists by label: any of those named, none a work item of no label", () => {
    const labelled = intakeIssueOf("one", { label_ids: ["l-bug", "l-ui"] });
    const bare = intakeIssueOf("two");
    expect(lists({ labels: ["l-ui"] }, labelled)).toBe(true);
    expect(lists({ labels: ["l-docs"] }, labelled)).toBe(false);
    expect(lists({ labels: ["none"] }, bare)).toBe(true);
    expect(lists({ labels: ["none"] }, labelled)).toBe(false);
    expect(lists({ labels: ["l-docs", "none"] }, bare)).toBe(true);
  });

  it("ANDs the filters: a work item each selects", () => {
    const row = intakeIssueOf("one", { priority: "high", assignee_ids: [ME], label_ids: ["l-bug"] });
    expect(lists({ priorities: ["high"], assignees: ["me"], labels: ["l-bug"] }, row)).toBe(true);
    expect(lists({ priorities: ["high"], assignees: ["me"], labels: ["none"] }, row)).toBe(false);
  });

  it("lists by the UTC day of creation, both ends in (C3): from the first's midnight UTC to the midnight after the last", () => {
    const early = intakeIssueOf("one", { created_at: "2026-11-02T00:30:00Z" });
    const late = intakeIssueOf("two", { created_at: "2026-11-01T23:30:00Z" });
    expect(lists({ created: { from: "2026-11-02", to: "2026-11-02" } }, early)).toBe(true);
    expect(lists({ created: { from: "2026-11-02", to: "2026-11-02" } }, late)).toBe(false);
    expect(lists({ created: { from: "2026-11-01", to: "2026-11-01" } }, late)).toBe(true);
    expect(lists({ created: { from: "2026-11-01", to: "2026-11-01" } }, early)).toBe(false);
    expect(lists({ created: { from: "2026-10-30", to: "2026-11-02" } }, early)).toBe(true);
    expect(lists({ created: { from: "2026-11-02" } }, early)).toBe(true);
    expect(lists({ created: { from: "2026-11-02" } }, late)).toBe(false);
    expect(lists({ created: { to: "2026-11-01" } }, late)).toBe(true);
    expect(lists({ created: { to: "2026-11-01" } }, early)).toBe(false);
    // the same moment in another offset is the same UTC day
    const offset = intakeIssueOf("three", { created_at: "2026-11-02T07:30:00+08:00" });
    expect(lists({ created: { to: "2026-11-01" } }, offset)).toBe(true);
    expect(lists({ created: {} }, late)).toBe(true);
  });

  it("lists by the UTC day of the last change, as the creation", () => {
    const changed = intakeIssueOf("one", { created_at: "2026-10-01T00:00:00Z", updated_at: "2026-11-05T23:59:59Z" });
    expect(lists({ updated: { from: "2026-11-05", to: "2026-11-05" } }, changed)).toBe(true);
    expect(lists({ updated: { from: "2026-11-06" } }, changed)).toBe(false);
    expect(lists({ updated: { to: "2026-11-04" } }, changed)).toBe(false);
    expect(lists({ created: { from: "2026-11-05" } }, changed)).toBe(false);
  });

  it("never by the tab, which the statuses select, nor by the order", () => {
    expect(lists({ tab: "closed" })).toBe(true);
    expect(lists({ order: "sequence_id" })).toBe(true);
  });
});
