import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the project stories. The page version and the
// API version of a story call the same function (M3 design 2).

/**
 * What a project holds once created: what its creator gave, the identifier
 * upper-cased, the workspace's time zone when he gave none.
 */
export interface NewProject {
  name: string;
  identifier: string;
  description: string;
  network: number;
  timezone: string;
  logo_props: Record<string, unknown>;
}

/** A member a new project has, an active admin, and his place in his sidebar (M3 design 3.18). */
export interface NewMember {
  email: string;
  sort_order: number;
}

/**
 * The six states of a new project, in their order, as expectProjectCreated reads them: Plane's DEFAULT_STATES,
 * Backlog the default (M3 design 3.17), each written with the project.
 */
const newStates = [
  { name: "Backlog", color: "#60646C", sequence: 15000, group: "backlog", default: true, with_the_project: true },
  { name: "Todo", color: "#60646C", sequence: 25000, group: "unstarted", default: false, with_the_project: true },
  { name: "In Progress", color: "#F59E0B", sequence: 35000, group: "started", default: false, with_the_project: true },
  { name: "Done", color: "#46A758", sequence: 45000, group: "completed", default: false, with_the_project: true },
  { name: "Cancelled", color: "#9AA4BC", sequence: 55000, group: "cancelled", default: false, with_the_project: true },
  { name: "Triage", color: "#4E5355", sequence: 65000, group: "triage", default: false, with_the_project: true },
];

/**
 * P1, P4, W3: the undeleted project of p.identifier in the workspace of slug holds p, is not archived, has no work
 * item numbered yet (last_issue_sequence 0) and is led by the account of leadEmail, or by no one when it is null. A
 * deleted project's identifier is free again (M3 design 4.6), so a deleted project of it is not read. Its members are
 * exactly `members`, each an active admin (role 20) with his display settings in it at his place; its states are the
 * six of a new project. Every row is in the project's workspace; the account of creatorEmail wrote every row, at the
 * project's creation, and none has changed since. Returns the project's id.
 */
export async function expectProjectCreated(
  db: Database,
  slug: string,
  p: NewProject,
  creatorEmail: string,
  leadEmail: string | null,
  members: NewMember[]
): Promise<string> {
  const [project] = await db.query<{ id: string }>(
    `SELECT p.id, p.name, p.identifier, p.description, p.network, p.timezone, p.logo_props, p.last_issue_sequence,
            p.archived_at, l.email AS lead, c.email AS creator,
            p.updated_by_id = p.created_by_id AND p.updated_at = p.created_at AS unchanged
       FROM projects p
       JOIN workspaces w ON w.id = p.workspace_id
       JOIN users c ON c.id = p.created_by_id
       LEFT JOIN users l ON l.id = p.project_lead_id
      WHERE w.slug = $1 AND p.identifier = $2 AND p.deleted_at IS NULL`,
    [slug, p.identifier]
  );
  expect(project, `the project ${p.identifier} of ${slug}`).toEqual({
    ...p,
    id: expect.any(String),
    last_issue_sequence: 0,
    archived_at: null,
    lead: leadEmail,
    creator: creatorEmail,
    unchanged: true,
  });
  const id = project?.id as string;
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  expect(
    await db.query(
      `SELECT u.email, m.role, m.is_active, s.sort_order,
              m.workspace_id = p.workspace_id AND m.created_by_id = p.created_by_id
                AND m.updated_by_id = p.created_by_id AND m.created_at = p.created_at AND m.updated_at = p.created_at
                AND m.deleted_at IS NULL
              AND s.workspace_id = p.workspace_id AND s.created_by_id = p.created_by_id
                AND s.updated_by_id = p.created_by_id AND s.created_at = p.created_at AND s.updated_at = p.created_at
                AND s.deleted_at IS NULL AS with_the_project
         FROM project_members m
         JOIN projects p ON p.id = m.project_id
         JOIN users u ON u.id = m.member_id
         LEFT JOIN project_user_properties s ON s.project_id = m.project_id AND s.user_id = m.member_id
        WHERE m.project_id = $1 ORDER BY u.email COLLATE "C"`,
      [id]
    ),
    `the members of ${p.identifier} and their display settings`
  ).toEqual(
    members
      .toSorted((a, b) => (a.email < b.email ? -1 : 1))
      .map((m) => ({ email: m.email, role: 20, is_active: true, sort_order: m.sort_order, with_the_project: true }))
  );
  expect(
    await db.query(
      `SELECT s.name, s.color, s.sequence, s."group", s."default",
              s.workspace_id = p.workspace_id AND s.created_by_id = p.created_by_id
                AND s.updated_by_id = p.created_by_id AND s.created_at = p.created_at AND s.updated_at = p.created_at
                AND s.deleted_at IS NULL AS with_the_project
         FROM states s JOIN projects p ON p.id = s.project_id
        WHERE s.project_id = $1 ORDER BY s.sequence`,
      [id]
    ),
    `the states of ${p.identifier}`
  ).toEqual(newStates);
  return id;
}

/** How many rows the project tables have. */
export interface ProjectCounts {
  projects: number;
  members: number;
  preferences: number;
  states: number;
}

export async function countProjects(db: Database): Promise<ProjectCounts> {
  const [counts] = await db.query<ProjectCounts>(
    `SELECT (SELECT count(*)::int FROM projects) AS projects,
            (SELECT count(*)::int FROM project_members) AS members,
            (SELECT count(*)::int FROM project_user_properties) AS preferences,
            (SELECT count(*)::int FROM states) AS states`
  );
  if (!counts) {
    throw new Error("the counts query returned no row");
  }
  return counts;
}

/**
 * What expectMember and expectMembers read a membership (m) from: its project (p), its member's account (u), the
 * account that wrote it last (b), and his undeleted display settings in the project (s), if any, with the account that
 * wrote them last (sb).
 */
const membershipsWithSettings = `project_members m
       JOIN projects p ON p.id = m.project_id
       JOIN users u ON u.id = m.member_id
       JOIN users b ON b.id = m.updated_by_id
       LEFT JOIN project_user_properties s ON s.project_id = m.project_id AND s.user_id = m.member_id AND s.deleted_at IS NULL
       LEFT JOIN users sb ON sb.id = s.updated_by_id`;

/** A membership of a project as expectMember reads it. */
export interface Membership {
  role: number;
  is_active: boolean;
  /** His place in his sidebar; null when he has no display settings in the project. */
  sort_order: number | null;
  /** The address of the account that wrote the membership last, and his display settings in the project, if any. */
  by: string;
}

/**
 * P2, P3, P4: the account of email's undeleted membership of the project of projectId is want, or he has none when
 * want is null. The membership and his display settings in the project are rows of the project's workspace, both
 * written last by the account of want.by, and the settings were written with the membership, when he became a member,
 * or before it.
 */
export async function expectMember(
  db: Database,
  projectId: string,
  email: string,
  want: Membership | null
): Promise<void> {
  const rows = await db.query(
    `SELECT m.role, m.is_active, s.sort_order, b.email AS by, sb.email AS settings_by,
            m.workspace_id = p.workspace_id AND (s.id IS NULL OR (s.workspace_id = p.workspace_id AND s.created_at <= m.updated_at))
              AS in_its_workspace
       FROM ${membershipsWithSettings}
      WHERE m.project_id = $1 AND u.email = $2 AND m.deleted_at IS NULL`,
    [projectId, email]
  );
  expect(rows, `the membership of ${email} in ${projectId}`).toEqual(
    want === null ? [] : [{ ...want, settings_by: want.sort_order === null ? null : want.by, in_its_workspace: true }]
  );
}

/**
 * P4, W3: the project of projectId is deleted by the account of adminEmail, and with it, at the same moment and by the
 * same account, every row under it: of each table whose foreign key names projects (the catalog's list, so a table a
 * later phase adds is read too), by its project_id. Each table has such a row, and none is left undeleted.
 */
export async function expectProjectDeleted(db: Database, projectId: string, adminEmail: string): Promise<void> {
  const [project] = await db.query(
    `SELECT p.deleted_at IS NOT NULL AS deleted, u.email AS by FROM projects p JOIN users u ON u.id = p.updated_by_id WHERE p.id = $1`,
    [projectId]
  );
  expect(project, `the project ${projectId}`).toEqual({ deleted: true, by: adminEmail });
  // The labels are left out until the stories can write one: P7b's createLabel gives them the way, and P4 and W3 then
  // write and check them.
  const tables = await db.query<{ name: string }>(
    `SELECT DISTINCT c.conrelid::regclass::text AS name FROM pg_constraint c
      WHERE c.contype = 'f' AND c.confrelid = 'projects'::regclass AND c.conrelid <> 'labels'::regclass ORDER BY 1`
  );
  expect(tables.length, "the tables under projects").toBeGreaterThan(0);
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  const rows = await Promise.all(
    tables.map(async ({ name }) => {
      const [counts] = await db.query<{ with_it: number; other: number }>(
        `SELECT count(*) FILTER (WHERE t.deleted_at = p.deleted_at AND t.updated_by_id = p.updated_by_id)::int AS with_it,
                count(*) FILTER (WHERE t.deleted_at IS DISTINCT FROM p.deleted_at
                                    OR t.updated_by_id IS DISTINCT FROM p.updated_by_id)::int AS other
           FROM ${name} t JOIN projects p ON p.id = t.project_id WHERE p.id = $1`,
        [projectId]
      );
      return { table: name, deletedWithIt: (counts?.with_it ?? 0) > 0, other: counts?.other };
    })
  );
  expect(rows, `the rows under ${projectId}`).toEqual(
    tables.map(({ name }) => ({ table: name, deletedWithIt: true, other: 0 }))
  );
}

/** A membership of a project as expectMembers reads it, its member by address. */
export interface MemberRow {
  email: string;
  role: number;
  is_active: boolean;
  /** The address of the account that wrote the membership last. */
  by: string;
  /** His place in his sidebar, and who wrote his display settings in the project last. */
  sort_order: number;
  settings_by: string;
}

/**
 * P5: the undeleted memberships of the project of projectId, ended ones too, are exactly want, in their members'
 * addresses' order, each beside his undeleted display settings in the project: an ended membership keeps them. The
 * project has no other undeleted display settings, so that settings written without a membership show. Every row is
 * of the project's workspace.
 */
export async function expectMembers(db: Database, projectId: string, want: MemberRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT u.email, m.role, m.is_active, b.email AS by, s.sort_order, sb.email AS settings_by,
              m.workspace_id = p.workspace_id AND s.workspace_id = p.workspace_id AS in_its_workspace
         FROM ${membershipsWithSettings}
        WHERE m.project_id = $1 AND m.deleted_at IS NULL ORDER BY u.email COLLATE "C"`,
      [projectId]
    ),
    `the memberships of ${projectId}`
  ).toEqual(
    want
      .toSorted((a, b) => (a.email < b.email ? -1 : 1))
      .map(({ email, role, is_active, by, sort_order, settings_by }) => ({
        email,
        role,
        is_active,
        by,
        sort_order,
        settings_by,
        in_its_workspace: true,
      }))
  );
  const [settings] = await db.query<{ count: number }>(
    `SELECT count(*)::int AS count FROM project_user_properties WHERE project_id = $1 AND deleted_at IS NULL`,
    [projectId]
  );
  expect(settings?.count, `the display settings in ${projectId}`).toBe(want.length);
}

/** A state of a project as expectStates reads it. */
export interface StateRow {
  name: string;
  color: string;
  group: string;
  sequence: number;
  default: boolean;
  deleted: boolean;
  /** The address of the account that wrote the state last: created, changed, deleted it, or made it the default or not. */
  by: string;
}

/** The states of a new project that the account of creatorEmail created, as its creation made them (newStates). */
export function statesOfANewProject(creatorEmail: string): StateRow[] {
  return newStates.map(({ name, color, sequence, group, default: isDefault }) => ({
    name,
    color,
    group,
    sequence,
    default: isDefault,
    deleted: false,
    by: creatorEmail,
  }));
}

/**
 * P6: the states of the project of projectId, deleted ones too, its triage state among them, are exactly want, by
 * sequence, then name. Each is a row of the project's workspace, and a deleted one was deleted at the moment of its
 * last write, by its writer (M3 design 3.17).
 */
export async function expectStates(db: Database, projectId: string, want: StateRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT s.name, s.color, s."group" AS group, s.sequence, s."default" AS default, s.deleted_at IS NOT NULL AS deleted,
              b.email AS by, s.workspace_id = p.workspace_id AS in_its_workspace,
              coalesce(s.deleted_at = s.updated_at, true) AS deleted_with_its_last_write
         FROM states s
         JOIN projects p ON p.id = s.project_id
         JOIN users b ON b.id = s.updated_by_id
        WHERE s.project_id = $1 ORDER BY s.sequence, s.name COLLATE "C"`,
      [projectId]
    ),
    `the states of ${projectId}`
  ).toEqual(
    want
      .toSorted((a, b) => a.sequence - b.sequence || (a.name < b.name ? -1 : 1))
      .map(({ name, color, group, sequence, default: isDefault, deleted, by }) => ({
        name,
        color,
        group,
        sequence,
        default: isDefault,
        deleted,
        by,
        in_its_workspace: true,
        deleted_with_its_last_write: true,
      }))
  );
}
