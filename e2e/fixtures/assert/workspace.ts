import { expect } from "@playwright/test";

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
