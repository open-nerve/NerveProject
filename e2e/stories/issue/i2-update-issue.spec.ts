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

// I2, changing a work item (M4 design 2, 3.7, 3.13): the API version, with P1b's assertions (its name, state,
// priority, assignees and labels, each change's rows of its activity, completed_at, who changed it last and when) and
// P2's (its dates and their rows, the dates held in order once merged, an archived work item not changed). The
// description is P4's, the page version P14's.

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
