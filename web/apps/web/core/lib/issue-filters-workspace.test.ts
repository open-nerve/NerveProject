/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's pages filter by the contract's 11 words (M7/P6b spec 2.4, R5): the project page's ten without a cycle or
// a module, and the workspace's projects; their list is flat, nerve refusing any grouping (422). The filters go back to
// the controls' expression with the projects in it, so that a view made through the API keeps every word it holds.
import { expect, it } from "vitest";
import type { TWorkItemFilterExpression } from "@nerve/types";
import { toFilterExpression, toMemberFilters, toWorkspaceFilters, workspaceListQuery } from "./issue-filters";

const A = "019a0000-0000-7000-8000-000000000001";
const B = "019a0000-0000-7000-8000-000000000002";
const S = "019a0000-0000-7000-8000-0000000000a1";

/** Every word the contract gives a workspace's view, each with a value its list takes. */
const eleven = {
  state_id: [S],
  state_group: ["started"],
  priority: ["urgent", "none"],
  label_id: [A, "none"],
  assignee_id: ["me", B],
  created_by_id: ["me"],
  mention_id: ["me"],
  subscriber_id: [B],
  start_date: ["2026-10-01.."],
  target_date: ["none"],
  project_id: [A, B],
};

it("takes the contract's 11 words from the controls' expression, and no cycle or module", () => {
  const expression: TWorkItemFilterExpression = {
    and: [
      { state_id__in: S },
      { state_group__in: "started" },
      { priority__in: "urgent,None" },
      { label_id__in: `${A},None` },
      { assignee_id__in: `me,${B}` },
      { created_by_id__in: "me" },
      { mention_id__in: "me" },
      { subscriber_id__in: B },
      { start_date__range: "2026-10-01," },
      { target_date__exact: "None" },
      { project_id__in: `${A},${B}` },
      { cycle_id__in: A },
      { module_id__in: B },
    ],
  };
  const filters = toWorkspaceFilters(expression);
  expect(filters).toEqual(eleven);
  expect(Object.keys(filters)).toHaveLength(11);
});

it("checks each word as the project page's do: a bad value or a second project condition clears its word", () => {
  expect(toWorkspaceFilters({ and: [{ priority__in: "broken" }, { project_id__in: A.toUpperCase() }] })).toEqual({
    project_id: [A],
  });
  expect(
    toWorkspaceFilters({ and: [{ project_id__in: A }, { project_id__exact: B }, { priority__in: "low" }] })
  ).toEqual({ priority: ["low"] });
  expect(toWorkspaceFilters({ project_id__exact: B })).toEqual({ project_id: [B] });
});

it("gives the workspace's filters back to the controls with their projects, every word kept both ways", () => {
  const expression = toFilterExpression(eleven);
  expect(expression).toMatchObject({ and: expect.arrayContaining([{ project_id__in: `${A},${B}` }]) });
  expect(toWorkspaceFilters(expression)).toEqual(eleven);
});

it("gives a member view's filters back to the controls as before, its project condition last", () => {
  const expression = { and: [{ project_id__in: A.toUpperCase() }, { priority__in: "high" }] };
  expect(toFilterExpression(toMemberFilters(expression))).toEqual({
    and: [{ priority__in: "high" }, { project_id__in: A }],
  });
  expect(toFilterExpression(toMemberFilters({ project_id__in: A }))).toEqual({ and: [{ project_id__in: A }] });
});

it.each([
  ["a list", { layout: "list", group_by: "state" }],
  ["a board", { layout: "kanban", group_by: "priority", sub_group_by: "state" }],
  ["a calendar", { layout: "calendar" }],
  ["the spreadsheet", { layout: "spreadsheet", group_by: "project" }],
])("asks for a flat workspace list whatever %s's display groups by", (_name, display) => {
  const query = workspaceListQuery(eleven, { ...display, order_by: "-assignees__first_name", sub_issue: false });
  expect(query).toEqual({ ...eleven, order_by: "-assignee_name", sub_issue: "false" });
  expect(Object.keys(query)[0]).toBe("project_id");
  expect(query).not.toHaveProperty("group_by");
  expect(query).not.toHaveProperty("sub_group_by");
});

it("asks for none of a cycle or a module, nor a value nerve would refuse, whatever it is given", () => {
  const given = { priority: ["broken"], cycle_id: [A], module_id: [B], project_id: ["none"] } as never;
  expect(workspaceListQuery(given)).toEqual({ order_by: "-created_at", sub_issue: "true" });
});
