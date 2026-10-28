/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, test } from "vitest";
import { listContext, listGroup, listIssue, listRow } from "@/store/issue/fake-issues";
import { confirmGroups, type PagedGroups, type ConfirmedIssueChange } from "./paged-groups";

const afterTime = "2026-10-28T00:00:00.000001Z";
// G01–16: frozen old/new membership × row presence × new key inside/outside.
// The original row wins over frozen membership; complete absence wins over both.
const cases = [
  ["G01", true, true, true, true, 5, 5, true],
  ["G02", true, true, true, false, 5, 5, false],
  ["G03", true, true, false, true, 5, 6, true],
  ["G04", true, true, false, false, 5, 6, false],
  ["G05", true, false, true, true, 4, 4, false],
  ["G06", true, false, true, false, 4, 4, false],
  ["G07", true, false, false, true, 4, 5, false],
  ["G08", true, false, false, false, 4, 5, false],
  ["G09", false, true, true, true, 5, 5, true],
  ["G10", false, true, true, false, 5, 5, false],
  ["G11", false, true, false, true, 6, 6, true],
  ["G12", false, true, false, false, 6, 6, false],
  ["G13", false, false, true, true, 4, 4, false],
  ["G14", false, false, true, false, 4, 4, false],
  ["G15", false, false, false, true, 5, 5, false],
  ["G16", false, false, false, false, 5, 5, false],
] as const;
for (const [name, old, next, present, inside, unfinishedTotal, completeTotal, visible] of cases) {
  for (const complete of [false, true])
    test(`${name} complete=${complete}`, () => {
      const before = listIssue("x", 4, { state_id: old ? "A" : "B" });
      const after = listIssue("x", inside ? 6 : 20, { state_id: next ? "A" : "B", updated_at: afterTime });
      const value: PagedGroups = {
        total: 9,
        groups: new Map([
          ["A", listGroup(5, present ? [listRow("x", 4)] : [], complete)],
          ["B", listGroup(4, [], false, 0)],
        ]),
      };
      const original = structuredClone(value);
      const result = confirmGroups(
        value,
        { id: "x", beforeSnapshot: { kind: "known", issue: before }, after },
        listContext({ group_by: "state_id" })
      );
      expect(result).toEqual({
        value: {
          total: 9,
          groups: new Map([
            [
              "A",
              {
                ...original.groups.get("A")!,
                total: complete ? completeTotal : unfinishedTotal,
                rows: (complete ? next : visible) ? [listRow("x", inside ? 6 : 20, afterTime)] : [],
              },
            ],
            ["B", { ...original.groups.get("B")!, total: old === next ? 4 : old ? 5 : 3 }],
          ]),
        },
        verifyIds: [],
      });
      expect(value).toEqual(original);
    });
}

test("I2-8 createsOmittedEmptyGroupAndLaneCell", () => {
  const after = listIssue("new", 30, { label_ids: ["L"], assignee_ids: ["U"] });
  const result = confirmGroups(
    { total: 0, groups: new Map(), subgroupTotals: new Map() },
    { id: "new", beforeSnapshot: { kind: "absent" }, after },
    listContext({ group_by: "label_id", sub_group_by: "assignee_id" })
  );
  expect(result).toEqual({
    value: {
      total: 1,
      subgroupTotals: new Map([["U", 1]]),
      groups: new Map([
        [
          "L",
          {
            total: 1,
            rows: [],
            nextCursor: null,
            boundary: null,
            cursorBoundary: null,
            stale: false,
            subgroups: new Map([
              [
                "U",
                {
                  total: 1,
                  rows: [listRow("new", 30)],
                  nextCursor: null,
                  boundary: null,
                  cursorBoundary: null,
                  stale: false,
                },
              ],
            ]),
          },
        ],
      ]),
    },
    verifyIds: [],
  });
});

test("multivalueCountsOuterGroupsAndSubgroupTotalsIndependently", () => {
  const context = listContext({ group_by: "label_id", sub_group_by: "assignee_id" });
  const before = listIssue("x", 4, { label_ids: ["L", "M"], assignee_ids: ["U", "V"] });
  const after = { ...before, label_ids: ["M", "N"], assignee_ids: ["V"], updated_at: afterTime };
  const value: PagedGroups = {
    total: 9,
    subgroupTotals: new Map([
      ["U", 6],
      ["V", 7],
    ]),
    groups: new Map(
      ["L", "M", "N"].map((key) => [
        key,
        {
          ...listGroup(key === "N" ? 3 : 5),
          subgroups: new Map([
            ["U", listGroup(2)],
            ["V", listGroup(3)],
          ]),
        },
      ])
    ),
  };
  const result = confirmGroups(
    value,
    { id: "x", beforeSnapshot: { kind: "known", issue: before }, after },
    context
  ).value;
  expect(result.total).toBe(9);
  expect(result.subgroupTotals).toEqual(
    new Map([
      ["U", 5],
      ["V", 7],
    ])
  );
  expect(
    [...result.groups].map(([key, group]) => [
      key,
      group.total,
      [...group.subgroups!].map(([sub, cell]) => [sub, cell.total, cell.rows]),
    ])
  ).toEqual([
    [
      "L",
      4,
      [
        ["U", 1, []],
        ["V", 2, []],
      ],
    ],
    [
      "M",
      5,
      [
        ["U", 1, []],
        ["V", 3, [listRow("x", 4, afterTime)]],
      ],
    ],
    [
      "N",
      4,
      [
        ["U", 2, []],
        ["V", 4, [listRow("x", 4, afterTime)]],
      ],
    ],
  ]);
});

test.each([false, true])("archiveAndUnarchiveRespectTheListScope archived=%s", (archived) => {
  const before = listIssue("x", 4, { archived_at: archived ? afterTime : null });
  const after = { ...before, archived_at: archived ? null : afterTime, updated_at: afterTime };
  for (const queryArchived of [false, true]) {
    const belonged = archived === queryArchived;
    const value: PagedGroups = { total: 5, groups: new Map([["", listGroup(5, belonged ? [listRow("x", 4)] : [])]]) };
    const result = confirmGroups(
      value,
      { id: "x", beforeSnapshot: { kind: "known", issue: before }, after },
      listContext({ archived: String(queryArchived) })
    ).value;
    expect(result.total).toBe(belonged ? 4 : 6);
    expect(result.groups.get("")).toEqual({
      ...value.groups.get("")!,
      total: belonged ? 4 : 6,
      rows: belonged ? [] : [listRow("x", 4, afterTime)],
    });
  }
});

test("I2-1 alreadyAnsweredVersionOnlyPlacesAndDoesNotDoubleCount", () => {
  const after = listIssue("x", 4, { updated_at: afterTime });
  const value: PagedGroups = { total: 7, groups: new Map([["", listGroup(7, [listRow("x", 4, afterTime)])]]) };
  expect(confirmGroups(value, { id: "x", beforeSnapshot: { kind: "absent" }, after }, listContext())).toEqual({
    value,
    verifyIds: [],
  });
});

test("unknownBeforeMarksOnlyPotentiallyAffectedPartsWithoutGuessingCounts", () => {
  const value: PagedGroups = {
    total: 9,
    groups: new Map([
      ["A", listGroup(5)],
      ["B", listGroup(4)],
    ]),
  };
  const result = confirmGroups(
    value,
    { id: "x", beforeSnapshot: { kind: "unknown" }, after: listIssue("x", 4, { state_id: "B" }) },
    listContext({ group_by: "state_id" })
  ).value;
  expect(result.total).toBe(9);
  expect([...result.groups.values()].map((group) => [group.total, group.stale])).toEqual([
    [5, true],
    [4, true],
  ]);
  const known = confirmGroups(
    value,
    {
      id: "x",
      beforeSnapshot: { kind: "known", issue: listIssue("x", 4, { state_id: "A" }) },
      after: listIssue("x", 5, { state_id: "A", updated_at: afterTime }),
    },
    listContext({ group_by: "state_id" })
  ).value;
  expect(known.groups.get("B")).toEqual(value.groups.get("B"));
});

test.each([{ mention_id: ["me"] }, { subscriber_id: ["me"] }])("unobservableProjectFilterMarksStale %j", (query) => {
  const value: PagedGroups = { total: 5, groups: new Map([["", listGroup(5)]]) };
  const result = confirmGroups(
    value,
    { id: "x", beforeSnapshot: { kind: "absent" }, after: listIssue("x", 4) },
    listContext(query)
  ).value;
  expect(result).toEqual({ ...value, groups: new Map([["", { ...value.groups.get("")!, stale: true }]]) });
});

test("I2-9 dropOutsideWindowExtendsDisplayWithoutMovingServerCursor", () => {
  const value: PagedGroups = { total: 5, groups: new Map([["", listGroup(5, [listRow("edge", 10)])]]) };
  const result = confirmGroups(
    value,
    { id: "x", beforeSnapshot: { kind: "absent" }, after: listIssue("x", 100), drop: { group: "", afterId: "edge" } },
    listContext()
  ).value;
  expect(result.groups.get("")).toEqual({
    ...value.groups.get("")!,
    total: 6,
    rows: [listRow("edge", 10), listRow("x", 100)],
    boundary: { id: "x", key: 100 },
  });
});

test("I2-4 movingLastRowNeverRecomputesDisplayBoundary", () => {
  const before = listIssue("edge", 10);
  const value: PagedGroups = {
    total: 5,
    groups: new Map([["", listGroup(5, [listRow("a", 2), listRow("edge", 10)])]]),
  };
  const result = confirmGroups(
    value,
    {
      id: "edge",
      beforeSnapshot: { kind: "known", issue: before },
      after: { ...before, sort_order: 20, updated_at: afterTime },
    },
    listContext()
  ).value;
  expect(result.groups.get("")).toEqual({ ...value.groups.get("")!, rows: [listRow("a", 2)] });
});

test("D-204 confirmationDeletesWithoutAVersionAndSecondConfirmationIsInert", () => {
  const value: PagedGroups = { total: 1, groups: new Map([["", listGroup(1, [listRow("x", 4)], true)]]) };
  const change: ConfirmedIssueChange = {
    id: "x",
    beforeSnapshot: { kind: "known", issue: listIssue("x", 4) },
    after: null,
  };
  const once = confirmGroups(value, change, listContext()).value;
  expect(once).toEqual({ total: 0, groups: new Map([["", { ...value.groups.get("")!, total: 0, rows: [] }]]) });
  expect(confirmGroups(once, change, listContext()).value).toEqual(once);
});

test("nullCreatorUsesNoneGroupAndNeverMatchesMeFilter", () => {
  const after = listIssue("x", 4, { created_by_id: null });
  const created = confirmGroups(
    { total: 0, groups: new Map() },
    { id: "x", beforeSnapshot: { kind: "absent" }, after },
    listContext({ group_by: "created_by_id" })
  ).value;
  expect([...created.groups.keys()]).toEqual(["none"]);
  expect(created.total).toBe(1);
  expect(
    confirmGroups(
      { total: 0, groups: new Map() },
      { id: "x", beforeSnapshot: { kind: "absent" }, after },
      listContext({ created_by_id: ["me"] })
    ).value.total
  ).toBe(0);
});

test.each([
  [{ priority: ["high"] }, { priority: "high" }, true],
  [{ priority: ["high"] }, { priority: "none" }, false],
  [{ state_id: ["S"] }, { state_id: "S" }, true],
  [{ state_id: ["S"] }, { state_id: "T" }, false],
  [{ label_id: ["none", "L"] }, { label_ids: [] }, true],
  [{ label_id: ["L"] }, { label_ids: ["L", "M"] }, true],
  [{ label_id: ["L"] }, { label_ids: ["M"] }, false],
  [{ assignee_id: ["me"] }, { assignee_ids: ["actor"] }, true],
  [{ assignee_id: ["none"] }, { assignee_ids: [] }, true],
  [{ created_by_id: ["me"] }, { created_by_id: "actor" }, true],
  [{ sub_issue: "false" }, { parent_id: "parent" }, false],
  [
    { start_date: ["2026-10-01..", "..2026-10-31"], target_date: ["none"] },
    { start_date: "2026-10-10", target_date: null },
    true,
  ],
  [{ start_date: ["2026-10-01..", "..2026-10-31"] }, { start_date: "2026-11-01" }, false],
  [{ target_date: ["2026-10-10"] }, { target_date: null }, false],
  [{ target_date: ["2026-10-10"] }, { target_date: "2026-10-10" }, true],
] satisfies [
  import("@/services/issue/issue-list.service").ProjectIssueQuery,
  Partial<import("@nerve/api-client").Issue>,
  boolean,
][])("rowFilters %j %j", (query, fields, belongs) => {
  const value: PagedGroups = { total: 5, groups: new Map([["", listGroup(5)]]) };
  const result = confirmGroups(
    value,
    { id: "x", beforeSnapshot: { kind: "absent" }, after: listIssue("x", 4, fields) },
    listContext(query)
  ).value;
  expect(result.total).toBe(belongs ? 6 : 5);
  expect(result.groups.get("")?.stale).toBe(false);
  expect(result.groups.get("")?.rows).toEqual(belongs ? [listRow("x", 4)] : []);
});

test.each(["assigned", "created", "subscribed"] as const)(
  "memberRelationUsesTargetUserAndSubscribedStaysUnknown %s",
  (relation) => {
    const context: import("./paged-groups").ListContext = {
      ...listContext(),
      scope: "member",
      userID: "target",
      workspaceSlug: "ws",
      workspaceID: "id-ws",
      query: { relation, group_by: "project_id", order_by: "sort_order", project_id: ["A"] },
    };
    const value: PagedGroups = { total: 5, groups: new Map([["A", listGroup(5)]]) };
    const result = confirmGroups(
      value,
      {
        id: "x",
        beforeSnapshot: { kind: "absent" },
        after: listIssue("x", 4, { assignee_ids: ["target"], created_by_id: "target" }),
      },
      context
    ).value;
    expect(result.total).toBe(relation === "subscribed" ? 5 : 6);
    expect(result.groups.get("A")?.stale).toBe(relation === "subscribed");
    const excluded = confirmGroups(
      value,
      { id: "x", beforeSnapshot: { kind: "absent" }, after: listIssue("x", 4, { project_id: "B" }) },
      context
    ).value;
    expect(excluded).toEqual(value);
  }
);

test("dropIntoFullCalendarCellAndEmptyLaneStaysVisible", () => {
  const context = listContext({ group_by: "target_date", sub_group_by: "priority" });
  const value: PagedGroups = {
    total: 4,
    groups: new Map([
      ["2026-10-29", { ...listGroup(4), subgroups: new Map([["high", listGroup(4, [listRow("edge", 10)])]]) }],
    ]),
    subgroupTotals: new Map([["high", 4]]),
  };
  const result = confirmGroups(
    value,
    {
      id: "x",
      beforeSnapshot: { kind: "absent" },
      after: listIssue("x", 100, { target_date: "2026-10-29", priority: "high" }),
      drop: { group: "2026-10-29", subgroup: "high" },
    },
    context
  ).value;
  expect(result.total).toBe(5);
  expect(result.subgroupTotals).toEqual(new Map([["high", 5]]));
  expect(result.groups.get("2026-10-29")?.subgroups?.get("high")).toEqual({
    ...value.groups.get("2026-10-29")!.subgroups!.get("high")!,
    total: 5,
    rows: [listRow("edge", 10), listRow("x", 100)],
    boundary: { id: "x", key: 100 },
  });
});

test("ownAfterRowInDestinationDoesNotHideOldMembershipOutsideSourceWindow", () => {
  const before = listIssue("x", 40, { state_id: "A" });
  const after = { ...before, state_id: "B", sort_order: 4, updated_at: afterTime };
  const value: PagedGroups = {
    total: 9,
    groups: new Map([
      ["A", listGroup(5)],
      ["B", listGroup(5, [listRow("x", 4, afterTime)])],
    ]),
  };
  const result = confirmGroups(
    value,
    { id: "x", beforeSnapshot: { kind: "known", issue: before }, after },
    listContext({ group_by: "state_id" })
  ).value;
  expect([result.groups.get("A")?.total, result.groups.get("B")?.total, result.total]).toEqual([4, 5, 9]);
});

test("memberConfirmationKeepsSameUserWorkspacesIsolated", () => {
  const value: PagedGroups = { total: 5, groups: new Map([["", listGroup(5, [listRow("held", 2)])]]) };
  const issue = listIssue("x", 4, { assignee_ids: ["target"], workspace_id: "id-ws" });
  for (const workspaceID of ["id-ws", "other-ws"]) {
    const context = {
      ...listContext(),
      scope: "member" as const,
      workspaceSlug: workspaceID === "id-ws" ? "one" : "two",
      workspaceID,
      userID: "target",
      query: { relation: "assigned" as const, order_by: "sort_order" as const },
    };
    const result = confirmGroups(value, { id: "x", beforeSnapshot: { kind: "absent" }, after: issue }, context);
    if (workspaceID === "other-ws") {
      expect(result).toEqual({ value, verifyIds: [] });
      expect(confirmGroups(value, { id: "x", beforeSnapshot: { kind: "unknown" }, after: issue }, context)).toEqual({
        value,
        verifyIds: [],
      });
    } else
      expect(result.value).toEqual({
        total: 6,
        groups: new Map([["", { ...value.groups.get("")!, total: 6, rows: [listRow("held", 2), listRow("x", 4)] }]]),
      });
  }
});
