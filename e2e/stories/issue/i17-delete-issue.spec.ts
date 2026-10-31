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
import { collaborationRows } from "../../fixtures/assert/issue-collaboration";
import { expectIssueDeleted } from "../../fixtures/assert/issue";
import { createCommentReaction, createIssueComment, createIssueReaction } from "../../fixtures/comments";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  createIssueRelations,
  issueArchived,
  issueDeleted,
  issueRead,
  listIssueRelations,
  updateIssue,
} from "../../fixtures/issues";
import { completedLinkCrawl, createIssueLink } from "../../fixtures/links";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";
import { issueListProject, expectListPageQuiet } from "../../fixtures/issue-list-page";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { sentHeld } from "../../fixtures/settings-pages";

test("I17 (page): the confirmation names the whole subtree including archived descendants; one held deletion stays open until confirmed, deletes every catalog row by the current admin, and preserves the old branch and unrelated item", async ({
  api,
  db,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const writer = await newAccount(api, info, "writer");
  await inviteAndAccept(api, seed.member.token, seed.slug, writer, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: writer.id, role: 15 }]);
  const bug = await createLabel(api, seed.member.token, seed.project.id, { name: "Bug" });
  const description = `<mention-component entity_name="user_mention" entity_identifier="${seed.member.id}"></mention-component>`;
  const root = await createIssue(api, writer.token, seed.project.id, {
    name: "Delete release tree",
    description_html: description,
    assignee_ids: [seed.member.id],
    label_ids: [bug.id],
  });
  const child = await createIssue(api, writer.token, seed.project.id, {
    name: "Release notes",
    parent_id: root.id,
    description_html: description,
  });
  const grandchild = await createIssue(api, writer.token, seed.project.id, {
    name: "Archived review",
    parent_id: child.id,
    description_html: description,
    state_id: seed.states.Done,
  });
  const old = await createIssue(api, writer.token, seed.project.id, {
    name: "Already deleted",
    parent_id: root.id,
    description_html: description,
  });
  const neighbor = await createIssue(api, writer.token, seed.project.id, {
    name: "Keep neighbor",
    description_html: description,
  });
  const live = [root, child, grandchild];
  await Promise.all(
    [...live, old, neighbor].map(async (item) => {
      const link = await createIssueLink(api, writer.token, item.id, {
        url: `http://169.254.169.254/delete/${item.id}`,
      });
      await completedLinkCrawl(db, link.id, link.url);
      const comment = await createIssueComment(api, writer.token, item.id, "<p>Keep author evidence</p>");
      await createCommentReaction(api, writer.token, comment.id);
      await createIssueReaction(api, writer.token, item.id);
    })
  );
  expect((await issueArchived(api, writer.token, grandchild.id)).status).toBe(200);
  expect(await issueDeleted(api, writer.token, old.id)).toEqual({ status: 204 });
  const oldRows = await collaborationRows(db, [old.id]);
  const neighborRows = await collaborationRows(db, [neighbor.id]);
  for (const table of [...oldRows, ...neighborRows]) expect(table.rows.length, table.table).toBeGreaterThan(0);
  const ops = await createProject(api, seed.member.token, seed.slug, { name: "Ops", identifier: "OPS" });
  const deploy = await createIssue(api, seed.member.token, ops.id, { name: "Deploy stays" });
  const ship = await createIssue(api, seed.member.token, ops.id, { name: "Ship stays" });
  await createIssueRelations(api, seed.member.token, root.id, { relation_type: "relates_to", issue_ids: [deploy.id] });
  await createIssueRelations(api, seed.member.token, root.id, { relation_type: "blocking", issue_ids: [ship.id] });
  const watch = await openIssueDetail(page, seed, root);
  await page.getByRole("button", { name: `Options: ${root.name}`, exact: true }).click();
  await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Delete Work item", exact: true });
  await expect(dialog).toContainText(
    `Delete WEB-${root.sequence_id} and all its sub-work items, including archived descendants?`
  );
  const path = `/api/v0/issues/${root.id}`;
  const held = await sentHeld(page, "DELETE", path, () =>
    dialog.getByRole("button", { name: "Delete", exact: true }).dblclick()
  );
  try {
    await expect
      .poll(() => db.query("SELECT deleted_at IS NOT NULL AS deleted FROM issues WHERE id=$1", [root.id]))
      .toEqual([{ deleted: true }]);
    await expectIssueDeleted(db, root.id, seed.member.email);
    await expect(dialog.getByRole("heading", { name: "Delete Work item" })).toBeVisible();
    await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeDisabled();
    await page.keyboard.press("Escape");
    await expect(dialog.getByRole("heading", { name: "Delete Work item" })).toBeVisible();
    await expect(page).toHaveURL(`/${seed.slug}/browse/WEB-${root.sequence_id}`);
    expect(watch.apiRequests.filter((request) => request === `DELETE ${path}`)).toHaveLength(1);
  } finally {
    expect((await held.release()).status()).toBe(204);
  }
  await expect(dialog).toHaveCount(0);
  await expect(page).toHaveURL(seed.path);
  await expect(page.getByText(neighbor.name, { exact: true })).toBeVisible();
  expect(await collaborationRows(db, [old.id])).toEqual(oldRows);
  expect(await collaborationRows(db, [neighbor.id])).toEqual(neighborRows);
  expect(await Promise.all(live.map(({ id }) => issueRead(api, seed.member.token, id)))).toEqual(
    live.map(() => ({ status: 404, code: "issue.not_found" }))
  );
  expect(await listIssueRelations(api, seed.member.token, deploy.id)).toEqual([]);
  expect(await listIssueRelations(api, seed.member.token, ship.id)).toEqual([]);
  const authors = await collaborationRows(
    db,
    live.map(({ id }) => id)
  );
  for (const table of authors) {
    expect(table.rows.length, table.table).toBeGreaterThan(0);
    expect(
      table.rows.every((row) => row.created_by_id === writer.id && row.updated_by_id === seed.member.id),
      table.table
    ).toBe(true);
  }
  await expectListPageQuiet(page, watch);
});

// I17, deleting a work item and the work items under it (M4 design 2, 3.4, 3.6, 3.9, 3.10): the API version, with
// P3's assertions (the work item, two levels of work items under it, an archived one among them, and their assignees,
// labels, subscribers, activity and relations either way deleted at one moment by its deleter; who may delete it).
// Comments, reactions, mentions and versions are furnished through P4's API; links are P7's and the page version P14's.

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
  const description = `<mention-component entity_name="user_mention" entity_identifier="${bob.id}"></mention-component>`;
  const plan = await createIssue(api, ann.token, web.id, {
    name: "Plan the release",
    description_html: description,
    assignee_ids: [bob.id],
    label_ids: [bug.id],
  });
  const notes = await createIssue(api, ann.token, web.id, {
    name: "Write the notes",
    description_html: description,
    parent_id: plan.id,
  });
  const mirrors = await createIssue(api, ann.token, web.id, {
    name: "Check the mirrors",
    description_html: description,
    state_id: done,
    parent_id: notes.id,
  });
  const liveLinks = await Promise.all(
    [plan, notes, mirrors].map((item) =>
      createIssueLink(api, bob.token, item.id, { url: `http://169.254.169.254/furnish/${item.id}` })
    )
  );
  expect(liveLinks.map((link) => link.created_by_id)).toEqual([bob.id, bob.id, bob.id]);
  await Promise.all(
    [plan, notes, mirrors].map((item) =>
      createIssueComment(api, bob.token, item.id, "<p>Delete discussion</p>").then((comment) =>
        Promise.all([ann, bob].map((who) => createCommentReaction(api, who.token, comment.id)))
      )
    )
  );
  await Promise.all(
    [plan, notes, mirrors].flatMap((item) => [ann, bob].map((who) => createIssueReaction(api, who.token, item.id)))
  );
  // An already-deleted branch keeps every collaboration table's earlier deleter and exact timestamp.
  const old = await createIssue(api, bob.token, web.id, {
    name: "Old branch",
    parent_id: notes.id,
    description_html: description,
  });
  const neighbor = await createIssue(api, ann.token, web.id, {
    name: "Untouched neighbor",
    description_html: description,
  });
  await Promise.all(
    [old, neighbor].map(async (item) => {
      const link = await createIssueLink(api, bob.token, item.id, { url: `http://169.254.169.254/furnish/${item.id}` });
      // Freeze the complete row only after this exact seed's metadata job finishes.
      await completedLinkCrawl(db, link.id, link.url);
      const discussion = await createIssueComment(api, bob.token, item.id, "<p>Keep these stamps</p>");
      await createCommentReaction(api, ann.token, discussion.id);
      await createIssueReaction(api, bob.token, item.id);
      await updateIssue(api, bob.token, item.id, { description_html: `<p>Second author</p>${description}` });
    })
  );
  expect(await issueDeleted(api, bob.token, old.id)).toEqual({ status: 204 });
  const oldRows = await collaborationRows(db, [old.id]);
  const neighborRows = await collaborationRows(db, [neighbor.id]);
  for (const table of oldRows) {
    expect(table.rows.length, `${table.table} has an already-deleted row`).toBeGreaterThan(0);
    expect(table.rows.every((row) => row.deleted_at !== null && row.updated_by_id === bob.id)).toBe(true);
  }
  for (const table of neighborRows)
    expect(table.rows.length, `${table.table} has a neighboring row`).toBeGreaterThan(0);
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
  expect(
    await db.query(
      `SELECT l.id, l.created_by_id, l.updated_by_id, l.deleted_at = i.deleted_at AS with_root
         FROM issue_links l JOIN issues i ON i.id = $1 WHERE l.id = ANY($2::uuid[]) ORDER BY l.id`,
      [plan.id, liveLinks.map((link) => link.id)]
    )
  ).toEqual(
    liveLinks
      .toSorted((a, b) => a.id.localeCompare(b.id))
      .map((link) => ({
        id: link.id,
        created_by_id: bob.id,
        updated_by_id: ann.id,
        with_root: true,
      }))
  );
  expect(await collaborationRows(db, [old.id])).toEqual(oldRows);
  expect(await collaborationRows(db, [neighbor.id])).toEqual(neighborRows);
  // None of the three is found again, and neither Deploy the site nor Ship it lists a relation.
  const gone = [plan, notes, mirrors];
  expect(await Promise.all(gone.map(({ id }) => issueRead(api, ann.token, id)))).toEqual(
    gone.map(() => ({ status: 404, code: "issue.not_found" }))
  );
  expect(await listIssueRelations(api, ann.token, deploy.id)).toEqual([]);
  expect(await listIssueRelations(api, ann.token, ship.id)).toEqual([]);
});
