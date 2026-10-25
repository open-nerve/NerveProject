import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { bearer, newAccount } from "../../fixtures/auth";
import {
  commentDeleted,
  commentPage,
  commentUpdated,
  createCommentReaction,
  createIssueComment,
  createIssueReaction,
} from "../../fixtures/comments";
import { createIssue } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// I11 exercises comment authorship and both reaction resources through the real server.
test("I11 (API): comment authors edit and delete; admins delete others but cannot edit; own reaction removal preserves other actors and PATCH preserves embedded live reactions", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const ann = await newAccount(api, testInfo, "ann");
  const bob = await newAccount(api, testInfo, "bob");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await Promise.all([ann, bob].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(
    api,
    admin.token,
    project.id,
    [ann, bob].map((who) => ({ member_id: who.id, role: 15 }))
  );
  const issue = await createIssue(api, ann.token, project.id, { name: "Discuss the release" });
  const comment = await createIssueComment(api, ann.token, issue.id, "<p>Ready</p>");
  expect(comment).toMatchObject({ actor_id: ann.id, edited_at: null, reactions: [] });
  const initially = await db.query(`SELECT id FROM issue_activities WHERE issue_id = $1`, [issue.id]);

  await Promise.all([ann, bob].map((who) => createCommentReaction(api, who.token, comment.id)));
  const withReactions = (await commentPage(api, ann.token, issue.id)).data[0];
  expect(withReactions?.reactions.map((row) => row.actor_id).toSorted()).toEqual([ann.id, bob.id].toSorted());
  const same = await commentUpdated(api, ann.token, comment.id, comment.comment_html);
  expect(same).toEqual({ status: 200, comment: withReactions });
  const edited = await commentUpdated(api, ann.token, comment.id, "<p>Ready now</p>");
  expect(edited).toMatchObject({
    status: 200,
    comment: { comment_html: "<p>Ready now</p>", reactions: withReactions?.reactions },
  });
  if (!("comment" in edited)) throw new Error("successful edit omitted comment");
  expect(edited.comment.edited_at).not.toBeNull();
  expect(edited.comment.edited_at).toBe(edited.comment.updated_at);
  expect(await commentUpdated(api, admin.token, comment.id, "<p>Admin edit</p>")).toEqual({
    status: 403,
    code: "issue.not_author",
  });
  expect(await commentUpdated(api, bob.token, comment.id, "<p>Bob edit</p>")).toEqual({
    status: 403,
    code: "issue.not_author",
  });

  const removeCommentReaction = async (token: string) => {
    const { response, error } = await api.DELETE("/api/v0/issue-comments/{comment_id}/reactions/{reaction}", {
      params: { path: { comment_id: comment.id, reaction: "128077" } },
      headers: bearer(token),
    });
    return { status: response.status, code: error?.code };
  };
  const stamps = await db.query(`SELECT updated_at, updated_by_id FROM issues WHERE id = $1`, [issue.id]);
  expect(await removeCommentReaction(ann.token)).toEqual({ status: 204, code: undefined });
  expect(await removeCommentReaction(ann.token)).toEqual({ status: 404, code: "issue.reaction_not_found" });
  expect((await commentPage(api, ann.token, issue.id)).data[0]?.reactions).toEqual(
    withReactions?.reactions.filter((row) => row.actor_id === bob.id)
  );
  expect(await db.query(`SELECT updated_at, updated_by_id FROM issues WHERE id = $1`, [issue.id])).toEqual(stamps);

  await Promise.all([ann, bob].map((who) => createIssueReaction(api, who.token, issue.id)));
  const reactions = async () => {
    const { response, data } = await api.GET("/api/v0/issues/{issue_id}/reactions", {
      params: { path: { issue_id: issue.id } },
      headers: bearer(ann.token),
    });
    expect(response.status).toBe(200);
    expect(data).toEqual({ data: expect.any(Array) });
    return data?.data ?? [];
  };
  const before = await reactions();
  expect(before.map((row) => row.actor_id).toSorted()).toEqual([ann.id, bob.id].toSorted());
  const removeIssueReaction = async () =>
    api.DELETE("/api/v0/issues/{issue_id}/reactions/{reaction}", {
      params: { path: { issue_id: issue.id, reaction: "128077" } },
      headers: bearer(ann.token),
    });
  expect((await removeIssueReaction()).response.status).toBe(204);
  const absent = await removeIssueReaction();
  expect(absent.response.status).toBe(404);
  expect(absent.error?.code).toBe("issue.reaction_not_found");
  expect(await reactions()).toEqual(before.filter((row) => row.actor_id === bob.id));

  expect(await commentDeleted(api, bob.token, comment.id)).toEqual({ status: 403, code: "issue.not_author" });
  expect(await commentDeleted(api, admin.token, comment.id)).toEqual({ status: 204 });
  expect((await commentPage(api, ann.token, issue.id)).data).toEqual([]);
  const own = await createIssueComment(api, bob.token, issue.id, "<p>Own comment</p>");
  expect(await commentDeleted(api, bob.token, own.id)).toEqual({ status: 204 });
  expect(await commentDeleted(api, bob.token, own.id)).toEqual({ status: 404, code: "issue.comment_not_found" });
  expect(await db.query(`SELECT id FROM issue_activities WHERE issue_id = $1`, [issue.id])).toEqual(initially);
  const [deleted] = await db.query(
    `SELECT c.deleted_at IS NOT NULL AND c.updated_by_id = $2 AS deleted,
    bool_and(r.deleted_at = c.deleted_at AND r.updated_by_id = c.updated_by_id) FILTER (WHERE r.actor_id = $3) AS reactions_deleted
    FROM issue_comments c JOIN comment_reactions r ON r.comment_id = c.id WHERE c.id = $1 GROUP BY c.id`,
    [comment.id, admin.id, bob.id]
  );
  expect(deleted).toEqual({ deleted: true, reactions_deleted: true });
});
