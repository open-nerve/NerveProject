import { createProject, createWorkspace, readProjectPreferences, slugFor } from "../../fixtures/api";
import { createdFor, expectIssueCreated, expectIssuePage, madeBy } from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import { createIssue, issueCreated, listIssues, updateIssue } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

import {
  issueListProject,
  saveListPreferences,
  openIssueList,
  listGroup,
  expectListOrder,
  expectListPageQuiet,
} from "../../fixtures/issue-list-page";
import { recordFocuses } from "../../fixtures/browser";
import { sentHeld } from "../../fixtures/settings-pages";
import { sentWithin } from "../../fixtures/dialogs";

test("I7 (API): quick-add uses ordinary creation in the chosen state, after that state's greatest position plus 10000; blank names are refused", async ({
  api,
  db,
}, info) => {
  const member = await newAccount(api, info, "member");
  const slug = slugFor(info);
  await createWorkspace(api, member.token, { name: "Acme", slug });
  const project = await createProject(api, member.token, slug, { name: "Web", identifier: "WEB" });
  const states = await stateIds(api, member.token, project.id);
  const todo = states.Todo;
  if (!todo || !states.Backlog) throw new Error("expected Todo and Backlog");
  const first = await createIssue(api, member.token, project.id, { name: "First", state_id: todo });
  const last = await createIssue(api, member.token, project.id, { name: "Last", state_id: todo });
  const other = await createIssue(api, member.token, project.id, { name: "Other", state_id: states.Backlog });
  await updateIssue(api, member.token, first.id, { sort_order: 100 });
  await updateIssue(api, member.token, last.id, { sort_order: 300 });
  await updateIssue(api, member.token, other.id, { sort_order: 90000 });
  const added = await createIssue(api, member.token, project.id, { name: "  Quick add  ", state_id: todo });
  expect(added).toMatchObject({ name: "Quick add", state_id: todo, sort_order: 10300, sequence_id: 4 });
  await expectIssueCreated(db, added.id, {
    ...madeBy(member.email, "Quick add"),
    sequence_id: 4,
    state: "Todo",
    sort_order: 10300,
    by: member.email,
    activity: createdFor(),
  });
  expectIssuePage(await listIssues(api, member.token, project.id, { state_id: [todo], order_by: "sort_order" }), [
    first.id,
    last.id,
    added.id,
  ]);
  expect(await issueCreated(api, member.token, project.id, { name: "   ", state_id: todo })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "name", code: "too_short" }],
  });
});

test("I7 (page): chosen-state quick add describes refusal before focus, keeps pending input and sends one confirmed creation at the state tail", async ({
  api,
  page,
  db,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const { member, project, states } = seed;
  const first = await createIssue(api, member.token, project.id, { name: "First", state_id: states.Todo });
  const last = await createIssue(api, member.token, project.id, { name: "Last", state_id: states.Todo });
  const other = await createIssue(api, member.token, project.id, { name: "Other", state_id: states.Backlog });
  await updateIssue(api, member.token, first.id, { sort_order: 100 });
  await updateIssue(api, member.token, last.id, { sort_order: 300 });
  await updateIssue(api, member.token, other.id, { sort_order: 90000 });
  const prefs = await readProjectPreferences(api, member.token, project.id);
  await saveListPreferences(api, seed, {
    filters: {},
    display_filters: {
      ...prefs.display_filters,
      layout: "list",
      group_by: "state_id",
      order_by: "sort_order",
      show_empty_groups: false,
    },
  });
  const { watch } = await openIssueList(page, seed);
  const todo = listGroup(page, "Todo");
  await expectListOrder(todo, [first.id, last.id]);
  await todo.getByRole("button", { name: "New work item", exact: true }).click();
  const name = todo.getByRole("textbox", { name: "Work item title" });
  await expect(name).toBeFocused();
  const focuses = await recordFocuses(page);
  await name.fill("   ");
  await name.evaluate((element: HTMLInputElement) => element.blur());
  const post = `/api/v0/projects/${project.id}/issues`;
  // Native form submission with focus outside makes the refusal's first focus observable.
  // Later attempts below use Enter on the real field.
  expect(
    await sentWithin(page, "POST", post, () =>
      name.evaluate((element: HTMLInputElement) => element.form!.requestSubmit())
    )
  ).toBe(false);
  await expect(name).toHaveAttribute("aria-invalid", "true");
  await expect(name).toHaveAccessibleDescription("Work item title is required.");
  await expect(name).toBeFocused();
  const refused = (await focuses()).filter((focus) => focus.invalid === "true");
  expect(refused.at(-1)?.description).toBe("Work item title is required.");

  let refuseNext = true;
  await page.route(`**${post}`, async (route) => {
    if (route.request().method() !== "POST" || !refuseNext) return route.fallback();
    refuseNext = false;
    await route.fulfill({
      status: 422,
      contentType: "application/problem+json",
      body: JSON.stringify({
        type: "about:blank",
        title: "Validation failed",
        status: 422,
        code: "validation_failed",
        errors: [{ field: "name", code: "too_short" }],
      }),
    });
  });
  await name.fill("  Quick add  ");
  await name.press("Enter");
  await expect(name).toHaveAccessibleDescription("Too short");
  await expect(name).toHaveValue("  Quick add  ");
  await expect(name).toBeFocused();
  const attempts = () => watch.apiRequests.filter((request) => request === `POST ${post}`).length;
  const before = attempts();
  const held = await sentHeld(page, "POST", post, () => name.press("Enter"));
  expect(held.body).toMatchObject({ name: "  Quick add  ", state_id: states.Todo });
  await expect(name).toBeDisabled();
  await expect(name).toHaveValue("  Quick add  ");
  expect(await sentWithin(page, "POST", post, () => page.keyboard.press("Enter"))).toBe(false);
  expect(attempts()).toBe(before + 1);
  await expectListOrder(todo, [first.id, last.id]);
  const response = await held.release();
  expect(response.status()).toBe(201);
  const added = await response.json();
  await expectListOrder(todo, [first.id, last.id, added.id]);
  await expect(name).toHaveValue("");
  await expect(name).toBeFocused();
  await expectIssueCreated(db, added.id, {
    ...madeBy(member.email, "Quick add"),
    sequence_id: 4,
    state: "Todo",
    sort_order: 10300,
    by: member.email,
    activity: createdFor(),
  });
  await expectListPageQuiet(page, watch, browserName, 1, {
    failures: [`422 POST ${post}`],
    errors:
      browserName === "chromium" || browserName === "webkit"
        ? ["Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)"]
        : [],
  });
});
