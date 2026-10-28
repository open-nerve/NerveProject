/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueListPage } from "@nerve/api-client";
import type { MemberIssueQuery, ProjectIssueQuery } from "@/services/issue/issue-list.service";
import { compareIssueTime, compareRows, issueSortKey, type IssueCatalog } from "./issue-order";

import { beforeIssue, issueMembership, issueParts, subgroupField, type Membership } from "./issue-membership";

export type SortKey = string | number | bigint | null;
export type Boundary = { key: SortKey; id: string };
export type Row = Boundary & { updatedAt: string };
export type Group = {
  total: number;
  rows: Row[];
  nextCursor: string | null;
  boundary: Boundary | null;
  cursorBoundary: Boundary | null;
  stale: boolean;
  subgroups?: Map<string, Group>;
};
export type PagedGroups = {
  total: number;
  groups: Map<string, Group>;
  subgroupTotals?: Map<string, number>;
  read?: PageReadMeta;
};
export type PageRange = { lower: Boundary | null; upper: Boundary | null; complete: boolean };
export type PagePart = { group: string; subgroup?: string };
export type CounterRef =
  | { kind: "outer" }
  | { kind: "group"; group: string }
  | { kind: "cell"; group: string; subgroup: string }
  | { kind: "subgroup"; subgroup: string };
export type PageReadMeta = {
  part: PagePart | null;
  ranges: Map<string, PageRange>;
  first: boolean;
  resetCounters: CounterRef[];
  rowParts: PagePart[];
};
export type PageAnswer = PagedGroups & { read: PageReadMeta };
export type ListContext = (
  | { scope: "project"; projectID: string; query: ProjectIssueQuery }
  | { scope: "member"; userID: string; workspaceSlug: string; workspaceID: string; query: MemberIssueQuery }
) & { catalog: IssueCatalog; currentUserID: string; pendingIDs: ReadonlySet<string> };

/** Stable identity for a group or cell; cannot collide on delimiter characters. */
export function pagePartKey(part: PagePart): string {
  return JSON.stringify([part.group, part.subgroup]);
}

/** Empty containers are placeholders, never proof that an unrequested part was read. */
function emptyGroup(): Group {
  return { total: 0, rows: [], nextCursor: null, boundary: null, cursorBoundary: null, stale: false };
}

function checkedTotal(total: number): number {
  if (!Number.isSafeInteger(total) || total < 0) throw new Error("Unsafe page count");
  return total;
}

function copyBoundary(boundary: Boundary | null): Boundary | null {
  return boundary === null ? null : { id: boundary.id, key: boundary.key };
}

/** Decode only a raw response. A merged model must never be passed as read evidence.
 * Input ranges hold the cursor boundary captured when issuing the request. Output
 * ranges, rowParts and resetCounters describe this response alone. For whole reads,
 * input rowParts/counters may name known empty groups omitted by the server.
 */
export function decodePage(page: IssueListPage, meta: PageReadMeta, context: ListContext): PageAnswer {
  checkedTotal(page.total);
  const read: PageReadMeta = {
    part: meta.part ? { ...meta.part } : null,
    first: meta.first,
    ranges: new Map(),
    resetCounters: [],
    rowParts: [],
  };
  // A selected page's total belongs to its group/cell, never to outer or its parent.
  const answer: PageAnswer = { total: meta.part === null ? page.total : 0, groups: new Map(), read };
  const counterKeys = new Set<string>();
  function counter(ref: CounterRef) {
    const key = JSON.stringify([ref.kind, "group" in ref ? ref.group : null, "subgroup" in ref ? ref.subgroup : null]);
    if (!counterKeys.has(key)) {
      counterKeys.add(key);
      read.resetCounters.push({ ...ref });
    }
  }
  function decodedGroupAt(part: PagePart): Group {
    let group = answer.groups.get(part.group);
    if (!group) {
      group = emptyGroup();
      answer.groups.set(part.group, group);
    }
    if (part.subgroup === undefined) return group;
    group.subgroups ??= new Map();
    let cell = group.subgroups.get(part.subgroup);
    if (!cell) {
      cell = emptyGroup();
      group.subgroups.set(part.subgroup, cell);
    }
    return cell;
  }
  function decodeRows(part: PagePart, total: number, data: Issue[], nextCursor: string | null) {
    const key = pagePartKey(part);
    const originalLower = meta.ranges.get(key)?.lower;
    if (!meta.first && !originalLower) throw new Error("Missing continuation boundary");
    const lower = meta.first ? null : copyBoundary(originalLower ?? null);
    const rows = data.map((issue): Row => {
      // Both counts are checked even when this list sorts by another field.
      checkedTotal(issue.sub_issues_count);
      checkedTotal(issue.link_count);
      compareIssueTime(issue.updated_at, issue.updated_at);
      return {
        id: issue.id,
        key: issueSortKey(issue, context.query.order_by ?? "-created_at", context.catalog),
        updatedAt: issue.updated_at,
      };
    });
    const last = rows.at(-1);
    const upper = last ? { id: last.id, key: last.key } : null;
    const group = decodedGroupAt(part);
    group.total = checkedTotal(total);
    group.rows = rows;
    group.nextCursor = nextCursor;
    group.boundary = copyBoundary(upper ?? lower);
    group.cursorBoundary = copyBoundary(upper ?? lower);
    read.ranges.set(key, { lower, upper, complete: nextCursor === null });
    read.rowParts.push({ ...part });
    counter(
      part.subgroup === undefined
        ? { kind: "group", group: part.group }
        : { kind: "cell", group: part.group, subgroup: part.subgroup }
    );
  }
  if (meta.part !== null) {
    decodeRows(meta.part, page.total, page.data ?? [], page.next_cursor ?? null);
    return answer;
  }
  counter({ kind: "outer" });
  if (page.groups !== undefined) {
    for (const group of page.groups) {
      const part = { group: group.key };
      const target = decodedGroupAt(part);
      target.total = checkedTotal(group.total);
      counter({ kind: "group", group: group.key });
      if (group.subgroups !== undefined) {
        target.subgroups = new Map();
        for (const cell of group.subgroups) {
          decodeRows({ ...part, subgroup: cell.key }, cell.total, cell.data, cell.next_cursor);
        }
      } else decodeRows(part, group.total, group.data ?? [], group.next_cursor ?? null);
    }
  } else decodeRows({ group: "" }, page.total, page.data ?? [], page.next_cursor ?? null);
  if (page.subgroup_totals !== undefined) {
    answer.subgroupTotals = new Map();
    for (const subgroup of page.subgroup_totals) {
      answer.subgroupTotals.set(subgroup.key, checkedTotal(subgroup.total));
      counter({ kind: "subgroup", subgroup: subgroup.key });
    }
  }
  // Whole responses reset omitted known groups/cells to zero. Selected responses
  // return above, so unrelated parts cannot accidentally acquire read evidence.
  for (const part of meta.rowParts) {
    if (!read.ranges.has(pagePartKey(part))) decodeRows(part, 0, [], null);
  }
  for (const ref of meta.resetCounters) {
    if (ref.kind === "subgroup") {
      if (answer.subgroupTotals) {
        if (!answer.subgroupTotals.has(ref.subgroup)) answer.subgroupTotals.set(ref.subgroup, 0);
        counter(ref);
      }
    } else if (ref.kind !== "outer") {
      decodedGroupAt(ref);
      counter(ref);
    }
  }
  for (const group of answer.groups.keys()) counter({ kind: "group", group });
  return answer;
}

export type Before = { kind: "known"; issue: Issue } | { kind: "absent" } | { kind: "unknown" };
export type ConfirmedIssueChange = {
  id: string;
  beforeSnapshot: Before;
  after: Issue | import("@nerve/api-client").IssueDetail | null;
  drop?: PagePart & { beforeId?: string; afterId?: string };
};
export type GroupChangeResult = { value: PagedGroups; verifyIds: string[] };

/** Internal operations are shared with raw replay; neither path mutates its inputs. */
export function groupAt(value: PagedGroups, part: PagePart): Group | undefined {
  const group = value.groups.get(part.group);
  return part.subgroup === undefined ? group : group?.subgroups?.get(part.subgroup);
}
function ensureGroup(value: PagedGroups, part: PagePart): Group {
  let group = value.groups.get(part.group);
  if (!group) value.groups.set(part.group, (group = emptyGroup()));
  if (part.subgroup === undefined) return group;
  group.subgroups ??= new Map();
  let cell = group.subgroups.get(part.subgroup);
  if (!cell) group.subgroups.set(part.subgroup, (cell = emptyGroup()));
  return cell;
}
export function rowParts(value: PagedGroups): PagePart[] {
  return [...value.groups].flatMap(([group, data]) =>
    data.subgroups ? [...data.subgroups.keys()].map((subgroup) => ({ group, subgroup })) : [{ group }]
  );
}
function counters(value: PagedGroups): CounterRef[] {
  const refs: CounterRef[] = [{ kind: "outer" }];
  for (const [group, data] of value.groups) {
    refs.push({ kind: "group", group });
    for (const subgroup of data.subgroups?.keys() ?? []) refs.push({ kind: "cell", group, subgroup });
  }
  for (const subgroup of value.subgroupTotals?.keys() ?? []) refs.push({ kind: "subgroup", subgroup });
  return refs;
}
export function partCounter(part: PagePart): CounterRef {
  return part.subgroup === undefined
    ? { kind: "group", group: part.group }
    : { kind: "cell", group: part.group, subgroup: part.subgroup };
}
function counterValue(value: PagedGroups, ref: CounterRef): number {
  return ref.kind === "outer"
    ? value.total
    : ref.kind === "subgroup"
      ? (value.subgroupTotals?.get(ref.subgroup) ?? 0)
      : (groupAt(value, ref)?.total ?? 0);
}
function setCounter(value: PagedGroups, ref: CounterRef, total: number) {
  checkedTotal(total);
  if (ref.kind === "outer") value.total = total;
  else if (ref.kind === "subgroup") (value.subgroupTotals ??= new Map()).set(ref.subgroup, total);
  else ensureGroup(value, ref).total = total;
}
function farther(a: Boundary | null, b: Boundary | null, context: ListContext): Boundary | null {
  return copyBoundary(!a || (b && compareRows(a, b, context.query.order_by ?? "-created_at") < 0) ? b : a);
}
export function counterParts(value: PagedGroups, ref: CounterRef): PagePart[] {
  if (ref.kind === "cell") return [{ group: ref.group, subgroup: ref.subgroup }];
  if (ref.kind === "group")
    return value.groups.get(ref.group)?.subgroups
      ? [...value.groups.get(ref.group)!.subgroups!.keys()].map((subgroup) => ({ group: ref.group, subgroup }))
      : [{ group: ref.group }];
  return rowParts(value).filter((part) => ref.kind === "outer" || part.subgroup === ref.subgroup);
}
/** Complete means the whole group here. Raw continuation proof uses ranges instead. */
function priorMembership(
  value: PagedGroups,
  change: ConfirmedIssueChange,
  context: ListContext,
  ref: CounterRef
): Membership {
  const groups = counterParts(value, ref).map((part) => groupAt(value, part));
  const row = groups.flatMap((group) => group?.rows ?? []).find((item) => item.id === change.id);
  if (row)
    return change.after && compareIssueTime(row.updatedAt, change.after.updated_at) >= 0
      ? issueMembership(change.after, context, ref)
      : true;
  if (groups.every((group) => !group || group.nextCursor === null)) return false;
  return issueMembership(beforeIssue(change.beforeSnapshot), context, ref);
}

/** Merge owns exactly the response write set; continuation totals replace local totals. */
export function mergePages(value: PagedGroups, answer: PageAnswer, context: ListContext): PagedGroups {
  const merged = structuredClone(answer.read.part === null ? answer : value);
  const parts = answer.read.part === null ? rowParts(value).concat(answer.read.rowParts) : answer.read.rowParts;
  for (const part of parts) {
    const old = groupAt(value, part);
    const raw = groupAt(answer, part) ?? emptyGroup();
    const target = ensureGroup(merged, part);
    const rows = new Map<string, Row>();
    for (const row of answer.read.first ? [] : (old?.rows ?? [])) rows.set(row.id, { ...row });
    for (const row of raw.rows) rows.set(row.id, { ...row });
    target.rows = [...rows.values()].toSorted((a, b) => compareRows(a, b, context.query.order_by ?? "-created_at"));
    target.boundary = farther(old?.boundary ?? null, raw.boundary, context);
    target.cursorBoundary = copyBoundary(raw.cursorBoundary);
    target.nextCursor = raw.nextCursor;
    target.stale = raw.stale;
  }
  for (const ref of answer.read.resetCounters) setCounter(merged, ref, counterValue(answer, ref));
  return merged;
}

/** Shared placement/count mechanics; replay supplies only its response-owned write set. */
export function applyGroupChange(
  value: PagedGroups,
  change: ConfirmedIssueChange,
  context: ListContext,
  options?: { answer: PageAnswer; phase: "before" | "after" | "unknown"; old: (ref: CounterRef) => Membership }
): GroupChangeResult {
  const before = beforeIssue(change.beforeSnapshot);
  // Workspace is immutable on an issue. A foreign confirmation cannot affect
  // this member scope even when its frozen before snapshot was unavailable.
  const scopedIssue = change.after ?? before;
  if (context.scope === "member" && scopedIssue && scopedIssue.workspace_id !== context.workspaceID)
    return { value, verifyIds: [] };
  const result = structuredClone(value);
  const relevant = issueParts(before, context).concat(issueParts(change.after, context));
  if (change.drop) relevant.push(change.drop);
  const whole = !options || options.answer.read.part === null;
  if (whole)
    for (const part of relevant) {
      ensureGroup(result, part);
      if (part.subgroup !== undefined)
        (result.subgroupTotals ??= new Map()).set(part.subgroup, result.subgroupTotals?.get(part.subgroup) ?? 0);
    }
  const refs = whole ? counters(result) : options.answer.read.resetCounters;
  const parts = whole ? rowParts(result) : options.answer.read.rowParts;
  const uncertain = new Set<string>();
  function mark(ref: CounterRef) {
    for (const part of counterParts(result, ref)) {
      const member = partCounter(part);
      if (
        issueMembership(before, context, member) !== false ||
        issueMembership(change.after, context, member) !== false ||
        groupAt(result, part)?.rows.some((row) => row.id === change.id)
      )
        uncertain.add(pagePartKey(part));
    }
  }
  for (const ref of refs) {
    const next = issueMembership(change.after, context, ref);
    const old =
      options?.phase === "after" ? next : options ? options.old(ref) : priorMembership(value, change, context, ref);
    const affected = old !== false || next !== false;
    if (options?.phase === "unknown") {
      if (affected) mark(ref);
      continue;
    }
    if (old === undefined || next === undefined) {
      if (affected) mark(ref);
      continue;
    }
    if (options?.phase !== "after") setCounter(result, ref, counterValue(result, ref) + Number(next) - Number(old));
  }
  for (const part of parts) {
    const group = ensureGroup(result, part);
    const ref = partCounter(part);
    const next = issueMembership(change.after, context, ref);
    const old =
      options?.phase === "after" ? next : options ? options.old(ref) : priorMembership(value, change, context, ref);
    const affected = old !== false || next !== false || group.rows.some((row) => row.id === change.id);
    if ((options?.phase === "unknown" || old === undefined || next === undefined) && affected) {
      group.stale = true;
      continue;
    }
    if (uncertain.has(pagePartKey(part))) group.stale = true;
    if (!affected) continue;
    group.rows = group.rows.filter((row) => row.id !== change.id);
    if (!next || !change.after) continue;
    const row: Row = {
      id: change.id,
      key: issueSortKey(change.after, context.query.order_by ?? "-created_at", context.catalog),
      updatedAt: change.after.updated_at,
    };
    const drop = change.drop && pagePartKey(change.drop) === pagePartKey(part);
    if (drop) group.boundary = farther(group.boundary, row, context);
    if (
      drop ||
      group.nextCursor === null ||
      (group.boundary && compareRows(row, group.boundary, context.query.order_by ?? "-created_at") <= 0)
    ) {
      group.rows.push(row);
      group.rows = group.rows.toSorted((a, b) => compareRows(a, b, context.query.order_by ?? "-created_at"));
    }
  }
  // A lane parent has no rows but must report uncertainty in its independent count.
  if (subgroupField(context))
    for (const [key, group] of result.groups) {
      if ([...(group.subgroups?.keys() ?? [])].some((subgroup) => uncertain.has(pagePartKey({ group: key, subgroup }))))
        group.stale = true;
    }
  return { value: result, verifyIds: [] };
}
export function confirmGroups(
  value: PagedGroups,
  change: ConfirmedIssueChange,
  context: ListContext
): GroupChangeResult {
  return applyGroupChange(value, change, context);
}
