/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { DEFAULT_GLOBAL_VIEWS_LIST } from "../workspace";
import { ISSUE_DISPLAY_PROPERTIES_KEYS } from "./common";
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, ISSUE_STORE_TO_FILTERS_MAP } from "./filter";

// A workspace's pages filter by the contract's 11 words and draw only the spreadsheet (M7/P6b spec 2.4, R5, R8), whose
// columns on a page of no project never show a cycle or the modules, so its display panel offers neither (re-97 m1).
describe("the workspace's pages' display options", () => {
  const workspace = ISSUE_DISPLAY_FILTERS_BY_PAGE.workspace_issues;

  it("filter by the contract's 11 words", () => {
    expect(workspace.filters).toEqual([
      "priority",
      "state_group",
      "state_id",
      "label_id",
      "assignee_id",
      "created_by_id",
      "mention_id",
      "subscriber_id",
      "project_id",
      "start_date",
      "target_date",
    ]);
  });

  it("are the spreadsheet's alone, every property but a cycle and the modules", () => {
    expect(Object.keys(workspace.layoutOptions)).toEqual(["spreadsheet"]);
    expect(workspace.layoutOptions.spreadsheet!.display_properties).toEqual(
      ISSUE_DISPLAY_PROPERTIES_KEYS.filter((key) => key !== "cycle" && key !== "modules")
    );
  });

  it("are the GLOBAL store type's, under their new name alone", () => {
    expect(ISSUE_STORE_TO_FILTERS_MAP[EIssuesStoreType.GLOBAL]).toBe(workspace);
    expect(ISSUE_DISPLAY_FILTERS_BY_PAGE.my_issues).toBeUndefined();
  });
});

// The four default views and each one's own term, `me` in it, fixed in its query (M7/P6b spec 2.5, R4).
it("lists the four default views, each with its own term", () => {
  expect(DEFAULT_GLOBAL_VIEWS_LIST.map(({ key, own }) => [key, own])).toEqual([
    ["all-issues", undefined],
    ["assigned", "assignee_id"],
    ["created", "created_by_id"],
    ["subscribed", "subscriber_id"],
  ]);
});
