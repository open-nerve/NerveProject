/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueListPage } from "@nerve/api-client";
import type { MemberIssueQuery, ProjectIssueQuery } from "@/services/issue/issue-list.service";
import { compareIssueTime, issueSortKey, type IssueCatalog } from "./issue-order";

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
  | { scope: "member"; userID: string; workspaceSlug: string; query: MemberIssueQuery }
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
  function groupAt(part: PagePart): Group {
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
    const group = groupAt(part);
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
      const target = groupAt(part);
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
      groupAt(ref);
      counter(ref);
    }
  }
  for (const group of answer.groups.keys()) counter({ kind: "group", group });
  return answer;
}
