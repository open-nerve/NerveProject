import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import { answer, type Api } from "./api";
import { bearer } from "./auth";
import type { Database } from "./db";
import { issueRead } from "./issues";

type RecentVisit = components["schemas"]["RecentVisit"];
export type RecentVisitCreate = components["schemas"]["RecentVisitCreate"];
type RecentVisitKind = components["schemas"]["RecentVisitKind"];

/**
 * Records the caller of token's visit of a work item or a project through the navigation API (204, new or revisited):
 * an admin, a member or a guest of the target's workspace who sees the target; its workspace and project are the
 * target's (M7 P3d spec 2.4, 2.5).
 */
export async function recordVisit(api: Api, token: string, body: RecentVisitCreate): Promise<void> {
  const { error, response } = await api.POST("/api/v0/me/recent-visits", { body, headers: bearer(token) });
  expect(
    response.status,
    `record the visit of the ${body.entity_type} ${body.entity_id}: ${JSON.stringify(error)}`
  ).toBe(204);
}

/**
 * The calls on recent visits whose answer a story compares, each by the caller of token: its status, and the visits a
 * list answers, or the problem's code and fields; a visit recorded answers 204 and nothing more (M7 P3d spec 2.4).
 */
export function recentVisitCalls(api: Api) {
  return {
    record: async (token: string, body: RecentVisitCreate) => {
      const { error, response } = await api.POST("/api/v0/me/recent-visits", { body, headers: bearer(token) });
      return answer(response, error);
    },
    list: async (token: string, slug: string, entity_type?: RecentVisitKind) => {
      const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/recent-visits", {
        params: { path: { slug }, query: { entity_type } },
        headers: bearer(token),
      });
      return data ? { status: response.status, visits: data.data } : answer(response, error);
    },
  };
}

/**
 * The SQL of column, a moment, as the API writes one: RFC 3339 in UTC, its fraction of a second without trailing zeros
 * and none when it is whole (Go's time.RFC3339Nano); null for null. A stored moment so compares with an answered one.
 */
function asAnswered(column: string): string {
  return `regexp_replace(to_char(${column} AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US'), '\\.?0+$', '') || 'Z'`;
}

/** A recent visit as stored (recentVisitRows). */
export interface RecentVisitRow {
  id: string;
  type: RecentVisitKind;
  identifier: string;
  project: string;
  created: string;
  visited: string;
  deleted: string | null;
  writer: string | null;
  stamped: boolean;
}

/**
 * The recent visits of the account of userId in the workspace of workspaceId as stored (user_recent_visits; M7 P3d
 * spec 2.3), deleted ones too, the first made first: each one's id; its kind, its target's id and the target's project;
 * the moments it was made, last visited and deleted, null while it is not, each as the API writes a moment; the
 * account that wrote it last; and whether its moments are a visit's: made by its owner, written last no earlier than
 * it was made, at its last visit while it is undeleted and at its deletion once deleted. A visit the cap took is none
 * of them: it is deleted for good (spec 2.5). Read-only.
 */
export async function recentVisitRows(db: Database, userId: string, workspaceId: string): Promise<RecentVisitRow[]> {
  return db.query<RecentVisitRow>(
    `SELECT id::text AS id, entity_type AS type, entity_identifier::text AS identifier, project_id::text AS project,
            ${asAnswered("created_at")} AS created, ${asAnswered("visited_at")} AS visited,
            ${asAnswered("deleted_at")} AS deleted, updated_by_id::text AS writer,
            created_by_id = user_id AND updated_at >= created_at
              AND CASE WHEN deleted_at IS NULL THEN visited_at = updated_at ELSE deleted_at = updated_at END AS stamped
       FROM user_recent_visits WHERE user_id = $1 AND workspace_id = $2 ORDER BY created_at, id`,
    [userId, workspaceId]
  );
}

/**
 * The visit a list answers for row (recentVisitRows), undeleted: its project and, a work item's, the work item, each as
 * the caller of token, who sees them, reads it now (M7 P3d spec 2.4, 2.5).
 */
export async function listedNow(api: Api, token: string, row: RecentVisitRow): Promise<RecentVisit> {
  const [project, issue] = await Promise.all([
    api.GET("/api/v0/projects/{project_id}", { params: { path: { project_id: row.project } }, headers: bearer(token) }),
    row.type === "issue" ? issueRead(api, token, row.identifier) : null,
  ]);
  if (!project.data) {
    throw new Error(`the project ${row.project} of the visit ${row.id}: ${JSON.stringify(project.error)}`);
  }
  if (issue && !("issue" in issue)) {
    throw new Error(`the work item ${row.identifier} of the visit ${row.id}: ${JSON.stringify(issue)}`);
  }
  const { id, name, identifier, logo_props } = project.data;
  return {
    id: row.id,
    entity_type: row.type,
    entity_id: row.identifier,
    visited_at: row.visited,
    project: { id, name, identifier, logo_props },
    issue: issue && {
      id: issue.issue.id,
      sequence_id: issue.issue.sequence_id,
      name: issue.issue.name,
      state_id: issue.issue.state_id,
      priority: issue.issue.priority,
      assignee_ids: issue.issue.assignee_ids,
    },
  };
}

/**
 * The moment the work item of issueId was deleted, as recentVisitRows writes a moment, null while it is not, and the
 * account that wrote it last: what its deletion's cascade stamps every account's visit of it with (M7 P3d spec 2.6).
 * Read-only.
 */
export async function issueDeletion(db: Database, issueId: string): Promise<{ at: string | null; by: string | null }> {
  const [row] = await db.query<{ at: string | null; by: string | null }>(
    `SELECT ${asAnswered("deleted_at")} AS at, updated_by_id::text AS by FROM issues WHERE id = $1`,
    [issueId]
  );
  if (!row) {
    throw new Error(`no work item ${issueId}`);
  }
  return row;
}
