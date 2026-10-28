/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { expect, test } from "vitest";
import { issuePageOf, listContext, listGroup, listIssue, listRow, pageMeta } from "@/store/issue/fake-issues";
import {
  confirmGroups as confirmWithSources,
  decodePage,
  mergePages,
  pagePartKey,
  type ConfirmedIssueChange,
  type PagedGroups,
} from "./paged-groups";
import { replayGroups } from "./paged-groups.replay";

// Hand-built initial fixtures represent source 0 held before dispatch; decoded
// partial/whole replies have positive identities and never inherit this permission.
const confirmGroups = (...args: Parameters<typeof confirmWithSources>) =>
  confirmWithSources(args[0], args[1], args[2], new Set([0]));

const context = listContext({ group_by: "state_id" });
const before = listIssue("x", 4, { state_id: "A" });
const after = { ...before, state_id: "B", updated_at: "2026-10-28T00:00:00.000001Z" };
const change: ConfirmedIssueChange = { id: "x", beforeSnapshot: { kind: "known", issue: before }, after };
const initial = (): PagedGroups => ({
  total: 9,
  groups: new Map([
    ["A", listGroup(5)],
    ["B", listGroup(4)],
  ]),
});
const whole = (dataA = [before], dataB: typeof dataA = [], countA = 5, countB = 4) =>
  decodePage(
    {
      total: 9,
      groups: [
        { key: "A", total: countA, data: dataA, next_cursor: "A-next" },
        { key: "B", total: countB, data: dataB, next_cursor: "B-next" },
      ],
    },
    pageMeta(),
    context
  );

test("R3-1 I2-3 M3-1 rawBeforeAnswerOverridesAlreadyMergedAfterRowEvidence", () => {
  const confirmed = confirmGroups(initial(), { ...change, drop: { group: "B" } }, context).value;
  const raw = decodePage(
    { ...issuePageOf([listIssue("tail", 10, { state_id: "B" })], "next-B"), total: 4 },
    pageMeta({ group: "B" }, { id: "start", key: 0 }),
    context
  );
  const frozen = structuredClone(raw);
  const merged = mergePages(confirmed, raw, context);
  expect(merged.groups.get("B")?.total).toBe(4);
  const result = replayGroups(merged, change, raw, context);
  expect(result.value.groups.get("B")?.total).toBe(5);
  expect(result.value.groups.get("A")?.total).toBe(4);
  expect(result.value.total).toBe(9);
  expect(result.value.groups.get("B")?.rows).toEqual([listRow("x", 4, after.updated_at), listRow("tail", 10)]);
  expect(result.verifyIds).toEqual(["x"]);
  expect(raw).toEqual(frozen);
});

test("R3-2 I2-1 rawAfterAnswerOnlyPlacesWithNoSecondDelta", () => {
  const raw = whole([], [after], 4, 5);
  const result = replayGroups(mergePages(initial(), raw, context), change, raw, context);
  expect(result.value.total).toBe(9);
  expect([...result.value.groups.values()].map((group) => [group.total, group.rows])).toEqual([
    [4, []],
    [5, [listRow("x", 4, after.updated_at)]],
  ]);
  expect(result.verifyIds).toEqual([]);
});

test("R3-3 rawBeforeVersionAppliesOnceToIndependentCounters", () => {
  const raw = whole();
  const result = replayGroups(mergePages(initial(), raw, context), change, raw, context);
  expect([...result.value.groups.values()].map((group) => group.total)).toEqual([4, 5]);
  expect(result.value.total).toBe(9);
  expect(result.verifyIds).toEqual([]);
});

test("R3-4 I2-6 missingAfterPositionInfersBeforeAndRequestsOneVerification", () => {
  const multiContext = listContext({ group_by: "label_id" });
  const issue = listIssue("x", 4, { label_ids: ["L", "M"] });
  const raw = decodePage({ total: 0, groups: [] }, pageMeta(), multiContext);
  const result = replayGroups(raw, { id: "x", beforeSnapshot: { kind: "absent" }, after: issue }, raw, multiContext);
  expect(result.value.total).toBe(1);
  expect([...result.value.groups.values()].map((group) => [group.total, group.rows])).toEqual([
    [1, [listRow("x", 4)]],
    [1, [listRow("x", 4)]],
  ]);
  expect(result.verifyIds).toEqual(["x"]);
});

test("R3-5 missingBeforePositionInfersAfterAndUnknownRangeOnlyMarksAnsweredAffectedPart", () => {
  const raw = decodePage({ ...issuePageOf([]), total: 4 }, pageMeta({ group: "A" }), context);
  const confirmed = confirmGroups(initial(), change, context).value;
  const result = replayGroups(mergePages(confirmed, raw, context), change, raw, context);
  expect(result.value.groups.get("A")?.total).toBe(4);
  expect(result.value.groups.get("B")).toEqual(confirmed.groups.get("B"));
  expect(result.verifyIds).toEqual(["x"]);
  const far = {
    ...change,
    beforeSnapshot: { kind: "known" as const, issue: { ...before, sort_order: 40 } },
    after: { ...after, sort_order: 50 },
  };
  const partial = decodePage(
    { ...issuePageOf([listIssue("tail", 20)], "next"), total: 4 },
    pageMeta({ group: "B" }, { id: "edge", key: 10 }),
    context
  );
  const merged = mergePages(confirmed, partial, context);
  expect(replayGroups(merged, far, partial, context)).toEqual({
    value: {
      ...merged,
      groups: new Map([
        ["A", merged.groups.get("A")!],
        ["B", { ...merged.groups.get("B")!, stale: true }],
      ]),
    },
    verifyIds: [],
  });
});

test("partialReplayAdjustsOnlyCountersResetByItsAnswer", () => {
  const raw = decodePage(
    { ...issuePageOf([listIssue("tail", 10, { state_id: "B" })], "next"), total: 4 },
    pageMeta({ group: "B" }),
    context
  );
  const confirmed = confirmGroups(initial(), change, context).value;
  expect([confirmed.groups.get("A")?.total, confirmed.groups.get("B")?.total, confirmed.total]).toEqual([4, 5, 9]);
  const result = replayGroups(mergePages(confirmed, raw, context), change, raw, context).value;
  expect([result.groups.get("A")?.total, result.groups.get("B")?.total, result.total]).toEqual([4, 5, 9]);
  expect(result.groups.get("A")).toEqual(confirmed.groups.get("A"));
});

test.each([
  ["A", "B"],
  ["B", "A"],
])("twoPartsInFlightKeepEachOthersConfirmedState %s then %s", (first, second) => {
  const answers = new Map([
    ["A", decodePage({ ...issuePageOf([before], "more-A"), total: 5 }, pageMeta({ group: "A" }), context)],
    [
      "B",
      decodePage({ ...issuePageOf([listIssue("tail", 10)], "more-B"), total: 4 }, pageMeta({ group: "B" }), context),
    ],
  ]);
  let value = confirmGroups(initial(), change, context).value;
  for (const key of [first, second]) {
    const raw = answers.get(key)!;
    value = replayGroups(mergePages(value, raw, context), change, raw, context).value;
  }
  expect(value.total).toBe(9);
  expect([...value.groups.values()].map((group) => [group.total, group.rows])).toEqual([
    [4, []],
    [5, [listRow("x", 4, after.updated_at), listRow("tail", 10)]],
  ]);
});

test("oneCellReplayPreservesUnansweredOuterSubgroupTotalsAndRows", () => {
  const lane = listContext({ group_by: "label_id", sub_group_by: "assignee_id" });
  const old = listIssue("x", 4, { label_ids: ["L", "M"], assignee_ids: ["U"] });
  const next = { ...old, label_ids: ["M", "N"], updated_at: after.updated_at };
  const value: PagedGroups = {
    total: 9,
    subgroupTotals: new Map([["U", 8]]),
    groups: new Map(
      ["L", "M", "N"].map((key) => [key, { ...listGroup(5), subgroups: new Map([["U", listGroup(4)]]) }])
    ),
  };
  const edit: ConfirmedIssueChange = { id: "x", beforeSnapshot: { kind: "known", issue: old }, after: next };
  const confirmed = confirmGroups(value, edit, lane).value;
  const raw = decodePage(
    { ...issuePageOf([listIssue("tail", 10)], "more"), total: 4 },
    pageMeta({ group: "N", subgroup: "U" }),
    lane
  );
  const result = replayGroups(mergePages(confirmed, raw, lane), edit, raw, lane).value;
  expect(result.total).toBe(9);
  expect(result.subgroupTotals).toEqual(new Map([["U", 8]]));
  expect(result.groups.get("L")).toEqual(confirmed.groups.get("L"));
  expect(result.groups.get("M")).toEqual(confirmed.groups.get("M"));
  expect(result.groups.get("N")?.total).toBe(6);
  expect(result.groups.get("N")?.subgroups?.get("U")).toEqual({
    total: 5,
    rows: [listRow("x", 4, after.updated_at), listRow("tail", 10)],
    nextCursor: "more",
    boundary: { id: "tail", key: 10 },
    cursorBoundary: { id: "tail", key: 10 },
    stale: false,
  });
});

test("R-TAIL finalPageCannotProveAbsenceBeforeItsLower", () => {
  const raw = decodePage({ ...issuePageOf([]), total: 4 }, pageMeta({ group: "A" }, { id: "edge", key: 10 }), context);
  const result = replayGroups(mergePages(initial(), raw, context), change, raw, context);
  expect(result.value.groups.get("A")?.total).toBe(4);
  expect(result.value.groups.get("A")?.stale).toBe(true);
  expect(result.verifyIds).toEqual([]);
});

test("R-DROP M3-4 mergeKeepsExpandedWindowDeduplicatesAndUsesServerCursorLower", () => {
  const base: PagedGroups = { total: 5, groups: new Map([["", listGroup(5, [listRow("edge", 10)])]]) };
  const ctx = listContext();
  const dropped = confirmGroups(
    base,
    { id: "x", beforeSnapshot: { kind: "absent" }, after: listIssue("x", 100), drop: { group: "" } },
    ctx
  ).value;
  const raw = decodePage(
    { ...issuePageOf([listIssue("middle", 20)], "cursor-2"), total: 6 },
    pageMeta({ group: "" }, dropped.groups.get("")!.cursorBoundary),
    ctx
  );
  expect(raw.read.ranges.get(pagePartKey({ group: "" }))?.lower).toEqual({ id: "edge", key: 10 });
  const merged = mergePages(dropped, raw, ctx);
  expect(merged.groups.get("")).toEqual({
    total: 6,
    rows: [listRow("edge", 10), listRow("middle", 20), listRow("x", 100)],
    nextCursor: "cursor-2",
    boundary: { id: "x", key: 100 },
    cursorBoundary: { id: "middle", key: 20 },
    stale: false,
  });
  const tail = decodePage(
    { ...issuePageOf([listIssue("x", 100)]), total: 6 },
    pageMeta({ group: "" }, merged.groups.get("")!.cursorBoundary),
    ctx
  );
  expect(mergePages(merged, tail, ctx).groups.get("")?.rows).toEqual([
    listRow("edge", 10),
    listRow("middle", 20),
    listRow("x", 100),
  ]);
});

test("mergeFirstPartReplacesOnlyItsRowsAndWholeClearsOmittedGroups", () => {
  const value = initial();
  value.groups.get("A")!.rows = [listRow("old", 1)];
  value.groups.get("B")!.rows = [listRow("other", 2)];
  const raw = decodePage({ ...issuePageOf([before]), total: 5 }, pageMeta({ group: "A" }), context);
  const merged = mergePages(value, raw, context);
  expect(merged.groups.get("A")?.rows).toEqual([listRow("x", 4)]);
  expect(merged.groups.get("B")).toEqual(value.groups.get("B"));
  const empty = decodePage({ total: 0, groups: [] }, pageMeta(), context);
  const cleared = mergePages(value, empty, context);
  expect(cleared.total).toBe(0);
  expect([...cleared.groups.values()].every((group) => group.total === 0 && group.rows.length === 0)).toBe(true);
  expect(value.groups.get("A")?.rows).toEqual([listRow("old", 1)]);
});

test.each(["before", "after", "unknown"] as const)("D-204 rawDeletionEvidence=%s", (kind) => {
  const deletion: ConfirmedIssueChange = { ...change, after: null };
  const raw =
    kind === "before"
      ? whole()
      : decodePage(
          { ...issuePageOf([]), total: 4 },
          pageMeta({ group: "A" }, kind === "unknown" ? { id: "edge", key: 10 } : null),
          context
        );
  const result = replayGroups(mergePages(initial(), raw, context), deletion, raw, context);
  expect(result.value.groups.get("A")?.total).toBe(4);
  expect(result.value.groups.get("A")?.rows).toEqual([]);
  expect(result.value.groups.get("A")?.stale).toBe(kind === "unknown");
  expect(result.value.total).toBe(kind === "before" ? 8 : 9);
  expect(result.verifyIds).toEqual([]);
});

test("unknownWholeRangeMarksOldAndNewGroupsButLeavesUnrelatedGroupFresh", () => {
  const raw = decodePage(
    {
      total: 12,
      groups: ["A", "B", "C"].map((key) => ({
        key,
        total: 4,
        data: [listIssue(key, 10, { state_id: key })],
        next_cursor: `next-${key}`,
      })),
    },
    pageMeta(),
    context
  );
  const edit: ConfirmedIssueChange = {
    ...change,
    beforeSnapshot: { kind: "known", issue: { ...before, sort_order: 40 } },
    after: { ...after, sort_order: 50 },
  };
  const result = replayGroups(raw, edit, raw, context).value;
  expect([...result.groups].map(([key, group]) => [key, group.total, group.stale])).toEqual([
    ["A", 4, true],
    ["B", 4, true],
    ["C", 4, false],
  ]);
  expect(result.groups.get("C")).toEqual(raw.groups.get("C"));
});

test("provenAfterAnswerDoesNotNeedUnknownBeforeToPlaceWithoutStale", () => {
  const raw = whole([], [after], 4, 5);
  const result = replayGroups(raw, { ...change, beforeSnapshot: { kind: "unknown" } }, raw, context);
  expect(result).toMatchObject({ value: raw, verifyIds: [] });
});

test("partialDeletionDoesNotSubtractUnansweredOuterAndLaneCountersAgain", () => {
  const lane = listContext({ group_by: "label_id", sub_group_by: "assignee_id" });
  const issue = listIssue("x", 4, { label_ids: ["L", "M"], assignee_ids: ["U", "V"] });
  const value: PagedGroups = {
    total: 9,
    subgroupTotals: new Map([
      ["U", 8],
      ["V", 7],
    ]),
    groups: new Map(
      ["L", "M"].map((key) => [
        key,
        {
          ...listGroup(5),
          subgroups: new Map([
            ["U", listGroup(4, [listRow("x", 4)])],
            ["V", listGroup(3, [listRow("x", 4)])],
          ]),
        },
      ])
    ),
  };
  const deletion: ConfirmedIssueChange = { id: "x", beforeSnapshot: { kind: "known", issue }, after: null };
  const confirmed = confirmGroups(value, deletion, lane).value;
  expect(confirmed.total).toBe(8);
  expect(confirmed.subgroupTotals).toEqual(
    new Map([
      ["U", 7],
      ["V", 6],
    ])
  );
  const raw = decodePage({ ...issuePageOf([issue], "more"), total: 4 }, pageMeta({ group: "L", subgroup: "U" }), lane);
  const result = replayGroups(mergePages(confirmed, raw, lane), deletion, raw, lane);
  expect(result.value.total).toBe(8);
  expect(result.value.subgroupTotals).toEqual(
    new Map([
      ["U", 7],
      ["V", 6],
    ])
  );
  expect(result.value.groups.get("M")).toEqual(confirmed.groups.get("M"));
  expect(result.value.groups.get("L")?.total).toBe(4);
  expect(result.value.groups.get("L")?.subgroups?.get("V")).toEqual(confirmed.groups.get("L")?.subgroups?.get("V"));
  expect(result.value.groups.get("L")?.subgroups?.get("U")).toEqual({
    ...confirmed.groups.get("L")!.subgroups!.get("U")!,
    nextCursor: "more",
    cursorBoundary: { id: "x", key: 4 },
  });
  expect(result.verifyIds).toEqual([]);
});

test("memberReplayKeepsSameUserWorkspacesIsolated", () => {
  const issue = listIssue("x", 4, { assignee_ids: ["target"], workspace_id: "id-ws" });
  for (const workspaceID of ["id-ws", "other-ws"]) {
    const member = {
      ...listContext(),
      scope: "member" as const,
      workspaceSlug: workspaceID === "id-ws" ? "one" : "two",
      workspaceID,
      userID: "target",
      query: { relation: "assigned" as const, order_by: "sort_order" as const },
    };
    const raw = decodePage(
      { ...issuePageOf([listIssue("held", 10, { workspace_id: workspaceID })], "more"), total: 5 },
      pageMeta(),
      member
    );
    const result = replayGroups(raw, { id: "x", beforeSnapshot: { kind: "absent" }, after: issue }, raw, member);
    if (workspaceID === "other-ws") {
      expect(result).toMatchObject({ value: raw, verifyIds: [] });
      expect(replayGroups(raw, { id: "x", beforeSnapshot: { kind: "unknown" }, after: issue }, raw, member)).toEqual({
        value: raw,
        verifyIds: [],
      });
    } else {
      expect(result.value.total).toBe(6);
      expect(result.value.groups.get("")?.rows).toEqual([listRow("x", 4), listRow("held", 10)]);
      expect(result.verifyIds).toEqual(["x"]);
    }
  }
});

test.each([
  ["L", "M"],
  ["M", "L"],
])("I1 twoPartialAfterAnswersBeforeConfirmationKeepSharedSource %s then %s", (first, second) => {
  const labels = listContext({ group_by: "label_id" });
  const issue = listIssue("x", 4, { label_ids: ["L", "M"], updated_at: after.updated_at });
  let value: PagedGroups = {
    total: 9,
    groups: new Map([
      ["L", listGroup(5)],
      ["M", listGroup(4)],
    ]),
  };
  for (const group of [first, second]) {
    const answer = decodePage(
      { ...issuePageOf([issue], "more"), total: group === "L" ? 6 : 5 },
      pageMeta({ group }),
      labels
    );
    value = mergePages(value, answer, labels);
  }
  const confirmed = confirmGroups(value, { id: "x", beforeSnapshot: { kind: "absent" }, after: issue }, labels);
  expect([
    confirmed.value.total,
    confirmed.value.groups.get("L")?.total,
    confirmed.value.groups.get("M")?.total,
  ]).toEqual([10, 6, 5]);
  expect(confirmed.verifyIds).toEqual([]);
  expect([...confirmed.value.groups.values()].map((group) => group.rows)).toEqual([
    [listRow("x", 4, after.updated_at)],
    [listRow("x", 4, after.updated_at)],
  ]);
});

test("I1 directConfirmationReturnsOneVerificationForMissingRowsInMultipleResetCounters", () => {
  const labels = listContext({ group_by: "label_id" });
  const issue = listIssue("x", 4, { label_ids: ["L", "M"], updated_at: after.updated_at });
  let value: PagedGroups = {
    total: 9,
    groups: new Map([
      ["L", listGroup(5)],
      ["M", listGroup(4)],
    ]),
  };
  for (const group of ["L", "M"])
    value = mergePages(
      value,
      decodePage(
        { ...issuePageOf([listIssue("tail", 10)], "more"), total: group === "L" ? 5 : 4 },
        pageMeta({ group }),
        labels
      ),
      labels
    );
  const result = confirmGroups(value, { id: "x", beforeSnapshot: { kind: "absent" }, after: issue }, labels);
  expect([result.value.total, result.value.groups.get("L")?.total, result.value.groups.get("M")?.total]).toEqual([
    10, 6, 5,
  ]);
  expect(result.verifyIds).toEqual(["x"]);
});

test("I1 continuationAbsenceCannotProveDeleteAndWholeRefreshReplacesCounterEvidence", () => {
  const value: PagedGroups = { total: 1, groups: new Map([["A", listGroup(1, [listRow("x", 4)])]]) };
  const raw = decodePage({ ...issuePageOf([]), total: 0 }, pageMeta({ group: "A" }, { id: "edge", key: 10 }), context);
  const deletion: ConfirmedIssueChange = { ...change, after: null };
  const confirmed = confirmGroups(mergePages(value, raw, context), deletion, context);
  expect([confirmed.value.total, confirmed.value.groups.get("A")?.total]).toEqual([0, 0]);
  expect(confirmed.value.groups.get("A")?.stale).toBe(true);
  expect(confirmed.verifyIds).toEqual([]);
  expect(confirmGroups(confirmed.value, deletion, context)).toEqual(confirmed);
  const freshWhole = decodePage({ total: 0, groups: [] }, pageMeta(), context);
  const refreshed = mergePages(confirmed.value, freshWhole, context);
  expect(refreshed.counterEvidence).toBeUndefined();
  expect(refreshed.groups.get("A")?.stale).toBe(false);
  expect(confirmGroups(refreshed, deletion, context).value.total).toBe(0);
});
