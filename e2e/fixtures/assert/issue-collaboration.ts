import { expect } from "@playwright/test";

import type { Database } from "../db";
import { changed } from "./issue";

/** Description audit expectations share the work-item activity factory. */
export async function expectDescriptionActivity(db: Database, issueId: string, old: string, html: string) {
  const rows = await db.query(
    `SELECT field, verb, old_value, new_value, comment, NULL AS named, NULL AS was
       FROM issue_activities WHERE issue_id = $1 AND field = 'description' AND deleted_at IS NULL
       ORDER BY created_at, id`,
    [issueId]
  );
  expect(rows).toEqual([changed("description", old, html, "updated the description to")]);
}

/** Read all collaboration rows, retaining timestamp precision beyond JavaScript Date. */
export async function collaborationRows(db: Database, issueIds: string[]) {
  const tables = [
    "issue_mentions",
    "issue_comments",
    "issue_reactions",
    "comment_reactions",
    "issue_description_versions",
  ];
  return Promise.all(
    tables.map(async (table) => ({
      table,
      rows: await db.query(
        table === "comment_reactions"
          ? `SELECT r.*, r.created_at::text AS creation_stamp, r.updated_at::text AS update_stamp, r.deleted_at::text AS deletion_stamp FROM comment_reactions r JOIN issue_comments c ON c.id = r.comment_id WHERE c.issue_id = ANY($1::uuid[]) ORDER BY r.id`
          : `SELECT r.*, r.created_at::text AS creation_stamp, r.updated_at::text AS update_stamp, r.deleted_at::text AS deletion_stamp FROM ${table} r WHERE issue_id = ANY($1::uuid[]) ORDER BY id`,
        [issueIds]
      ),
    }))
  );
}

/** Initial descriptions, filtered mentions and their saved version share creation stamps. */
export async function expectInitialDescription(db: Database, issueId: string, html: string, mentions: string[]) {
  const [issue] = await db.query(`SELECT description_html, description_stripped FROM issues WHERE id = $1`, [issueId]);
  expect(issue).toMatchObject({ description_html: html });
  const mentioned = await db.query<{ mention_id: string; stamped: boolean }>(
    `SELECT m.mention_id, m.created_at = i.created_at AND m.updated_at = i.updated_at
       AND m.created_by_id = i.created_by_id AND m.updated_by_id = i.updated_by_id AS stamped
       FROM issue_mentions m JOIN issues i ON i.id = m.issue_id WHERE i.id = $1 AND m.deleted_at IS NULL ORDER BY m.mention_id`,
    [issueId]
  );
  expect(mentioned).toEqual(mentions.toSorted().map((mention_id) => ({ mention_id, stamped: true })));
  const versions = await db.query(
    `SELECT v.description_html, v.last_saved_at = i.created_at AND v.created_at = i.created_at
       AND v.updated_at = i.updated_at AND v.owned_by_id = i.created_by_id
       AND v.created_by_id = i.created_by_id AND v.updated_by_id = i.updated_by_id AS stamped
       FROM issue_description_versions v JOIN issues i ON i.id = v.issue_id WHERE i.id = $1 AND v.deleted_at IS NULL`,
    [issueId]
  );
  expect(versions).toEqual([{ description_html: html, stamped: true }]);
}
