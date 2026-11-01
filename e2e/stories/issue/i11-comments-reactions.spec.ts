import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { bearer, newAccount, accountId, emailFor, shownNameOf } from "../../fixtures/auth";
import {
  commentDeleted,
  commentPage,
  commentUpdated,
  createCommentReaction,
  createIssueComment,
  createIssueReaction,
} from "../../fixtures/comments";
import { createIssue, issueArchived } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { expectListPageQuiet, issueListProject } from "../../fixtures/issue-list-page";
import { answerTo, registerOnboarded, sentHeld } from "../../fixtures/settings-pages";

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

test("I11 (page): real comment input ignores IME and empty submission, retains a field refusal, and confirms one create and author edit", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Comment input" });
  const watch = await openIssueDetail(page, seed, issue);
  const editor = page.locator(`[id="editor-container-add_comment_${issue.id}"] [contenteditable="true"]`);
  const submit = page.getByRole("button", { name: "Comment", exact: true });
  await expect(editor).toBeVisible();
  await expect(editor).toHaveClass(/prose-brand/);
  await expect(submit).toBeDisabled();
  await editor.press("Enter");
  await editor.fill("中文输入");
  await editor.dispatchEvent("keydown", { key: "Enter", isComposing: true, bubbles: true });
  await editor.dispatchEvent("keydown", { key: "Enter", keyCode: 229, bubbles: true });
  await expect(editor).toHaveText("中文输入");
  const path = `/api/v0/issues/${issue.id}/comments`;
  expect(watch.apiRequests.filter((request) => request === `POST ${path}`)).toEqual([]);
  let refused = false;
  await page.route(
    (url) => url.pathname === path,
    async (route) => {
      if (route.request().method() !== "POST" || refused) return route.fallback();
      refused = true;
      await route.fulfill({
        status: 422,
        contentType: "application/problem+json",
        body: JSON.stringify({
          type: "about:blank",
          title: "Validation failed",
          status: 422,
          code: "validation_failed",
          errors: [{ field: "comment_html", code: "required" }],
        }),
      });
    }
  );
  expect((await answerTo(page, "POST", path, () => submit.click())).status()).toBe(422);
  const field = page.getByRole("textbox", { name: "Comment", exact: true });
  await expect(field).toHaveAttribute("aria-invalid", "true");
  await expect(field).toBeFocused();
  await expect(field).toHaveAccessibleDescription("Required");
  await expect(editor).toHaveText("中文输入");
  await editor.fill("Confirmed comment");
  const held = await sentHeld(page, "POST", path, () => submit.dblclick());
  expect(held.body).toEqual({ comment_html: expect.stringContaining("Confirmed comment") });
  await expect(submit).toBeDisabled();
  expect(watch.apiRequests.filter((request) => request === `POST ${path}`)).toHaveLength(2);
  const created = await held.release();
  expect(created.status()).toBe(201);
  const comment = await created.json();
  const row = page.locator(`[id="comment-${comment.id}"]`);
  await expect(row.getByText("Confirmed comment", { exact: true })).toBeVisible();
  await expect(editor).toBeEmpty();
  const options = row.getByRole("button", {
    name: `Options for ${shownNameOf(seed.member.email)}’s comment`,
    exact: true,
  });
  await expect(options.locator("button")).toHaveCount(0);
  await options.focus();
  await page.keyboard.press("Enter");
  await page.getByRole("menuitem", { name: "Edit", exact: true }).press("Enter");
  const editing = row.locator('[contenteditable="true"]');
  await expect(editing).toBeFocused();
  await editing.fill("Edited comment");
  await editing.dispatchEvent("keydown", { key: "Enter", isComposing: true, bubbles: true });
  await editing.dispatchEvent("keydown", { key: "Enter", keyCode: 229, bubbles: true });
  const editPath = `/api/v0/issue-comments/${comment.id}`;
  expect(watch.apiRequests.filter((request) => request === `PATCH ${editPath}`)).toEqual([]);
  const editHeld = await sentHeld(page, "PATCH", editPath, () => editing.press("Enter"));
  await expect(row.getByRole("button", { name: "Save", exact: true })).toBeDisabled();
  expect(editHeld.body).toEqual({ comment_html: expect.stringContaining("Edited comment") });
  expect((await editHeld.release()).status()).toBe(200);
  await expect(row.getByText("Edited comment", { exact: true })).toBeVisible();
  await expect(row.locator('[contenteditable="true"]')).toHaveCount(0);
  expect(watch.apiRequests.filter((request) => request === `PATCH ${editPath}`)).toHaveLength(1);
  await expectListPageQuiet(page, watch, {
    failures: [`422 POST ${path}`],
    errors:
      browserName === "firefox"
        ? []
        : ["Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)"],
  });
});

test("I11 (page): a named comment menu lets a project administrator delete another author but never edit", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const email = emailFor(info, "comment-author");
  const tokens = await registerOnboarded(api, email);
  const author = { email, token: tokens.access_token, id: await accountId(api, tokens.access_token) };
  await inviteAndAccept(api, seed.member.token, seed.slug, author, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: author.id, role: 15 }]);
  const issue = await createIssue(api, author.token, seed.project.id, { name: "Admin moderates comments" });
  const comment = await createIssueComment(api, author.token, issue.id, "<p>Author content</p>");
  const watch = await openIssueDetail(page, seed, issue);
  const row = page.locator(`[id="comment-${comment.id}"]`);
  const options = row.getByRole("button", { name: `Options for ${shownNameOf(email)}’s comment`, exact: true });
  await options.focus();
  await page.keyboard.press("Space");
  await expect(page.getByRole("menuitem", { name: "Edit", exact: true })).toHaveCount(0);
  await expect(page.getByRole("menuitem", { name: "Delete", exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(options).toBeFocused();
  await options.press("Enter");
  const path = `/api/v0/issue-comments/${comment.id}`;
  const held = await sentHeld(page, "DELETE", path, () =>
    page.getByRole("menuitem", { name: "Delete", exact: true }).press("Enter")
  );
  await expect(row.getByText("Author content", { exact: true })).toBeVisible();
  await expect(options).toBeDisabled();
  expect((await held.release()).status()).toBe(204);
  await expect(row).toHaveCount(0);
  expect(watch.apiRequests.filter((request) => request === `DELETE ${path}`)).toHaveLength(1);
  expect((await commentPage(api, author.token, issue.id)).data).toEqual([]);
  await expectListPageQuiet(page, watch);
});

test("I11 (page): both reactions remove only the current actor after confirmation and remain visible without writes when archived", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const other = await newAccount(api, info, "reaction-other");
  await inviteAndAccept(api, seed.member.token, seed.slug, other, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: other.id, role: 15 }]);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Confirmed reactions",
    state_id: seed.states.Done,
  });
  const comment = await createIssueComment(api, other.token, issue.id, "<p>Other person's comment</p>");
  await Promise.all(
    [seed.member, other].flatMap((who) => [
      createIssueReaction(api, who.token, issue.id),
      createCommentReaction(api, who.token, comment.id),
    ])
  );
  const watch = await openIssueDetail(page, seed, issue);
  const row = page.locator(`[id="comment-${comment.id}"]`);
  const commentReaction = row.getByRole("button", { name: "👍 2", exact: true });
  const issueGroup = page.getByRole("button", { name: "Add reaction", exact: true }).locator("..");
  const issueReaction = issueGroup.getByRole("button", { name: "👍 2", exact: true });
  await expect(commentReaction).toBeVisible();
  await expect(row.locator("button button")).toHaveCount(0);
  await expect(issueGroup.locator("button button")).toHaveCount(0);
  await expect(issueReaction.locator("xpath=ancestor::button")).toHaveCount(0);
  const removeComment = `/api/v0/issue-comments/${comment.id}/reactions/128077`;
  const held = await sentHeld(page, "DELETE", removeComment, () => commentReaction.dblclick());
  await expect(commentReaction).toBeDisabled();
  await expect(commentReaction).toHaveText("👍2");
  await expect(page.getByRole("searchbox", { name: "Search", exact: true })).toHaveCount(0);
  expect((await held.release()).status()).toBe(204);
  await expect(row.getByRole("button", { name: "👍 1", exact: true })).toBeEnabled();
  expect(
    (await commentPage(api, seed.member.token, issue.id)).data[0]?.reactions.map((reaction) => reaction.actor_id)
  ).toEqual([other.id]);
  const removeIssue = `/api/v0/issues/${issue.id}/reactions/128077`;
  const issueHeld = await sentHeld(page, "DELETE", removeIssue, () => issueReaction.dblclick());
  await expect(issueReaction).toBeDisabled();
  expect((await issueHeld.release()).status()).toBe(204);
  await expect(page.getByRole("button", { name: "👍 1", exact: true })).toHaveCount(2);
  const remaining = await api.GET("/api/v0/issues/{issue_id}/reactions", {
    params: { path: { issue_id: issue.id } },
    headers: bearer(seed.member.token),
  });
  expect(remaining.data?.data.map((reaction) => reaction.actor_id)).toEqual([other.id]);
  expect(watch.apiRequests.filter((request) => request === `DELETE ${removeIssue}`)).toHaveLength(1);
  expect(watch.apiRequests.filter((request) => request === `DELETE ${removeComment}`)).toHaveLength(1);
  expect(await issueArchived(api, seed.member.token, issue.id)).toMatchObject({ status: 200 });
  await page.reload();
  const archivedReactions = page.getByRole("button", { name: "👍 1", exact: true });
  await expect(archivedReactions).toHaveCount(2);
  await expect(archivedReactions.nth(0)).toBeDisabled();
  await expect(archivedReactions.nth(1)).toBeDisabled();
  await expect(page.getByRole("button", { name: /^Add reaction/ })).toHaveCount(0);
  const writes = watch.apiRequests.filter((request) => !request.startsWith("GET "));
  await archivedReactions.nth(0).dispatchEvent("click");
  await archivedReactions.nth(1).dispatchEvent("click");
  expect(watch.apiRequests.filter((request) => !request.startsWith("GET "))).toEqual(writes);
  await expectListPageQuiet(page, watch);
});
