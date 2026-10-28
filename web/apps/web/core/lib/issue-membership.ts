/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueListGroupField } from "@nerve/api-client";
import type { Before, CounterRef, ListContext, PagePart } from "./paged-groups";

export type Membership = boolean | undefined;
const alternatives = (filter: string[] | undefined, values: string[], me: string): boolean =>
  !filter?.length || filter.some((value) => values.includes(value === "me" ? me : value));
function dates(filter: string[] | undefined, value: string | null): boolean {
  return (
    !filter?.length ||
    filter.every((condition) => {
      if (condition === "none") return value === null;
      if (value === null) return false;
      if (!condition.includes("..")) return value === condition;
      const [lower, upper] = condition.split("..");
      return (!lower || value >= lower) && (!upper || value <= upper);
    })
  );
}
function groupKeys(issue: Issue, field: IssueListGroupField | undefined, context: ListContext): string[] | undefined {
  switch (field) {
    case undefined:
      return [""];
    case "label_id":
      return issue.label_ids.length ? [...new Set(issue.label_ids)] : ["none"];
    case "assignee_id":
      return issue.assignee_ids.length ? [...new Set(issue.assignee_ids)] : ["none"];
    case "state_group": {
      const group = context.catalog.states.get(issue.state_id)?.group;
      return group ? [group] : undefined;
    }
    case "target_date":
      return [issue.target_date ?? "none"];
    default:
      return [issue[field] ?? "none"];
  }
}
export function subgroupField(context: ListContext): IssueListGroupField | undefined {
  return "sub_group_by" in context.query ? context.query.sub_group_by : undefined;
}
/** Unknown relationship tables stay unknown; a row never invents mention/subscriber data. */
export function issueMembership(
  issue: Issue | null | undefined,
  context: ListContext,
  ref: CounterRef = { kind: "outer" }
): Membership {
  if (issue === undefined) return undefined;
  if (issue === null) return false;
  const q = context.query;
  let unknown = false;
  if (context.scope === "project") {
    const p = context.query;
    if (issue.project_id !== context.projectID || (issue.archived_at !== null) !== (p.archived === "true"))
      return false;
    if (
      !alternatives(p.state_id, [issue.state_id], context.currentUserID) ||
      !alternatives(p.assignee_id, issue.assignee_ids.length ? issue.assignee_ids : ["none"], context.currentUserID) ||
      !alternatives(p.created_by_id, issue.created_by_id ? [issue.created_by_id] : [], context.currentUserID)
    )
      return false;
    unknown = !!(p.mention_id?.length || p.subscriber_id?.length);
  } else {
    const p = context.query;
    if (issue.workspace_id !== context.workspaceID) return false;
    if (issue.archived_at !== null || !alternatives(p.project_id, [issue.project_id], context.currentUserID))
      return false;
    if (p.relation === "assigned" && !issue.assignee_ids.includes(context.userID)) return false;
    if (p.relation === "created" && issue.created_by_id !== context.userID) return false;
    unknown = p.relation === "subscribed";
  }
  if (
    !alternatives(q.priority, [issue.priority], context.currentUserID) ||
    !alternatives(q.label_id, issue.label_ids.length ? issue.label_ids : ["none"], context.currentUserID) ||
    !dates(q.start_date, issue.start_date) ||
    !dates(q.target_date, issue.target_date) ||
    (q.sub_issue === "false" && issue.parent_id !== null)
  )
    return false;
  if (q.state_group?.length) {
    const state = context.catalog.states.get(issue.state_id);
    if (!state) unknown = true;
    else if (!q.state_group.includes(state.group)) return false;
  }
  if (ref.kind === "group" || ref.kind === "cell") {
    const groups = groupKeys(issue, q.group_by, context);
    if (!groups) unknown = true;
    else if (!groups.includes(ref.group)) return false;
  }
  if (ref.kind === "cell" || ref.kind === "subgroup") {
    const groups = groupKeys(issue, subgroupField(context), context);
    if (!groups) unknown = true;
    else if (!groups.includes(ref.subgroup)) return false;
  }
  return unknown ? undefined : true;
}
export function issueParts(issue: Issue | null | undefined, context: ListContext): PagePart[] {
  if (!issue || issueMembership(issue, context) === false) return [];
  const groups = groupKeys(issue, context.query.group_by, context) ?? [];
  const subField = subgroupField(context);
  const subgroups = subField ? (groupKeys(issue, subField, context) ?? []) : undefined;
  return groups.flatMap((group) => (subgroups ? subgroups.map((subgroup) => ({ group, subgroup })) : [{ group }]));
}

/** Preserve the distinction between a create and an unavailable frozen snapshot. */
export function beforeIssue(before: Before): Issue | null | undefined {
  return before.kind === "known" ? before.issue : before.kind === "absent" ? null : undefined;
}
