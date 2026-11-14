/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueCreate, IssueUpdate } from "@nerve/api-client";
import type { IssueChange, ModuleSetChange } from "@nerve/types";

/** The fields of a creation the web sends; whatever else the caller passed stays out of the request. */
export const createBody = (input: IssueCreate): IssueCreate => ({
  name: input.name,
  description_html: input.description_html,
  state_id: input.state_id,
  priority: input.priority,
  assignee_ids: input.assignee_ids,
  label_ids: input.label_ids,
  start_date: input.start_date,
  target_date: input.target_date,
  parent_id: input.parent_id,
  cycle_id: input.cycle_id,
  module_ids: input.module_ids,
});

/**
 * The fields of an update the web sends; `skip_activity` goes only with `description_html` alone. `module_ids` is the
 * work item's whole new set, which only a change's request (changeBody) works out, from the set held when its turn
 * comes (P7a spec 2.2).
 */
export function updateBody(input: IssueUpdate): IssueUpdate {
  const body: IssueUpdate = {
    name: input.name,
    description_html: input.description_html,
    state_id: input.state_id,
    priority: input.priority,
    assignee_ids: input.assignee_ids,
    label_ids: input.label_ids,
    sort_order: input.sort_order,
    start_date: input.start_date,
    target_date: input.target_date,
    parent_id: input.parent_id,
    cycle_id: input.cycle_id,
    module_ids: input.module_ids,
    skip_activity: input.skip_activity,
  };
  if (
    body.skip_activity !== undefined &&
    (typeof body.skip_activity !== "boolean" ||
      body.description_html === undefined ||
      Object.entries(body).some(
        ([key, value]) => value !== undefined && !["description_html", "skip_activity"].includes(key)
      ))
  )
    throw new Error("skip_activity requires description_html alone");
  return body;
}

/** The set of modules a change makes of the set held: its removals taken out, its additions not in it put after. */
function modulesAfter(held: readonly string[], change: ModuleSetChange): string[] {
  const removed = new Set(change.remove);
  const modules = held.filter((module) => !removed.has(module));
  for (const module of change.add ?? []) if (!modules.includes(module)) modules.push(module);
  return modules;
}

/**
 * A change's request, worked out in its turn from the modules the work item then holds: each field named, the modules
 * as the whole set they make, whatever else a caller slipped into the change left out.
 */
export const changeBody = (change: IssueChange, heldModules: readonly string[]): IssueUpdate => ({
  name: change.name,
  description_html: change.description_html,
  state_id: change.state_id,
  priority: change.priority,
  assignee_ids: change.assignee_ids,
  label_ids: change.label_ids,
  sort_order: change.sort_order,
  start_date: change.start_date,
  target_date: change.target_date,
  parent_id: change.parent_id,
  cycle_id: change.cycle_id,
  module_ids: change.modules ? modulesAfter(heldModules, change.modules) : undefined,
  skip_activity: change.skip_activity,
});

/**
 * The fields a held list's membership and display depend on, as one comparable value: the root's observed reads and
 * the writes' confirmation compare the same snapshot.
 */
export const membershipFields = (issue: Issue): string =>
  JSON.stringify([
    issue.project_id,
    issue.workspace_id,
    issue.name,
    issue.sort_order,
    issue.state_id,
    issue.priority,
    issue.label_ids,
    issue.assignee_ids,
    issue.cycle_id,
    issue.module_ids,
    issue.parent_id,
    issue.created_by_id,
    issue.start_date,
    issue.target_date,
    issue.archived_at,
    issue.sub_issues_count,
    issue.link_count,
    issue.attachment_count,
  ]);
