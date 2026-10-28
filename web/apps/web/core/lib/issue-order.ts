/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueListOrder, Label, MemberUser, State } from "@nerve/api-client";
import type { Boundary, SortKey } from "./paged-groups";

export type IssueCatalog = {
  states: ReadonlyMap<string, State>;
  labels: ReadonlyMap<string, Label>;
  members: ReadonlyMap<string, MemberUser>;
};

/** Match the server's ASCII fold and PostgreSQL C collation, without locale rules. */
export function issueNameKey(text: string): string {
  return text.replace(/[A-Z]/g, (letter) => letter.toLowerCase());
}

export function compareCodePoints(a: string, b: string): number {
  const left = Array.from(a, (character) => character.codePointAt(0)!);
  const right = Array.from(b, (character) => character.codePointAt(0)!);
  for (let index = 0; index < Math.min(left.length, right.length); index++) {
    if (left[index] !== right[index]) return left[index]! < right[index]! ? -1 : 1;
  }
  return Math.sign(left.length - right.length);
}

function validDate(date: string): boolean {
  const parsed = Date.parse(`${date}T00:00:00Z`);
  return (
    /^\d{4}-\d{2}-\d{2}$/.test(date) && Number.isFinite(parsed) && new Date(parsed).toISOString().slice(0, 10) === date
  );
}

function timeNanoseconds(time: string): bigint {
  const match =
    /^(\d{4}-\d{2}-\d{2})T([01]\d|2[0-3]):([0-5]\d):([0-5]\d)(?:\.(\d{1,9}))?(Z|[+-](?:[01]\d|2[0-3]):[0-5]\d)$/.exec(
      time
    );
  if (!match || !validDate(match[1]!)) throw new Error(`Invalid issue timestamp: ${time}`);
  // Date.parse sees only whole seconds; the original fraction never passes through number.
  const milliseconds = Date.parse(`${match[1]}T${match[2]}:${match[3]}:${match[4]}${match[6]}`);
  if (!Number.isFinite(milliseconds)) throw new Error(`Invalid issue timestamp: ${time}`);
  return BigInt(milliseconds) * 1_000_000n + BigInt((match[5] ?? "").padEnd(9, "0"));
}

export function compareIssueTime(a: string, b: string): number {
  const left = timeNanoseconds(a);
  const right = timeNanoseconds(b);
  return left < right ? -1 : left > right ? 1 : 0;
}

function minimumName(names: (string | undefined)[]): string | null {
  const keys = names.filter((name): name is string => name !== undefined).map(issueNameKey);
  return keys.length ? keys.toSorted(compareCodePoints)[0]! : null;
}

function exactCount(count: number): bigint {
  if (!Number.isSafeInteger(count) || count < 0) throw new Error("Unsafe issue count");
  return BigInt(count);
}

export function issueSortKey(issue: Issue, order: IssueListOrder, catalog: IssueCatalog): SortKey {
  switch (order.replace(/^-/, "")) {
    case "created_at":
      timeNanoseconds(issue.created_at);
      return issue.created_at;
    case "updated_at":
      timeNanoseconds(issue.updated_at);
      return issue.updated_at;
    case "start_date":
    case "target_date": {
      const date = order.endsWith("start_date") ? issue.start_date : issue.target_date;
      if (date !== null && !validDate(date)) throw new Error(`Invalid issue date: ${date}`);
      return date;
    }
    case "priority":
      return ["urgent", "high", "medium", "low", "none"].indexOf(issue.priority);
    case "state_name":
      return minimumName([catalog.states.get(issue.state_id)?.name]);
    case "label_name":
      return minimumName(issue.label_ids.map((id) => catalog.labels.get(id)?.name));
    case "assignee_name":
      return minimumName(issue.assignee_ids.map((id) => catalog.members.get(id)?.display_name));
    case "sub_issues_count":
      return exactCount(issue.sub_issues_count);
    case "link_count":
      return exactCount(issue.link_count);
    case "sort_order":
      if (!Number.isFinite(issue.sort_order)) throw new Error("Invalid issue sort order");
      return issue.sort_order;
    default:
      throw new Error(`Invalid issue order: ${order}`);
  }
}

/** NULL is last in either direction; the UUID tie breaker follows the key direction. */
export function compareRows(a: Boundary, b: Boundary, order: IssueListOrder): number {
  const direction = order.startsWith("-") ? -1 : 1;
  if (a.key === null && b.key !== null) return 1;
  if (a.key !== null && b.key === null) return -1;
  let compared = 0;
  if (a.key !== null && b.key !== null) {
    if (typeof a.key !== typeof b.key) throw new Error("Incompatible issue sort keys");
    if (typeof a.key === "string" && typeof b.key === "string") {
      compared =
        order.endsWith("created_at") || order.endsWith("updated_at")
          ? compareIssueTime(a.key, b.key)
          : compareCodePoints(a.key, b.key);
    } else compared = a.key < b.key ? -1 : a.key > b.key ? 1 : 0;
  }
  return (compared || compareCodePoints(a.id, b.id)) * direction;
}
