import {
  addProjectMembers,
  changeProject,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import { commentPage, createIssueComment } from "../../fixtures/comments";
import { createIssue, issueCreated, issueRead, listIssues } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

test("I20 (API): a member creates his issue before becoming a guest; own-only hides another detail and list row, creation is forbidden but his own comment is accepted", async ({
  api,
  db,
}, info) => {
  const admin = await newAccount(api, info, "admin");
  const guest = await newAccount(api, info, "guest");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, guest, 15);
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  expect((await changeProject(api, admin.token, project.id, { guest_view_all_features: false })).status).toBe(200);
  await addProjectMembers(api, admin.token, project.id, [{ member_id: guest.id, role: 15 }]);
  const own = await createIssue(api, guest.token, project.id, { name: "My issue" });
  const other = await createIssue(api, admin.token, project.id, { name: "Another issue" });
  expect(
    await projectMemberWrites(api, project.id).change(
      admin.token,
      await projectMembershipOf(api, admin.token, project.id, guest.id),
      5
    )
  ).toEqual({ status: 200 });
  await changeRole(api, admin.token, slug, guest.id, 5);
  expect(await issueRead(api, guest.token, other.id)).toEqual({ status: 404, code: "issue.not_found" });
  expect(await issueCreated(api, guest.token, project.id, { name: "Forbidden" })).toEqual({
    status: 403,
    code: "forbidden",
  });
  expectIssuePage(await listIssues(api, guest.token, project.id), [own.id]);
  const comment = await createIssueComment(api, guest.token, own.id, "<p>My guest comment</p>");
  expect(comment).toMatchObject({ actor_id: guest.id, issue_id: own.id, comment_html: "<p>My guest comment</p>" });
  expect((await commentPage(api, guest.token, own.id)).data).toEqual([comment]);
  expect(
    await db.query(
      "SELECT actor_id::text, issue_id::text, comment_html FROM issue_comments WHERE id=$1 AND deleted_at IS NULL",
      [comment.id]
    )
  ).toEqual([{ actor_id: guest.id, issue_id: own.id, comment_html: "<p>My guest comment</p>" }]);
  expect((await changeProject(api, admin.token, project.id, { guest_view_all_features: true })).status).toBe(200);
  expectIssuePage(await listIssues(api, guest.token, project.id), [other.id, own.id]);
  expect(await issueRead(api, guest.token, other.id)).toMatchObject({ status: 200, issue: { id: other.id } });
  expect(await issueCreated(api, guest.token, project.id, { name: "Still forbidden" })).toEqual({
    status: 403,
    code: "forbidden",
  });
});
