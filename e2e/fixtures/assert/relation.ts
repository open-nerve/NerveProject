import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the relation stories. The page version and the
// API version of a story call the same function (M4 design 2).

/**
 * A relation as stored: "<work item> <type> <other work item>", the work items by name, a work item blocking another
 * stored as the other's blocked_by; the address of the account that wrote it last; and whether it is deleted.
 */
export interface StoredRelation {
  relation: string;
  by: string;
  deleted: boolean;
}

/**
 * I9: the relations stored of the work items of issueIds, either way, are want, in the order they were made (M4 design
 * 3.10): one row a pair of work items, a work item blocking another stored as the other's blocked_by. Each row is of
 * the workspace and the project of its work item (issue_id), the other work item of the same workspace; an undeleted
 * one is as it was made, by the account that made it, and a deleted one was written last at its deletion.
 */
export async function expectRelations(db: Database, issueIds: string[], want: StoredRelation[]): Promise<void> {
  const rows = await db.query(
    `SELECT i.name || ' ' || r.relation_type || ' ' || o.name AS relation, u.email AS by,
            r.deleted_at IS NOT NULL AS deleted,
            r.workspace_id = i.workspace_id AND r.project_id = i.project_id AND o.workspace_id = i.workspace_id
              AND (r.deleted_at IS NULL AND r.updated_by_id = r.created_by_id AND r.updated_at = r.created_at
                   OR r.deleted_at = r.updated_at) AS stamped
       FROM issue_relations r JOIN issues i ON i.id = r.issue_id JOIN issues o ON o.id = r.related_issue_id
       JOIN users u ON u.id = r.updated_by_id
      WHERE r.issue_id = ANY($1::uuid[]) OR r.related_issue_id = ANY($1::uuid[])
      ORDER BY r.created_at, r.id`,
    [issueIds]
  );
  expect(rows, `the relations of ${issueIds.join(", ")}`).toEqual(want.map((r) => ({ ...r, stamped: true })));
}
