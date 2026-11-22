/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { NotificationFilters } from "./filters";
import { filterKey, filtersOf, listQuery } from "./filters";

// The filters of the caller's notifications, as the page holds them, and what nerve is asked with them (P4 spec 2.2,
// R10, R21; the contract's traps T2–T5, T7, T8): the key of a list names exactly what it fetches, and turns back into
// the filters it names (re-93 m5); the list's query never sends what nerve refuses.

/** The page's first filters: all, no type, none of the three toggles. */
const all: NotificationFilters = { tab: "all", types: [], unread: false, archived: false, snoozed: false };
const withFilters = (fields: Partial<NotificationFilters>): NotificationFilters => ({ ...all, ...fields });

describe("filterKey", () => {
  it("is the same whatever the order of the types, each named once", () => {
    const key = filterKey(withFilters({ types: ["assigned", "created"] }));
    expect(filterKey(withFilters({ types: ["created", "assigned"] }))).toBe(key);
    expect(filterKey(withFilters({ types: ["created", "assigned", "created"] }))).toBe(key);
  });

  it("names no type in the mentions tab: the all tab's selection lists the same mentions", () => {
    const mentions = withFilters({ tab: "mentions" });
    expect(filterKey({ ...mentions, types: ["assigned", "subscribed"] })).toBe(filterKey(mentions));
  });

  it("tells every other filter apart", () => {
    const filters = [
      all,
      withFilters({ tab: "mentions" }),
      withFilters({ types: ["assigned"] }),
      withFilters({ types: ["created"] }),
      withFilters({ types: ["subscribed"] }),
      withFilters({ types: ["assigned", "created", "subscribed"] }),
      withFilters({ unread: true }),
      withFilters({ archived: true }),
      withFilters({ snoozed: true }),
      withFilters({ tab: "mentions", unread: true }),
      withFilters({ unread: true, archived: true }),
    ];
    const keys = filters.map(filterKey);
    expect(new Set(keys).size).toBe(filters.length);
  });
});

describe("filtersOf", () => {
  it.each<[string, NotificationFilters]>([
    ["the first filters", all],
    ["the mentions tab", withFilters({ tab: "mentions" })],
    ["one type", withFilters({ types: ["subscribed"] })],
    ["two types", withFilters({ types: ["created", "subscribed"] })],
    ["every type", withFilters({ types: ["assigned", "created", "subscribed"] })],
    ["unread", withFilters({ unread: true })],
    ["archived", withFilters({ archived: true })],
    ["snoozed", withFilters({ snoozed: true })],
    [
      "every toggle, in the mentions tab",
      withFilters({ tab: "mentions", unread: true, archived: true, snoozed: true }),
    ],
  ])("gives back the filters their key names: %s", (_, filters) => {
    expect(filtersOf(filterKey(filters))).toEqual(filters);
  });

  it("gives back the filters as their key names them: the types in order, each once, none in the mentions tab", () => {
    const shuffled = withFilters({ types: ["subscribed", "assigned", "subscribed"], unread: true });
    expect(filtersOf(filterKey(shuffled))).toEqual(withFilters({ types: ["assigned", "subscribed"], unread: true }));
    const mentions = withFilters({ tab: "mentions", types: ["created"], archived: true });
    expect(filtersOf(filterKey(mentions))).toEqual(withFilters({ tab: "mentions", archived: true }));
  });
});

describe("listQuery", () => {
  it("always sends the tab, archived and snoozed as false or true (T7), and pages of 100 (T3, R21)", () => {
    expect(listQuery(all)).toEqual({ tab: "all", archived: "false", snoozed: "false", limit: 100 });
    expect(listQuery(withFilters({ archived: true }))).toEqual({
      tab: "all",
      archived: "true",
      snoozed: "false",
      limit: 100,
    });
    expect(listQuery(withFilters({ snoozed: true }))).toEqual({
      tab: "all",
      archived: "false",
      snoozed: "true",
      limit: 100,
    });
  });

  it("sends the all tab's types, in order and each once, joined by commas", () => {
    expect(listQuery(withFilters({ types: ["subscribed", "assigned", "subscribed"] }))).toEqual({
      tab: "all",
      type: "assigned,subscribed",
      archived: "false",
      snoozed: "false",
      limit: 100,
    });
  });

  it("never sends type in the mentions tab, whatever the all tab's selection (T2, R10)", () => {
    const query = listQuery(withFilters({ tab: "mentions", types: ["assigned", "created"] }));
    expect(query).toEqual({ tab: "mentions", archived: "false", snoozed: "false", limit: 100 });
    expect(Object.keys(query)).not.toContain("type");
  });

  it("never sends an empty type: no type selected, no type (T8)", () => {
    expect(Object.keys(listQuery(all))).not.toContain("type");
    expect(Object.keys(listQuery(withFilters({ types: [] }), "c-2"))).not.toContain("type");
  });

  it("asks for the unread alone with read=false, and sends no read otherwise (T5)", () => {
    expect(listQuery(withFilters({ unread: true }))).toEqual({
      tab: "all",
      read: "false",
      archived: "false",
      snoozed: "false",
      limit: 100,
    });
    expect(Object.keys(listQuery(all))).not.toContain("read");
  });

  it("sends a cursor only when given one, as it is (T4)", () => {
    expect(listQuery(all, "opaque+/=")).toEqual({
      tab: "all",
      archived: "false",
      snoozed: "false",
      limit: 100,
      cursor: "opaque+/=",
    });
    expect(Object.keys(listQuery(all))).not.toContain("cursor");
  });

  it("names every value as a string but the page size, a number", () => {
    const query = listQuery(withFilters({ types: ["created"], unread: true, archived: true }), "c-1");
    const { limit, ...strings } = query;
    expect(limit).toBe(100);
    expect(Object.values(strings).every((value) => typeof value === "string")).toBe(true);
    expect(Object.keys(strings).toSorted()).toEqual(["archived", "cursor", "read", "snoozed", "tab", "type"]);
  });
});
