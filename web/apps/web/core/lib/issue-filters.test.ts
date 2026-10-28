/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import { memberListQuery, projectListQuery, toFilterExpression, toProjectFilters } from "./issue-filters";
it("translatesOneLevelAndWithoutCollapsingRepeatedDateConditions", () => {
  expect(
    toProjectFilters({
      and: [{ state_id__in: "a,b" }, { target_date__range: "2026-10-01," }, { target_date__range: ",2026-10-31" }],
    })
  ).toEqual({ state_id: ["a", "b"], target_date: ["2026-10-01..", "..2026-10-31"] });
  expect(
    toProjectFilters(toFilterExpression({ label_id: ["none", "L"], target_date: ["2026-10-01..", "..2026-10-31"] }))
  ).toEqual({ label_id: ["none", "L"], target_date: ["2026-10-01..", "..2026-10-31"] });
});
it("narrowsDisplayVocabularyAndMemberScope", () => {
  expect(
    projectListQuery(
      {},
      { layout: "kanban", group_by: "labels", sub_group_by: "assignees", order_by: "-labels__name", sub_issue: false }
    )
  ).toEqual({ group_by: "label_id", sub_group_by: "assignee_id", order_by: "-label_name", sub_issue: "false" });
  expect(
    memberListQuery({ state_id: ["S"], label_id: ["none"] }, { group_by: "project", order_by: "broken" }, "created")
  ).toEqual({
    relation: "created",
    label_id: ["none"],
    group_by: "project_id",
    order_by: "-created_at",
    sub_issue: "true",
  });
});

it("F01 preserves subscriber filters on readback", () => {
  expect(toProjectFilters(toFilterExpression({ subscriber_id: ["me"] }))).toEqual({ subscriber_id: ["me"] });
});

it("F01 clears incompatible date none", () => {
  expect(toProjectFilters({ and: [{ target_date__exact: "none" }, { target_date__exact: "2026-10-01" }] })).toEqual({});
});

it("F01 avoids an orphan subgroup", () => {
  expect(projectListQuery({}, { layout: "kanban", sub_group_by: "labels" })).toEqual({
    order_by: "-created_at",
    sub_issue: "true",
  });
});
