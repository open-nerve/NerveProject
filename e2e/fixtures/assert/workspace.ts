import { expect } from "@playwright/test";

import type { WorkspacePreferences } from "../api";
import type { Database } from "../db";

// Database assertions of the workspace stories. The page version and the
// API version of a story call the same function (M3 design 2).

/** What a workspace was created with: the values it must hold. */
export interface NewWorkspace {
  name: string;
  slug: string;
  organization_size: string | null;
  timezone: string;
}

/**
 * W1, W10: the workspace of w.slug is new and holds w; the account of
 * adminEmail, a lowercased address, created it and is its only member, an
 * active admin (role 20), since the moment it was created. Returns the
 * workspace's id.
 */
export async function expectWorkspaceCreated(db: Database, adminEmail: string, w: NewWorkspace): Promise<string> {
  const admins = await db.query<{ id: string }>("SELECT id FROM users WHERE email = $1", [adminEmail]);
  expect(admins, `the account of ${adminEmail}`).toHaveLength(1);
  const admin = admins[0]?.id;
  const workspaces = await db.query<{ id: string; created_at: Date; updated_at: Date }>(
    `SELECT id, name, slug, organization_size, timezone, created_by_id, updated_by_id, created_at, updated_at, deleted_at
       FROM workspaces WHERE slug = $1`,
    [w.slug]
  );
  expect(workspaces, `the workspaces of slug ${w.slug}`).toEqual([
    {
      ...w,
      id: expect.any(String),
      created_by_id: admin,
      updated_by_id: admin,
      created_at: expect.any(Date),
      updated_at: workspaces[0]?.created_at,
      deleted_at: null,
    },
  ]);
  const id = workspaces[0]?.id as string;
  expect(
    await db.query(
      `SELECT member_id, role, is_active, created_by_id, updated_by_id, created_at, updated_at, deleted_at
         FROM workspace_members WHERE workspace_id = $1`,
      [id]
    ),
    `the members of ${w.slug}`
  ).toEqual([
    {
      member_id: admin,
      role: 20,
      is_active: true,
      created_by_id: admin,
      updated_by_id: admin,
      created_at: workspaces[0]?.created_at,
      updated_at: workspaces[0]?.created_at,
      deleted_at: null,
    },
  ]);
  return id;
}

/** How many workspaces and memberships there are. */
export interface WorkspaceCounts {
  workspaces: number;
  members: number;
}

export async function countWorkspaces(db: Database): Promise<WorkspaceCounts> {
  const [counts] = await db.query<WorkspaceCounts>(
    `SELECT (SELECT count(*)::int FROM workspaces) AS workspaces,
            (SELECT count(*)::int FROM workspace_members) AS members`
  );
  if (!counts) {
    throw new Error("the counts query returned no row");
  }
  return counts;
}

/** W1, W10: a refused creation added no workspace and no membership. */
export async function expectNoWorkspaceAdded(db: Database, before: WorkspaceCounts): Promise<void> {
  expect(await countWorkspaces(db)).toEqual(before);
}

/**
 * W8: the settings rows of the account of email in the workspace of slug:
 * none while want is null, else exactly one, undeleted, holding want and
 * written by that account. Returns the row's id, or null.
 */
export async function expectPreferences(
  db: Database,
  slug: string,
  email: string,
  want: WorkspacePreferences | null
): Promise<string | null> {
  const rows = await db.query<{ id: string }>(
    `SELECT p.id, p.navigation_control_preference, p.navigation_project_limit, p.deleted_at,
            p.created_by_id = u.id AND p.updated_by_id = u.id AS written_by_the_account
       FROM workspace_user_properties p
       JOIN workspaces w ON w.id = p.workspace_id
       JOIN users u ON u.id = p.user_id
      WHERE w.slug = $1 AND u.email = $2`,
    [slug, email]
  );
  if (want === null) {
    expect(rows, `the settings of ${email} in ${slug}`).toEqual([]);
    return null;
  }
  expect(rows, `the settings of ${email} in ${slug}`).toEqual([
    { ...want, id: expect.any(String), deleted_at: null, written_by_the_account: true },
  ]);
  return rows[0]?.id ?? null;
}
