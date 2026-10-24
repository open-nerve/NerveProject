import { changeWorkspacePreferences, createLabel, createProject, invite, inviteAndAccept, type Api } from "./api";

import { accountId } from "./auth";
import { createIssueComment } from "./comments";
import { createIssue, createIssueRelations } from "./issues";

// The rows a deletion story furnishes a workspace or a project with first, so that its deletion has a row of each
// table to write (expectWorkspaceDeleted, expectProjectDeleted).

/**
 * Gives the project of projectId a row of each of the work items' tables: a work item that the caller of token, a
 * member of it, creates, assigned to the account of assigneeId, a member of it who is not its guest, and labelled the
 * label of labelId, one of its labels; the creator and the assignee its subscribers, and its activity the creation
 * and the assignee; and a second work item, which the first blocks: their relation, and a row of each one's activity
 * on it.
 */
export async function furnishIssueRows(
  api: Api,
  token: string,
  projectId: string,
  assigneeId: string,
  labelId: string
): Promise<void> {
  const fix = await createIssue(api, token, projectId, {
    name: "Fix the login",
    assignee_ids: [assigneeId],
    label_ids: [labelId],
  });
  await createIssueComment(api, token, fix.id, "<p>Ready to fix</p>");
  const test = await createIssue(api, token, projectId, { name: "Test the login" });
  await createIssueRelations(api, token, fix.id, { relation_type: "blocking", issue_ids: [test.id] });
}

/**
 * Gives the workspace of slug, as its admin of adminToken, a row of each table its deletion writes
 * (expectWorkspaceDeleted): the membership of member, by an invitation he accepts (deleted as he accepts it); a pending
 * invitation of inviteeEmail; the admin's display settings; and the project Web, with its own rows: its label Bug
 * among them, and a work item assigned to the admin and labelled Bug, with its own (furnishIssueRows).
 */
export async function furnishWorkspace(
  api: Api,
  adminToken: string,
  slug: string,
  member: { email: string; token: string },
  inviteeEmail: string
): Promise<void> {
  await inviteAndAccept(api, adminToken, slug, member, 15);
  await invite(api, adminToken, slug, [{ email: inviteeEmail, role: 15 }]);
  await changeWorkspacePreferences(api, adminToken, slug, { navigation_project_limit: 3 });
  const web = await createProject(api, adminToken, slug, { name: "Web", identifier: "WEB" });
  const bug = await createLabel(api, adminToken, web.id, { name: "Bug" });
  await furnishIssueRows(api, adminToken, web.id, await accountId(api, adminToken), bug.id);
}
