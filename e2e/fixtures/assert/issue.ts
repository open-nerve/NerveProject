import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the work-item stories. The page version and the
// API version of a story call the same function (M4 design 2).

/** A row of a work item's activity as expectIssueCreated reads it: the account a row names, by its address. */
export interface ActivityRow {
  field: string | null;
  verb: string;
  old_value: string | null;
  new_value: string | null;
  /** The address of the account the row names (new_identifier), for an assignee's row; else null. */
  named: string | null;
  comment: string;
}

/**
 * A work item as created: its number, name, state (by name), priority and sort order; whether it is completed; the
 * address of the account that created it; its assignees and subscribers by address, its labels by name, each in
 * order; and its activity, in its order.
 */
export interface CreatedIssue {
  sequence_id: number;
  name: string;
  state: string;
  priority: string;
  sort_order: number;
  completed: boolean;
  by: string;
  assignees: string[];
  labels: string[];
  subscribers: string[];
  activity: ActivityRow[];
}

/**
 * I1: the work item of issueId holds want, as created (M4 design 3.7, 3.13): undeleted, not archived, numbered
 * want.sequence_id, which its project's last_issue_sequence is now. It and every row of its assignees, labels,
 * subscribers and activity are of its project and workspace, made and last written by the account of want.by, the
 * activity's actor too, at one moment.
 */
export async function expectIssueCreated(db: Database, issueId: string, want: CreatedIssue): Promise<void> {
  const [issue] = await db.query(
    `SELECT i.sequence_id, i.name, s.name AS state, i.priority, i.sort_order, i.completed_at IS NOT NULL AS completed,
            c.email AS by, p.last_issue_sequence,
            i.workspace_id = p.workspace_id AND i.updated_by_id = i.created_by_id AND i.updated_at = i.created_at
              AND i.deleted_at IS NULL AND i.archived_at IS NULL AS as_created
       FROM issues i JOIN projects p ON p.id = i.project_id JOIN states s ON s.id = i.state_id
       JOIN users c ON c.id = i.created_by_id
      WHERE i.id = $1`,
    [issueId]
  );
  const { assignees, labels, subscribers, activity, ...row } = want;
  expect(issue, `the work item ${issueId}`).toEqual({
    ...row,
    last_issue_sequence: want.sequence_id,
    as_created: true,
  });
  const named = async (sql: string) => (await db.query<{ v: string }>(sql, [issueId])).map(({ v }) => v);
  expect(
    {
      assignees: await named(`SELECT u.email AS v FROM issue_assignees r JOIN users u ON u.id = r.assignee_id
                                WHERE r.issue_id = $1 ORDER BY u.email COLLATE "C"`),
      labels: await named(`SELECT l.name AS v FROM issue_labels r JOIN labels l ON l.id = r.label_id
                             WHERE r.issue_id = $1 ORDER BY l.name COLLATE "C"`),
      subscribers: await named(`SELECT u.email AS v FROM issue_subscribers r JOIN users u ON u.id = r.subscriber_id
                                  WHERE r.issue_id = $1 ORDER BY u.email COLLATE "C"`),
      activity: await db.query(
        `SELECT a.field, a.verb, a.old_value, a.new_value, u.email AS named, a.comment
           FROM issue_activities a LEFT JOIN users u ON u.id = a.new_identifier
          WHERE a.issue_id = $1 ORDER BY a.created_at, a.id`,
        [issueId]
      ),
    },
    `the rows of the work item ${issueId}`
  ).toEqual({ assignees, labels, subscribers, activity });
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  const [stamps] = await db.query<{ rows: number; stamped: number }>(
    `SELECT count(*)::int AS rows,
            count(*) FILTER (WHERE r.workspace_id = i.workspace_id AND r.project_id = i.project_id
                               AND r.created_by_id = i.created_by_id AND r.updated_by_id = i.created_by_id
                               AND r.actor IS NOT DISTINCT FROM i.created_by_id
                               AND r.created_at = i.created_at AND r.updated_at = i.created_at AND r.deleted_at IS NULL)::int
              AS stamped
       FROM issues i, LATERAL (
              SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id AS actor, created_at, updated_at, deleted_at
                FROM issue_assignees WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_labels WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, created_by_id, created_at, updated_at, deleted_at
                FROM issue_subscribers WHERE issue_id = i.id
              UNION ALL SELECT workspace_id, project_id, created_by_id, updated_by_id, actor_id, created_at, updated_at, deleted_at
                FROM issue_activities WHERE issue_id = i.id) r
      WHERE i.id = $1`,
    [issueId]
  );
  expect(stamps, `the rows of the work item ${issueId}, each made with it`).toEqual({
    rows: assignees.length + labels.length + subscribers.length + activity.length,
    stamped: assignees.length + labels.length + subscribers.length + activity.length,
  });
}
