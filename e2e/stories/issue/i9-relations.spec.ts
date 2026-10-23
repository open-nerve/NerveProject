import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { createdFor, expectIssueUpdated, itemRef, relatedRow, unrelatedRow } from "../../fixtures/assert/issue";
import { expectRelations } from "../../fixtures/assert/relation";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  createIssueRelations,
  issueRelationDeleted,
  issueRelationsCreated,
  listIssueRelations,
} from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// I9, relations (M4 design 2, 3.10, 3.13): the API version, with P3's assertions (a relation within a project and one
// across two, one row each, a work item blocking another stored as the other's blocked_by, each relation's two rows of
// activity, the reverse one of the other's project, the refusals, the lists, a deletion from either work item). The
// page version is P14's.

test("I9 (API): a member relates a work item blocking another of its project and one of another project, each stored as the other's blocked_by, of the other's project: one row each, and each relation a row of the activity of each work item, of the work item's project, as the work item reads it; the other work item lists it as blocked_by; a pair related already, either way, and a work item of a project she does not see are refused at once (422); a list holds no relation to a work item its caller does not see; a member of one of the two projects deletes the relation across them, a row more on each work item: it is listed no more, nor found again (404)", async ({
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
  const [fixRef, checkRef, deployRef] = [itemRef("WEB", fix), itemRef("WEB", check), itemRef("OPS", deploy)];
  // A work item of ann's, as it is while only its relations change.
  const asMade = (name: string) => ({
    name,
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    parent: null,
    completed: false,
    assignees: [],
    labels: [],
    subscribers: [ann.email],
  });

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
    ...asMade("Fix the login"),
    by: ann.email,
    activity: [...createdFor(), relatedRow("blocking", checkRef), relatedRow("blocking", deployRef)],
    wrote: 1,
  };
  await expectIssueUpdated(db, fix.id, fixRelated);
  await expectIssueUpdated(db, check.id, {
    ...asMade("Test the login"),
    by: ann.email,
    activity: [...createdFor(), relatedRow("blocked_by", fixRef)],
    wrote: 1,
  });
  const deployRelated = {
    ...asMade("Deploy the site"),
    by: ann.email,
    activity: [...createdFor(), relatedRow("blocked_by", fixRef)],
    wrote: 1,
  };
  await expectIssueUpdated(db, deploy.id, deployRelated);
  // bob lists Test the login's relations: Fix the login blocks it.
  expect(await listIssueRelations(api, bob.token, check.id)).toMatchObject([
    { relation_type: "blocked_by", issue: { id: fix.id } },
  ]);

  // Probe the cache, of Lab, which ann does not see, and Test the login, which Fix the login relates to already, the
  // other way round, are refused at once; nothing is written.
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
  await expectRelations(db, [fix.id, check.id], stored);

  // The admin relates Fix the login to Probe the cache: ann's list of Fix the login's relations holds it not, his does.
  await createIssueRelations(api, admin.token, fix.id, { relation_type: "relates_to", issue_ids: [probe.id] });
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
  // site's, which he may not change, it is deleted as he may change Fix the login. The relation is deleted by him, and
  // each work item has a row more, his, as it reads the relation, Deploy the site's of Ops. Neither lists it, and it
  // is not found again.
  expect(await issueRelationDeleted(api, bob.token, across.id)).toEqual({ status: 204 });
  await expectRelations(
    db,
    [deploy.id],
    [{ relation: "Deploy the site blocked_by Fix the login", by: bob.email, deleted: true }]
  );
  await expectIssueUpdated(db, fix.id, {
    ...fixRelated,
    by: bob.email,
    activity: [
      ...fixRelated.activity,
      relatedRow("relates_to", itemRef("LAB", probe)),
      unrelatedRow("blocking", deployRef),
    ],
  });
  await expectIssueUpdated(db, deploy.id, {
    ...deployRelated,
    by: bob.email,
    activity: [...deployRelated.activity, unrelatedRow("blocked_by", fixRef)],
  });
  expect(await listIssueRelations(api, ann.token, deploy.id)).toEqual([]);
  expect(await listIssueRelations(api, ann.token, fix.id)).toMatchObject([
    { relation_type: "blocking", issue: { id: check.id } },
  ]);
  expect(await issueRelationDeleted(api, bob.token, across.id)).toEqual({
    status: 404,
    code: "issue.relation_not_found",
  });
});
