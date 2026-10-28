/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue } from "@nerve/api-client";
import { beforeIssue, issueMembership, issueParts, type Membership } from "./issue-membership";
import { compareIssueTime, compareRows, issueSortKey } from "./issue-order";
import {
  applyGroupChange,
  counterParts,
  groupAt,
  pagePartKey,
  partCounter,
  rowParts,
  type Boundary,
  type PageRange,
  type ConfirmedIssueChange,
  type CounterRef,
  type GroupChangeResult,
  type ListContext,
  type PageAnswer,
  type PagePart,
  type PagedGroups,
} from "./paged-groups";

function covers(answer: PageAnswer, part: PagePart, issue: Issue, context: ListContext): boolean {
  const range = answer.read.ranges.get(pagePartKey(part));
  if (!range) return answer.read.part === null && !groupAt(answer, part);
  const row = { id: issue.id, key: issueSortKey(issue, context.query.order_by ?? "-created_at", context.catalog) };
  return coversRange(range, row, context);
}

function coversRange(range: PageRange, row: Boundary, context: ListContext): boolean {
  const order = context.query.order_by ?? "-created_at";
  return (
    (!range.lower || compareRows(row, range.lower, order) > 0) &&
    (range.complete || (!!range.upper && compareRows(row, range.upper, order) <= 0))
  );
}

/** Evidence is the untouched answer, never the rows already merged or confirmed locally. */
export function replayGroups(
  value: PagedGroups,
  change: ConfirmedIssueChange,
  answer: PageAnswer,
  context: ListContext
): GroupChangeResult {
  const before = beforeIssue(change.beforeSnapshot);
  const rows = rowParts(answer)
    .flatMap((part) => groupAt(answer, part)?.rows ?? [])
    .filter((row) => row.id === change.id);
  let phase: "before" | "after" | "unknown" = "unknown";
  let inferred = false;
  if (rows.length)
    phase =
      change.after && rows.some((row) => compareIssueTime(row.updatedAt, change.after!.updated_at) >= 0)
        ? "after"
        : "before";
  else {
    const covered = (issue: Issue) =>
      issueParts(issue, context).some(
        (part) => issueMembership(issue, context, partCounter(part)) === true && covers(answer, part, issue, context)
      );
    if (change.after && covered(change.after)) {
      phase = "before";
      inferred = true;
    } else if (before && covered(before)) {
      phase = "after";
      inferred = true;
    }
  }
  const old = (ref: CounterRef): Membership => {
    const parts = counterParts(answer, ref);
    if (parts.some((part) => groupAt(answer, part)?.rows.some((row) => row.id === change.id))) return true;
    const membership = issueMembership(before, context, ref);
    if (membership === false) return false;
    const relevant = before
      ? issueParts(before, context).filter(
          (part) =>
            ref.kind === "outer" ||
            (ref.kind === "group" && ref.group === part.group) ||
            (ref.kind === "subgroup" && ref.subgroup === part.subgroup) ||
            (ref.kind === "cell" && ref.group === part.group && ref.subgroup === part.subgroup)
        )
      : parts;
    if (
      relevant.length &&
      relevant.every((part) => {
        const range = answer.read.ranges.get(pagePartKey(part));
        return before ? covers(answer, part, before, context) : !!range && range.lower === null && range.complete;
      })
    )
      return false;
    return membership;
  };
  const result = applyGroupChange(value, change, context, { answer, phase, old });
  if (inferred && change.after) result.verifyIds = [change.id];
  return result;
}
