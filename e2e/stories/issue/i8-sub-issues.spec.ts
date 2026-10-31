import {
  addProjectMembers,
  amidAnotherWorkspace,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import {
  archivedRow,
  byId,
  createdFor,
  expectIssueArchival,
  expectIssueCreated,
  expectIssueUpdated,
  itemRef,
  madeBy,
  parentRow,
} from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  issueArchived,
  issueRead,
  issueUpdated,
  subIssuesAdded,
  updateIssue,
} from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";
import { issueListProject, expectListPageQuiet } from "../../fixtures/issue-list-page";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { answerTo, sentHeld } from "../../fixtures/settings-pages";

test("I8 (page): native searches add two existing children and create one; progress is confirmed, descendants cannot be parents, and moving and removing a parent update counts and activity", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const top = await createIssue(api, seed.member.token, seed.project.id, { name: "Release plan" });
  const first = await createIssue(api, seed.member.token, seed.project.id, { name: "Write notes" });
  const done = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Done build",
    state_id: seed.states.Done,
  });
  const alternate = await createIssue(api, seed.member.token, seed.project.id, { name: "Next release" });
  const child = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Nested child",
    parent_id: first.id,
  });
  await createIssue(api, seed.member.token, seed.project.id, { name: "Nested grandchild", parent_id: child.id });
  const watch = await openIssueDetail(page, seed, top);
  const add = page.getByRole("button", { name: "Add sub-work item", exact: true });
  // Opening the actual widget's menu is also a keyboard-accessible, named entry.
  await add.click();
  const searchPath = `/api/v0/workspaces/${seed.slug}/issue-search`;
  const search = await answerTo(page, "GET", searchPath, () =>
    page.getByRole("menuitem", { name: "Add existing", exact: true }).click()
  );
  expect(Object.fromEntries(new URL(search.url()).searchParams)).toEqual({
    project_id: seed.project.id,
    for: "sub_issue",
    issue_id: top.id,
  });
  const existing = page.getByRole("dialog", { name: "Add existing work item", exact: true });
  await existing.getByRole("option").filter({ hasText: "Write notes" }).click();
  await existing.getByRole("option").filter({ hasText: "Done build" }).click();
  const held = await sentHeld(page, "POST", `/api/v0/issues/${top.id}/sub-issues`, () =>
    existing.getByRole("button", { name: "Add selected work items", exact: true }).click()
  );
  try {
    expect(held.body).toEqual({ sub_issue_ids: [first.id, done.id] });
    await expect
      .poll(() => db.query("SELECT id FROM issues WHERE parent_id=$1 ORDER BY id", [top.id]))
      .toEqual([first, done].toSorted(byId).map(({ id }) => ({ id })));
    await expect(existing).toHaveCount(1);
    await expect(page.locator(`#issue-${first.id}`)).toHaveCount(0);
  } finally {
    expect((await held.release()).status()).toBe(200);
  }
  await expect(existing).toHaveCount(0);
  await expect(page.locator(`#issue-${first.id}`)).toBeVisible();
  await expect(page.locator(`#issue-${done.id}`)).toBeVisible();
  await expect(page.getByText("1/2 Done", { exact: true })).toBeVisible();
  const topRef = itemRef("WEB", top);
  await expectIssueUpdated(db, first.id, {
    ...madeBy(seed.member.email, first.name),
    parent: top.name,
    by: seed.member.email,
    activity: [...createdFor(), parentRow(null, topRef)],
    wrote: 1,
  });

  await add.click();
  await page.getByRole("menuitem", { name: "Create new", exact: true }).click();
  const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
  await form.getByRole("textbox", { name: "Title", exact: true }).fill("Draft notice");
  const created = await answerTo(page, "POST", `/api/v0/projects/${seed.project.id}/issues`, () =>
    form.getByRole("button", { name: "Save", exact: true }).click()
  );
  expect(created.status()).toBe(201);
  const createdRow = (
    await db.query<{ id: string; sequence_id: string; sort_order: number }>(
      "SELECT id, sequence_id, sort_order FROM issues WHERE project_id=$1 AND name=$2",
      [seed.project.id, "Draft notice"]
    )
  )[0]!;
  await expect(form).toHaveCount(0);
  await expect(page.locator(`#issue-${createdRow.id}`)).toBeVisible();
  await expect(page.getByText("1/3 Done", { exact: true })).toBeVisible();
  await expectIssueCreated(db, createdRow.id, {
    ...madeBy(seed.member.email, "Draft notice"),
    sequence_id: Number(createdRow.sequence_id),
    sort_order: createdRow.sort_order,
    parent: top.name,
    by: seed.member.email,
    activity: createdFor(),
  });

  // Follow the child's own native address; the picker must exclude it and every level below it.
  const childHref = await page.locator(`#issue-${first.id}`).getAttribute("href");
  expect(childHref).toBe(`/${seed.slug}/browse/WEB-${first.sequence_id}`);
  await page.goto(childHref!);
  const changeParent = page.getByRole("button", { name: "Add parent work item", exact: true });
  const parentSearch = await answerTo(page, "GET", searchPath, () => changeParent.click());
  expect(Object.fromEntries(new URL(parentSearch.url()).searchParams)).toEqual({
    project_id: seed.project.id,
    for: "parent",
    issue_id: first.id,
  });
  const picker = page.getByRole("dialog", { name: "Add parent work item", exact: true });
  await expect(picker.getByRole("option").filter({ hasText: alternate.name })).toBeVisible();
  await Promise.all(
    [first.name, top.name, "Nested child", "Nested grandchild"].map((excluded) =>
      expect(picker.getByRole("option").filter({ hasText: excluded })).toHaveCount(0)
    )
  );
  const moved = await sentHeld(page, "PATCH", `/api/v0/issues/${first.id}`, () =>
    picker.getByRole("option").filter({ hasText: alternate.name }).click()
  );
  try {
    expect(moved.body).toEqual({ parent_id: alternate.id });
    await expect
      .poll(() => db.query("SELECT parent_id FROM issues WHERE id=$1", [first.id]))
      .toEqual([{ parent_id: alternate.id }]);
    await expect(picker).toHaveCount(1);
    await page.keyboard.press("Escape");
    await expect(picker).toHaveCount(1);
  } finally {
    expect((await moved.release()).status()).toBe(200);
  }
  await expect(picker).toHaveCount(0);
  const parentControl = changeParent.locator("..");
  await expect(parentControl.getByRole("link")).toHaveAttribute(
    "href",
    `/${seed.slug}/browse/WEB-${alternate.sequence_id}`
  );
  const movedActivity = [...createdFor(), parentRow(null, topRef), parentRow(topRef, itemRef("WEB", alternate))];
  await expectIssueUpdated(db, first.id, {
    ...madeBy(seed.member.email, first.name),
    parent: alternate.name,
    by: seed.member.email,
    activity: movedActivity,
    wrote: 1,
  });
  expect(await issueRead(api, seed.member.token, top.id)).toMatchObject({ issue: { sub_issues_count: 2 } });
  expect(await issueRead(api, seed.member.token, alternate.id)).toMatchObject({ issue: { sub_issues_count: 1 } });
  expect(
    (
      await answerTo(page, "PATCH", `/api/v0/issues/${first.id}`, () =>
        parentControl.getByRole("button", { name: "Remove", exact: true }).click()
      )
    ).status()
  ).toBe(200);
  await expect(parentControl.getByRole("link")).toHaveCount(0);
  await expectIssueUpdated(db, first.id, {
    ...madeBy(seed.member.email, first.name),
    parent: null,
    by: seed.member.email,
    activity: [...movedActivity, parentRow(itemRef("WEB", alternate), null)],
    wrote: 1,
  });
  expect(await issueRead(api, seed.member.token, alternate.id)).toMatchObject({ issue: { sub_issues_count: 0 } });
  await expectListPageQuiet(page, watch, browserName, 2);
});

// I8, the work items under a work item (M4 design 2, 3.9, 3.13): the API version, with P3's assertions (work items
// put under one, one created under it, moved under another and from under any, each move a row of the moved work
// item's activity, the count of the work items directly under one, a work item above refused under one below it, an
// archived work item taking none). The page version is P14's.

test("I8 (API): a member puts two work items under one, named out of their ids' order: they answer in id order, each moved with a row of its activity, hers; put under it again, nothing is written; one she creates under it has no row of its parent; another member moves one under another work item by changing it, and puts one under that other, each row's old value the work item it was under; she takes one from under any; each work item counts those directly under it; a work item is refused under one below it, either way (422), and an archived one takes none (409) nor becomes a parent (422), writing nothing", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann and bob, Web's members, each a member of Acme.
  const [ann, bob] = await Promise.all(["ann", "bob"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !bob) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, bob].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: bob.id, role: 15 },
  ]);
  const { Done: done } = await stateIds(api, ann.token, web.id);
  if (!done) {
    throw new Error("Web's states hold no Done");
  }
  // ann's Plan the release, Write the notes and Tag the build, in Backlog, and Ship it, in Done.
  const plan = await createIssue(api, ann.token, web.id, { name: "Plan the release" });
  const notes = await createIssue(api, ann.token, web.id, { name: "Write the notes" });
  const tag = await createIssue(api, ann.token, web.id, { name: "Tag the build" });
  const ship = await createIssue(api, ann.token, web.id, { name: "Ship it", state_id: done });
  const [planRef, shipRef] = [itemRef("WEB", plan), itemRef("WEB", ship)];
  const counted = async (issueId: string, count: number) =>
    expect(await issueRead(api, ann.token, issueId)).toMatchObject({ status: 200, issue: { sub_issues_count: count } });

  // ann puts Tag the build and Write the notes under Plan the release, named out of their ids' order: they answer in
  // id order, under it; each has one row more, hers, from under none to under Plan the release.
  expect(await subIssuesAdded(api, ann.token, plan.id, [tag.id, notes.id].toSorted().toReversed())).toMatchObject({
    status: 200,
    issues: [notes, tag].toSorted(byId).map(({ id }) => ({ id, parent_id: plan.id })),
  });
  const notesUnder = {
    ...madeBy(ann.email, "Write the notes"),
    parent: "Plan the release",
    by: ann.email,
    activity: [...createdFor(), parentRow(null, planRef)],
    wrote: 1,
  };
  await expectIssueUpdated(db, notes.id, notesUnder);
  await expectIssueUpdated(db, tag.id, { ...notesUnder, name: "Tag the build" });
  await counted(plan.id, 2);
  // Put under it again, Write the notes does not move: none answered, nothing written.
  expect(await subIssuesAdded(api, ann.token, plan.id, [notes.id])).toEqual({ status: 200, issues: [] });
  await expectIssueUpdated(db, notes.id, notesUnder);

  // ann creates Draft the notice under Plan the release: under it, its activity only its creation's row.
  const draft = await createIssue(api, ann.token, web.id, { name: "Draft the notice", parent_id: plan.id });
  await expectIssueCreated(db, draft.id, {
    ...madeBy(ann.email, "Draft the notice"),
    sequence_id: 5,
    sort_order: draft.sort_order,
    parent: "Plan the release",
    by: ann.email,
    activity: createdFor(),
  });
  await counted(plan.id, 3);

  // bob moves Write the notes under Ship it by changing it, and puts Tag the build under Ship it: each a row of his,
  // from under Plan the release to under Ship it. Tag the build's is the row of a work item put under another, which
  // Plane wrote with the work item moved as the one it was under.
  expect(await updateIssue(api, bob.token, notes.id, { parent_id: ship.id })).toMatchObject({ parent_id: ship.id });
  const notesMoved = {
    ...notesUnder,
    parent: "Ship it",
    by: bob.email,
    activity: [...notesUnder.activity, parentRow(planRef, shipRef)],
  };
  await expectIssueUpdated(db, notes.id, notesMoved);
  expect(await subIssuesAdded(api, bob.token, ship.id, [tag.id])).toMatchObject({
    status: 200,
    issues: [{ id: tag.id, parent_id: ship.id }],
  });
  await expectIssueUpdated(db, tag.id, { ...notesMoved, name: "Tag the build" });
  // ann takes Draft the notice from under any: its row's new value is "".
  await updateIssue(api, ann.token, draft.id, { parent_id: null });
  const draftAlone = {
    ...madeBy(ann.email, "Draft the notice"),
    parent: null,
    by: ann.email,
    activity: [...createdFor(), parentRow(planRef, null)],
    wrote: 1,
  };
  await expectIssueUpdated(db, draft.id, draftAlone);
  await counted(plan.id, 0);

  // Check the mirrors, ann's, under Write the notes: Ship it counts the two directly under it, not the one under one of
  // them. It is above Check the mirrors: neither put under it nor made its parent by changing it (422), unchanged.
  const mirrors = await createIssue(api, ann.token, web.id, { name: "Check the mirrors", parent_id: notes.id });
  await counted(ship.id, 2);
  expect(await subIssuesAdded(api, ann.token, mirrors.id, [ship.id])).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "sub_issue_ids[0]", code: "not_allowed" }],
  });
  expect(await issueUpdated(api, ann.token, ship.id, { parent_id: mirrors.id })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "parent_id", code: "not_allowed" }],
  });
  await expectIssueArchival(db, ship.id, { archived: false, by: ann.email, activity: createdFor() });

  // ann archives Ship it: it takes Draft the notice under it no more (409 issue.archived), nor does changing Draft the
  // notice make it its parent (422); neither writes anything.
  expect((await issueArchived(api, ann.token, ship.id)).status).toBe(200);
  expect(await subIssuesAdded(api, ann.token, ship.id, [draft.id])).toEqual({ status: 409, code: "issue.archived" });
  expect(await issueUpdated(api, ann.token, draft.id, { parent_id: ship.id })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "parent_id", code: "not_allowed" }],
  });
  await expectIssueArchival(db, ship.id, { archived: true, by: ann.email, activity: [...createdFor(), archivedRow] });
  await expectIssueUpdated(db, draft.id, draftAlone);
});
