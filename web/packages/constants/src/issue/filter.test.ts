/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, ISSUE_STORE_TO_FILTERS_MAP } from "./filter";

// The project page filters, groups and sub-groups its work items by their cycle and their modules (P7b spec 2.5, A8).
describe("the project page's display options", () => {
  const project = ISSUE_DISPLAY_FILTERS_BY_PAGE.issues;

  it("filter by the cycle and the modules", () => {
    expect(project.filters).toEqual(expect.arrayContaining(["cycle_id", "module_id"]));
  });

  it("group the list, and group and sub-group the board, by the cycle and the module, before no grouping", () => {
    const { list, kanban } = project.layoutOptions;
    expect(list!.display_filters.group_by!.slice(-3)).toEqual(["cycle", "module", null]);
    expect(kanban!.display_filters.group_by!.slice(-2)).toEqual(["cycle", "module"]);
    expect(kanban!.display_filters.sub_group_by!.slice(-3)).toEqual(["cycle", "module", null]);
  });
});

// The archive lists the project's archived work items through the project's list (`archived=true`), whose words it
// shares: it filters and groups its one layout, the list, by the cycle and the module too (P8 spec 2.4).
describe("the archived page's display options", () => {
  const archived = ISSUE_DISPLAY_FILTERS_BY_PAGE.archived_issues;

  it("filter by the cycle and the modules", () => {
    expect(archived.filters).toEqual(expect.arrayContaining(["cycle_id", "module_id"]));
  });

  it("group the list by the cycle and the module, before no grouping", () => {
    expect(Object.keys(archived.layoutOptions)).toEqual(["list"]);
    expect(archived.layoutOptions.list!.display_filters.group_by!.slice(-3)).toEqual(["cycle", "module", null]);
  });
});

// A cycle's and a module's page list their own cycle's or module's work items under options of their own (P5b R9,
// P6b R12): filtering or grouping by the page's own cycle or module would name it everywhere, and nerve refuses it on
// the page's preferences. Each is the project page's options without the page's own filter and grouping, every other
// option the same (P7b spec 2.5, A8).
describe.each([
  { page: "cycle_issues", type: EIssuesStoreType.CYCLE, filter: "cycle_id", grouping: "cycle", other: "module" },
  { page: "module_issues", type: EIssuesStoreType.MODULE, filter: "module_id", grouping: "module", other: "cycle" },
])("the $page page's display options", ({ page, type, filter, grouping, other }) => {
  const own = ISSUE_DISPLAY_FILTERS_BY_PAGE[page];

  it("are the page's own", () => {
    expect(own).toBeDefined();
    expect(own).not.toBe(ISSUE_DISPLAY_FILTERS_BY_PAGE.issues);
    expect(own).not.toBe(ISSUE_DISPLAY_FILTERS_BY_PAGE[`${other}_issues`]);
    expect(ISSUE_STORE_TO_FILTERS_MAP[type]).toBe(own);
  });

  it(`are the project page's without ${filter} and ${grouping}, every other option the same`, () => {
    const lists = new Set(["filters", "group_by", "sub_group_by"]);
    const withoutOwn: unknown = JSON.parse(JSON.stringify(ISSUE_DISPLAY_FILTERS_BY_PAGE.issues), (key, value) =>
      lists.has(key) ? (value as unknown[]).filter((word) => word !== filter && word !== grouping) : value
    );
    expect(own).toStrictEqual(withoutOwn);
  });

  it.each(["list", "kanban"] as const)(`group the %s by anything but the page's own, by the ${other} too`, (layout) => {
    const options = ISSUE_STORE_TO_FILTERS_MAP[type]!.layoutOptions[layout]!.display_filters;
    const groupings = [...(options.group_by ?? []), ...(options.sub_group_by ?? [])];
    expect(groupings.length).toBeGreaterThan(0);
    expect(groupings.filter((option) => new RegExp(grouping, "i").test(String(option)))).toEqual([]);
    expect(groupings).toContain(other);
  });

  it(`filter by anything but the page's own, by the ${other} too`, () => {
    expect(own.filters).not.toContain(filter);
    expect(own.filters).toContain(`${other}_id`);
  });
});
