/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, it } from "vitest";
import type { ProjectIssueFilters } from "@nerve/api-client";
import {
  workItemFiltersAdapter,
  FilterInstance,
  WorkItemFilterStore,
  buildWorkItemFilterExpressionFromConditions,
} from "@nerve/shared-state";
import { EIssuesStoreType } from "@nerve/types";
import type { TWorkItemFilterProperty, TWorkItemFilterExpression } from "@nerve/types";
const A = "019a0000-0000-7000-8000-000000000001";
const B = "019a0000-0000-7000-8000-000000000002";
import { listContext, listIssue } from "@/store/issue/fake-issues";
import { issueMembership } from "./issue-membership";
import { memberListQuery, projectListQuery, toFilterExpression, toProjectFilters } from "./issue-filters";
it("translatesOneLevelAndWithoutCollapsingRepeatedDateConditions", () => {
  expect(
    toProjectFilters({
      and: [
        { state_id__in: `${A},${B}` },
        { target_date__range: "2026-10-01," },
        { target_date__range: ",2026-10-31" },
      ],
    })
  ).toEqual({ state_id: [A, B], target_date: ["2026-10-01..", "..2026-10-31"] });
  expect(
    toProjectFilters(toFilterExpression({ label_id: ["none", A], target_date: ["2026-10-01..", "..2026-10-31"] }))
  ).toEqual({ label_id: ["none", A], target_date: ["2026-10-01..", "..2026-10-31"] });
});
it("narrowsDisplayVocabularyAndMemberScope", () => {
  expect(
    projectListQuery(
      {},
      { layout: "kanban", group_by: "labels", sub_group_by: "assignees", order_by: "-labels__name", sub_issue: false }
    )
  ).toEqual({ group_by: "label_id", sub_group_by: "assignee_id", order_by: "-label_name", sub_issue: "false" });
  expect(
    memberListQuery({ state_id: [A], label_id: ["none"] }, { group_by: "project", order_by: "broken" }, "created")
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

// F01-01..08: each scalar/collection property accepts its own wire vocabulary.
it.each([
  ["state_id", A, `${A},${B}`, [A, B]],
  [
    "state_group",
    "started",
    "backlog,unstarted,started,completed,cancelled",
    ["backlog", "unstarted", "started", "completed", "cancelled"],
  ],
  ["priority", "none", "urgent,high,medium,low,none", ["urgent", "high", "medium", "low", "none"]],
  ["label_id", "None", `None,${A}`, ["none", A]],
  ["assignee_id", "me", `None,me,${A}`, ["none", "me", A]],
  ["created_by_id", "me", `me,${A}`, ["me", A]],
  ["mention_id", "me", `me,${A}`, ["me", A]],
  ["subscriber_id", "me", `me,${A}`, ["me", A]],
])("F01 property %s supports exact/in and rejects range", (field, single, input, values) => {
  expect(toProjectFilters({ [`${field}__exact`]: single })).toEqual({ [field]: [single === "None" ? "none" : single] });
  expect(toProjectFilters({ [`${field}__in`]: input })).toEqual({ [field]: values });
  expect(toProjectFilters({ [`${field}__range`]: input })).toEqual({});
  expect(toProjectFilters({ [`${field}__exact`]: input })).toEqual({});
});

it.each([
  ["state", "state_id", A],
  ["labels", "label_id", A],
  ["assignees", "assignee_id", "me"],
  ["created_by", "created_by_id", "me"],
  ["mentions", "mention_id", "me"],
  ["subscriber", "subscriber_id", "me"],
])("F01-09 maps the UI property %s", (property, field, value) => {
  expect(toProjectFilters({ [`${property}__in`]: value })).toEqual({ [field]: [value] });
});

it.each(["start_date", "target_date"])("F01-10 preserves each date condition and range bound for %s", (field) => {
  expect(
    toProjectFilters({
      and: [
        { [`${field}__exact`]: "2026-10-10" },
        { [`${field}__range`]: "2026-10-01,2026-10-31" },
        { [`${field}__range`]: "2026-10-01," },
        { [`${field}__range`]: ",2026-10-31" },
        { [`${field}__exact`]: "2026-10-10" },
      ],
    })
  ).toEqual({ [field]: ["2026-10-10", "2026-10-01..2026-10-31", "2026-10-01..", "..2026-10-31", "2026-10-10"] });
  expect(toProjectFilters({ [`${field}__exact`]: "None" })).toEqual({ [field]: ["none"] });
  expect(toProjectFilters({ [`${field}__in`]: "2026-10-10" })).toEqual({});
});

it.each([
  ["state_id", "me"],
  ["label_id", "me"],
  ["created_by_id", "none"],
  ["mention_id", "none"],
  ["subscriber_id", "none"],
  ["priority", "highest"],
  ["state_group", "done"],
  ["assignee_id", "not-a-uuid"],
])("F01-11 clears invalid values for %s", (field, value) => {
  expect(toProjectFilters({ [`${field}__in`]: value })).toEqual({});
  expect(toProjectFilters({ [`${field}__in`]: `,${value}` })).toEqual({});
});

it.each([
  "2026-02-29",
  "2026-13-01",
  "2026-1-01",
  "2026-10-32",
  "..",
  "2026-10-31..2026-10-01",
  "2026-10-01..2026-02-30",
  "2026-10-01;after",
  "1_weeks;after;fromnow",
])("F01-12 clears bad stored date %s", (value) => {
  expect(toFilterExpression({ target_date: [value], priority: ["high"] })).toEqual({ and: [{ priority__in: "high" }] });
  expect(projectListQuery({ target_date: [value] })).toEqual({ order_by: "-created_at", sub_issue: "true" });
});

it("F01-13 clears an unrepresentable repeated non-date AND as a whole field", () => {
  expect(toProjectFilters({ and: [{ label_id__in: A }, { label_id__in: B }, { priority__in: "high" }] })).toEqual({
    priority: ["high"],
  });
  expect(toProjectFilters({ and: [{ priority__exact: "high" }, { priority__in: "high,low" }] })).toEqual({});
});

it("F01-14 clears malformed fields and unsupported structures without leaking HTTP parameters", () => {
  const damaged = JSON.parse(
    '{"priority":["high","broken"],"assignee_id":"me","state_group":null,"start_date":[7],"label_id":[],"cycle_id":["x"],"archived":["true"],"order_by":"broken"}'
  );
  expect(toFilterExpression(damaged)).toEqual({});
  expect(projectListQuery(damaged)).toEqual({ order_by: "-created_at", sub_issue: "true" });
  for (const value of [null, 42, "broken", [], { and: null }, { and: [{ or: [] }] }]) {
    expect(toProjectFilters(JSON.parse(JSON.stringify(value)))).toEqual({});
    expect(toFilterExpression(JSON.parse(JSON.stringify(value)))).toEqual({});
  }
  expect(toProjectFilters({ and: [{ target_date__exact: "none" }, { target_date__exact: "none" }] })).toEqual({});
  expect(toFilterExpression({ target_date: ["none", "2026-10-10"] })).toEqual({});
});

it("F01-15 reads every saved field through the real UI adapter without comparing temporary node IDs", () => {
  const saved: ProjectIssueFilters = {
    state_id: [A, B],
    state_group: ["started"],
    priority: ["none"],
    label_id: ["none", A],
    assignee_id: ["none", "me", A],
    created_by_id: ["me"],
    mention_id: [B],
    subscriber_id: ["me"],
    start_date: ["2026-10-01..", "..2026-10-31"],
    target_date: ["2026-10-10", "2026-10-01..2026-10-31"],
  };
  const internal = workItemFiltersAdapter.toInternal(toFilterExpression(saved));
  expect(internal).not.toBeNull();
  if (!internal) throw new Error("Missing filter tree");
  expect(toProjectFilters(workItemFiltersAdapter.toExternal(internal))).toEqual(saved);
});

it("F01-16 clears the last condition through the real filter instance", () => {
  const emitted: TWorkItemFilterExpression[] = [];
  const instance = new FilterInstance<TWorkItemFilterProperty, TWorkItemFilterExpression>({
    adapter: workItemFiltersAdapter,
    initialExpression: { priority__in: "high" },
    onExpressionChange: (expression) => emitted.push(expression),
  });
  instance.removeCondition(instance.allConditions[0]!.id);
  expect(emitted).toEqual([{}]);
  expect(toProjectFilters(emitted[0]!)).toEqual({});
});

it("F01-17 narrows member fields and retains its required relation", () => {
  expect(
    memberListQuery(
      {
        state_id: [A],
        assignee_id: [A],
        created_by_id: [A],
        mention_id: [A],
        subscriber_id: [A],
        project_id: [A],
        state_group: ["started"],
        priority: ["low"],
        label_id: ["none", B],
        start_date: ["2026-10-01..", "..2026-10-31"],
      },
      { layout: "kanban", group_by: "state_id", sub_group_by: "label_id" },
      "subscribed"
    )
  ).toEqual({
    relation: "subscribed",
    project_id: [A],
    state_group: ["started"],
    priority: ["low"],
    label_id: ["none", B],
    start_date: ["2026-10-01..", "..2026-10-31"],
    order_by: "-created_at",
    sub_issue: "true",
  });
  expect(memberListQuery({ project_id: ["none"], priority: ["broken"] }, {}, "assigned")).toEqual({
    relation: "assigned",
    order_by: "-created_at",
    sub_issue: "true",
  });
});

it.each(["project", "project_id", "unknown", "constructor", "toString", "__proto__"])(
  "F01-18 rejects project grouping %s and its orphan subgroup",
  (group_by) => {
    expect(projectListQuery({}, { layout: "kanban", group_by, sub_group_by: "label_id" })).toEqual({
      order_by: "-created_at",
      sub_issue: "true",
    });
  }
);

it("F01-19 preserves both open range slots through the real UI adapter", () => {
  const expression = { and: [{ start_date__range: "2026-10-01," }, { start_date__range: ",2026-10-31" }] };
  const internal = workItemFiltersAdapter.toInternal(expression);
  if (!internal) throw new Error("Missing filter tree");
  expect(workItemFiltersAdapter.toExternal(internal)).toEqual(expression);
});

it("F01-20 retains a stored subscriber while a real instance edits another condition and saves again", () => {
  const emitted: TWorkItemFilterExpression[] = [];
  const store = new WorkItemFilterStore();
  const instance = store.getOrCreateFilter({
    entityType: EIssuesStoreType.PROJECT,
    entityId: "project",
    initialExpression: { and: [{ subscriber_id__in: "me" }, { priority__in: "high" }] },
    onExpressionChange: (expression) => emitted.push(expression),
  });
  store.updateFilterValueFromSidebar(EIssuesStoreType.PROJECT, "project", {
    property: "priority",
    operator: "in",
    value: ["low"],
  });
  expect(toProjectFilters(emitted.at(-1)!)).toEqual({ subscriber_id: ["me"], priority: ["low"] });
  store.updateFilterValueFromSidebar(EIssuesStoreType.PROJECT, "project", {
    property: "state_group",
    operator: "in",
    value: ["started"],
  });
  expect(toProjectFilters(emitted.at(-1)!)).toEqual({
    subscriber_id: ["me"],
    priority: ["low"],
    state_group: ["started"],
  });
  const priority = instance.allConditions.find((condition) => condition.property === "priority")!;
  instance.updateConditionProperty(priority.id, "label_id", "in");
  instance.updateConditionValue(priority.id, ["none", A]);
  expect(toProjectFilters(emitted.at(-1)!)).toEqual({
    subscriber_id: ["me"],
    label_id: ["none", A],
    state_group: ["started"],
  });
  instance.updateConditionOperator(priority.id, "exact");
  instance.updateConditionValue(priority.id, "none");
  expect(toProjectFilters(emitted.at(-1)!)).toEqual({
    subscriber_id: ["me"],
    label_id: ["none"],
    state_group: ["started"],
  });
  expect(
    toProjectFilters(
      buildWorkItemFilterExpressionFromConditions({
        conditions: [
          { property: "subscriber_id", operator: "in", value: ["me"] },
          { property: "target_date", operator: "range", value: ["", "2026-10-31"] },
        ],
      })!
    )
  ).toEqual({ subscriber_id: ["me"], target_date: ["..2026-10-31"] });
});

it.each([null, { order_by: null }, { order_by: 4 }, { order_by: [] }])(
  "F01-21 defaults damaged display settings %j",
  (display) => {
    const damaged = JSON.parse(JSON.stringify(display));
    expect(projectListQuery({}, damaged)).toEqual({ order_by: "-created_at", sub_issue: "true" });
    expect(memberListQuery({}, damaged, "assigned")).toEqual({
      relation: "assigned",
      order_by: "-created_at",
      sub_issue: "true",
    });
  }
);

it.each([
  ["2026-09-30", false],
  ["2026-10-01", true],
  ["2026-10-31", true],
  ["2026-11-01", false],
  [null, false],
])("F01-22 repeated date AND includes both endpoints and rejects either outside bound: %s", (start_date, want) => {
  const query = projectListQuery(
    toProjectFilters({ and: [{ start_date__range: "2026-10-01," }, { start_date__range: ",2026-10-31" }] })
  );
  expect(issueMembership(listIssue("row", 1, { start_date }), listContext(query))).toBe(want);
});

it.each(["start_date", "target_date"] as const)("fix1 I1 rejects year zero throughout %s boundaries", (field) => {
  for (const value of ["0000-01-01", "0000-01-01..", "..0000-12-31", "0000-01-01..0001-01-01"]) {
    const filters = { [field]: [value, "2026-10-01"], priority: ["high"] };
    const expression = {
      and: [
        { [`${field}__${value.includes("..") ? "range" : "exact"}`]: value.replace("..", ",") },
        { priority__in: "high" },
      ],
    };
    expect.soft(toProjectFilters(expression)).toEqual({ priority: ["high"] });
    expect.soft(toFilterExpression(filters)).toEqual({ and: [{ priority__in: "high" }] });
    expect.soft(projectListQuery(filters)).toEqual({ priority: ["high"], order_by: "-created_at", sub_issue: "true" });
    expect
      .soft(memberListQuery(filters, {}, "assigned"))
      .toEqual({ relation: "assigned", priority: ["high"], order_by: "-created_at", sub_issue: "true" });
  }
});

it.each(["start_date", "target_date"] as const)("fix1 I1 accepts both supported year endpoints in %s", (field) => {
  for (const value of ["0001-01-01", "9999-12-31", "0001-01-01..", "..9999-12-31", "0001-01-01..9999-12-31"]) {
    const filters = { [field]: [value], priority: ["high"] };
    expect(toProjectFilters(toFilterExpression(filters))).toEqual(filters);
    expect(projectListQuery(filters)).toEqual({ ...filters, order_by: "-created_at", sub_issue: "true" });
    expect(memberListQuery(filters, {}, "created")).toEqual({
      ...filters,
      relation: "created",
      order_by: "-created_at",
      sub_issue: "true",
    });
  }
});

it.each(["state_id", "label_id", "assignee_id", "created_by_id", "mention_id", "subscriber_id"] as const)(
  "fix1 I3 canonicalizes accepted UUIDs at each %s vocabulary boundary",
  (field) => {
    const input = { [field]: [A.toUpperCase()] };
    expect.soft(toProjectFilters({ [`${field}__in`]: A.toUpperCase() })).toEqual({ [field]: [A] });
    expect.soft(toFilterExpression(input)).toEqual({ and: [{ [`${field}__in`]: A }] });
    expect.soft(projectListQuery(input)).toEqual({ [field]: [A], order_by: "-created_at", sub_issue: "true" });
  }
);

it("fix1 I3 converted project and member UUID filters match canonical issue rows", () => {
  const row = listIssue("row", 1, { state_id: A, assignee_ids: [A], label_ids: [A], created_by_id: A });
  const query = projectListQuery(
    toProjectFilters(
      toFilterExpression({
        state_id: [A.toUpperCase()],
        label_id: [A.toUpperCase()],
        assignee_id: [A.toUpperCase()],
        created_by_id: [A.toUpperCase()],
      })
    )
  );
  expect.soft(issueMembership(row, listContext(query))).toBe(true);
  const memberQuery = memberListQuery({ project_id: [A.toUpperCase()], label_id: [A.toUpperCase()] }, {}, "created");
  expect
    .soft(memberQuery)
    .toEqual({ relation: "created", project_id: [A], label_id: [A], order_by: "-created_at", sub_issue: "true" });
  expect
    .soft(
      issueMembership(
        { ...row, project_id: A },
        {
          ...listContext(),
          scope: "member",
          workspaceSlug: "ws",
          workspaceID: row.workspace_id,
          userID: A,
          query: memberQuery,
        }
      )
    )
    .toBe(true);
});

it("extracts member project conditions without admitting them to project filters", async () => {
  const { toMemberFilters } = await import("./issue-filters");
  const expression = { and: [{ project_id__in: A.toUpperCase() }, { priority__in: "high" }] };
  expect(toMemberFilters(expression)).toEqual({ project_id: [A], priority: ["high"] });
  expect(toProjectFilters(expression)).toEqual({ priority: ["high"] });
  expect(toMemberFilters({ and: [{ project_id__in: A }, { project_id__exact: A }, { priority__in: "low" }] })).toEqual({
    priority: ["low"],
  });
  expect(
    toMemberFilters(JSON.parse(JSON.stringify({ and: [{ and: [{ project_id__in: A }] }, { priority__in: "high" }] })))
  ).toEqual({ priority: ["high"] });
});
