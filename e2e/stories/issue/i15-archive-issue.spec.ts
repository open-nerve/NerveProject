import {
  addProjectMembers,
  amidAnotherWorkspace,
  changeProject,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import {
  archivedRow,
  changed,
  createdFor,
  expectIssueArchival,
  expectIssueCreated,
  expectIssueDeleted,
  expectIssuePage,
  itemRef,
  linkCreatedRow,
  madeBy,
  relatedRow,
  restoredRow,
} from "../../fixtures/assert/issue";
import {
  commentDeleted,
  commentUpdated,
  createCommentReaction,
  createIssueComment,
  createIssueReaction,
} from "../../fixtures/comments";
import { accountId, bearer, emailFor, newAccount } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import { modalDialog, sentWithin } from "../../fixtures/dialogs";
import { addCondition } from "../../fixtures/issue-filters-page";
import { expectListPageQuiet, issueListProject, openIssueList } from "../../fixtures/issue-list-page";
import { answerTo, registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
import {
  createIssue,
  createIssueRelations,
  issueArchived,
  issueDeleted,
  issueRelationDeleted,
  issueRelationsCreated,
  issueUnarchived,
  issueUpdated,
  issuesListed,
  listIssues,
  subIssuesAdded,
  updateIssue,
} from "../../fixtures/issues";
import { createIssueLink } from "../../fixtures/links";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I15, archiving and restoring a work item (M4 design 2, 3.8, 3.13): the API version, with P2's assertions (archiving
// and restoring, their two rows of activity, issue.not_archivable, an archived work item not changed, a guest
// refused) and P3's (an archived work item takes no work item under it, relates to none and keeps its relations). The
// archived list is P5's, the page version P13's. The archived work item's deletion at the end is P2's deleteIssue
// (3.6), whose rows expectIssueDeleted reads from the catalog: the work item under it and its relations either way
// among them, from P3, and its link from P7.

test("I15 (API): a member's work item in a started state is not archived (409); moved to a completed state, she archives it, and archiving it again writes nothing; it is not changed while it is archived (409); a guest who sees it neither restores nor archives it (403); she restores it, and restoring it again writes nothing; each archiving and restoring one row of its activity, hers at its moment; restored, a work item is put under it and it is related to two others, either way; archived again, it takes no work item under it, relates to none and keeps its relations (409), and the admin deletes it, with every row under it, at one moment", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann, Web's member; gus, its guest; each a member of Acme.
  const [ann, gus] = await Promise.all(["ann", "gus"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !gus) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, gus].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  // Web shows its guests every work item: gus sees ann's, and is refused as a guest (403), not as one who sees no
  // such work item (404).
  expect((await changeProject(api, admin.token, web.id, { guest_view_all_features: true })).status).toBe(200);
  const bug = await createLabel(api, admin.token, web.id, { name: "Bug" });
  const { "In Progress": inProgress, Done: done } = await stateIds(api, ann.token, web.id);
  if (!inProgress || !done) {
    throw new Error("Web's states hold no In Progress or no Done");
  }

  // ann's Fix the login, in In Progress, assigned to her and labelled Bug: a row of each table under it. In a state of
  // the started group it is not archived.
  const { id } = await createIssue(api, ann.token, web.id, {
    name: "Fix the login",
    description_html: `<mention-component entity_name="user_mention" entity_identifier="${ann.id}"></mention-component>`,
    state_id: inProgress,
    assignee_ids: [ann.id],
    label_ids: [bug.id],
  });
  const created = createdFor(ann);
  const asCreated = { archived: false, by: ann.email, activity: created };
  await expectIssueArchival(db, id, asCreated);
  expect(await issueArchived(api, ann.token, id)).toEqual({ status: 409, code: "issue.not_archivable" });
  await expectIssueArchival(db, id, asCreated);

  // P7's link is ann's too; the final admin deletion must stamp it with the other dependent rows.
  const link = await createIssueLink(api, ann.token, id, { url: `http://169.254.169.254/i15/${id}` });
  const linked = [...created, linkCreatedRow(link.url)];
  await expectIssueArchival(db, id, { ...asCreated, activity: linked });

  // Moved to Done, she archives it: archived at its moment, one row of its activity; archived again, nothing is
  // written, and the answer is the work item as it is.
  const comment = await createIssueComment(api, ann.token, id, "<p>Archive discussion</p>");
  await createCommentReaction(api, ann.token, comment.id);
  await createIssueReaction(api, ann.token, id);
  await updateIssue(api, ann.token, id, { state_id: done });
  const toDone = changed("state", "In Progress", "Done", "updated the state to", "Done", "In Progress");
  const archiving = await issueArchived(api, ann.token, id);
  expect(archiving).toMatchObject({ status: 200, issue: { id, archived_at: expect.any(String) } });
  const archived = { archived: true, by: ann.email, activity: [...linked, toDone, archivedRow] };
  await expectIssueArchival(db, id, archived);
  expect(await issueArchived(api, ann.token, id)).toEqual(archiving);
  await expectIssueArchival(db, id, archived);
  expectIssuePage(await listIssues(api, ann.token, web.id, { archived: "true" }), [id]);
  expectIssuePage(await listIssues(api, ann.token, web.id), []);
  expect(await issuesListed(api, gus.token, web.id, { archived: "true" })).toEqual({ status: 403, code: "forbidden" });

  // While it is archived, she does not change it, and gus, who may not, does not restore it.
  expect(await issueUpdated(api, ann.token, id, { name: "Fix the sign-in" })).toEqual({
    status: 409,
    code: "issue.archived",
  });
  expect(await issueUnarchived(api, gus.token, id)).toEqual({ status: 403, code: "forbidden" });
  await expectIssueArchival(db, id, archived);

  // She restores it: one row; restored again, nothing is written. gus does not archive it either.
  expect(await issueUnarchived(api, ann.token, id)).toMatchObject({ status: 200, issue: { id, archived_at: null } });
  const restored = { archived: false, by: ann.email, activity: [...archived.activity, restoredRow] };
  await expectIssueArchival(db, id, restored);
  expectIssuePage(await listIssues(api, ann.token, web.id, { archived: "true" }), []);
  expectIssuePage(await listIssues(api, ann.token, web.id), [id]);
  expect(await issueUnarchived(api, ann.token, id)).toMatchObject({ status: 200, issue: { id, archived_at: null } });
  expect(await issueArchived(api, gus.token, id)).toEqual({ status: 403, code: "forbidden" });
  await expectIssueArchival(db, id, restored);

  // Restored, it takes ann's Write the test under it, and she relates it to her Write the spec, stored from it, and
  // blocking her Ship the fix, stored from Ship the fix as its blocked_by: a row more of its activity for each
  // relation.
  const part = await createIssue(api, ann.token, web.id, { name: "Write the test" });
  const spec = await createIssue(api, ann.token, web.id, { name: "Write the spec" });
  const ship = await createIssue(api, ann.token, web.id, { name: "Ship the fix" });
  expect(await subIssuesAdded(api, ann.token, id, [part.id])).toMatchObject({ status: 200, issues: [{ id: part.id }] });
  const [toSpec] = await createIssueRelations(api, ann.token, id, {
    relation_type: "relates_to",
    issue_ids: [spec.id],
  });
  if (!toSpec) {
    throw new Error("relating Fix the login to Write the spec answered no relation");
  }
  const [toShip] = await createIssueRelations(api, ann.token, id, { relation_type: "blocking", issue_ids: [ship.id] });
  if (!toShip) {
    throw new Error("relating Fix the login to Ship the fix answered no relation");
  }
  const related = {
    ...restored,
    activity: [
      ...restored.activity,
      relatedRow("relates_to", itemRef("WEB", spec)),
      relatedRow("blocking", itemRef("WEB", ship)),
    ],
  };
  await expectIssueArchival(db, id, related);

  // Archived again, one row more, hers at its moment: it takes Fix the logout under it no more, relates to it no more,
  // and neither of its relations is deleted (409 issue.archived), the one stored from it nor the one stored from Ship
  // the fix; none of the four writes anything.
  expect((await issueArchived(api, ann.token, id)).status).toBe(200);
  const archivedAgain = { archived: true, by: ann.email, activity: [...related.activity, archivedRow] };
  const rejectedComment = await api.POST("/api/v0/issues/{issue_id}/comments", {
    params: { path: { issue_id: id } },
    body: { comment_html: "<p>Archived</p>" },
    headers: bearer(ann.token),
  });
  expect(rejectedComment.response.status).toBe(409);
  expect(rejectedComment.error?.code).toBe("issue.archived");
  expect(await commentUpdated(api, ann.token, comment.id, "<p>Archived edit</p>")).toEqual({
    status: 409,
    code: "issue.archived",
  });
  expect(await commentDeleted(api, ann.token, comment.id)).toEqual({ status: 409, code: "issue.archived" });

  await expectIssueArchival(db, id, archivedAgain);
  const logout = await createIssue(api, ann.token, web.id, { name: "Fix the logout" });
  // The existing sequence is unchanged: four live contrast rows, and exactly one archived row.
  expectIssuePage(await listIssues(api, ann.token, web.id, { archived: "true" }), [id]);
  const active = await listIssues(api, ann.token, web.id);
  expect(active.total).toBe(4);
  expect(active.data?.map((row) => row.id).toSorted()).toEqual([part.id, spec.id, ship.id, logout.id].toSorted());
  const refused = { status: 409, code: "issue.archived" };
  expect(await subIssuesAdded(api, ann.token, id, [logout.id])).toEqual(refused);
  expect(
    await issueRelationsCreated(api, ann.token, id, { relation_type: "relates_to", issue_ids: [logout.id] })
  ).toEqual(refused);
  expect(await issueRelationDeleted(api, ann.token, toSpec.id)).toEqual(refused);
  expect(await issueRelationDeleted(api, ann.token, toShip.id)).toEqual(refused);
  await expectIssueArchival(db, id, archivedAgain);
  await expectIssueCreated(db, logout.id, {
    ...madeBy(ann.email, "Fix the logout"),
    sequence_id: 5,
    sort_order: logout.sort_order,
    by: ann.email,
    activity: createdFor(),
  });

  // It is deleted as any other: the admin deletes it, and with it every row under it, Write the test and its relations
  // either way among them.
  expect(await issueDeleted(api, admin.token, id)).toEqual({ status: 204 });
  await expectIssueDeleted(db, id, admin.email);
});

test("I15 (page): a terminal row archives once while pending, its saved archive filter survives reload, its detail is read-only and restore returns it once", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const { member, project, states, slug } = seed;
  const item = await createIssue(api, member.token, project.id, {
    name: "Archive this terminal item",
    state_id: states.Done,
    priority: "high",
    description_html: "<p>Keep this description</p>",
  });
  const contrast = await createIssue(api, member.token, project.id, {
    name: "Other archive",
    state_id: states.Done,
    priority: "low",
  });
  expect((await issueArchived(api, member.token, contrast.id)).status).toBe(200);
  const { watch } = await openIssueList(page, seed);
  await page.getByRole("button", { name: `Options: ${item.name}`, exact: true }).click();
  await page.getByRole("menuitem", { name: "Archive", exact: true }).click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName(`Archive Work item WEB ${item.sequence_id}`);
  const archive = `/api/v0/issues/${item.id}/archive`;
  const held = await sentHeld(page, "POST", archive, () =>
    dialog.getByRole("button", { name: "Archive", exact: true }).dblclick()
  );
  await expect(dialog.getByRole("button", { name: "Archiving", exact: true })).toBeDisabled();
  await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeDisabled();
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(1);
  expect(watch.apiRequests.filter((request) => request === `POST ${archive}`)).toHaveLength(1);
  expect((await held.release()).status()).toBe(200);
  await expect(dialog).toHaveCount(0);
  await expect(page.getByText("Archive success", { exact: true })).toHaveCount(1);
  await expect(page.locator(`[id^="issue_${item.id}_"]`)).toHaveCount(0);
  await expectIssueArchival(db, item.id, {
    archived: true,
    by: member.email,
    activity: [...createdFor(), archivedRow],
  });

  const archives = `/${slug}/projects/${project.id}/archives/issues`;
  await page.goto(archives);
  await expect(page.locator(`[id^="issue_${item.id}_"]`)).toBeAttached();
  await addCondition(page, "Priority");
  await page.getByRole("option", { name: "High", exact: true }).click();
  await page.keyboard.press("Escape");
  await expect(page.locator(`[id^="issue_${contrast.id}_"]`)).toHaveCount(0);
  await page.reload();
  await expect(page.locator(`[id^="issue_${item.id}_"]`)).toBeAttached();
  await expect(page.locator(`[id^="issue_${contrast.id}_"]`)).toHaveCount(0);
  expect(
    watch.apiRequests.filter((request) => request.startsWith("PATCH ") && request.endsWith("/preferences"))
  ).toEqual([]);
  await page.goto(`${archives}/${item.id}`);
  await expect(page.getByText(item.name, { exact: true })).toBeVisible();
  await expect(page.getByText("Keep this description", { exact: true })).toBeVisible();
  await expect(page.locator('textarea, [contenteditable="true"]')).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add comment", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add link", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add relation", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add sub-work item", exact: true })).toHaveCount(0);
  const writesBefore = watch.apiRequests.filter((request) =>
    /^(PATCH|POST|DELETE) \/api\/v0\/(issues|comments)/.test(request)
  );
  expect(writesBefore).toEqual([`POST ${archive}`]);
  await page.getByRole("button", { name: `Options: ${item.name}`, exact: true }).click();
  const restore = `/api/v0/issues/${item.id}/unarchive`;
  expect(
    (
      await answerTo(page, "POST", restore, () => page.getByRole("menuitem", { name: "Restore", exact: true }).click())
    ).status()
  ).toBe(200);
  await expect(page.getByText("Restore success", { exact: true })).toHaveCount(1);
  await expectIssueArchival(db, item.id, {
    archived: false,
    by: member.email,
    activity: [...createdFor(), archivedRow, restoredRow],
  });
  expect(watch.apiRequests.filter((request) => request === `POST ${restore}`)).toHaveLength(1);
  await page.goto(seed.path);
  await expect(page.locator(`[id^="issue_${item.id}_"]`)).toBeAttached();
  await expectListPageQuiet(page, watch, browserName, 5);
});

test("I15 (page): a nonterminal row offers no archive; a refused archive keeps its dialog and a late confirmed archive after Back cannot change the new page", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const { member, project, states, slug } = seed;
  const active = await createIssue(api, member.token, project.id, {
    name: "Still in progress",
    state_id: states["In Progress"],
  });
  const item = await createIssue(api, member.token, project.id, { name: "Terminal to refuse", state_id: states.Done });
  const { watch } = await openIssueList(page, seed);
  await page.getByRole("button", { name: `Options: ${active.name}`, exact: true }).click();
  await expect(page.getByRole("menuitem", { name: /^Archive/ })).toHaveCount(0);
  await page.keyboard.press("Escape");
  const archive = `/api/v0/issues/${item.id}/archive`;
  let refuse = true;
  await page.route(`**${archive}`, async (route) => {
    if (route.request().method() !== "POST" || !refuse) return route.fallback();
    refuse = false;
    await route.fulfill({
      status: 403,
      contentType: "application/problem+json",
      body: JSON.stringify({ type: "about:blank", title: "Forbidden", status: 403, code: "forbidden" }),
    });
  });
  await page.getByRole("button", { name: `Options: ${item.name}`, exact: true }).click();
  await page.getByRole("menuitem", { name: "Archive", exact: true }).click();
  const dialog = modalDialog(page);
  expect(
    (
      await answerTo(page, "POST", archive, () => dialog.getByRole("button", { name: "Archive", exact: true }).click())
    ).status()
  ).toBe(403);
  await expect(dialog).toHaveCount(1);
  await expect(page.getByText("Your role does not allow this.", { exact: true })).toHaveCount(1);
  await expectIssueArchival(db, item.id, { archived: false, by: member.email, activity: createdFor() });
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  // Establish a client-side Back target, then keep only the POST's confirmed response in flight.
  await page
    .getByRole("complementary", { name: "Main sidebar", exact: true })
    .getByRole("link", { name: "Projects", exact: true })
    .click();
  await expect(page).toHaveURL(`/${slug}/projects`);
  await page.getByRole("heading", { name: "Web", exact: true }).click();
  await expect(page).toHaveURL(seed.path);
  await page.getByRole("button", { name: `Options: ${item.name}`, exact: true }).click();
  await page.getByRole("menuitem", { name: "Archive", exact: true }).click();
  const held = await sentHeld(page, "POST", archive, () =>
    modalDialog(page).getByRole("button", { name: "Archive", exact: true }).click()
  );
  await page.goBack();
  await expect(page).toHaveURL(`/${slug}/projects`);
  await expect(modalDialog(page)).toHaveCount(0);
  expect((await held.release()).status()).toBe(200);
  await expect(page).toHaveURL(`/${slug}/projects`);
  await expect(page.getByText("Archive success", { exact: true })).toHaveCount(0);
  await expectIssueArchival(db, item.id, {
    archived: true,
    by: member.email,
    activity: [...createdFor(), archivedRow],
  });
  expect(watch.apiRequests.filter((request) => request === `POST ${archive}`)).toHaveLength(2);
  await expectListPageQuiet(page, watch, browserName, 1, {
    failures: [`403 POST ${archive}`],
    errors:
      browserName === "chromium" || browserName === "webkit"
        ? ["Failed to load resource: the server responded with a status of 403 (Forbidden)"]
        : [],
  });
});

test("I15 (page): an admin deletes an archived item through its read-only detail and every dependent row shares the deletion stamp", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const { member, project, states, slug } = seed;
  const label = await createLabel(api, member.token, project.id, { name: "Complete graph" });
  const item = await createIssue(api, member.token, project.id, {
    name: "Archived parent to delete",
    state_id: states.Done,
    assignee_ids: [member.id],
    label_ids: [label.id],
    description_html: `<mention-component entity_name="user_mention" entity_identifier="${member.id}"></mention-component>`,
  });
  const child = await createIssue(api, member.token, project.id, { name: "Dependent child" });
  const one = await createIssue(api, member.token, project.id, { name: "Related forward" });
  const two = await createIssue(api, member.token, project.id, { name: "Related backward" });
  expect((await subIssuesAdded(api, member.token, item.id, [child.id])).status).toBe(200);
  await createIssueRelations(api, member.token, item.id, { relation_type: "relates_to", issue_ids: [one.id] });
  await createIssueRelations(api, member.token, item.id, { relation_type: "blocking", issue_ids: [two.id] });
  await createIssueLink(api, member.token, item.id, { url: `http://169.254.169.254/i15-page/${item.id}` });
  const comment = await createIssueComment(api, member.token, item.id, "<p>Read-only discussion</p>");
  await createCommentReaction(api, member.token, comment.id);
  await createIssueReaction(api, member.token, item.id);
  expect((await issueArchived(api, member.token, item.id)).status).toBe(200);
  const archives = `/${slug}/projects/${project.id}/archives/issues`;
  const { watch } = await openIssueList(page, { ...seed, path: `${archives}/${item.id}` });
  await expect(page.getByText(item.name, { exact: true })).toBeVisible();
  await expect(page.getByText("Read-only discussion", { exact: true })).toBeVisible();
  await expect(page.locator('textarea, [contenteditable="true"]')).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add relation", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add link", exact: true })).toHaveCount(0);
  const path = `/api/v0/issues/${item.id}`;
  // Header and detail each load the core; the child loader also verifies the parent before rendering.
  // Their initial core requests may coalesce. A stable read-only interaction must not restart any of them.
  const parentReads = watch.apiRequests.filter((request) => request === `GET ${path}`).length;
  expect(await sentWithin(page, "PATCH", path, () => page.getByText(item.name, { exact: true }).dblclick())).toBe(
    false
  );
  expect(watch.apiRequests.filter((request) => request === `GET ${path}`)).toHaveLength(parentReads);
  expect(watch.apiRequests.filter((request) => request === `GET ${path}/sub-issues`)).toHaveLength(1);
  await page.getByRole("button", { name: `Options: ${item.name}`, exact: true }).click();
  await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
  const dialog = modalDialog(page);
  expect(
    (
      await answerTo(page, "DELETE", path, () => dialog.getByRole("button", { name: "Delete", exact: true }).click())
    ).status()
  ).toBe(204);
  await expect(page).toHaveURL(archives);
  await expectIssueDeleted(db, item.id, member.email);
  expect(
    watch.apiRequests.filter((request) => /^(PATCH|POST|DELETE) \/api\/v0\/(issues|comments)/.test(request))
  ).toEqual([`DELETE ${path}`]);
  await expectListPageQuiet(page, watch, browserName, 1);
});

test("I15 (page): a parent with children stays mounted in active detail and peek without restarting its child read", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const parent = await createIssue(api, seed.member.token, seed.project.id, { name: "Stable parent" });
  const child = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Stable child",
    parent_id: parent.id,
  });
  const { watch } = await openIssueList(page, seed);
  await page.locator(`#issue-${parent.id}`).click();
  const peek = page.locator("#full-screen-portal");
  await expect(peek.getByText(child.name, { exact: true })).toBeVisible();
  const parentRead = `GET /api/v0/issues/${parent.id}`;
  const childrenRead = `${parentRead}/sub-issues`;
  await peek.getByRole("button", { name: "Add relation", exact: true }).click();
  await expect(page.getByRole("menuitem", { name: "Relates to", exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  expect(watch.apiRequests.filter((request) => request === childrenRead)).toHaveLength(1);
  expect(watch.apiRequests.filter((request) => request === parentRead)).toHaveLength(2);
  await peek.getByRole("button", { name: "Close the peek view", exact: true }).click();
  await page.goto(`/${seed.slug}/browse/WEB-${parent.sequence_id}`);
  await expect(page.getByText(child.name, { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Add relation", exact: true }).click();
  await expect(page.getByRole("menuitem", { name: "Relates to", exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  expect(watch.apiRequests.filter((request) => request === childrenRead)).toHaveLength(2);
  expect(watch.apiRequests.filter((request) => request === parentRead)).toHaveLength(3);
  await expectListPageQuiet(page, watch, browserName, 2);
});

test("I15 (page): a project guest may read a visible archived detail without write controls but never mounts the archived list", async ({
  api,
  page,
  signedInPage,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const { member, project, states, slug } = seed;
  const tokens = await registerOnboarded(api, emailFor(info, "archive-guest"));
  const guest = {
    id: await accountId(api, tokens.access_token),
    token: tokens.access_token,
    email: emailFor(info, "archive-guest"),
  };
  await inviteAndAccept(api, member.token, slug, guest, 5);
  await addProjectMembers(api, member.token, project.id, [{ member_id: guest.id, role: 5 }]);
  await changeProject(api, member.token, project.id, { guest_view_all_features: true });
  const item = await createIssue(api, member.token, project.id, {
    name: "Visible guest archive",
    state_id: states.Done,
    description_html: "<p>Guest reads this</p>",
  });
  expect((await issueArchived(api, member.token, item.id)).status).toBe(200);
  await signedInPage(tokens);
  const watch = await watchPage(page);
  const archives = `/${slug}/projects/${project.id}/archives/issues`;
  await page.goto(`${archives}/${item.id}`);
  await expect(page.getByText(item.name, { exact: true })).toBeVisible();
  await expect(page.getByText("Guest reads this", { exact: true })).toBeVisible();
  await expect(page.locator('textarea, [contenteditable="true"]')).toHaveCount(0);
  await page.getByRole("button", { name: `Options: ${item.name}`, exact: true }).click();
  await expect(page.getByRole("menuitem", { name: "Restore", exact: true })).toHaveCount(0);
  await expect(page.getByRole("menuitem", { name: "Delete", exact: true })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await page.goto(archives);
  await expect(page).toHaveTitle(/Archived work items/);
  await expect(page.getByRole("button", { name: "Display", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add filter", exact: true })).toHaveCount(0);
  expect(watch.apiRequests.filter((request) => request === `GET /api/v0/projects/${project.id}/issues`)).toEqual([]);
  expect(
    watch.apiRequests.filter((request) => /^(PATCH|POST|DELETE) \/api\/v0\/(issues|comments)/.test(request))
  ).toEqual([]);
  await expectListPageQuiet(page, watch, browserName, 2);
});
