import {
  addProjectMembers,
  changeProject,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { createdFor, expectIssueUpdated, itemRef, madeBy, relatedRow, unrelatedRow } from "../../fixtures/assert/issue";
import { expectRelations } from "../../fixtures/assert/relation";
import { accountId, emailFor, newAccount } from "../../fixtures/auth";
import {
  createIssue,
  createIssueRelations,
  issueRelationDeleted,
  issueRelationsCreated,
  listIssueRelations,
  issueUpdated,
} from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";
import { issueListProject, expectListPageQuiet } from "../../fixtures/issue-list-page";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { answerTo, registerOnboarded, sentHeld, transitionsEnded } from "../../fixtures/settings-pages";

test("I9 (page): native candidates create Blocking and cross-project Relates to; the reverse end shows Blocked by, and a member authorized only at the other end deletes the cross-project relation with both activities", async ({
  api,
  db,
  page,
  browser,
}, info) => {
  const seed = await issueListProject(api, info);
  const ops = await createProject(api, seed.member.token, seed.slug, { name: "Ops", identifier: "OPS" });
  const source = await createIssue(api, seed.member.token, seed.project.id, { name: "Implement protocol" });
  const target = await createIssue(api, seed.member.token, seed.project.id, { name: "Verify protocol" });
  const cross = await createIssue(api, seed.member.token, ops.id, { name: "Deploy protocol" });
  const email = emailFor(info, "ops-member");
  const token = (await registerOnboarded(api, email)).access_token;
  const peer = { email, token, id: await accountId(api, token) };
  await inviteAndAccept(api, seed.member.token, seed.slug, peer, 15);
  await addProjectMembers(api, seed.member.token, ops.id, [{ member_id: peer.id, role: 15 }]);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: peer.id, role: 5 }]);
  expect((await changeProject(api, seed.member.token, seed.project.id, { guest_view_all_features: true })).status).toBe(
    200
  );
  expect(await issueUpdated(api, peer.token, source.id, { name: "Not permitted" })).toEqual({
    status: 403,
    code: "forbidden",
  });
  const watch = await openIssueDetail(page, seed, source);
  const searchPath = `/api/v0/workspaces/${seed.slug}/issue-search`;
  const modal = page.getByRole("dialog", { name: "Add existing work item", exact: true });
  const firstSearch = await answerTo(page, "GET", searchPath, async () => {
    await page.getByRole("button", { name: "Add relation", exact: true }).click();
    await page.getByRole("menuitem", { name: "Blocking", exact: true }).click();
  });
  expect(Object.fromEntries(new URL(firstSearch.url()).searchParams)).toEqual({
    for: "relation",
    issue_id: source.id,
    project_id: seed.project.id,
  });
  await expect(modal.getByRole("option").filter({ hasText: source.name })).toHaveCount(0);
  await modal.getByRole("option").filter({ hasText: target.name }).click();
  const held = await sentHeld(page, "POST", `/api/v0/issues/${source.id}/relations`, () =>
    modal.getByRole("button", { name: "Add selected work items", exact: true }).click()
  );
  try {
    expect(held.body).toEqual({ relation_type: "blocking", issue_ids: [target.id] });
    await expect
      .poll(() =>
        db.query("SELECT relation_type FROM issue_relations WHERE related_issue_id=$1 AND deleted_at IS NULL", [
          source.id,
        ])
      )
      .toEqual([{ relation_type: "blocked_by" }]);
    await expect(modal).toHaveCount(1);
    await expect(page.locator(`#issue-${target.id}`)).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Relations 1", exact: true })).toHaveCount(0);
  } finally {
    expect((await held.release()).status()).toBe(201);
  }
  await expect(modal).toHaveCount(0);
  await page.getByRole("button", { name: "Relations 1", exact: true }).click();
  await expect(page.locator(`#issue-${target.id}`)).toBeVisible();
  await page.getByRole("button", { name: "Add relation", exact: true }).click();
  await answerTo(page, "GET", searchPath, () =>
    page.getByRole("menuitem", { name: "Relates to", exact: true }).click()
  );
  await expect(modal.getByRole("option").filter({ hasText: target.name })).toHaveCount(0);
  const workspaceSearch = await answerTo(page, "GET", searchPath, () =>
    modal.getByRole("switch", { name: "Workspace level" }).click()
  );
  expect(Object.fromEntries(new URL(workspaceSearch.url()).searchParams)).toEqual({
    for: "relation",
    issue_id: source.id,
  });
  await modal.getByRole("option").filter({ hasText: cross.name }).click();
  expect(
    (
      await answerTo(page, "POST", `/api/v0/issues/${source.id}/relations`, () =>
        modal.getByRole("button", { name: "Add selected work items", exact: true }).click()
      )
    ).status()
  ).toBe(201);
  await expect(modal).toHaveCount(0);
  await expect(page.locator(`#issue-${cross.id}`)).toBeVisible();
  await expectRelations(
    db,
    [source.id],
    [
      { relation: `${target.name} blocked_by ${source.name}`, by: seed.member.email, deleted: false },
      { relation: `${source.name} relates_to ${cross.name}`, by: seed.member.email, deleted: false },
    ]
  );
  const sourceRef = itemRef("WEB", source),
    targetRef = itemRef("WEB", target),
    crossRef = itemRef("OPS", cross);
  const sourceActivity = [...createdFor(), relatedRow("blocking", targetRef), relatedRow("relates_to", crossRef)];
  const crossActivity = [...createdFor(), relatedRow("relates_to", sourceRef)];
  await expectIssueUpdated(db, source.id, {
    ...madeBy(seed.member.email, source.name),
    by: seed.member.email,
    activity: sourceActivity,
    wrote: 1,
  });
  await expectIssueUpdated(db, target.id, {
    ...madeBy(seed.member.email, target.name),
    by: seed.member.email,
    activity: [...createdFor(), relatedRow("blocked_by", sourceRef)],
    wrote: 1,
  });
  await expectIssueUpdated(db, cross.id, {
    ...madeBy(seed.member.email, cross.name),
    by: seed.member.email,
    activity: crossActivity,
    wrote: 1,
  });
  await page.goto(`/${seed.slug}/browse/WEB-${target.sequence_id}`);
  await page.getByRole("button", { name: "Relations 1", exact: true }).click();
  await expect(page.getByRole("button", { name: "Blocked by", exact: true })).toBeVisible();
  await expect(page.locator(`#issue-${source.id}`)).toBeVisible();
  await expectListPageQuiet(page, watch);

  const context = await browser.newContext({ baseURL: new URL(page.url()).origin });
  try {
    const otherPage = await context.newPage();
    const otherWatch = await openIssueDetail(otherPage, { ...seed, project: ops, member: peer }, cross);
    const relationsButton = otherPage.getByRole("button", { name: "Relations 1", exact: true });
    await relationsButton.click();
    // Finish revealing the row before Playwright scrolls and clicks its menu through the clip.
    await transitionsEnded(relationsButton.locator("../.."));
    const row = otherPage.locator(`#issue-${source.id}`);
    await expect(row).toBeVisible();
    const relations = await listIssueRelations(api, peer.token, cross.id);
    const relation = relations.find((r) => r.issue.id === source.id);
    expect(relation).toBeDefined();
    if (!relation) throw new Error("the reverse end did not list its relation");
    await row.locator('button[aria-haspopup="menu"]').click();
    expect(
      (
        await answerTo(otherPage, "DELETE", `/api/v0/issue-relations/${relation.id}`, () =>
          otherPage.getByRole("menuitem", { name: "Remove relation", exact: true }).click()
        )
      ).status()
    ).toBe(204);
    await expect(row).toHaveCount(0);
    await expectListPageQuiet(otherPage, otherWatch);
  } finally {
    await context.close();
  }
  await expectRelations(
    db,
    [source.id],
    [
      { relation: `${target.name} blocked_by ${source.name}`, by: seed.member.email, deleted: false },
      { relation: `${source.name} relates_to ${cross.name}`, by: peer.email, deleted: true },
    ]
  );
  await expectIssueUpdated(db, source.id, {
    ...madeBy(seed.member.email, source.name),
    by: peer.email,
    activity: [...sourceActivity, unrelatedRow("relates_to", crossRef)],
    wrote: 1,
  });
  await expectIssueUpdated(db, cross.id, {
    ...madeBy(seed.member.email, cross.name),
    by: peer.email,
    activity: [...crossActivity, unrelatedRow("relates_to", sourceRef)],
    wrote: 1,
  });
  expect(await listIssueRelations(api, peer.token, cross.id)).toEqual([]);
});

// I9, relations (M4 design 2, 3.10, 3.13): the API version, with P3's assertions (a relation within a project and one
// across two, one row each, a work item blocking another stored as the other's blocked_by, each relation's two rows of
// activity, the reverse one of the other's project, the refusals, the lists, a deletion from either work item). The
// page version is P14's.

test("I9 (API): a member relates a work item blocking another of its project and one of another project, each stored as the other's blocked_by, of the other's project: one row each, and each relation a row of the activity of each work item, of the work item's project, as the work item reads it; the other work item lists it as blocked_by; a pair related already, either way, and a work item of a project she does not see are refused at once (422); a list holds no relation to a work item its caller does not see; a member of one of the two projects deletes the relation across them, stored from the work item he may not change, and one stored from the work item he may change to one he does not see, a row more on each work item: neither is listed any more, nor found again (404)", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann, Web's member and Ops' one; bob, Web's member alone; each a member of Acme.
  const [ann, bob] = await Promise.all(["ann", "bob"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !bob) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, bob].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const [web, ops, lab] = await amidAnotherWorkspace(api, admin.token, testInfo, async () => [
    await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" }),
    await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" }),
    // Lab is private, and the admin its one member.
    await createProject(api, admin.token, slug, { name: "Lab", identifier: "LAB", network: 0 }),
  ]);
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: bob.id, role: 15 },
  ]);
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: ann.id, role: 15 }]);
  // ann's Fix the login and Test the login in Web, Deploy the site in Ops; the admin's Probe the cache in Lab.
  const fix = await createIssue(api, ann.token, web.id, { name: "Fix the login" });
  const check = await createIssue(api, ann.token, web.id, { name: "Test the login" });
  const deploy = await createIssue(api, ann.token, ops.id, { name: "Deploy the site" });
  const probe = await createIssue(api, admin.token, lab.id, { name: "Probe the cache" });
  const [fixRef, checkRef] = [itemRef("WEB", fix), itemRef("WEB", check)];
  const [deployRef, probeRef] = [itemRef("OPS", deploy), itemRef("LAB", probe)];

  // ann relates Fix the login blocking Test the login, then blocking Deploy the site: each answered as Fix the login
  // reads it, each stored as the other's blocked_by, of the other's project: Deploy the site's of Ops. Each work item
  // has one row more for each relation, hers, of its own project, as it reads the relation: Deploy the site's of Ops,
  // blocked_by.
  expect(
    await createIssueRelations(api, ann.token, fix.id, { relation_type: "blocking", issue_ids: [check.id] })
  ).toMatchObject([{ relation_type: "blocking", issue: { id: check.id, project_id: web.id } }]);
  const [across] = await createIssueRelations(api, ann.token, fix.id, {
    relation_type: "blocking",
    issue_ids: [deploy.id],
  });
  if (!across) {
    throw new Error("relating Fix the login to Deploy the site answered no relation");
  }
  expect(across).toMatchObject({ relation_type: "blocking", issue: { id: deploy.id, project_id: ops.id } });
  const stored = [
    { relation: "Test the login blocked_by Fix the login", by: ann.email, deleted: false },
    { relation: "Deploy the site blocked_by Fix the login", by: ann.email, deleted: false },
  ];
  await expectRelations(db, [fix.id], stored);
  const fixRelated = {
    ...madeBy(ann.email, "Fix the login"),
    by: ann.email,
    activity: [...createdFor(), relatedRow("blocking", checkRef), relatedRow("blocking", deployRef)],
    wrote: 1,
  };
  await expectIssueUpdated(db, fix.id, fixRelated);
  await expectIssueUpdated(db, check.id, {
    ...madeBy(ann.email, "Test the login"),
    by: ann.email,
    activity: [...createdFor(), relatedRow("blocked_by", fixRef)],
    wrote: 1,
  });
  const deployRelated = {
    ...madeBy(ann.email, "Deploy the site"),
    by: ann.email,
    activity: [...createdFor(), relatedRow("blocked_by", fixRef)],
    wrote: 1,
  };
  await expectIssueUpdated(db, deploy.id, deployRelated);
  // bob lists Test the login's relations: Fix the login blocks it.
  expect(await listIssueRelations(api, bob.token, check.id)).toMatchObject([
    { relation_type: "blocked_by", issue: { id: fix.id } },
  ]);

  // ann relates Test the login to Probe the cache, of Lab, which she does not see, and to Fix the login, which relates
  // to it already, the other way round, by the relation stored from Test the login: both are refused at once. Nor does
  // she relate Fix the login to Deploy the site again, by another type, their relation stored from Deploy the site. No
  // relation is written.
  expect(
    await issueRelationsCreated(api, ann.token, check.id, { relation_type: "duplicate", issue_ids: [probe.id, fix.id] })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [
      { field: "issue_ids[0]", code: "not_allowed" },
      { field: "issue_ids[1]", code: "duplicate" },
    ],
  });
  expect(
    await issueRelationsCreated(api, ann.token, fix.id, { relation_type: "relates_to", issue_ids: [deploy.id] })
  ).toEqual({ status: 422, code: "validation_failed", errors: [{ field: "issue_ids[0]", code: "duplicate" }] });
  await expectRelations(db, [fix.id, check.id], stored);

  // The admin relates Fix the login to Probe the cache, stored from Fix the login: ann's list of Fix the login's
  // relations holds it not, his does.
  const [toProbe] = await createIssueRelations(api, admin.token, fix.id, {
    relation_type: "relates_to",
    issue_ids: [probe.id],
  });
  if (!toProbe) {
    throw new Error("relating Fix the login to Probe the cache answered no relation");
  }
  expect(await listIssueRelations(api, ann.token, fix.id)).toMatchObject([
    { relation_type: "blocking", issue: { id: check.id } },
    { relation_type: "blocking", issue: { id: deploy.id } },
  ]);
  expect(await listIssueRelations(api, admin.token, fix.id)).toMatchObject([
    { relation_type: "blocking", issue: { id: check.id } },
    { relation_type: "blocking", issue: { id: deploy.id } },
    { relation_type: "relates_to", issue: { id: probe.id } },
  ]);

  // bob, who is no member of Ops, deletes the relation of Fix the login and Deploy the site: stored as Deploy the
  // site's, which he may not change, it is deleted as he may change Fix the login. Then he deletes the relation of Fix
  // the login to Probe the cache, which he does not see: stored as Fix the login's, it is deleted as he may change
  // that. Each relation is deleted by him, and each of its work items has a row more, his, as it reads the relation, of
  // its own project: Deploy the site's of Ops, Probe the cache's of Lab. None of the three lists them, and neither is
  // found again.
  expect(await issueRelationDeleted(api, bob.token, across.id)).toEqual({ status: 204 });
  expect(await issueRelationDeleted(api, bob.token, toProbe.id)).toEqual({ status: 204 });
  await expectRelations(
    db,
    [deploy.id, probe.id],
    [
      { relation: "Deploy the site blocked_by Fix the login", by: bob.email, deleted: true },
      { relation: "Fix the login relates_to Probe the cache", by: bob.email, deleted: true },
    ]
  );
  await expectIssueUpdated(db, fix.id, {
    ...fixRelated,
    by: bob.email,
    activity: [
      ...fixRelated.activity,
      relatedRow("relates_to", probeRef),
      unrelatedRow("blocking", deployRef),
      unrelatedRow("relates_to", probeRef),
    ],
  });
  await expectIssueUpdated(db, deploy.id, {
    ...deployRelated,
    by: bob.email,
    activity: [...deployRelated.activity, unrelatedRow("blocked_by", fixRef)],
  });
  await expectIssueUpdated(db, probe.id, {
    ...madeBy(admin.email, "Probe the cache"),
    by: bob.email,
    activity: [...createdFor(), relatedRow("relates_to", fixRef), unrelatedRow("relates_to", fixRef)],
    wrote: 1,
  });
  expect(await listIssueRelations(api, ann.token, deploy.id)).toEqual([]);
  expect(await listIssueRelations(api, admin.token, probe.id)).toEqual([]);
  expect(await listIssueRelations(api, admin.token, fix.id)).toMatchObject([
    { relation_type: "blocking", issue: { id: check.id } },
  ]);
  const notFound = { status: 404, code: "issue.relation_not_found" };
  expect(await issueRelationDeleted(api, bob.token, across.id)).toEqual(notFound);
  expect(await issueRelationDeleted(api, bob.token, toProbe.id)).toEqual(notFound);
});
