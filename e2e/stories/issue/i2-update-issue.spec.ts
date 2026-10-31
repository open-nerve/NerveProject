import {
  addProjectMembers,
  amidAnotherWorkspace,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import {
  archivedRow,
  assigned,
  byId,
  changed,
  createdFor,
  expectIssueArchival,
  expectIssueCreated,
  expectIssueUpdated,
} from "../../fixtures/assert/issue";
import { newAccount, shownNameOf } from "../../fixtures/auth";
import { createIssue, issueArchived, issueUpdated, updateIssue, type IssueUpdate } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";
import { recordFocuses } from "../../fixtures/browser";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { expectListPageQuiet, issueListProject } from "../../fixtures/issue-list-page";
import { answerTo, bodiesSentTo } from "../../fixtures/settings-pages";

// I2, changing a work item (M4 design 2, 3.7, 3.13): the API version, with P1b's assertions (its name, state,
// priority, assignees and labels, each change's rows of its activity, completed_at, who changed it last and when) and
// P2's (its dates and their rows, the dates held in order once merged, an archived work item not changed). The
// description cases are in i2-description.spec.ts; P14's real detail page version follows.

test("I2 (page): the detail announces title refusals before focus, saves each property through its real control and clears selected collections and dates", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  test.setTimeout(60_000);
  const seed = await issueListProject(api, info);
  const colleague = await newAccount(api, info, "colleague");
  await inviteAndAccept(api, seed.member.token, seed.slug, colleague, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: colleague.id, role: 15 }]);
  const bug = await createLabel(api, seed.member.token, seed.project.id, { name: "Bug" });
  const ui = await createLabel(api, seed.member.token, seed.project.id, { name: "UI" });
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Original detail" });
  const path = `/api/v0/issues/${issue.id}`;
  const watch = await openIssueDetail(page, seed, issue);
  const bodies = await bodiesSentTo(page, "PATCH", path);
  const title = page.getByRole("textbox", { name: "Work item title", exact: true });
  const titleId = await title.getAttribute("id");
  const state = page.getByRole("button", { name: "State: Backlog", exact: true });
  // Ordinary revalidation keeps focus where the person put it. Only a refused save refocuses its field.
  await (
    [
      ["   ", "Title is required"],
      ["x".repeat(256), "Too long"],
    ] as const
  ).reduce(async (previous, [value, description]) => {
    await previous;
    await title.fill(value);
    await state.focus();
    const focuses = await recordFocuses(page);
    await expect(title).toBeFocused();
    expect((await focuses()).at(-1)).toEqual({ id: titleId, invalid: "true", description });
    expect(bodies).toEqual([]);
  }, Promise.resolve());
  const focuses = await recordFocuses(page);
  const refusal = await answerTo(page, "PATCH", path, async () => {
    await title.fill("Invalid\u0000title");
    await state.focus();
  });
  expect(refusal.status()).toBe(422);
  await expect(title).toBeFocused();
  expect((await focuses()).at(-1)).toEqual({
    id: titleId,
    invalid: "true",
    description: "The format is not valid",
  });
  expect(await db.query("SELECT name FROM issues WHERE id=$1", [issue.id])).toEqual([{ name: "Original detail" }]);

  const patches: IssueUpdate[] = [];
  const change = async (body: IssueUpdate, act: () => Promise<void>) => {
    const previous = bodies.length;
    expect((await answerTo(page, "PATCH", path, act)).status()).toBe(200);
    expect(bodies.slice(previous)).toEqual([body]);
    patches.push(body);
  };
  await change({ name: "Edited detail" }, () => title.fill("Edited detail"));
  await expect(title).not.toHaveAttribute("aria-invalid", "true");
  await state.click();
  await change({ state_id: seed.states.Done }, () => page.getByRole("option", { name: "Done", exact: true }).click());
  await expect(page.getByRole("button", { name: "State: Done", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Priority: None", exact: true }).click();
  await change({ priority: "high" }, () => page.getByRole("option", { name: "High", exact: true }).click());
  const members = page.getByText("Assignees", { exact: true }).locator("../..").getByRole("button");
  await members.click();
  await change({ assignee_ids: [seed.member.id] }, () => page.getByRole("option", { name: /You$/ }).click());
  await change({ assignee_ids: [seed.member.id, colleague.id] }, () =>
    page
      .getByRole("option")
      .filter({ hasText: shownNameOf(colleague.email) })
      .click()
  );
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Labels: Edited detail", exact: true }).click();
  await change({ label_ids: [bug.id] }, () => page.getByRole("option", { name: "Bug", exact: true }).click());
  await change({ label_ids: [bug.id, ui.id] }, () => page.getByRole("option", { name: "UI", exact: true }).click());
  await page.keyboard.press("Escape");
  await (
    [
      ["Add start date", 10, { start_date: "2026-11-10" }],
      ["Add due date", 20, { target_date: "2026-11-20" }],
    ] as const
  ).reduce(async (previous, [label, day, body]) => {
    await previous;
    await page.getByRole("button", { name: label, exact: true }).click();
    const calendar = page.getByRole("dialog", { name: label, exact: true });
    await calendar.getByRole("combobox", { name: "Choose the Year" }).selectOption("2026");
    await calendar.getByRole("combobox", { name: "Choose the Month" }).selectOption("10");
    await change(body, () =>
      calendar.getByRole("button", { name: new RegExp(`November ${day}(?:th|st|nd|rd), 2026`) }).click()
    );
  }, Promise.resolve());
  expect(
    await db.query(
      `SELECT i.name, s.name AS state, i.priority, to_char(i.start_date, 'YYYY-MM-DD') AS start_date,
              to_char(i.target_date, 'YYYY-MM-DD') AS target_date, i.completed_at IS NOT NULL AS completed
         FROM issues i JOIN states s ON s.id=i.state_id WHERE i.id=$1`,
      [issue.id]
    )
  ).toEqual([
    {
      name: "Edited detail",
      state: "Done",
      priority: "high",
      start_date: "2026-11-10",
      target_date: "2026-11-20",
      completed: true,
    },
  ]);
  expect(
    await db.query(
      "SELECT assignee_id::text AS id FROM issue_assignees WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY assignee_id",
      [issue.id]
    )
  ).toEqual([seed.member.id, colleague.id].toSorted().map((id) => ({ id })));
  expect(
    await db.query(
      "SELECT label_id::text AS id FROM issue_labels WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY label_id",
      [issue.id]
    )
  ).toEqual([bug.id, ui.id].toSorted().map((id) => ({ id })));

  await members.click();
  await change({ assignee_ids: [colleague.id] }, () => page.getByRole("option", { name: /You$/ }).click());
  await change({ assignee_ids: [] }, () =>
    page
      .getByRole("option")
      .filter({ hasText: shownNameOf(colleague.email) })
      .click()
  );
  await page.keyboard.press("Escape");
  const labels = page.getByText("Labels", { exact: true }).locator("../..");
  await change({ label_ids: [ui.id] }, () => labels.getByRole("button", { name: "Bug", exact: true }).click());
  await change({ label_ids: [] }, () => labels.getByRole("button", { name: "UI", exact: true }).click());
  await page.getByRole("button", { name: "Add start date", exact: true }).hover();
  await change({ start_date: null }, () =>
    page.getByRole("button", { name: "Clear date: Add start date", exact: true }).click()
  );
  await page.getByRole("button", { name: "Add due date", exact: true }).hover();
  await change({ target_date: null }, () =>
    page.getByRole("button", { name: "Clear date: Add due date", exact: true }).click()
  );
  expect(
    await db.query(
      `SELECT start_date, target_date,
       (SELECT count(*)::int FROM issue_assignees WHERE issue_id=i.id AND deleted_at IS NULL) AS assignees,
       (SELECT count(*)::int FROM issue_labels WHERE issue_id=i.id AND deleted_at IS NULL) AS labels
     FROM issues i WHERE id=$1`,
      [issue.id]
    )
  ).toEqual([{ start_date: null, target_date: null, assignees: 0, labels: 0 }]);
  expect(bodies).toEqual([{ name: "Invalid\u0000title" }, ...patches]);
  expect(
    await page
      .locator("[tabindex]")
      .evaluateAll((nodes) => nodes.filter((node) => Number(node.getAttribute("tabindex")) > 0).length)
  ).toBe(0);
  await expectListPageQuiet(page, watch, {
    failures: [`422 PATCH ${path}`],
    errors:
      browserName === "chromium"
        ? ["Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)"]
        : [],
  });
});

test("I2 (API): a member renames a work item; another makes it high and moves it to a completed state, which completes it; the first assigns it to another account instead of himself and labels it otherwise, the others kept, taking off rows the second wrote; its values sent as they are write nothing; an account and a label added again and a new label, the two labels out of their ids' order, are added in id order; a label, then an account, taken off alone each change it, the rows taken off before keeping their stamps; moving it back completes it no more; its dates are set, then a start date alone after the target date kept and a target date alone before the start date kept are refused (422), both moved past the target date it had are taken, and the target date is cleared; problems of the values are refused before any read, alike for a work item that is not there, and a state and a label the project lacks under the lock, each set at once (422), writing nothing; a work item sharing an account and a label with it is left as created; an archived work item is not changed (409): each change one row of its activity a field, a label or an account added or taken off, all its rows the changer's at its moment", async ({
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
  const [bug, ui, docs, login] = await Promise.all(
    ["Bug", "UI", "Docs", "Login"].map((name) => createLabel(api, admin.token, web.id, { name }))
  );
  if (!bug || !ui || !docs || !login) {
    throw new Error("the labels were not created");
  }
  const { Done: done, "In Progress": inProgress } = await stateIds(api, ann.token, web.id);
  if (!done || !inProgress) {
    throw new Error("Web's states hold no Done or no In Progress");
  }
  // ann's work item, in Backlog, of no priority, assigned to bob and to her and labelled Bug and Docs; and her Fix the
  // logout, assigned to bob and labelled Bug, which none of the changes below names.
  const { id } = await createIssue(api, ann.token, web.id, {
    name: "Fix the login",
    assignee_ids: [bob.id, ann.id],
    label_ids: [bug.id, docs.id],
  });
  const logout = await createIssue(api, ann.token, web.id, {
    name: "Fix the logout",
    assignee_ids: [bob.id],
    label_ids: [bug.id],
  });
  const created = createdFor(bob, ann);

  // bob renames it.
  expect(await updateIssue(api, bob.token, id, { name: "  Fix the sign-in " })).toMatchObject({
    name: "Fix the sign-in",
  });
  const renamed = changed("name", "Fix the login", "Fix the sign-in", "updated the name to");
  await expectIssueUpdated(db, id, {
    name: "Fix the sign-in",
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    parent: null,
    completed: false,
    by: bob.email,
    assignees: [ann.email, bob.email].toSorted(),
    labels: ["Bug", "Docs"],
    subscribers: [ann.email, bob.email].toSorted(),
    activity: [...created, renamed],
    wrote: 1,
  });

  // ann makes it high and moves it to Done, of the completed group: completed, the priority's row before the state's.
  expect(await updateIssue(api, ann.token, id, { state_id: done, priority: "high" })).toMatchObject({
    state_id: done,
    priority: "high",
  });
  const toDone = [
    changed("priority", "none", "high", "updated the priority to"),
    changed("state", "Backlog", "Done", "updated the state to", "Done", "Backlog"),
  ];
  await expectIssueUpdated(db, id, {
    name: "Fix the sign-in",
    state: "Done",
    priority: "high",
    start_date: null,
    target_date: null,
    parent: null,
    completed: true,
    by: ann.email,
    assignees: [ann.email, bob.email].toSorted(),
    labels: ["Bug", "Docs"],
    subscribers: [ann.email, bob.email].toSorted(),
    activity: [...created, renamed, ...toDone],
    wrote: 2,
  });

  // bob assigns it to the admin instead of him and labels it UI instead of Bug, ann and Docs kept: the admin
  // subscribed, bob kept subscribed; four rows of activity, the label added, the label taken off, the account added,
  // the account taken off; the two rows taken off, ann's, soft-deleted and bob's now; nine rows in all. The work item
  // answers its assignees and labels in id order.
  const assignedIds = [ann.id, admin.id].toSorted();
  const labelledIds = [docs.id, ui.id].toSorted();
  expect(
    await updateIssue(api, bob.token, id, { assignee_ids: [ann.id, admin.id], label_ids: [docs.id, ui.id] })
  ).toMatchObject({ assignee_ids: assignedIds, label_ids: labelledIds });
  // The rows that take Bug and bob off it, written by this change and each again by a later one.
  const bugRemoved = changed("labels", "Bug", "", "removed label ", null, "Bug");
  const bobUnassigned = changed("assignees", shownNameOf(bob.email), "", "removed assignee ", null, bob.email);
  const afterSwap = {
    name: "Fix the sign-in",
    state: "Done",
    priority: "high",
    start_date: null,
    target_date: null,
    parent: null,
    completed: true,
    by: bob.email,
    assignees: [admin.email, ann.email].toSorted(),
    labels: ["Docs", "UI"],
    subscribers: [admin.email, ann.email, bob.email].toSorted(),
    activity: [
      ...created,
      renamed,
      ...toDone,
      changed("labels", "", "UI", "added label ", "UI"),
      bugRemoved,
      assigned(admin.email),
      bobUnassigned,
    ],
    wrote: 9,
  };
  await expectIssueUpdated(db, id, afterSwap);

  // ann sends its name, state, priority, assignees and labels as they are, the sets in the reverse of their order:
  // nothing changes, nothing is written, and it stays bob's last change.
  expect(
    await updateIssue(api, ann.token, id, {
      name: "Fix the sign-in",
      state_id: done,
      priority: "high",
      assignee_ids: assignedIds.toReversed(),
      label_ids: labelledIds.toReversed(),
    })
  ).toMatchObject({ name: "Fix the sign-in", state_id: done, assignee_ids: assignedIds, label_ids: labelledIds });
  await expectIssueUpdated(db, id, afterSwap);

  // ann assigns bob again and labels it Bug again and Login, the two in the reverse of their ids' order: their rows of
  // activity in id order, then bob's; bob, subscribed still, is not subscribed again; six rows.
  const again = [bug, login].toSorted(byId);
  expect(
    await updateIssue(api, ann.token, id, {
      assignee_ids: [ann.id, admin.id, bob.id],
      label_ids: [docs.id, ui.id, ...again.map((label) => label.id).toReversed()],
    })
  ).toMatchObject({
    assignee_ids: [ann.id, admin.id, bob.id].toSorted(),
    label_ids: [docs.id, ui.id, bug.id, login.id].toSorted(),
  });
  const addedAgain = {
    ...afterSwap,
    by: ann.email,
    assignees: [admin.email, ann.email, bob.email].toSorted(),
    labels: ["Bug", "Docs", "Login", "UI"],
    activity: [
      ...afterSwap.activity,
      ...again.map((label) => changed("labels", "", label.name, "added label ", label.name)),
      assigned(bob.email),
    ],
    wrote: 6,
  };
  await expectIssueUpdated(db, id, addedAgain);

  // bob takes Bug off it, nothing else: a change, its two rows; the Bug he took off before keeps that change's stamps.
  await updateIssue(api, bob.token, id, { label_ids: [docs.id, login.id, ui.id] });
  const bugOff = {
    ...addedAgain,
    by: bob.email,
    labels: ["Docs", "Login", "UI"],
    activity: [...addedAgain.activity, bugRemoved],
    wrote: 2,
  };
  await expectIssueUpdated(db, id, bugOff);

  // ann takes bob off it, nothing else: a change too; the bob taken off before keeps that change's stamps.
  await updateIssue(api, ann.token, id, { assignee_ids: [ann.id, admin.id] });
  const bobOff = {
    ...bugOff,
    by: ann.email,
    assignees: [admin.email, ann.email].toSorted(),
    activity: [...bugOff.activity, bobUnassigned],
    wrote: 2,
  };
  await expectIssueUpdated(db, id, bobOff);

  // bob moves it back to In Progress: completed no more.
  await updateIssue(api, bob.token, id, { state_id: inProgress });
  const movedBack = {
    ...bobOff,
    state: "In Progress",
    completed: false,
    by: bob.email,
    activity: [
      ...bobOff.activity,
      changed("state", "Done", "In Progress", "updated the state to", "In Progress", "Done"),
    ],
    wrote: 1,
  };
  await expectIssueUpdated(db, id, movedBack);

  // ann gives it a start date and a target date, the 12th and the 19th of October: a row each, the start date's first.
  await updateIssue(api, ann.token, id, { start_date: "2026-10-12", target_date: "2026-10-19" });
  const dated = {
    ...movedBack,
    start_date: "2026-10-12",
    target_date: "2026-10-19",
    by: ann.email,
    activity: [
      ...movedBack.activity,
      changed("start_date", "", "2026-10-12", "updated the start date to "),
      changed("target_date", "", "2026-10-19", "updated the target date to"),
    ],
    wrote: 2,
  };
  await expectIssueUpdated(db, id, dated);
  // The dates it then has are held in order, each given one merged with the one it keeps: a start date alone after the
  // target date is refused at start_date, a target date alone before the start date at target_date, writing nothing;
  // both moved past the target date it had, in order, are taken.
  expect(await issueUpdated(api, bob.token, id, { start_date: "2026-10-20" })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "start_date", code: "not_allowed" }],
  });
  expect(await issueUpdated(api, bob.token, id, { target_date: "2026-10-11" })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "target_date", code: "not_allowed" }],
  });
  await expectIssueUpdated(db, id, dated);
  await updateIssue(api, bob.token, id, { start_date: "2026-10-20", target_date: "2026-10-26" });
  const redated = {
    ...dated,
    start_date: "2026-10-20",
    target_date: "2026-10-26",
    by: bob.email,
    activity: [
      ...dated.activity,
      changed("start_date", "2026-10-12", "2026-10-20", "updated the start date to "),
      changed("target_date", "2026-10-19", "2026-10-26", "updated the target date to"),
    ],
    wrote: 2,
  };
  await expectIssueUpdated(db, id, redated);
  // bob clears its target date: its row's new value is "".
  await updateIssue(api, bob.token, id, { target_date: null });
  const cleared = {
    ...redated,
    target_date: null,
    activity: [...redated.activity, changed("target_date", "2026-10-26", "", "updated the target date to")],
    wrote: 1,
  };
  await expectIssueUpdated(db, id, cleared);

  // A priority in another case (the contract's values are lower case), an account and a label named twice are refused
  // before the work item is read, the three at once, as they are for a work item that is not there; a state and a
  // label Web lacks under the lock, the two at once. Neither refused change writes anything: it stays bob's last change.
  const malformed: IssueUpdate = {
    name: "Fix it",
    // @ts-expect-error -- a priority in another case breaks the contract on purpose
    priority: "High",
    assignee_ids: [ann.id, ann.id],
    label_ids: [ui.id, ui.id],
  };
  const refusedValues = {
    status: 422,
    code: "validation_failed",
    errors: [
      { field: "priority", code: "invalid_format" },
      { field: "assignee_ids[1]", code: "duplicate" },
      { field: "label_ids[1]", code: "duplicate" },
    ],
  };
  expect(await issueUpdated(api, ann.token, id, malformed)).toEqual(refusedValues);
  expect(await issueUpdated(api, ann.token, crypto.randomUUID(), malformed)).toEqual(refusedValues);
  expect(
    await issueUpdated(api, ann.token, id, {
      name: "Fix it",
      state_id: crypto.randomUUID(),
      label_ids: [crypto.randomUUID()],
    })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [
      { field: "state_id", code: "not_allowed" },
      { field: "label_ids[0]", code: "not_allowed" },
    ],
  });
  await expectIssueUpdated(db, id, cleared);

  // Fix the logout, which shares bob and Bug with it, is as ann created it: no change above wrote a row of it.
  await expectIssueCreated(db, logout.id, {
    sequence_id: 2,
    name: "Fix the logout",
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: logout.sort_order,
    parent: null,
    completed: false,
    by: ann.email,
    assignees: [bob.email],
    labels: ["Bug"],
    subscribers: [ann.email, bob.email].toSorted(),
    activity: createdFor(bob),
  });

  // An archived work item is not changed (409 issue.archived): ann's Ship it, in Done, which she archives; bob's
  // change writes nothing, so it stays ann's archiving.
  const shipped = await createIssue(api, ann.token, web.id, { name: "Ship it", state_id: done });
  expect((await issueArchived(api, ann.token, shipped.id)).status).toBe(200);
  const shippedArchived = { archived: true, by: ann.email, activity: [...createdFor(), archivedRow] };
  await expectIssueArchival(db, shipped.id, shippedArchived);
  expect(await issueUpdated(api, bob.token, shipped.id, { target_date: "2026-10-19" })).toEqual({
    status: 409,
    code: "issue.archived",
  });
  await expectIssueArchival(db, shipped.id, shippedArchived);
});
