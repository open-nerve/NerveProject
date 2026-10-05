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

/** An invitation as a story expects it in the database. */
export interface InvitationRow {
  email: string;
  role: number;
  accepted: boolean;
  responded: boolean;
  deleted: boolean;
}

/** The columns of workspace_member_invites (M3 design 4.4): no token among them, the server keeps none (3.8). */
const invitationColumns = [
  "accepted",
  "created_at",
  "created_by_id",
  "deleted_at",
  "email",
  "id",
  "responded_at",
  "role",
  "updated_at",
  "updated_by_id",
  "workspace_id",
];

/**
 * W4, W5, W6: the invitations of the workspace of slug, deleted ones too, are want, in any order. The table
 * holds no token: its rows have the columns of the design, and none holds any of tokens. Every one was made by
 * the account of inviterEmail; an acceptance deletes the invitation at the moment of the answer.
 */
export async function expectInvitations(
  db: Database,
  slug: string,
  inviterEmail: string,
  want: InvitationRow[],
  tokens: string[] = []
): Promise<void> {
  const rows = await db.query<
    Record<string, unknown> & { email: string; accepted: boolean; responded_at: Date | null; deleted_at: Date | null }
  >(
    `SELECT i.* FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id
      WHERE w.slug = $1 ORDER BY i.email COLLATE "C"`,
    [slug]
  );
  for (const row of rows) {
    expect(Object.keys(row).toSorted(), "the columns of workspace_member_invites").toEqual(invitationColumns);
    const values = new Set(Object.values(row).map(String));
    expect(
      tokens.filter((token) => values.has(token)),
      `the invitation of ${row.email} holds no token`
    ).toEqual([]);
  }
  expect(
    rows.map((r) => ({
      email: r.email,
      role: r.role,
      accepted: r.accepted,
      responded: r.responded_at !== null,
      deleted: r.deleted_at !== null,
    })),
    `the invitations of ${slug}`
  ).toEqual(want.toSorted((a, b) => (a.email < b.email ? -1 : 1)));
  const [inviter] = await db.query<{ id: string }>("SELECT id FROM users WHERE email = $1", [inviterEmail]);
  expect(
    rows.map((r) => r.created_by_id),
    `the invitations of ${slug}, made by ${inviterEmail}`
  ).toEqual(rows.map(() => inviter?.id));
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  const answeredApart = await db.query<{ email: string }>(
    `SELECT i.email FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id
      WHERE w.slug = $1 AND i.accepted AND i.deleted_at IS DISTINCT FROM i.responded_at`,
    [slug]
  );
  expect(answeredApart, `the accepted invitations of ${slug}, deleted when answered`).toEqual([]);
}

/**
 * W5, W6: the membership of the account of email in the workspace of slug: none while want is null, else one,
 * undeleted, holding want.
 */
export async function expectMembership(
  db: Database,
  slug: string,
  email: string,
  want: { role: number; is_active: boolean } | null
): Promise<void> {
  const rows = await db.query(
    `SELECT m.role, m.is_active FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id JOIN users u ON u.id = m.member_id
      WHERE w.slug = $1 AND u.email = $2 AND m.deleted_at IS NULL`,
    [slug, email]
  );
  expect(rows, `the membership of ${email} in ${slug}`).toEqual(want === null ? [] : [want]);
}

/** The tables whose rows belong to a workspace and are deleted with it (M3 design 3.6, 4.12); P7b adds the labels. */
export const workspaceTables = [
  "workspace_members",
  "workspace_member_invites",
  "workspace_user_properties",
  "projects",
  "project_members",
  "project_user_properties",
  "states",
] as const;

/**
 * The tables whose rows a story can delete on their own before their workspace, which keep that moment: an
 * invitation, when it is accepted or deleted, or its address's membership ends (M3 design 3.8); a project, its
 * memberships, its members' display settings and its states, when the project is deleted (P4b).
 */
export const deletedAloneTables: (typeof workspaceTables)[number][] = [
  "workspace_member_invites",
  "projects",
  "project_members",
  "project_user_properties",
  "states",
];

/**
 * W2, W3: the workspace of slug is deleted by the account of adminEmail, and with it, at the same moment and by
 * the same account, every row under it that was not deleted before: its memberships, invitations and display
 * settings, its projects, their memberships, their members' display settings and their states. Each table has
 * such a row; none is left undeleted; each table of deletedAlone, whose rows the story deleted alone, has rows
 * deleted earlier, which kept their moment, and every row of the others carries the workspace's.
 */
export async function expectWorkspaceDeleted(
  db: Database,
  slug: string,
  adminEmail: string,
  deletedAlone: (typeof workspaceTables)[number][]
): Promise<void> {
  const [w] = await db.query<{ id: string; deleted_at: Date | null; updated_by_id: string; admin: string | null }>(
    `SELECT w.id, w.deleted_at, w.updated_by_id, (SELECT id FROM users WHERE email = $2) AS admin FROM workspaces w WHERE w.slug = $1`,
    [slug, adminEmail]
  );
  expect(w?.deleted_at, `${slug} deleted`).toBeInstanceOf(Date);
  expect(w?.admin, `the account of ${adminEmail}`).toEqual(expect.any(String));
  expect(w?.updated_by_id, `${slug} deleted by ${adminEmail}`).toBe(w?.admin);
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  const tables = await Promise.all(
    workspaceTables.map(async (table) => {
      const [counts] = await db.query<{
        with_it: number;
        by_another: number;
        undeleted_or_later: number;
        earlier: number;
      }>(
        `SELECT count(*) FILTER (WHERE t.deleted_at = w.deleted_at)::int AS with_it,
                count(*) FILTER (WHERE t.deleted_at = w.deleted_at
                                   AND t.updated_by_id IS DISTINCT FROM w.updated_by_id)::int AS by_another,
                count(*) FILTER (WHERE t.deleted_at IS NULL OR t.deleted_at > w.deleted_at)::int AS undeleted_or_later,
                count(*) FILTER (WHERE t.deleted_at < w.deleted_at)::int AS earlier
           FROM ${table} t JOIN workspaces w ON w.id = t.workspace_id WHERE w.id = $1`,
        [w?.id]
      );
      return {
        table,
        deletedWithIt: (counts?.with_it ?? 0) > 0,
        deletedByAnother: counts?.by_another,
        undeletedOrLater: counts?.undeleted_or_later,
        deletedEarlier: (counts?.earlier ?? 0) > 0,
      };
    })
  );
  expect(tables, `the rows under ${slug}`).toEqual(
    workspaceTables.map((table) => ({
      table,
      deletedWithIt: true,
      deletedByAnother: 0,
      undeletedOrLater: 0,
      deletedEarlier: deletedAlone.includes(table),
    }))
  );
}

/**
 * W2, W7, W12: the membership of the account of email in the workspace of slug has ended, by the account of byEmail:
 * its row kept, inactive, with its role; his memberships of the projects of projects (identifiers; each active
 * before, as the caller shows), ended with it, at the same moment and by the same account, each row kept with its
 * role; he has no other membership of the workspace's projects that is active; and no invitation to his address in
 * the workspace is pending (M3 design 3.6, 3.8).
 */
export async function expectMembershipEnded(
  db: Database,
  slug: string,
  email: string,
  byEmail: string,
  role: number,
  projects: { identifier: string; role: number }[]
): Promise<void> {
  expect(
    await db.query(
      `SELECT m.role, m.is_active, m.deleted_at, b.email AS by
         FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
         JOIN users u ON u.id = m.member_id JOIN users b ON b.id = m.updated_by_id
        WHERE w.slug = $1 AND u.email = $2`,
      [slug, email]
    ),
    `the membership of ${email} in ${slug}`
  ).toEqual([{ role, is_active: false, deleted_at: null, by: byEmail }]);
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  expect(
    await db.query(
      `SELECT p.identifier, pm.role, pm.is_active, pm.deleted_at, b.email AS by, pm.updated_at = m.updated_at AS with_it
         FROM project_members pm JOIN projects p ON p.id = pm.project_id
         JOIN workspace_members m ON m.workspace_id = pm.workspace_id AND m.member_id = pm.member_id
         JOIN workspaces w ON w.id = m.workspace_id JOIN users u ON u.id = m.member_id JOIN users b ON b.id = pm.updated_by_id
        WHERE w.slug = $1 AND u.email = $2 AND (pm.is_active OR pm.updated_at = m.updated_at)
        ORDER BY p.identifier COLLATE "C"`,
      [slug, email]
    ),
    `the memberships of ${email} of the projects of ${slug}, active or ended with it`
  ).toEqual(
    projects
      .toSorted((a, b) => (a.identifier < b.identifier ? -1 : 1))
      .map((p) => ({
        identifier: p.identifier,
        role: p.role,
        is_active: false,
        deleted_at: null,
        by: byEmail,
        with_it: true,
      }))
  );
  expect(
    await db.query(
      `SELECT i.email FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id
        WHERE w.slug = $1 AND i.email = $2 AND i.responded_at IS NULL AND i.deleted_at IS NULL`,
      [slug, email]
    ),
    `the pending invitations to ${email} in ${slug}`
  ).toEqual([]);
}

/**
 * W2, W7, W12, before an ending: who wrote last the membership of the account of email in the workspace of slug
 * ("workspace") and each of his active memberships of its projects (by identifier), each the account of an address in
 * writers, so that a claim of the ending's writing them can fail.
 */
export async function expectWrittenLastBy(
  db: Database,
  slug: string,
  email: string,
  writers: Record<string, string>
): Promise<void> {
  const rows = await db.query<{ of: string; by: string }>(
    `SELECT 'workspace' AS of, b.email AS by
       FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
       JOIN users u ON u.id = m.member_id JOIN users b ON b.id = m.updated_by_id
      WHERE w.slug = $1 AND u.email = $2 AND m.deleted_at IS NULL
     UNION ALL
     SELECT p.identifier, b.email
       FROM project_members pm JOIN projects p ON p.id = pm.project_id JOIN workspaces w ON w.id = p.workspace_id
       JOIN users u ON u.id = pm.member_id JOIN users b ON b.id = pm.updated_by_id
      WHERE w.slug = $1 AND u.email = $2 AND pm.is_active AND pm.deleted_at IS NULL`,
    [slug, email]
  );
  expect(
    Object.fromEntries(rows.map((r) => [r.of, r.by])),
    `who wrote last the memberships of ${email} in ${slug}`
  ).toEqual(writers);
}
