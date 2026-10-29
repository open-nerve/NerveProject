/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue } from "@nerve/api-client";
import { beforeIssue, issueMembership, issueParts, type Membership } from "./issue-membership";
import { compareIssueTime, compareRows, issueSortKey } from "./issue-order";
import type {
  PageAnswer,
  PagedGroups,
  ConfirmedIssueChange,
  CounterRef,
  ListContext,
  PagePart,
  PageRange,
  Row,
} from "./paged-groups";

export type RowEvidence = { rows: Row[]; complete: boolean };
export type ReadEvidence = { whole: boolean; parts: { part: PagePart; rows: Row[]; range: PageRange }[] };
export type CounterEvidence = {
  sourceID: number;
  held: RowEvidence;
  read?: ReadEvidence;
  confirmed: Map<string, { updatedAt: string | null; member: boolean }>;
};
export function counterKey(ref: CounterRef): string {
  return JSON.stringify([ref.kind, "group" in ref ? ref.group : null, "subgroup" in ref ? ref.subgroup : null]);
}
function matches(ref: CounterRef, part: PagePart): boolean {
  return (
    ref.kind === "outer" ||
    (ref.kind === "group" && ref.group === part.group) ||
    (ref.kind === "subgroup" && ref.subgroup === part.subgroup) ||
    (ref.kind === "cell" && ref.group === part.group && ref.subgroup === part.subgroup)
  );
}
function covers(read: ReadEvidence, part: PagePart, issue: Issue, context: ListContext): boolean {
  const entry = read.parts.find((item) => item.part.group === part.group && item.part.subgroup === part.subgroup);
  if (!entry) return read.whole;
  const row = { id: issue.id, key: issueSortKey(issue, context.query.order_by ?? "-created_at", context.catalog) };
  const order = context.query.order_by ?? "-created_at";
  return (
    (!entry.range.lower || compareRows(row, entry.range.lower, order) > 0) &&
    (entry.range.complete || (!!entry.range.upper && compareRows(row, entry.range.upper, order) <= 0))
  );
}
/** The original read's version relation, shared by replay and a reset counter. */
export function answerTiming(
  read: ReadEvidence,
  change: ConfirmedIssueChange,
  context: ListContext
): {
  phase: "before" | "after" | "newer" | "unknown";
  inferred: boolean;
} {
  const rows = read.parts.flatMap((entry) => entry.rows).filter((row) => row.id === change.id);
  if (rows.length)
    return {
      phase:
        change.after && rows.some((row) => compareIssueTime(row.updatedAt, change.after!.updated_at) > 0)
          ? "newer"
          : change.after && rows.some((row) => compareIssueTime(row.updatedAt, change.after!.updated_at) === 0)
            ? "after"
            : "before",
      inferred: false,
    };
  const covered = (issue: Issue) =>
    issueParts(issue, context).some((part) => {
      const ref: CounterRef =
        part.subgroup === undefined
          ? { kind: "group", group: part.group }
          : { kind: "cell", group: part.group, subgroup: part.subgroup };
      return issueMembership(issue, context, ref) === true && covers(read, part, issue, context);
    });
  const before = beforeIssue(change.beforeSnapshot);
  if (change.after && covered(change.after)) return { phase: "before", inferred: true };
  if (before && covered(before)) return { phase: "after", inferred: true };
  return { phase: "unknown", inferred: false };
}
export function priorAnswerMembership(
  read: ReadEvidence,
  change: ConfirmedIssueChange,
  context: ListContext,
  ref: CounterRef
): Membership {
  const parts = read.parts.filter((entry) => matches(ref, entry.part));
  if (parts.some((entry) => entry.rows.some((row) => row.id === change.id))) return true;
  const before = beforeIssue(change.beforeSnapshot);
  const membership = issueMembership(before, context, ref);
  if (membership === false) return false;
  const relevant = before
    ? issueParts(before, context).filter((part) => matches(ref, part))
    : parts.map((entry) => entry.part);
  if (
    relevant.length &&
    relevant.every((part) =>
      before
        ? covers(read, part, before, context)
        : parts.some(
            (entry) =>
              entry.part.group === part.group &&
              entry.part.subgroup === part.subgroup &&
              entry.range.lower === null &&
              entry.range.complete
          )
    )
  )
    return false;
  return membership;
}
/** Held rows predate subsequent partial mutations; they prove only their own count. */
export function priorRowMembership(
  held: RowEvidence,
  change: ConfirmedIssueChange,
  context: ListContext,
  ref: CounterRef
): Membership {
  const row = held.rows.find((item) => item.id === change.id);
  if (row)
    return change.after && compareIssueTime(row.updatedAt, change.after.updated_at) === 0
      ? issueMembership(change.after, context, ref)
      : true;
  if (held.complete) return false;
  return issueMembership(beforeIssue(change.beforeSnapshot), context, ref);
}
export function priorCounterMembership(
  source: CounterEvidence,
  change: ConfirmedIssueChange,
  context: ListContext,
  ref: CounterRef,
  heldSources: ReadonlySet<number>
): { old: Membership; inferred: boolean } {
  const fact = source.confirmed.get(change.id);
  if (fact)
    return {
      old:
        fact.updatedAt === null
          ? false
          : change.after && compareIssueTime(fact.updatedAt, change.after.updated_at) === 0
            ? issueMembership(change.after, context, ref)
            : fact.member,
      inferred: false,
    };
  if (heldSources.has(source.sourceID))
    return { old: priorRowMembership(source.held, change, context, ref), inferred: false };
  if (!source.read) return { old: undefined, inferred: false };
  const timing = answerTiming(source.read, change, context);
  return {
    old:
      timing.phase === "unknown" || timing.phase === "newer"
        ? undefined
        : timing.phase === "after"
          ? issueMembership(change.after, context, ref)
          : priorAnswerMembership(source.read, change, context, ref),
    inferred: timing.inferred && change.after !== null,
  };
}

/** Only currently held source identities authorize a future operation's frozen Before. */
export function heldCounterSources(value: PagedGroups | undefined): ReadonlySet<number> {
  if (!value) return new Set();
  return new Set(
    value.counterEvidence
      ? [...value.counterEvidence.values()].map((source) => source.sourceID)
      : [value.read?.sourceID ?? 0]
  );
}

/** Extract untouched raw evidence before any rows are merged or confirmed. */
export function pageEvidence(answer: PageAnswer): ReadEvidence {
  return {
    whole: answer.read.part === null,
    parts: answer.read.rowParts.map((part) => {
      const group = answer.groups.get(part.group);
      const rows = (part.subgroup === undefined ? group : group?.subgroups?.get(part.subgroup))?.rows ?? [];
      return { part, rows, range: answer.read.ranges.get(JSON.stringify([part.group, part.subgroup]))! };
    }),
  };
}

/** A strictly later fact supersedes this operation only for this counter's own source. */
export function counterSupersedes(source: CounterEvidence, change: ConfirmedIssueChange): boolean {
  if (!change.after) return false;
  const newer = (version: string) => compareIssueTime(version, change.after!.updated_at) > 0;
  const fact = source.confirmed.get(change.id);
  return Boolean(
    (fact?.updatedAt && newer(fact.updatedAt)) ||
    source.held.rows.some((row) => row.id === change.id && newer(row.updatedAt)) ||
    source.read?.parts.some((entry) => entry.rows.some((row) => row.id === change.id && newer(row.updatedAt)))
  );
}
