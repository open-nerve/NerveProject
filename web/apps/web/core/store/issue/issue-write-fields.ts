/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueCreate, IssueUpdate } from "@nerve/api-client";

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
});

/** The fields of an update the web sends; `skip_activity` goes only with `description_html` alone. */
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
