import { expect } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  changeWorkspacePreferences,
  createLabel,
  createProject,
  invite,
  inviteAndAccept,
  projectMemberWrites,
  type Api,
} from "./api";

import { accountId } from "./auth";
import { createCommentReaction, createIssueComment, createIssueReaction } from "./comments";
import { changeCyclePreferences, createCycle } from "./cycles";
import type { Database } from "./db";
import { submitIntakeIssue, triageIntakeIssue } from "./intake";
import { createDraft, createIssue, createIssueRelations, updateIssue } from "./issues";
import { createIssueLink } from "./links";
import { changeModulePreferences, createModule, createModuleLink } from "./modules";
import { furnishIssueFiles, storedFile } from "./storage";

// The rows a deletion story furnishes a workspace or a project with first, so that its deletion has a row of each
// table to write (expectWorkspaceDeleted, expectProjectDeleted).

/** The cycle and the module furnishPlanningRows creates in a project, which furnishIssueRows puts its work item in. */
export interface PlanningRows {
  cycleId: string;
  moduleId: string;
}

/**
 * Waits, a bounded while, for the account of receiverId to have a notification of the work item of issueId: nerve
 * sends a write's notifications from its notification.fanout job, after the write has answered (M7 P1b spec 2.9), and
 * a fan-out still pending when the work item is deleted, or when the account no longer reads it, sends him nothing.
 */
async function notified(db: Database, receiverId: string, issueId: string): Promise<void> {
  await expect
    .poll(
      async () =>
        (
          await db.query<{ count: number }>(
            "SELECT count(*)::int AS count FROM notifications WHERE receiver_id = $1 AND issue_id = $2 AND deleted_at IS NULL",
            [receiverId, issueId]
          )
        )[0]?.count,
      { timeout: 10_000, intervals: [20, 50, 100], message: `the notification of ${issueId} sent to ${receiverId}` }
    )
    .toBeGreaterThan(0);
}

/**
 * Gives the project of projectId a row of each of the work items' tables: a work item that the caller of token, a
 * member of it, creates, assigned to the account of assigneeId, a member of it who is not its guest, and labelled the
 * label of labelId, one of its labels; the creator and the assignee its subscribers, and its activity the creation
 * and the assignee; and a second work item, assigned to the account of notifiedId, a member of it who is neither its
 * guest nor the caller, which the first blocks: their relation, and a row of each one's activity on it. The first also
 * carries a description mention, its initial version, a link, a comment and both reaction resources, all created
 * through their APIs so deletion catalog assertions cover the collaboration tables, links and a project draft. The
 * second's creation sends its assignee his notifications of it (M7 P1b), which the furnishing waits for in db
 * (notified), so that a deletion finds them written. The caller then puts the first in the cycle and the module of
 * planning, furnishPlanningRows' of the project (whose cycles, modules and intake it turned on), through its own
 * fields: a row of each of the memberships' tables (M6 P3a spec 2.5); and submits a third to the project's intake,
 * where it waits for a triage: an intake row (M7 P2a spec 2.10).
 */
export async function furnishIssueRows(
  api: Api,
  db: Database,
  token: string,
  workspaceSlug: string,
  projectId: string,
  assigneeId: string,
  notifiedId: string,
  labelId: string,
  planning: PlanningRows
): Promise<void> {
  const draft = await createDraft(api, token, workspaceSlug, {
    project_id: projectId,
    payload: { name: "Unfinished" },
  });
  await storedFile(api, token, "DRAFT_ISSUE_ATTACHMENT", { kind: "draft", id: draft.id });
  const fix = await createIssue(api, token, projectId, {
    name: "Fix the login",
    description_html: `<mention-component entity_name="user_mention" entity_identifier="${assigneeId}"></mention-component>`,
    assignee_ids: [assigneeId],
    label_ids: [labelId],
  });
  await createIssueLink(api, token, fix.id, { url: `http://169.254.169.254/furnish/${fix.id}` });
  const comment = await createIssueComment(api, token, fix.id, "<p>Ready to fix</p>");
  await furnishIssueFiles(api, token, fix.id, comment.id);
  await createCommentReaction(api, token, comment.id);
  await createIssueReaction(api, token, fix.id);
  const test = await createIssue(api, token, projectId, { name: "Test the login", assignee_ids: [notifiedId] });
  await createIssueRelations(api, token, fix.id, { relation_type: "blocking", issue_ids: [test.id] });
  await updateIssue(api, token, fix.id, { cycle_id: planning.cycleId, module_ids: [planning.moduleId] });
  await submitIntakeIssue(api, token, projectId, { name: "Support the login" });
  await notified(db, notifiedId, test.id);
}

/**
 * Gives the project of projectId a row of each of the planning tables: its cycles and modules turned on by the caller of
 * adminToken, an admin of it (they are off by default), and its intake, for a work item to be submitted to
 * (furnishIssueRows); a cycle he creates, the draft Sprint 1; and a module he creates, Release 1, which he leads and is
 * the member of, with a link; and his display settings on each one's page, which his first change stores (M6 P3b spec
 * 2.5). It answers the two, for work items to join (furnishIssueRows): a deletion story furnishes its planning rows
 * before its work items' rows.
 */
export async function furnishPlanningRows(api: Api, adminToken: string, projectId: string): Promise<PlanningRows> {
  const changed = await changeProject(api, adminToken, projectId, {
    cycle_view: true,
    module_view: true,
    intake_view: true,
  });
  expect(changed.status, `turn on the cycles, modules and intake of ${projectId}: ${JSON.stringify(changed)}`).toBe(
    200
  );
  const sprint = await createCycle(api, adminToken, projectId, { name: "Sprint 1" });
  const admin = await accountId(api, adminToken);
  const release = await createModule(api, adminToken, projectId, {
    name: "Release 1",
    lead_id: admin,
    member_ids: [admin],
  });
  await createModuleLink(api, adminToken, release.id, { url: `http://169.254.169.254/furnish/${release.id}` });
  await changeCyclePreferences(api, adminToken, sprint.id, { filters: { priority: ["high"] } });
  await changeModulePreferences(api, adminToken, release.id, { filters: { priority: ["high"] } });
  return { cycleId: sprint.id, moduleId: release.id };
}

/**
 * Puts under the work item of parentId, of the project of projectId, a work item that was in the project's intake, so
 * that its intake row is under that work item, which a deletion of it writes (expectIssueDeleted; M7 P2a spec 2.10):
 * the caller of adminToken, an admin of the project, turns its intake on (it is off by default), submits the work item
 * to it and accepts it, which moves it to the project's default state, out of the intake (P2a spec 2.4, 2.5); a work
 * item as any other then, he puts it under the work item of parentId.
 */
export async function furnishAcceptedIntake(
  api: Api,
  adminToken: string,
  projectId: string,
  parentId: string
): Promise<void> {
  const changed = await changeProject(api, adminToken, projectId, { intake_view: true });
  expect(changed.status, `turn on the intake of ${projectId}: ${JSON.stringify(changed)}`).toBe(200);
  const request = await submitIntakeIssue(api, adminToken, projectId, { name: "Accepted request" });
  await triageIntakeIssue(api, adminToken, request.id, { status: "accepted" });
  await updateIssue(api, adminToken, request.id, { parent_id: parentId });
}

/**
 * Gives the workspace of slug, as its admin of adminToken, a row of each table its deletion writes
 * (expectWorkspaceDeleted): the membership of member, by an invitation he accepts (deleted as he accepts it); a pending
 * invitation of inviteeEmail; the admin's display settings; and the project Web, with its own rows: its label Bug
 * among them, a cycle and a module with its member and link (furnishPlanningRows), and a work item assigned to the admin
 * and labelled Bug, in the two, with its own (furnishIssueRows). The member is Web's for a while: the admin adds him,
 * assigns him the second work item, which sends him its notifications, and once they are written he leaves Web, so
 * that the only active membership of his in the workspace is the one he wrote (W2).
 */
export async function furnishWorkspace(
  api: Api,
  db: Database,
  adminToken: string,
  slug: string,
  member: { email: string; token: string },
  inviteeEmail: string
): Promise<void> {
  await inviteAndAccept(api, adminToken, slug, member, 15);
  await invite(api, adminToken, slug, [{ email: inviteeEmail, role: 15 }]);
  await changeWorkspacePreferences(api, adminToken, slug, { navigation_project_limit: 3 });
  const web = await createProject(api, adminToken, slug, { name: "Web", identifier: "WEB" });
  const memberId = await accountId(api, member.token);
  await addProjectMembers(api, adminToken, web.id, [{ member_id: memberId, role: 15 }]);
  const bug = await createLabel(api, adminToken, web.id, { name: "Bug" });
  const draft = await createDraft(api, adminToken, slug, { payload: { name: "Unassigned" } });
  await storedFile(api, adminToken, "DRAFT_ISSUE_ATTACHMENT", { kind: "draft", id: draft.id });
  const planning = await furnishPlanningRows(api, adminToken, web.id);
  const adminId = await accountId(api, adminToken);
  await furnishIssueRows(api, db, adminToken, slug, web.id, adminId, memberId, bug.id, planning);
  expect(await projectMemberWrites(api, web.id).leave(member.token), `${member.email} leaves Web`).toEqual({
    status: 204,
  });
}
