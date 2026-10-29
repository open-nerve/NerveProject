/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueListPage } from "@nerve/api-client";
import { expect, test } from "vitest";
import { issueOf, issuePageOf } from "@/store/issue/fake-issues";
import { decodePage, pagePartKey, type ListContext, type PageReadMeta } from "./paged-groups";

const context: ListContext = {
  scope: "project",
  projectID: "A",
  query: { order_by: "sort_order" },
  currentUserID: "actor",
  catalog: { states: new Map(), labels: new Map(), members: new Map() },
  pendingIDs: new Set(),
};
const first = (): PageReadMeta => ({
  sourceID: 1,
  part: null,
  ranges: new Map(),
  first: true,
  resetCounters: [],
  rowParts: [],
});
const row = (id: string, key: number) => ({ ...issueOf(id), sort_order: key });
const part = { group: "high", subgroup: "label" };
const partKey = pagePartKey(part);
function continuation(): PageReadMeta {
  return {
    sourceID: 2,
    part: { ...part },
    first: false,
    resetCounters: [],
    rowParts: [],
    ranges: new Map([
      [
        partKey,
        {
          lower: { id: "before", key: 10 },
          upper: null,
          complete: false,
        },
      ],
    ]),
  };
}

test("decodesFlatPageIntoRowsWithoutRetainingIssues", () => {
  const page = issuePageOf([row("a", 1), row("b", 2)], "opaque:do/not/parse");
  page.total = 8;
  const meta = first();
  const answer = decodePage(page, meta, context);
  expect(answer.total).toBe(8);
  expect(answer.groups.get("")).toEqual({
    total: 8,
    rows: [
      { id: "a", key: 1, updatedAt: issueOf("a").updated_at },
      { id: "b", key: 2, updatedAt: issueOf("b").updated_at },
    ],
    nextCursor: "opaque:do/not/parse",
    boundary: { id: "b", key: 2 },
    cursorBoundary: { id: "b", key: 2 },
    stale: false,
  });
  expect(answer.read.ranges.get(pagePartKey({ group: "" }))).toEqual({
    lower: null,
    upper: { id: "b", key: 2 },
    complete: false,
  });
  expect(answer.read.resetCounters).toEqual([{ kind: "outer" }, { kind: "group", group: "" }]);
  expect(answer.read.rowParts).toEqual([{ group: "" }]);
  page.data![0]!.updated_at = "2030-01-01T00:00:00Z";
  page.data![0]!.sort_order = 30;
  meta.ranges.clear();
  expect(answer.groups.get("")?.rows[0]).toEqual({ id: "a", key: 1, updatedAt: "2026-10-28T00:00:00Z" });
});

test("decodesMissingGroupsAsEmpty", () => {
  const meta = first();
  meta.rowParts = [{ group: "low" }, { group: "high", subgroup: "missing" }];
  meta.resetCounters = [{ kind: "subgroup", subgroup: "missing" }];
  const answer = decodePage({ total: 0, groups: [], subgroup_totals: [] }, meta, context);
  const empty = { total: 0, rows: [], nextCursor: null, boundary: null, cursorBoundary: null, stale: false };
  expect(answer.groups.get("low")).toEqual(empty);
  expect(answer.groups.get("high")?.subgroups?.get("missing")).toEqual(empty);
  expect(answer.subgroupTotals?.get("missing")).toBe(0);
  expect(answer.read.ranges.get(pagePartKey({ group: "high", subgroup: "missing" }))).toEqual({
    lower: null,
    upper: null,
    complete: true,
  });
});

test("selectsGroupedAndLaneBranchesByPresenceEvenWhenEmpty", () => {
  const page: IssueListPage = {
    total: 4,
    groups: [
      { key: "high", total: 3, subgroups: [{ key: "label", total: 2, data: [row("a", 1)], next_cursor: "more" }] },
      { key: "low", total: 1, subgroups: [] },
    ],
    subgroup_totals: [{ key: "label", total: 2 }],
  };
  const answer = decodePage(page, first(), context);
  expect(answer.groups.has("")).toBe(false);
  expect(answer.groups.get("high")?.total).toBe(3);
  expect(answer.groups.get("high")?.rows).toEqual([]);
  expect(answer.groups.get("low")?.subgroups).toEqual(new Map());
  expect(answer.groups.get("high")?.subgroups?.get("label")?.nextCursor).toBe("more");
  expect(answer.subgroupTotals).toEqual(new Map([["label", 2]]));
  expect(answer.read.rowParts).toEqual([part]);
  expect(answer.read.resetCounters).toEqual([
    { kind: "outer" },
    { kind: "group", group: "high" },
    { kind: "cell", ...part },
    { kind: "group", group: "low" },
    { kind: "subgroup", subgroup: "label" },
  ]);
  expect(decodePage({ total: 0, groups: [] }, first(), context).groups.size).toBe(0);
});

test("selectedGroupAndCellTotalsNeverClaimOuterCounters", () => {
  for (const selected of [{ group: "high" }, part]) {
    const meta = first();
    meta.part = selected;
    const answer = decodePage({ ...issuePageOf([row("a", 11)]), total: 7 }, meta, context);
    expect(answer.total).toBe(0);
    expect(answer.read.rowParts).toEqual([selected]);
    expect(answer.read.resetCounters).toEqual([{ kind: "subgroup" in selected ? "cell" : "group", ...selected }]);
    expect(answer.subgroupTotals).toBeUndefined();
    if ("subgroup" in selected) {
      expect(answer.groups.get("high")?.total).toBe(0);
      expect(answer.groups.get("high")?.subgroups?.get("label")?.total).toBe(7);
    } else expect(answer.groups.get("high")?.total).toBe(7);
  }
});

test("separatesCursorAndDisplayBoundary", () => {
  const answer = decodePage(issuePageOf([row("a", 11)], "next"), continuation(), context);
  const group = answer.groups.get("high")!.subgroups!.get("label")!;
  // A later confirmed drop can extend the display window without changing the response proof or cursor.
  group.boundary!.key = 100;
  expect(group.cursorBoundary).toEqual({ id: "a", key: 11 });
  expect(answer.read.ranges.get(partKey)).toEqual({
    lower: { id: "before", key: 10 },
    upper: { id: "a", key: 11 },
    complete: false,
  });
});

test("lastPageDoesNotCoverEarlierRows", () => {
  const meta = continuation();
  const answer = decodePage(issuePageOf([row("a", 11)]), meta, context);
  expect(answer.read.ranges.get(partKey)).toEqual({
    lower: { id: "before", key: 10 },
    upper: { id: "a", key: 11 },
    complete: true,
  });
  meta.ranges.get(partKey)!.lower!.key = 999;
  meta.part!.group = "elsewhere";
  expect(answer.read.ranges.get(partKey)?.lower).toEqual({ id: "before", key: 10 });
  expect(answer.read.part).toEqual(part);
});

test("emptyContinuationKeepsCursorBoundaryAndOnlyItsOwnRange", () => {
  const meta = continuation();
  meta.ranges.set(pagePartKey({ group: "other" }), { lower: null, upper: null, complete: true });
  const answer = decodePage(issuePageOf([]), meta, context);
  expect(answer.groups.get("high")?.subgroups?.get("label")?.cursorBoundary).toEqual({ id: "before", key: 10 });
  expect(answer.read.ranges).toEqual(
    new Map([[partKey, { lower: { id: "before", key: 10 }, upper: null, complete: true }]])
  );
  expect(answer.groups.get("high")?.subgroups?.get("label")?.nextCursor).toBeNull();
});

test("continuationRequiresItsOriginalLowerBoundary", () => {
  expect(() => decodePage(issuePageOf([]), { ...continuation(), ranges: new Map() }, context)).toThrow();
});

test("refusesUnsafeDecodedCounts", () => {
  const rounded: number = JSON.parse("9007199254740993");
  for (const issue of [
    { ...row("a", 1), sub_issues_count: rounded },
    { ...row("a", 1), link_count: rounded },
  ]) {
    expect(() => decodePage(issuePageOf([issue]), first(), context)).toThrow();
  }
  for (const page of [
    { ...issuePageOf([]), total: rounded },
    { total: 0, groups: [{ key: "high", total: rounded, data: [], next_cursor: null }] },
    { total: 0, groups: [], subgroup_totals: [{ key: "label", total: rounded }] },
    {
      total: 0,
      groups: [{ key: "high", total: 0, subgroups: [{ key: "label", total: rounded, data: [], next_cursor: null }] }],
    },
  ])
    expect(() => decodePage(page, first(), context)).toThrow();
});

test("rejectsInvalidRowVersionsBeforeTheyBecomeProof", () => {
  expect(() => decodePage(issuePageOf([{ ...row("a", 1), updated_at: "bad" }]), first(), context)).toThrow();
});

test("usesGeneratedMemberQueryAndDefaultOrder", () => {
  const member: ListContext = {
    scope: "member",
    userID: "member",
    workspaceSlug: "ws",
    workspaceID: "id-ws",
    query: { relation: "assigned" },
    currentUserID: "actor",
    catalog: context.catalog,
    pendingIDs: new Set(["a"]),
  };
  const answer = decodePage(issuePageOf([issueOf("a")]), first(), member);
  expect(answer.groups.get("")?.rows[0]?.key).toBe("2026-10-28T00:00:00Z");
});

test("wholeReadOwnsOmittedParentCountersExactlyOnce", () => {
  const meta = first();
  meta.rowParts = [{ group: "high", subgroup: "missing" }];
  meta.resetCounters = [
    { group: "high", kind: "group" },
    { kind: "cell", group: "high", subgroup: "missing" },
  ];
  const answer = decodePage({ total: 0, groups: [{ key: "high", total: 0, subgroups: [] }] }, meta, context);
  expect(answer.read.resetCounters.filter((ref) => ref.kind === "group")).toEqual([{ kind: "group", group: "high" }]);
  expect(answer.read.resetCounters.filter((ref) => ref.kind === "cell")).toEqual([
    { kind: "cell", group: "high", subgroup: "missing" },
  ]);
  meta.rowParts[0]!.group = "changed";
  meta.resetCounters[0] = { kind: "outer" };
  meta.resetCounters[1]!.kind = "outer";
  expect(answer.read.rowParts).toEqual([{ group: "high", subgroup: "missing" }]);
  expect(answer.read.resetCounters.filter((ref) => ref.kind === "cell")).toEqual([
    { kind: "cell", group: "high", subgroup: "missing" },
  ]);

  const omitted = first();
  omitted.rowParts = [{ group: "absent", subgroup: "label" }];
  expect(decodePage({ total: 0, groups: [] }, omitted, context).read.resetCounters).toContainEqual({
    kind: "group",
    group: "absent",
  });
});

test("groupedFlatAnswersKeepIndependentOpaqueCursors", () => {
  const answer = decodePage(
    {
      total: 3,
      groups: [
        { key: "high", total: 2, data: [row("a", 1)], next_cursor: "opaque-A" },
        { key: "low", total: 1, data: [row("b", 2)], next_cursor: null },
      ],
    },
    first(),
    context
  );
  expect(answer.groups.get("high")?.nextCursor).toBe("opaque-A");
  expect(answer.groups.get("low")?.nextCursor).toBeNull();
  expect(answer.read.ranges.get(pagePartKey({ group: "high" }))?.complete).toBe(false);
  expect(answer.read.ranges.get(pagePartKey({ group: "low" }))?.complete).toBe(true);
});
