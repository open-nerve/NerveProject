import type { TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  archiveProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { createdFor, expectIssueCreated } from "../../fixtures/assert/issue";
import { expectInitialDescription } from "../../fixtures/assert/issue-collaboration";
import { bearer, newAccount } from "../../fixtures/auth";
import {
  createDraft,
  createIssue,
  draftDeleted,
  draftPublished,
  draftRead,
  draftsListed,
  draftUpdated,
  issueCreated,
  issueRead,
  type IssueCreate,
} from "../../fixtures/issues";
import { createState } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I14 uses only HTTP for its preconditions. Database reads check exact publication audit stamps.
async function draftWorld(api: Api, testInfo: TestInfo) {
  const admin = await newAccount(api, testInfo, "admin");
  const member = await newAccount(api, testInfo, "member");
  const other = await newAccount(api, testInfo, "other");
  const guest = await newAccount(api, testInfo, "guest");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await Promise.all([member, other, guest].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB", network: 0 });
  await addProjectMembers(api, admin.token, project.id, [
    { member_id: member.id, role: 15 },
    { member_id: other.id, role: 15 },
    { member_id: guest.id, role: 5 },
  ]);
  return { admin, member, other, guest, slug, project };
}

test("I14 (API): a member saves missing-name input, receives create's 400 on publish, replaces the payload and publishes its description and parent at one moment; the consumed draft disappears and cannot publish twice", async ({
  api,
  db,
}, testInfo) => {
  const { member, slug, project } = await draftWorld(api, testInfo);
  // Retaining this invalid priority after PATCH would expose an incorrect payload merge.
  const incomplete = { description_html: "<p>Unfinished</p>", priority: "not-a-priority" };
  const draft = await createDraft(api, member.token, slug, { project_id: project.id, payload: incomplete });
  expect(draft).toMatchObject({ project_id: project.id, payload: incomplete });
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 200, draft });
  const refused = await draftPublished(api, member.token, draft.id);
  expect(refused).toMatchObject({ status: 400, code: "bad_request" });
  // Only this deliberately invalid call narrows around IssueCreate's required name.
  expect(refused).toEqual(await issueCreated(api, member.token, project.id, incomplete as IssueCreate));
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 200, draft });

  const parent = await createIssue(api, member.token, project.id, { name: "Release" });
  const description = "<p>Ship the saved plan</p>";
  const payload = { name: "Ship it", description_html: description, parent_id: parent.id };
  expect(await draftUpdated(api, member.token, draft.id, { payload })).toMatchObject({
    status: 200,
    draft: { id: draft.id, project_id: project.id, payload },
  });
  const saved = await draftRead(api, member.token, draft.id);
  expect(saved).toMatchObject({ status: 200 });
  if (!("draft" in saved)) throw new Error("the patched draft is missing");
  expect(saved.draft.payload).toEqual(payload);

  const published = await draftPublished(api, member.token, draft.id);
  expect(published.status, JSON.stringify(published)).toBe(201);
  if (!("issue" in published)) throw new Error("publish answered without the issue");
  const issue = published.issue;
  expect(issue).toMatchObject({
    name: "Ship it",
    description_html: description,
    project_id: project.id,
    parent_id: parent.id,
    sub_issues_count: 0,
  });
  expect(await issueRead(api, member.token, parent.id)).toMatchObject({
    status: 200,
    issue: { sub_issues_count: 1 },
  });
  await expectIssueCreated(db, issue.id, {
    sequence_id: 2,
    name: "Ship it",
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 75535,
    parent: "Release",
    completed: false,
    by: member.email,
    assignees: [],
    labels: [],
    subscribers: [member.email],
    activity: createdFor(),
  });
  await expectInitialDescription(db, issue.id, description, []);
  expect(await draftsListed(api, member.token, slug)).toEqual({
    status: 200,
    page: { data: [], next_cursor: null, total: 0 },
  });
  const missing = { status: 404, code: "issue.draft_not_found" };
  expect(await draftRead(api, member.token, draft.id)).toEqual(missing);
  expect(await draftPublished(api, member.token, draft.id)).toEqual(missing);
  // PostgreSQL compares microseconds directly; JavaScript Date would truncate them.
  expect(
    await db.query(
      `SELECT d.deleted_at = i.created_at AND d.updated_at = i.created_at AS same_moment,
              d.updated_by_id = i.created_by_id AND i.updated_by_id = i.created_by_id
                AND i.created_by_id = $3::uuid AS same_actor
         FROM draft_issues d JOIN issues i ON i.id = $2 WHERE d.id = $1`,
      [draft.id, issue.id, member.id]
    )
  ).toEqual([{ same_moment: true, same_actor: true }]);
  expect(await db.query(`SELECT count(*)::int AS count FROM issues WHERE project_id = $1`, [project.id])).toEqual([
    { count: 2 },
  ]);
});

test("I14 (API): a deleted saved state is revalidated (422), another member or admin sees no personal draft (404), a project guest cannot publish (403), and discarding creates no issue", async ({
  api,
  db,
}, testInfo) => {
  const { admin, member, other, guest, slug, project } = await draftWorld(api, testInfo);
  const state = await createState(api, admin.token, project.id, { name: "Review", color: "#0EA5E9", group: "started" });
  const draft = await createDraft(api, member.token, slug, {
    project_id: project.id,
    payload: { name: "Review release", state_id: state.id },
  });
  await Promise.all(
    [other, admin].map(async (who) => {
      const missing = { status: 404, code: "issue.draft_not_found" };
      expect(await draftRead(api, who.token, draft.id)).toEqual(missing);
      expect(await draftUpdated(api, who.token, draft.id, { payload: {} })).toEqual(missing);
      expect(await draftDeleted(api, who.token, draft.id)).toEqual(missing);
      expect(await draftPublished(api, who.token, draft.id)).toEqual(missing);
      expect(await draftsListed(api, who.token, slug)).toMatchObject({ status: 200, page: { data: [], total: 0 } });
    })
  );
  const removed = await api.DELETE("/api/v0/states/{state_id}", {
    params: { path: { state_id: state.id } },
    headers: bearer(admin.token),
  });
  expect(removed.response.status).toBe(204);
  expect(await draftPublished(api, member.token, draft.id)).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "state_id", code: "not_allowed" }],
  });
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 200, draft });
  const guestDraft = await createDraft(api, guest.token, slug, { project_id: project.id, payload: { name: "Guest" } });
  expect(await draftPublished(api, guest.token, guestDraft.id)).toEqual({ status: 403, code: "forbidden" });
  expect(await draftRead(api, guest.token, guestDraft.id)).toEqual({ status: 200, draft: guestDraft });
  expect(await draftDeleted(api, member.token, draft.id)).toEqual({ status: 204 });
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 404, code: "issue.draft_not_found" });
  expect(await draftsListed(api, member.token, slug)).toMatchObject({ status: 200, page: { data: [], total: 0 } });
  expect(await db.query(`SELECT count(*)::int AS count FROM issues WHERE project_id = $1`, [project.id])).toEqual([
    { count: 0 },
  ]);
});

test("I14 (API): archived and left private projects keep drafts in the personal list while publish applies current permissions before archive validation", async ({
  api,
}, testInfo) => {
  const { admin, member, guest, slug, project } = await draftWorld(api, testInfo);
  const draft = await createDraft(api, member.token, slug, { project_id: project.id, payload: { name: "Keep me" } });
  const guestDraft = await createDraft(api, guest.token, slug, { project_id: project.id, payload: { name: "Guest" } });
  await archiveProject(api, admin.token, project.id);
  expect(await draftsListed(api, member.token, slug)).toEqual({
    status: 200,
    page: { data: [draft], next_cursor: null, total: 1 },
  });
  expect(await draftPublished(api, member.token, draft.id)).toEqual({ status: 409, code: "project.archived" });
  expect(await draftPublished(api, guest.token, guestDraft.id)).toEqual({ status: 403, code: "forbidden" });
  // Leaving is allowed even after archive; the workspace membership still owns the draft.
  expect((await projectMemberWrites(api, project.id).leave(member.token)).status).toBe(204);
  expect(await draftsListed(api, member.token, slug)).toEqual({
    status: 200,
    page: { data: [draft], next_cursor: null, total: 1 },
  });
  expect(await draftRead(api, member.token, draft.id)).toEqual({ status: 200, draft });
  expect(await draftPublished(api, member.token, draft.id)).toEqual({ status: 404, code: "project.not_found" });
});
