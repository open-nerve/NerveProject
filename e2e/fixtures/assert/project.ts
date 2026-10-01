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
 * P1, W3: the project of p.identifier in the workspace of slug holds p, is neither archived nor deleted, has no
 * work item numbered yet (last_issue_sequence 0) and is led by the account of leadEmail, or by no one when it is
 * null. Its members are members, each an active admin (role 20) with his display settings in it at his place; its
 * states are the six of a new project. The account of creatorEmail wrote every row, at the project's creation, and
 * none has changed since. Returns the project's id.
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
            p.archived_at, p.deleted_at, l.email AS lead, c.email AS creator,
            p.updated_by_id = p.created_by_id AND p.updated_at = p.created_at AS unchanged
       FROM projects p
       JOIN workspaces w ON w.id = p.workspace_id
       JOIN users c ON c.id = p.created_by_id
       LEFT JOIN users l ON l.id = p.project_lead_id
      WHERE w.slug = $1 AND p.identifier = $2`,
    [slug, p.identifier]
  );
  expect(project, `the project ${p.identifier} of ${slug}`).toEqual({
    ...p,
    id: expect.any(String),
    last_issue_sequence: 0,
    archived_at: null,
    deleted_at: null,
    lead: leadEmail,
    creator: creatorEmail,
    unchanged: true,
  });
  const id = project?.id as string;
  // The database compares the moments: a Date holds milliseconds, a timestamptz microseconds.
  expect(
    await db.query(
      `SELECT u.email, m.role, m.is_active, s.sort_order,
              m.created_by_id = p.created_by_id AND m.updated_by_id = p.created_by_id AND m.created_at = p.created_at
                AND m.updated_at = p.created_at AND m.deleted_at IS NULL
              AND s.created_by_id = p.created_by_id AND s.updated_by_id = p.created_by_id AND s.created_at = p.created_at
                AND s.updated_at = p.created_at AND s.deleted_at IS NULL AS with_the_project
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
              s.created_by_id = p.created_by_id AND s.updated_by_id = p.created_by_id AND s.created_at = p.created_at
                AND s.updated_at = p.created_at AND s.deleted_at IS NULL AS with_the_project
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
