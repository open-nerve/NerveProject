import {
  addProjectMembers,
  amidAnotherWorkspace,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  projectMembershipOf,
  slugFor,
} from "../../fixtures/api";
import { expectIssueDeleted } from "../../fixtures/assert/issue";
import { createIssueComment, createIssueReaction } from "../../fixtures/comments";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  createIssueRelations,
  issueArchived,
  issueDeleted,
  issueRead,
  listIssueRelations,
} from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I17, deleting a work item and the work items under it (M4 design 2, 3.4, 3.6, 3.9, 3.10): the API version, with
// P3's assertions (the work item, two levels of work items under it, an archived one among them, and their assignees,
// labels, subscribers, activity and relations either way deleted at one moment by its deleter; who may delete it).
// Comments and issue reactions are furnished through P4's API; comment reactions, mentions and versions are later in P4, the links P7's, the page version P14's.

test("I17 (API): a member does not delete another's work item (403), a guest his own (403), nor a creator who has left the private project his own (404); a member deletes her own, which has two levels of work items under it, the lower archived, and relates to two work items of another project, one relation stored from each end: with it, at that moment and by her, every row under it, the work items under it and its relations either way among them; none of the three is found again (404), and the other work items list no relation", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann, bob, gus and lea, Web's members, ann Ops' too; each a member of Acme.
  const [ann, bob, gus, lea] = await Promise.all(
    ["ann", "bob", "gus", "lea"].map((label) => newAccount(api, testInfo, label))
  );
  if (!ann || !bob || !gus || !lea) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, bob, gus, lea].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  // Web is private: who leaves it sees none of its work items.
  const [web, ops] = await amidAnotherWorkspace(api, admin.token, testInfo, async () => [
    await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB", network: 0 }),
    await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" }),
  ]);
  await addProjectMembers(
    api,
    admin.token,
    web.id,
    [ann, bob, gus, lea].map(({ id }) => ({ member_id: id, role: 15 }))
  );
  await addProjectMembers(api, admin.token, ops.id, [{ member_id: ann.id, role: 15 }]);
  const bug = await createLabel(api, admin.token, web.id, { name: "Bug" });
  const { Done: done } = await stateIds(api, ann.token, web.id);
  if (!done) {
    throw new Error("Web's states hold no Done");
  }

  // ann's Plan the release, assigned to bob and labelled Bug; Write the notes under it, and Check the mirrors, in Done,
  // under that, archived. Plan the release relates to Deploy the site, of Ops, stored from it, and blocks Ship it, of
  // Ops, stored from Ship it as its blocked_by: under Plan the release, a row of each key of the catalog.
  const plan = await createIssue(api, ann.token, web.id, {
    name: "Plan the release",
    assignee_ids: [bob.id],
    label_ids: [bug.id],
  });
  const notes = await createIssue(api, ann.token, web.id, { name: "Write the notes", parent_id: plan.id });
  const mirrors = await createIssue(api, ann.token, web.id, {
    name: "Check the mirrors",
    state_id: done,
    parent_id: notes.id,
  });
  await Promise.all(
    [plan, notes, mirrors].map((item) => createIssueComment(api, bob.token, item.id, "<p>Delete discussion</p>"))
  );
  await Promise.all([plan, notes, mirrors].map((item) => createIssueReaction(api, bob.token, item.id)));
  expect((await issueArchived(api, ann.token, mirrors.id)).status).toBe(200);
  const deploy = await createIssue(api, ann.token, ops.id, { name: "Deploy the site" });
  const ship = await createIssue(api, ann.token, ops.id, { name: "Ship it" });
  await createIssueRelations(api, ann.token, plan.id, { relation_type: "relates_to", issue_ids: [deploy.id] });
  await createIssueRelations(api, ann.token, plan.id, { relation_type: "blocking", issue_ids: [ship.id] });

  // gus' Draft the notice, made while he was Web's member, before the admin made him its guest; lea's Tag the build,
  // made before she left Web.
  const notice = await createIssue(api, gus.token, web.id, { name: "Draft the notice" });
  const gusIn = await projectMembershipOf(api, admin.token, web.id, gus.id);
  expect(await projectMemberWrites(api, web.id).change(admin.token, gusIn, 5)).toEqual({ status: 200 });
  const build = await createIssue(api, lea.token, web.id, { name: "Tag the build" });
  expect(await projectMemberWrites(api, web.id).leave(lea.token)).toEqual({ status: 204 });

  // bob, a member, does not delete ann's work item (403), nor gus, a guest, his own (403); lea, who has left Web, finds
  // hers no more (404). Each is there still.
  expect(await issueDeleted(api, bob.token, plan.id)).toEqual({ status: 403, code: "forbidden" });
  expect(await issueDeleted(api, gus.token, notice.id)).toEqual({ status: 403, code: "forbidden" });
  expect(await issueDeleted(api, lea.token, build.id)).toEqual({ status: 404, code: "issue.not_found" });
  const kept = [plan, notice, build];
  expect(await Promise.all(kept.map(({ id }) => issueRead(api, admin.token, id)))).toMatchObject(
    kept.map(({ id }) => ({ status: 200, issue: { id } }))
  );

  // ann deletes Plan the release, her own: with it, at that moment and by her, the two work items under it and every
  // row under the three, its relations either way among them, the one stored from Ship it, of Ops, too.
  expect(await issueDeleted(api, ann.token, plan.id)).toEqual({ status: 204 });
  await expectIssueDeleted(db, plan.id, ann.email);
  // None of the three is found again, and neither Deploy the site nor Ship it lists a relation.
  const gone = [plan, notes, mirrors];
  expect(await Promise.all(gone.map(({ id }) => issueRead(api, ann.token, id)))).toEqual(
    gone.map(() => ({ status: 404, code: "issue.not_found" }))
  );
  expect(await listIssueRelations(api, ann.token, deploy.id)).toEqual([]);
  expect(await listIssueRelations(api, ann.token, ship.id)).toEqual([]);
});
