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
import { newAccount, shownNameOf } from "../../fixtures/auth";
import { createIssue, issueCreated } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";
import { inputTextIncludingControls, recordFocuses } from "../../fixtures/browser";
import { expectListPageQuiet, issueListProject, openIssueList } from "../../fixtures/issue-list-page";
import { answerTo, bodiesSentTo, holdAnswer } from "../../fixtures/settings-pages";

test("I1 (page): c opens a named form; field refusals retain it, native controls create one numbered work item with a mention, and a held confirmation cannot be submitted twice", async ({
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
  const { watch } = await openIssueList(page, seed);
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toBeVisible();
  await page.keyboard.press("c");
  const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
  await expect(form).toHaveCount(1);
  const title = form.getByRole("textbox", { name: "Title", exact: true });
  const save = form.getByRole("button", { name: "Save", exact: true });
  const path = `/api/v0/projects/${seed.project.id}/issues`;
  const bodies = await bodiesSentTo(page, "POST", path);
  await (
    [
      ["   ", "Title is required"],
      ["x".repeat(256), "Too long"],
    ] as const
  ).reduce(async (previous, [value, description]) => {
    await previous;
    await title.fill(value!);
    const focuses = await recordFocuses(page);
    await save.click();
    await expect(title).toBeFocused();
    expect((await focuses()).at(-1)).toEqual({ id: "name", invalid: "true", description });
    expect(bodies).toEqual([]);
    await expect(title).toBeVisible();
  }, Promise.resolve());
  await inputTextIncludingControls(title, "Invalid\u0000title");
  const focuses = await recordFocuses(page);
  expect((await answerTo(page, "POST", path, () => save.click())).status()).toBe(422);
  await expect(title).toBeFocused();
  expect((await focuses()).at(-1)).toEqual({ id: "name", invalid: "true", description: "The format is not valid" });
  expect(await db.query("SELECT id FROM issues WHERE project_id=$1", [seed.project.id])).toEqual([]);

  await title.fill("  Created through the form  ");
  const more = form.getByRole("switch", { name: "Create more", exact: true });
  await more.click();
  await expect(more).toBeChecked();
  await more.click();
  await expect(more).not.toBeChecked();
  const editor = form.locator('[contenteditable="true"]');
  await expect(editor).toBeVisible();
  await editor.fill("Notes ");
  await editor.press("End");
  await page.keyboard.type("@");
  await page
    .locator('[id^="mention-item-"]')
    .filter({ hasText: shownNameOf(colleague.email) })
    .click();
  await expect(editor.locator(".mention-component")).toHaveCount(1);
  await form.getByRole("button", { name: "State: Backlog", exact: true }).click();
  await page.getByRole("option", { name: "In Progress", exact: true }).click();
  await form.getByRole("button", { name: "Priority: None", exact: true }).click();
  await page.getByRole("option", { name: "High", exact: true }).click();
  await form.getByRole("button", { name: "Assignees", exact: true }).click();
  await page.getByRole("option", { name: /You$/ }).click();
  await page
    .getByRole("option")
    .filter({ hasText: shownNameOf(colleague.email) })
    .click();
  await title.click();
  await form.getByRole("button", { name: /Labels/ }).click();
  await page.getByRole("option", { name: "Bug", exact: true }).click();
  await page.getByRole("option", { name: "UI", exact: true }).click();
  await title.click();
  await (
    [
      ["Start date", 10],
      ["Due date", 20],
    ] as const
  ).reduce(async (previous, [label, day]) => {
    await previous;
    await form.getByRole("button", { name: label, exact: true }).click();
    const calendar = page.getByRole("dialog", { name: label, exact: true });
    await calendar.getByRole("combobox", { name: "Choose the Year" }).selectOption("2026");
    await calendar.getByRole("combobox", { name: "Choose the Month" }).selectOption("10");
    if (label === "Due date") {
      await expect(calendar.getByRole("button", { name: /November 9th, 2026/ })).toBeDisabled();
      await expect(title).toBeVisible();
    }
    await calendar.getByRole("button", { name: new RegExp(`November ${day}th, 2026`) }).click();
  }, Promise.resolve());
  expect(
    await form
      .locator("[tabindex]")
      .evaluateAll((nodes) => nodes.filter((node) => Number(node.getAttribute("tabindex")) > 0).length)
  ).toBe(0);
  const release = await holdAnswer(page, "POST", path);
  // Record after the holding route, so the request reaches this observer before route.fetch.
  const submissions = await bodiesSentTo(page, "POST", path);
  await save.dblclick();
  await expect.poll(() => submissions.length).toBe(1);
  await expect(form.getByRole("button", { name: "Saving", exact: true })).toBeDisabled();
  await expect(form.getByRole("button", { name: "Discard", exact: true })).toBeDisabled();
  await expect(title).toBeVisible();
  await expect
    .poll(() => db.query("SELECT name FROM issues WHERE project_id=$1", [seed.project.id]))
    .toEqual([{ name: "Created through the form" }]);
  const response = await answerTo(page, "POST", path, release);
  expect(response.status()).toBe(201);
  const issue = await response.json();
  await expect(
    page.getByRole("region", { name: "Notifications" }).getByRole("link", { name: "WEB-1", exact: true })
  ).toBeVisible();
  expect(bodies).toHaveLength(1);
  expect(submissions).toHaveLength(1);
  expect(submissions[0]).toMatchObject({
    name: "  Created through the form  ",
    state_id: seed.states["In Progress"],
    priority: "high",
    assignee_ids: [seed.member.id, colleague.id],
    label_ids: [bug.id, ui.id],
    start_date: "2026-11-10",
    target_date: "2026-11-20",
  });
  await expect(form).toHaveCount(0);
  const row = page.locator(`[id="issue-${issue.id}"]`);
  await expect(row).toContainText("WEB-1");
  await expect(row).toContainText("Created through the form");
  await expectIssueCreated(db, issue.id, {
    sequence_id: 1,
    name: "Created through the form",
    state: "In Progress",
    priority: "high",
    start_date: "2026-11-10",
    target_date: "2026-11-20",
    sort_order: 65535,
    parent: null,
    completed: false,
    by: seed.member.email,
    assignees: [seed.member.email, colleague.email].toSorted(),
    labels: ["Bug", "UI"],
    subscribers: [seed.member.email, colleague.email].toSorted(),
    activity: createdFor(seed.member, colleague),
  });
  const html = (submissions[0] as { description_html: string }).description_html;
  expect(html).toContain(`entity_identifier="${colleague.id}"`);
  await expectInitialDescription(db, issue.id, html, [colleague.id]);
  expect(await db.query("SELECT description_stripped FROM issues WHERE id=$1", [issue.id])).toEqual([
    { description_stripped: "Notes" },
  ]);
  expect(
    await db.query("SELECT last_issue_sequence::int AS sequence FROM projects WHERE id=$1", [seed.project.id])
  ).toEqual([{ sequence: 1 }]);
  await expectListPageQuiet(page, watch, {
    failures: [`422 POST ${path}`],
    errors:
      browserName === "chromium" || browserName === "webkit"
        ? ["Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)"]
        : [],
  });
});

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
