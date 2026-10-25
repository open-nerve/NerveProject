import {
  addProjectMembers,
  amidAnotherWorkspace,
  archiveProject,
  changeProject,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { byId, createdFor, expectIssueCreated } from "../../fixtures/assert/issue";
import { expectInitialDescription } from "../../fixtures/assert/issue-collaboration";
import { newAccount } from "../../fixtures/auth";
import { createIssue, issueCreated } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// I1, creating a work item (M4 design 2, 3.7, 3.13): the API version, with P1a's assertions (the work item's rows,
// its number, the default state and assignee, its assignees, labels and subscribers, the activity of its creation and
// assignees, the refusals) and P2's (its dates). The page version is P14's.

test("I1 (API): a member creates a work item in a state, of a priority, assigned to two members, labelled twice, from a start date to a target date: numbered 1, first in its state, with its assignees, labels and dates, the creator and the assignees its subscribers, its creation and each assignee in its activity, all at one moment; without a state or assignees, the next is in the default state and assigned to the project's default assignee while he is its member, the one after to no one once he has left; one in a cancelled state is not completed; an account or a label named twice, a priority in another case and a start date after the target date are refused at once (422), and so are a state, a label the project lacks and a guest assigned (422); a guest creates none (403), nor does anyone in an archived project (409)", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann and bob, Web's members; gus, its guest; each a member of Acme.
  const [ann, bob, gus] = await Promise.all(["ann", "bob", "gus"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !bob || !gus) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, bob, gus].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: bob.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  const bug = await createLabel(api, admin.token, web.id, { name: "Bug" });
  const ui = await createLabel(api, admin.token, web.id, { name: "UI" });
  const { "In Progress": inProgress, Cancelled: cancelled } = await stateIds(api, ann.token, web.id);
  if (!inProgress || !cancelled) {
    throw new Error("Web's states hold no In Progress or no Cancelled");
  }

  // ann creates Fix the login in In Progress, high, assigned to bob and the admin, labelled UI and Bug, from the 12th
  // to the 19th of October: the answer gives its accounts and labels in their ids' order.
  const assignees = [bob, admin].toSorted(byId);
  const description = `<p>Login notes</p><mention-component entity_name="user_mention" entity_identifier="${bob.id}"></mention-component><mention-component entity_name="user_mention" entity_identifier="${admin.id}"></mention-component><mention-component entity_name="user_mention" entity_identifier="${crypto.randomUUID()}"></mention-component>`;
  const first = await createIssue(api, ann.token, web.id, {
    name: "  Fix the login ",
    description_html: description,
    state_id: inProgress,
    priority: "high",
    assignee_ids: [bob.id, admin.id],
    label_ids: [ui.id, bug.id],
    start_date: "2026-10-12",
    target_date: "2026-10-19",
  });
  expect(first).toMatchObject({
    project_id: web.id,
    sequence_id: 1,
    name: "Fix the login",
    state_id: inProgress,
    start_date: "2026-10-12",
    target_date: "2026-10-19",
    assignee_ids: assignees.map((a) => a.id),
    label_ids: [ui, bug].toSorted(byId).map((l) => l.id),
  });
  await expectIssueCreated(db, first.id, {
    sequence_id: 1,
    name: "Fix the login",
    state: "In Progress",
    priority: "high",
    start_date: "2026-10-12",
    target_date: "2026-10-19",
    sort_order: 65535,
    parent: null,
    completed: false,
    by: ann.email,
    assignees: [admin.email, bob.email].toSorted(),
    labels: ["Bug", "UI"],
    subscribers: [admin.email, ann.email, bob.email].toSorted(),
    activity: createdFor(bob, admin),
  });

  expect(first.description_html).toBe(description);
  await expectInitialDescription(db, first.id, description, [bob.id, admin.id]);

  // The admin makes ann Web's default assignee; bob's next work item, without a state or assignees, is in Backlog,
  // the default state, assigned to her.
  expect((await changeProject(api, admin.token, web.id, { default_assignee_id: ann.id })).status).toBe(200);
  const second = await createIssue(api, bob.token, web.id, { name: "Plan the release" });
  await expectIssueCreated(db, second.id, {
    sequence_id: 2,
    name: "Plan the release",
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 65535,
    parent: null,
    completed: false,
    by: bob.email,
    assignees: [ann.email],
    labels: [],
    subscribers: [ann.email, bob.email].toSorted(),
    activity: createdFor(ann),
  });
  expect(second.description_html).toBe("<p></p>");
  await expectInitialDescription(db, second.id, "<p></p>", []);

  // Once she has left Web, the next is assigned to no one, after the second in Backlog.
  expect((await projectMemberWrites(api, web.id).leave(ann.token)).status).toBe(204);
  const third = await createIssue(api, bob.token, web.id, { name: "Write the notes" });
  await expectIssueCreated(db, third.id, {
    sequence_id: 3,
    name: "Write the notes",
    state: "Backlog",
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 75535,
    parent: null,
    completed: false,
    by: bob.email,
    assignees: [],
    labels: [],
    subscribers: [bob.email],
    activity: createdFor(),
  });

  // One in Cancelled, a state of the cancelled group, is not completed (M4 design 3.7).
  const fourth = await createIssue(api, bob.token, web.id, { name: "Drop the beta", state_id: cancelled });
  await expectIssueCreated(db, fourth.id, {
    sequence_id: 4,
    name: "Drop the beta",
    state: "Cancelled",
    priority: "none",
    start_date: null,
    target_date: null,
    sort_order: 65535,
    parent: null,
    completed: false,
    by: bob.email,
    assignees: [],
    labels: [],
    subscribers: [bob.email],
    activity: createdFor(),
  });

  // An account and a label named twice, a priority in another case (the contract's values are lower case) and a start
  // date after the target date are refused, the four at once; so are a state and a label Web does not have and gus,
  // Web's guest, as an assignee: every problem in one answer.
  expect(
    await issueCreated(api, bob.token, web.id, {
      name: "Fix",
      // @ts-expect-error -- a priority in another case breaks the contract on purpose
      priority: "High",
      assignee_ids: [bob.id, bob.id],
      label_ids: [bug.id, bug.id],
      start_date: "2026-10-20",
      target_date: "2026-10-19",
    })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [
      { field: "priority", code: "invalid_format" },
      { field: "assignee_ids[1]", code: "duplicate" },
      { field: "label_ids[1]", code: "duplicate" },
      { field: "target_date", code: "not_allowed" },
    ],
  });
  expect(
    await issueCreated(api, bob.token, web.id, {
      name: "Fix",
      state_id: crypto.randomUUID(),
      assignee_ids: [gus.id],
      label_ids: [crypto.randomUUID()],
    })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [
      { field: "state_id", code: "not_allowed" },
      { field: "assignee_ids[0]", code: "not_allowed" },
      { field: "label_ids[0]", code: "not_allowed" },
    ],
  });
  // gus creates none; in archived Web nobody creates one.
  expect(await issueCreated(api, gus.token, web.id, { name: "Fix" })).toEqual({ status: 403, code: "forbidden" });
  await archiveProject(api, admin.token, web.id);
  expect(await issueCreated(api, bob.token, web.id, { name: "Fix" })).toEqual({
    status: 409,
    code: "project.archived",
  });
});
