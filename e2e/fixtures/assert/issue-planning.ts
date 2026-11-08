import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of a work item's cycle and modules (M6 P3a spec 2.2, 2.4, 2.5): its rows of cycle_issues and
// module_issues, the rows of its activity they write, and its own last write. The page version and the API version
// of a story call the same function.

/** A row of a work item's cycle_issues or module_issues as expectCycleRows and expectModuleRows read it. */
export interface MembershipRow {
  /** The name of the cycle or the module it puts the work item in. */
  target: string;
  /** The moment it was added: that of the write of the work item that added it (lastWrite). */
  added: string;
  /** The moment it was removed: that of the write, or the deletion, that removed it; null while it is not. */
  removed: string | null;
  /** The address of the account that added it. */
  adder: string;
  /** The address of the account that wrote it last: that added it, or that removed it. */
  by: string;
}

/**
 * The rows of the work item of issueId in table, removed ones too, are exactly want, in the order they were added,
 * those added at one moment by name. Each is a row of the work item's project and workspace, in a cycle or a module of
 * that project, written only when it was added and, once, when it was removed (M6 P3a spec R4).
 */
async function expectMemberships(
  db: Database,
  table: "cycle_issues" | "module_issues",
  issueId: string,
  want: MembershipRow[]
): Promise<void> {
  const [column, targets] = table === "cycle_issues" ? ["cycle_id", "cycles"] : ["module_id", "modules"];
  expect(
    await db.query(
      `SELECT t.name AS target, r.created_at::text AS added, r.deleted_at::text AS removed, a.email AS adder,
              b.email AS by,
              r.workspace_id = i.workspace_id AND r.project_id = i.project_id AND t.project_id = i.project_id
                AS in_its_project,
              r.updated_at = coalesce(r.deleted_at, r.created_at) AS written_at_its_moments
         FROM ${table} r
         JOIN issues i ON i.id = r.issue_id
         JOIN ${targets} t ON t.id = r.${column}
         JOIN users a ON a.id = r.created_by_id
         JOIN users b ON b.id = r.updated_by_id
        WHERE r.issue_id = $1 ORDER BY r.created_at, t.name COLLATE "C"`,
      [issueId]
    ),
    `the ${table} rows of the work item ${issueId}`
  ).toEqual(want.map((row) => ({ ...row, in_its_project: true, written_at_its_moments: true })));
}

/** G4, G14: the rows of the work item of issueId in cycles are exactly want (expectMemberships). */
export function expectCycleRows(db: Database, issueId: string, want: MembershipRow[]): Promise<void> {
  return expectMemberships(db, "cycle_issues", issueId, want);
}

/** G11, G14: the rows of the work item of issueId in modules are exactly want (expectMemberships). */
export function expectModuleRows(db: Database, issueId: string, want: MembershipRow[]): Promise<void> {
  return expectMemberships(db, "module_issues", issueId, want);
}

/** A change of a work item's cycle or modules as a row of its activity says it (M6 P3a spec 2.4). */
export interface PlanningChange {
  field: "cycles" | "modules";
  verb: "created" | "updated" | "deleted";
  old_value: string;
  new_value: string;
  /** The name of the cycle or the module its new_identifier names, null for none. */
  named: string | null;
  /** As named, of its old_identifier. */
  was: string | null;
  comment: string;
}

/** A row of a work item's activity about its cycle or its modules, as expectPlanningActivity reads it. */
export interface PlanningActivityRow extends PlanningChange {
  /** The address of the account that acted: that wrote the work item, or deleted the cycle or the module. */
  actor: string;
  /** The moment it was written: that of the write or the deletion. */
  at: string;
}

/** The change of a work item's joining the cycle of name. */
export function cycleJoined(name: string): PlanningChange {
  return {
    field: "cycles",
    verb: "created",
    old_value: "",
    new_value: name,
    named: name,
    was: null,
    comment: "added cycle ",
  };
}

/** The change of a work item's moving from the cycle of from to the cycle of to. */
export function cycleMoved(from: string, to: string): PlanningChange {
  return {
    field: "cycles",
    verb: "updated",
    old_value: from,
    new_value: to,
    named: to,
    was: from,
    comment: "updated cycle from ",
  };
}

/** The change of a work item's leaving the cycle of name: taken out of it, or the cycle deleted. */
export function cycleLeft(name: string): PlanningChange {
  return {
    field: "cycles",
    verb: "deleted",
    old_value: name,
    new_value: "",
    named: null,
    was: name,
    comment: "removed this work item from cycle ",
  };
}

/** The change of a work item's joining the module of name. */
export function moduleJoined(name: string): PlanningChange {
  return {
    field: "modules",
    verb: "created",
    old_value: "",
    new_value: name,
    named: name,
    was: null,
    comment: "added module ",
  };
}

/** The change of a work item's leaving the module of name: taken out of it, or the module deleted. */
export function moduleLeft(name: string): PlanningChange {
  return {
    field: "modules",
    verb: "deleted",
    old_value: name,
    new_value: "",
    named: null,
    was: name,
    comment: "removed this work item from module ",
  };
}

/** changes, in order, as the rows of activity the account of actor wrote at the moment at. */
export function wrote(actor: string, at: string, ...changes: PlanningChange[]): PlanningActivityRow[] {
  return changes.map(({ field, verb, old_value, new_value, named, was, comment }) => ({
    field,
    verb,
    old_value,
    new_value,
    named,
    was,
    comment,
    actor,
    at,
  }));
}

/**
 * G4, G11, G14: the rows of the activity of the work item of issueId about its cycle and its modules, those deleted
 * with it too, are exactly want, in their order. Each is a row of the work item's project and workspace, made by the
 * account that acted.
 */
export async function expectPlanningActivity(
  db: Database,
  issueId: string,
  want: PlanningActivityRow[]
): Promise<void> {
  expect(
    await db.query(
      `SELECT a.field, a.verb, a.old_value, a.new_value, coalesce(nc.name, nm.name) AS named,
              coalesce(oc.name, om.name) AS was, a.comment, u.email AS actor, a.created_at::text AS at,
              a.workspace_id = i.workspace_id AND a.project_id = i.project_id AND a.created_by_id = a.actor_id
                AS of_its_work_item
         FROM issue_activities a
         JOIN issues i ON i.id = a.issue_id
         JOIN users u ON u.id = a.actor_id
         LEFT JOIN cycles nc ON nc.id = a.new_identifier LEFT JOIN modules nm ON nm.id = a.new_identifier
         LEFT JOIN cycles oc ON oc.id = a.old_identifier LEFT JOIN modules om ON om.id = a.old_identifier
        WHERE a.issue_id = $1 AND a.field IN ('cycles', 'modules') ORDER BY a.created_at, a.id`,
      [issueId]
    ),
    `the activity of the work item ${issueId} about its cycle and modules`
  ).toEqual(want.map((row) => ({ ...row, of_its_work_item: true })));
}

/** The last write of a work item: its moment, the address of the account that wrote it, and its deletion's moment. */
export interface LastWrite {
  at: string;
  by: string;
  /** The moment it was deleted, null while it is not. */
  deleted: string | null;
}

/** The last write of the work item of issueId (LastWrite). */
export async function lastWrite(db: Database, issueId: string): Promise<LastWrite> {
  const [row] = await db.query<{ at: string; by: string; deleted: string | null }>(
    `SELECT i.updated_at::text AS at, u.email AS by, i.deleted_at::text AS deleted
       FROM issues i JOIN users u ON u.id = i.updated_by_id WHERE i.id = $1`,
    [issueId]
  );
  if (!row) {
    throw new Error(`no work item ${issueId}`);
  }
  return row;
}

/**
 * The rows of the project of projectId that a write of its work items writes, deleted ones too: how many work items,
 * rows of their cycles and modules and rows of their activity it has, and the moment of the last write of any of the
 * first three. A refused write leaves them as they are.
 */
export async function workItemRows(db: Database, projectId: string) {
  const [row] = await db.query(
    `SELECT (SELECT count(*) FROM issues WHERE project_id = $1)::int AS issues,
            (SELECT count(*) FROM cycle_issues WHERE project_id = $1)::int AS cycle_issues,
            (SELECT count(*) FROM module_issues WHERE project_id = $1)::int AS module_issues,
            (SELECT count(*) FROM issue_activities WHERE project_id = $1)::int AS issue_activities,
            (SELECT max(updated_at)::text
               FROM (SELECT updated_at FROM issues WHERE project_id = $1
                     UNION ALL SELECT updated_at FROM cycle_issues WHERE project_id = $1
                     UNION ALL SELECT updated_at FROM module_issues WHERE project_id = $1) w) AS last_write`,
    [projectId]
  );
  return row;
}
