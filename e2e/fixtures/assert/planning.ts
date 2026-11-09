import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the planning stories (M6 design 2). The page version and the API version of a story call
// the same function.

/** A cycle of a project as expectCycles reads it. */
export interface CycleRow {
  name: string;
  description: string;
  /** Its days, as YYYY-MM-DD; both null for a draft. */
  start: string | null;
  end: string | null;
  sort_order: number;
  archived: boolean;
  deleted: boolean;
  /** The address of the account that owns it: its creator. */
  owner: string;
  /** The address of the account that wrote it last: created, changed, archived, unarchived or deleted it. */
  by: string;
}

/**
 * G1–G3, G8, G18: the cycles of the project of projectId, deleted ones too, are exactly want, by name. Each is a row
 * of the project's workspace, made by its owner; an archived one was archived, and a deleted one deleted, at a moment
 * of its writes, a deleted one at its last (M6 P1 spec 2.4).
 */
export async function expectCycles(db: Database, projectId: string, want: CycleRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT c.name, c.description, c.start_date::text AS start, c.end_date::text AS end, c.sort_order,
              c.archived_at IS NOT NULL AS archived, c.deleted_at IS NOT NULL AS deleted, o.email AS owner,
              b.email AS by,
              c.workspace_id = p.workspace_id AND c.created_by_id = c.owned_by_id AS in_its_project,
              coalesce(c.archived_at BETWEEN c.created_at AND c.updated_at, true)
                AND coalesce(c.deleted_at = c.updated_at, true) AS written_at_its_moments
         FROM cycles c
         JOIN projects p ON p.id = c.project_id
         JOIN users o ON o.id = c.owned_by_id
         JOIN users b ON b.id = c.updated_by_id
        WHERE c.project_id = $1 ORDER BY c.name COLLATE "C"`,
      [projectId]
    ),
    `the cycles of ${projectId}`
  ).toEqual(
    want
      .toSorted((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0))
      .map(({ name, description, start, end, sort_order, archived, deleted, owner, by }) => ({
        name,
        description,
        start,
        end,
        sort_order,
        archived,
        deleted,
        owner,
        by,
        in_its_project: true,
        written_at_its_moments: true,
      }))
  );
}

/** The moment of the last write of each cycle of the project of projectId, by name, deleted ones too. */
export async function cycleMoments(db: Database, projectId: string): Promise<Record<string, string>> {
  const rows = await db.query<{ name: string; at: string }>(
    `SELECT name, updated_at::text AS at FROM cycles WHERE project_id = $1`,
    [projectId]
  );
  return Object.fromEntries(rows.map((r) => [r.name, r.at]));
}

/** A module of a project as expectModules reads it. */
export interface ModuleRow {
  name: string;
  description: string;
  /** Its days, as YYYY-MM-DD, each null for none. */
  start: string | null;
  target: string | null;
  status: string;
  /** The address of the account that leads it, null for none. */
  lead: string | null;
  /** The addresses of the accounts of its undeleted members' rows, in order. */
  members: string[];
  sort_order: number;
  archived: boolean;
  deleted: boolean;
  /** The address of the account that created it. */
  creator: string;
  /** The address of the account that wrote it last: created, changed, archived, unarchived or deleted it. */
  by: string;
}

/**
 * G9, G13: the modules of the project of projectId, deleted ones too, are exactly want, by name. Each is a row of the
 * project's workspace; an archived one was archived, and a deleted one deleted, at a moment of its writes, a deleted
 * one at its last (M6 P2 spec 2.4).
 */
export async function expectModules(db: Database, projectId: string, want: ModuleRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT m.name, m.description, m.start_date::text AS start, m.target_date::text AS target, m.status,
              l.email AS lead,
              ARRAY(SELECT u.email::text FROM module_members mm JOIN users u ON u.id = mm.member_id
                     WHERE mm.module_id = m.id AND mm.deleted_at IS NULL ORDER BY u.email COLLATE "C") AS members,
              m.sort_order, m.archived_at IS NOT NULL AS archived, m.deleted_at IS NOT NULL AS deleted,
              c.email AS creator, b.email AS by,
              m.workspace_id = p.workspace_id AS in_its_project,
              coalesce(m.archived_at BETWEEN m.created_at AND m.updated_at, true)
                AND coalesce(m.deleted_at = m.updated_at, true) AS written_at_its_moments
         FROM modules m
         JOIN projects p ON p.id = m.project_id
         LEFT JOIN users l ON l.id = m.lead_id
         JOIN users c ON c.id = m.created_by_id
         JOIN users b ON b.id = m.updated_by_id
        WHERE m.project_id = $1 ORDER BY m.name COLLATE "C"`,
      [projectId]
    ),
    `the modules of ${projectId}`
  ).toEqual(
    want
      .toSorted((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0))
      .map((m) => ({
        name: m.name,
        description: m.description,
        start: m.start,
        target: m.target,
        status: m.status,
        lead: m.lead,
        members: m.members.toSorted(),
        sort_order: m.sort_order,
        archived: m.archived,
        deleted: m.deleted,
        creator: m.creator,
        by: m.by,
        in_its_project: true,
        written_at_its_moments: true,
      }))
  );
}

/** The moment of the last write of each module of the project of projectId, by name, deleted ones too. */
export async function moduleMoments(db: Database, projectId: string): Promise<Record<string, string>> {
  const rows = await db.query<{ name: string; at: string }>(
    `SELECT name, updated_at::text AS at FROM modules WHERE project_id = $1`,
    [projectId]
  );
  return Object.fromEntries(rows.map((r) => [r.name, r.at]));
}

/** A row of a module's members as expectModuleMembers reads it. */
export interface ModuleMemberRow {
  /** The address of the member's account. */
  member: string;
  /** The moment the row was added: that of the module's write that added it (moduleMoments). */
  added: string;
  /** The moment it was removed: that of the change or the deletion of the module that removed it; null while it is not. */
  removed: string | null;
  /** The address of the account that added it. */
  adder: string;
  /** The address of the account that wrote it last: that added it, or removed it. */
  by: string;
}

/**
 * G9, G13: the rows of the members of the module of moduleId, removed ones too, are exactly want, in the order they
 * were added, those added at one moment by address. Each is a row of the module's project and workspace, written
 * only when it was added and, once, when it was removed: a member a change keeps is left as it is (M6 P2 spec R5).
 */
export async function expectModuleMembers(db: Database, moduleId: string, want: ModuleMemberRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT u.email AS member, mm.created_at::text AS added, mm.deleted_at::text AS removed, a.email AS adder,
              b.email AS by,
              mm.workspace_id = m.workspace_id AND mm.project_id = m.project_id AS in_its_module,
              mm.updated_at = coalesce(mm.deleted_at, mm.created_at) AS written_at_its_moments
         FROM module_members mm
         JOIN modules m ON m.id = mm.module_id
         JOIN users u ON u.id = mm.member_id
         JOIN users a ON a.id = mm.created_by_id
         JOIN users b ON b.id = mm.updated_by_id
        WHERE mm.module_id = $1 ORDER BY mm.created_at, u.email COLLATE "C"`,
      [moduleId]
    ),
    `the members of the module ${moduleId}`
  ).toEqual(
    want.map(({ member, added, removed, adder, by }) => ({
      member,
      added,
      removed,
      adder,
      by,
      in_its_module: true,
      written_at_its_moments: true,
    }))
  );
}

/** A link of a module as expectModuleLinks reads it; its metadata, which the crawl writes, aside. */
export interface ModuleLinkRow {
  url: string;
  title: string | null;
  /** The address of the account that added it. */
  adder: string;
  /** The address of the account that wrote it last: added, changed or deleted it. */
  by: string;
  /** The moment it was deleted, by itself or with its module; null while it is not. */
  removed: string | null;
}

/**
 * G10, G13: the links of the module of moduleId, deleted ones too, are exactly want, by address, those of one address
 * in the order they were added. Each is a row of the module's project and workspace, a deleted one deleted at its
 * last write.
 */
export async function expectModuleLinks(db: Database, moduleId: string, want: ModuleLinkRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT l.url, l.title, a.email AS adder, b.email AS by, l.deleted_at::text AS removed,
              l.workspace_id = m.workspace_id AND l.project_id = m.project_id AS in_its_module,
              coalesce(l.deleted_at = l.updated_at, true) AS written_at_its_moments
         FROM module_links l
         JOIN modules m ON m.id = l.module_id
         JOIN users a ON a.id = l.created_by_id
         JOIN users b ON b.id = l.updated_by_id
        WHERE l.module_id = $1 ORDER BY l.url COLLATE "C", l.created_at`,
      [moduleId]
    ),
    `the links of the module ${moduleId}`
  ).toEqual(
    want
      .toSorted((a, b) => (a.url < b.url ? -1 : a.url > b.url ? 1 : 0))
      .map(({ url, title, adder, by, removed }) => ({
        url,
        title,
        adder,
        by,
        removed,
        in_its_module: true,
        written_at_its_moments: true,
      }))
  );
}

/** The moment of the last write of each link of the module of moduleId, by address, deleted ones too. */
export async function moduleLinkMoments(db: Database, moduleId: string): Promise<Record<string, string>> {
  const rows = await db.query<{ url: string; at: string }>(
    `SELECT url, updated_at::text AS at FROM module_links WHERE module_id = $1`,
    [moduleId]
  );
  return Object.fromEntries(rows.map((r) => [r.url, r.at]));
}

/** A row of an account's display settings on the page of a cycle or a module, as pageSettingsRows reads it. */
export interface PageSettingsRow {
  /** The address of the account whose settings it holds. */
  user: string;
  filters: unknown;
  display_filters: unknown;
  display_properties: unknown;
  /** The address of the account that wrote it last: its own, or the one that deleted its cycle or module. */
  by: string;
  /** The moment it was made, its account's first change on the page. */
  created: string;
  /** The moment it was written last. */
  updated: string;
  /** The moment it was deleted, null while it is not. */
  deleted: string | null;
}

/**
 * G15: the rows of the display settings on the page of the cycle or the module of id, kept in table, deleted ones too,
 * by account address, those of one account in the order they were made. Each must be a row of its page's project and
 * workspace, made by its own account (M6 P3b spec 2.3, 2.4).
 */
export async function pageSettingsRows(
  db: Database,
  table: "cycle_user_properties" | "module_user_properties",
  id: string
): Promise<PageSettingsRow[]> {
  const [column, targets] = table === "cycle_user_properties" ? ["cycle_id", "cycles"] : ["module_id", "modules"];
  const rows = await db.query<PageSettingsRow & { of_its_page: boolean }>(
    `SELECT u.email AS user, r.filters, r.display_filters, r.display_properties, b.email AS by,
            r.created_at::text AS created, r.updated_at::text AS updated, r.deleted_at::text AS deleted,
            r.workspace_id = t.workspace_id AND r.project_id = t.project_id AND r.created_by_id = r.user_id
              AS of_its_page
       FROM ${table} r
       JOIN ${targets} t ON t.id = r.${column}
       JOIN users u ON u.id = r.user_id
       JOIN users b ON b.id = r.updated_by_id
      WHERE r.${column} = $1 ORDER BY u.email COLLATE "C", r.created_at`,
    [id]
  );
  expect(
    rows.filter((r) => !r.of_its_page).map((r) => r.user),
    `the accounts of the rows of ${table} of ${id} that are not of its page or not made by their account`
  ).toEqual([]);
  return rows.map(({ user, filters, display_filters, display_properties, by, created, updated, deleted }) => ({
    user,
    filters,
    display_filters,
    display_properties,
    by,
    created,
    updated,
    deleted,
  }));
}
