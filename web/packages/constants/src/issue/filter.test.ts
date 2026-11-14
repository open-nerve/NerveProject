/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, ISSUE_STORE_TO_FILTERS_MAP } from "./filter";

// A module's page lists its module's work items under options of its own (P6b spec 2.6, R12): grouping by module would
// group them by the page's own module, so the page never offers it, now or once P7 adds modules to the vocabulary.
describe("the module page's display options", () => {
  it("are the module page's own", () => {
    const own = ISSUE_DISPLAY_FILTERS_BY_PAGE.module_issues;
    expect(own).toBeDefined();
    expect(own).not.toBe(ISSUE_DISPLAY_FILTERS_BY_PAGE.issues);
    expect(own).not.toBe(ISSUE_DISPLAY_FILTERS_BY_PAGE.cycle_issues);
    expect(ISSUE_STORE_TO_FILTERS_MAP[EIssuesStoreType.MODULE]).toBe(own);
  });

  it.each(["list", "kanban"] as const)("group the %s by anything but a module", (layout) => {
    const options = ISSUE_STORE_TO_FILTERS_MAP[EIssuesStoreType.MODULE]!.layoutOptions[layout]!.display_filters;
    const groupings = [...(options.group_by ?? []), ...(options.sub_group_by ?? [])];
    expect(groupings.length).toBeGreaterThan(0);
    expect(groupings.filter((option) => /module/i.test(String(option)))).toEqual([]);
  });
});
