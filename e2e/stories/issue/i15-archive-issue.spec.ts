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
  itemRef,
  madeBy,
  relatedRow,
  restoredRow,
} from "../../fixtures/assert/issue";
import { createCommentReaction, createIssueComment, createIssueReaction } from "../../fixtures/comments";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  createIssueRelations,
  issueArchived,
  issueDeleted,
  issueRelationDeleted,
  issueRelationsCreated,
  issueUnarchived,
  issueUpdated,
  subIssuesAdded,
  updateIssue,
} from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I15, archiving and restoring a work item (M4 design 2, 3.8, 3.13): the API version, with P2's assertions (archiving
// and restoring, their two rows of activity, issue.not_archivable, an archived work item not changed, a guest
// refused) and P3's (an archived work item takes no work item under it, relates to none and keeps its relations). The
// archived list is P5's, the page version P13's. The archived work item's deletion at the end is P2's deleteIssue
// (3.6), whose rows expectIssueDeleted reads from the catalog: the work item under it and its relations either way
// among them, from P3.

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

  // Moved to Done, she archives it: archived at its moment, one row of its activity; archived again, nothing is
  // written, and the answer is the work item as it is.
  const comment = await createIssueComment(api, ann.token, id, "<p>Archive discussion</p>");
  await createCommentReaction(api, ann.token, comment.id);
  await createIssueReaction(api, ann.token, id);
  await updateIssue(api, ann.token, id, { state_id: done });
  const toDone = changed("state", "In Progress", "Done", "updated the state to", "Done", "In Progress");
  const archiving = await issueArchived(api, ann.token, id);
  expect(archiving).toMatchObject({ status: 200, issue: { id, archived_at: expect.any(String) } });
  const archived = { archived: true, by: ann.email, activity: [...created, toDone, archivedRow] };
  await expectIssueArchival(db, id, archived);
  expect(await issueArchived(api, ann.token, id)).toEqual(archiving);
  await expectIssueArchival(db, id, archived);

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
  await expectIssueArchival(db, id, archivedAgain);
  const logout = await createIssue(api, ann.token, web.id, { name: "Fix the logout" });
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
