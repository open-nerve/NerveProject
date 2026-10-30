import type { ProjectDisplayFilters, ProjectDisplayProperties, ProjectIssueFilters } from "@nerve/api-client";
import type { Page } from "@playwright/test";

import {
  addProjectMembers,
  changeProjectPreferences,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  readProjectPreferences,
  slugFor,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { accountId, emailFor, newAccount, signInContext } from "../../fixtures/auth";
import { watchPage } from "../../fixtures/browser";
import { addCondition } from "../../fixtures/issue-filters-page";
import {
  expectListOrder,
  expectListPageQuiet,
  issueListProject,
  openIssueList,
  saveListPreferences,
} from "../../fixtures/issue-list-page";
import { createIssue, listIssues, type IssueQuery } from "../../fixtures/issues";
import { answerTo, bodiesSentTo, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

test("I5 (API): two members retain different complete list settings; me follows the caller, saved filters equal explicit query parameters and never apply implicitly", async ({
  api,
  db,
}, info) => {
  const admin = await newAccount(api, info, "admin");
  const a = await newAccount(api, info, "a"),
    b = await newAccount(api, info, "b");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await [a, b].reduce(async (before, who) => {
    await before;
    await inviteAndAccept(api, admin.token, slug, who, 15);
  }, Promise.resolve());
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(
    api,
    admin.token,
    project.id,
    [a, b].map((who) => ({ member_id: who.id, role: 15 as const }))
  );
  const states = await stateIds(api, admin.token, project.id);
  const started = states["In Progress"];
  if (!started) throw new Error("expected started state");
  const mine = await createIssue(api, admin.token, project.id, {
    name: "A's match",
    state_id: started,
    assignee_ids: [a.id],
    target_date: "2026-10-16",
  });
  const yours = await createIssue(api, admin.token, project.id, {
    name: "B's match",
    state_id: started,
    assignee_ids: [b.id],
    target_date: "2026-10-16",
  });
  const outside = await createIssue(api, admin.token, project.id, {
    name: "Outside range",
    state_id: started,
    assignee_ids: [a.id],
    target_date: "2026-10-25",
  });
  const backlog = await createIssue(api, admin.token, project.id, {
    name: "Backlog",
    assignee_ids: [a.id],
    target_date: "2026-10-16",
  });
  const label = await createLabel(api, admin.token, project.id, { name: "Labelled" });
  const labelled = await createIssue(api, admin.token, project.id, {
    name: "Has label",
    state_id: started,
    assignee_ids: [a.id],
    target_date: "2026-10-16",
    label_ids: [label.id],
  });
  const filters: ProjectIssueFilters = {
    state_group: ["started"],
    assignee_id: ["me"],
    label_id: ["none"],
    target_date: ["2026-10-01..2026-10-31", "2026-10-15..2026-10-20"],
  };
  const display: ProjectDisplayFilters = {
    layout: "kanban",
    group_by: "state_id",
    sub_group_by: null,
    order_by: "sort_order",
    sub_issue: false,
    show_empty_groups: false,
    calendar: { layout: "week", show_weekends: true },
  };
  const properties: ProjectDisplayProperties = {
    assignee: true,
    start_date: true,
    due_date: true,
    labels: false,
    key: true,
    priority: true,
    state: true,
    sub_issue_count: true,
    link: false,
    created_on: true,
    updated_on: true,
  };
  const savedA = {
    ...(await readProjectPreferences(api, a.token, project.id)),
    filters,
    display_filters: display,
    display_properties: properties,
  };
  expect(await changeProjectPreferences(api, a.token, project.id, savedA)).toEqual({
    status: 200,
    preferences: savedA,
  });
  const savedB = {
    ...(await readProjectPreferences(api, b.token, project.id)),
    filters: { priority: ["low"] },
    display_filters: { ...display, layout: "calendar", group_by: "priority", order_by: "priority" },
    display_properties: { ...properties, labels: true, priority: false },
  };
  expect(await changeProjectPreferences(api, b.token, project.id, savedB)).toEqual({
    status: 200,
    preferences: savedB,
  });
  // Deliberately write B last: a missing actor condition must not pass by returning A's defaults.
  expect(await readProjectPreferences(api, a.token, project.id)).toEqual(savedA);
  expect(await readProjectPreferences(api, b.token, project.id)).toEqual(savedB);
  await (
    [
      [a, savedA],
      [b, savedB],
    ] as const
  ).reduce(async (before, [who, saved]) => {
    await before;
    expect(
      await db.query(
        `SELECT filters, display_filters, display_properties, updated_by_id::text AS actor
         FROM project_user_properties WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL`,
        [project.id, who.id]
      )
    ).toEqual([
      {
        filters: saved.filters,
        display_filters: saved.display_filters,
        display_properties: saved.display_properties,
        actor: who.id,
      },
    ]);
  }, Promise.resolve());
  const loaded = await readProjectPreferences(api, a.token, project.id);
  // Settings intentionally use broad string vocabularies; narrow the two requested values by assertion.
  expect(loaded.display_filters.group_by).toBe("state_id");
  expect(loaded.display_filters.order_by).toBe("sort_order");
  const fromSaved: IssueQuery = {
    ...loaded.filters,
    group_by: "state_id",
    order_by: "sort_order",
    sub_issue: String(loaded.display_filters.sub_issue),
  };
  const explicit: IssueQuery = {
    state_group: ["started"],
    assignee_id: ["me"],
    label_id: ["none"],
    target_date: ["2026-10-01..2026-10-31", "2026-10-15..2026-10-20"],
    group_by: "state_id",
    order_by: "sort_order",
    sub_issue: "false",
  };
  const result = await listIssues(api, a.token, project.id, fromSaved);
  expect(result).toEqual(await listIssues(api, a.token, project.id, explicit));
  expect(result).toMatchObject({
    total: 1,
    groups: [{ key: started, total: 1, data: [{ id: mine.id }], next_cursor: null }],
  });
  expect(await listIssues(api, b.token, project.id, explicit)).toMatchObject({
    total: 1,
    groups: [{ data: [{ id: yours.id }] }],
  });
  const implicit = await listIssues(api, a.token, project.id);
  expect(implicit.total).toBe(5);
  expect(implicit.data?.map((i) => i.id).toSorted()).toEqual(
    [mine.id, yours.id, outside.id, backlog.id, labelled.id].toSorted()
  );
  expect(implicit).not.toHaveProperty("groups");
  expect(await changeProjectPreferences(api, a.token, project.id, { filters: {} })).toEqual({
    status: 200,
    preferences: { ...savedA, filters: {} },
  });
  expect(await readProjectPreferences(api, b.token, project.id)).toEqual(savedB);
  expect(
    await changeProjectPreferences(api, a.token, project.id, { display_filters: { ...display, group_by: "cycle_id" } })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "display_filters.group_by", code: "not_allowed" }],
  });
  expect(await changeProjectPreferences(api, a.token, project.id, { filters: { label_id: ["wrong"] } })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "filters.label_id[0]", code: "not_allowed" }],
  });
  expect(await readProjectPreferences(api, a.token, project.id)).toEqual({ ...savedA, filters: {} });
  expectIssuePage(await listIssues(api, a.token, project.id, { priority: ["low"] }), []);
});

async function addChoice(page: Page, name: string, value: string, preferencesPath: string) {
  await addCondition(page, name);
  const saved = await sentTo(page, "PATCH", preferencesPath, () =>
    page.getByRole("option", { name: value, exact: true }).click()
  );
  expect(saved.answer.status()).toBe(200);
  await page.keyboard.press("Escape");
  return saved.body;
}
async function addRange(page: Page, from: string, to: string, preferencesPath: string) {
  const row = await addCondition(page, "Target date");
  await page.keyboard.press("Escape");
  await row.getByRole("button", { name: "is", exact: true }).click();
  await page.getByRole("option", { name: "between", exact: true }).click();
  // Select calendar captions explicitly: this story remains independent of today's month.
  await page.getByRole("combobox", { name: "Choose the Year" }).selectOption("2026");
  await page.getByRole("combobox", { name: "Choose the Month" }).selectOption("9");
  await page.locator(`[data-day="${from}"] button`).click();
  await expect(row.locator('button[aria-invalid="true"]')).toHaveAccessibleDescription(
    "Choose both dates to apply the range"
  );
  const saved = await sentTo(page, "PATCH", preferencesPath, () => page.locator(`[data-day="${to}"] button`).click());
  expect(saved.answer.status()).toBe(200);
  await page.keyboard.press("Escape");
  return saved.body;
}
async function displayPanel(page: Page) {
  const button = page.getByRole("button", { name: "Display", exact: true });
  if ((await button.getAttribute("aria-expanded")) !== "true") await button.click();
  const id = await button.getAttribute("aria-controls");
  expect(id).toBeTruthy();
  return page.locator(`[id="${id}"]`);
}

test("I5 (page): the toolbar saves started, me, none and two intersecting date ranges; reload, another account and another project keep their own settings", async ({
  api,
  db,
  page,
  browser,
  browserName,
  nerve,
}, info) => {
  const seed = await issueListProject(api, info);
  const { member: a, project, states, slug } = seed;
  const bTokens = await registerOnboarded(api, emailFor(info, "other-member"));
  const b = { token: bTokens.access_token, id: await accountId(api, bTokens.access_token) };
  await inviteAndAccept(api, a.token, slug, { ...b, email: emailFor(info, "other-member") }, 15);
  await addProjectMembers(api, a.token, project.id, [{ member_id: b.id, role: 15 }]);
  const otherProject = await createProject(api, a.token, slug, { name: "Other", identifier: "OTHER" });
  const label = await createLabel(api, a.token, project.id, { name: "Labelled" });
  const mine = await createIssue(api, a.token, project.id, {
    name: "A's match",
    state_id: states["In Progress"],
    assignee_ids: [a.id],
    target_date: "2026-10-16",
  });
  const yours = await createIssue(api, a.token, project.id, {
    name: "B's match",
    state_id: states["In Progress"],
    assignee_ids: [b.id],
    target_date: "2026-10-16",
  });
  const outside = await createIssue(api, a.token, project.id, {
    name: "Outside range",
    state_id: states["In Progress"],
    assignee_ids: [a.id],
    target_date: "2026-10-25",
  });
  const backlog = await createIssue(api, a.token, project.id, {
    name: "Backlog",
    assignee_ids: [a.id],
    target_date: "2026-10-16",
  });
  const labelled = await createIssue(api, a.token, project.id, {
    name: "Has label",
    state_id: states["In Progress"],
    assignee_ids: [a.id],
    target_date: "2026-10-16",
    label_ids: [label.id],
  });
  const defaults = await readProjectPreferences(api, a.token, project.id);
  const initial = await saveListPreferences(api, seed, {
    filters: {},
    display_filters: { ...defaults.display_filters, layout: "list", group_by: null, sub_group_by: null },
  });
  const bBefore = await readProjectPreferences(api, b.token, project.id);
  const otherBefore = await readProjectPreferences(api, a.token, otherProject.id);
  const path = `/api/v0/me/projects/${project.id}/preferences`;
  const { watch, queries } = await openIssueList(page, seed);
  await expectListOrder(page.locator("body"), [labelled.id, backlog.id, outside.id, yours.id, mine.id]);
  expect(await addChoice(page, "State Group", "Started", path)).toEqual({ filters: { state_group: ["started"] } });
  expect(await addChoice(page, "Assignees", "Me", path)).toEqual({
    filters: { state_group: ["started"], assignee_id: ["me"] },
  });
  expect(await addChoice(page, "Label", "None", path)).toEqual({
    filters: { state_group: ["started"], assignee_id: ["me"], label_id: ["none"] },
  });
  await addRange(page, "2026-10-01", "2026-10-31", path);
  const filters: ProjectIssueFilters = {
    state_group: ["started"],
    assignee_id: ["me"],
    label_id: ["none"],
    target_date: ["2026-10-01..2026-10-31", "2026-10-15..2026-10-20"],
  };
  expect(await addRange(page, "2026-10-15", "2026-10-20", path)).toEqual({ filters });
  await expectListOrder(page.locator("body"), [mine.id]);
  await expect.poll(() => queries.at(-1)?.getAll("target_date")).toEqual(filters.target_date);
  expect(queries.at(-1)?.getAll("state_group")).toEqual(["started"]);
  expect(queries.at(-1)?.getAll("assignee_id")).toEqual(["me"]);
  expect(queries.at(-1)?.getAll("label_id")).toEqual(["none"]);
  expect(
    await db.query(
      "SELECT filters, display_filters, display_properties, updated_by_id::text AS actor FROM project_user_properties WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL",
      [project.id, a.id]
    )
  ).toEqual([
    { filters, display_filters: initial.display_filters, display_properties: initial.display_properties, actor: a.id },
  ]);
  await page.reload();
  await expectListOrder(page.locator("body"), [mine.id]);
  await expect(page.getByRole("button", { name: "Target date", exact: true })).toHaveCount(2);
  expect(await readProjectPreferences(api, b.token, project.id)).toEqual(bBefore);
  expect(await readProjectPreferences(api, a.token, otherProject.id)).toEqual(otherBefore);
  // A separate real session sees all five rows; A's saved filter must not apply to B.
  const context = await browser.newContext({ baseURL: nerve.baseURL });
  try {
    await signInContext(context, nerve.baseURL, bTokens);
    const bPage = await context.newPage();
    const bWatch = await watchPage(bPage);
    await bPage.goto(seed.path);
    await Promise.all(
      [mine, yours, outside, backlog, labelled].map((item) =>
        expect(bPage.locator(`[id^="issue_${item.id}_"]`)).toBeAttached()
      )
    );
    await expect(bPage.getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(0);
    await expectListPageQuiet(bPage, bWatch, browserName, 1);
  } finally {
    await context.close();
  }
  await page.goto(`/${slug}/projects/${otherProject.id}/issues`);
  await expect(page.getByRole("button", { name: "Add filter", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Remove filter", exact: true })).toHaveCount(0);
  await expectListPageQuiet(page, watch, browserName, 3);
});

test("I5 (page): all eleven display properties and layout, grouping, ordering and extra options save from their controls without losing untouched calendar values", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const { project, member } = seed;
  await createIssue(api, member.token, project.id, { name: "Visible preference target" });
  const defaults = await readProjectPreferences(api, member.token, project.id);
  const initial = await saveListPreferences(api, seed, {
    display_filters: {
      ...defaults.display_filters,
      layout: "list",
      group_by: null,
      sub_group_by: null,
      calendar: { layout: "week", show_weekends: false },
    },
  });
  const path = `/api/v0/me/projects/${project.id}/preferences`;
  const { watch, queries } = await openIssueList(page, seed);
  let panel = await displayPanel(page);
  const properties: [keyof ProjectDisplayProperties, string][] = [
    ["assignee", "Assignee"],
    ["start_date", "Start date"],
    ["due_date", "Due date"],
    ["labels", "Labels"],
    ["key", "ID"],
    ["priority", "Priority"],
    ["state", "State"],
    ["sub_issue_count", "Sub-work item count"],
    ["link", "Link"],
    ["created_on", "Created on"],
    ["updated_on", "Updated on"],
  ];
  const expected = { ...initial.display_properties };
  const propertyGroup = panel.getByRole("button", { name: "Display Properties", exact: true }).locator("../..");
  await properties.reduce(async (before, [key, name]) => {
    await before;
    const button = propertyGroup.getByRole("button", { name, exact: true });
    await expect(button).toHaveAttribute("aria-pressed", String(expected[key]));
    expected[key] = !expected[key];
    const sent = await sentTo(page, "PATCH", path, () => button.click());
    expect(sent.answer.status()).toBe(200);
    expect(sent.body).toEqual({ display_properties: expected });
    await expect(button).toHaveAttribute("aria-pressed", String(expected[key]));
  }, Promise.resolve());
  await page.keyboard.press("Escape");
  const board = await sentTo(page, "PATCH", path, () =>
    page.getByRole("button", { name: "Board Layout", exact: true }).click()
  );
  expect(board.answer.status()).toBe(200);
  await expect(page.getByRole("button", { name: "Board Layout", exact: true })).toHaveAttribute("aria-pressed", "true");
  panel = await displayPanel(page);
  const group = panel.getByRole("button", { name: "Group by", exact: true }).locator("../..");
  expect(
    (
      await answerTo(page, "PATCH", path, () => group.getByRole("button", { name: "Priority", exact: true }).click())
    ).status()
  ).toBe(200);
  const subgroup = panel.getByRole("button", { name: "Sub-group by", exact: true }).locator("../..");
  expect(
    (
      await answerTo(page, "PATCH", path, () => subgroup.getByRole("button", { name: "Labels", exact: true }).click())
    ).status()
  ).toBe(200);
  expect(
    (
      await answerTo(page, "PATCH", path, () =>
        panel.getByRole("button", { name: "Last updated", exact: true }).click()
      )
    ).status()
  ).toBe(200);
  await ["Show empty groups", "Show sub-work items"].reduce(async (before, name) => {
    await before;
    const button = panel.getByRole("button", { name, exact: true });
    if ((await button.getAttribute("aria-pressed")) !== "false")
      expect((await answerTo(page, "PATCH", path, () => button.click())).status()).toBe(200);
    await expect(button).toHaveAttribute("aria-pressed", "false");
  }, Promise.resolve());
  const saved = await readProjectPreferences(api, member.token, project.id);
  expect(saved).toEqual({
    ...initial,
    display_properties: expected,
    display_filters: {
      ...initial.display_filters,
      layout: "kanban",
      group_by: "priority",
      sub_group_by: "label_id",
      order_by: "-updated_at",
      sub_issue: false,
      show_empty_groups: false,
    },
  });
  expect(
    await db.query(
      "SELECT filters, display_filters, display_properties FROM project_user_properties WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL",
      [project.id, member.id]
    )
  ).toEqual([{ filters: saved.filters, display_filters: saved.display_filters, display_properties: expected }]);
  await expect.poll(() => queries.at(-1)?.get("sub_issue")).toBe("false");
  expect(queries.at(-1)?.get("group_by")).toBe("priority");
  expect(queries.at(-1)?.get("sub_group_by")).toBe("label_id");
  expect(queries.at(-1)?.get("order_by")).toBe("-updated_at");
  await page.reload();
  panel = await displayPanel(page);
  await Promise.all(
    properties.map(([key, name]) =>
      expect(
        panel
          .getByRole("button", { name: "Display Properties", exact: true })
          .locator("../..")
          .getByRole("button", { name, exact: true })
      ).toHaveAttribute("aria-pressed", String(expected[key]))
    )
  );
  await expectListPageQuiet(page, watch, browserName, 2);
});

test("I5 (page): a refused preference keeps confirmed UI and database; a confirmed change with a failed read retries only GET", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const { project, member } = seed;
  const row = await createIssue(api, member.token, project.id, { name: "Saved preference target" });
  const initial = await readProjectPreferences(api, member.token, project.id);
  const path = `/api/v0/me/projects/${project.id}/preferences`;
  const list = `/api/v0/projects/${project.id}/issues`;
  const { watch } = await openIssueList(page, seed);
  await expect(page.locator(`[id^="issue_${row.id}_"]`)).toBeAttached();
  const bodies = await bodiesSentTo(page, "PATCH", path);
  let refuse = true,
    failRead = false;
  await page.route(`**${path}`, async (route) => {
    if (route.request().method() !== "PATCH" || !refuse) return route.fallback();
    refuse = false;
    await route.fulfill({
      status: 403,
      contentType: "application/problem+json",
      body: JSON.stringify({ type: "about:blank", title: "Forbidden", status: 403, code: "forbidden" }),
    });
  });
  await page.route(
    (url) => url.pathname === list,
    async (route) => {
      if (route.request().method() !== "GET" || !failRead) return route.fallback();
      failRead = false;
      await route.fulfill({
        status: 503,
        contentType: "application/problem+json",
        body: JSON.stringify({ type: "about:blank", title: "Server busy", status: 503, code: "server_busy" }),
      });
    }
  );
  const panel = await displayPanel(page);
  const priority = panel
    .getByRole("button", { name: "Display Properties", exact: true })
    .locator("../..")
    .getByRole("button", { name: "Priority", exact: true });
  expect((await answerTo(page, "PATCH", path, () => priority.click())).status()).toBe(403);
  await expect(priority).toHaveAttribute("aria-pressed", String(initial.display_properties.priority));
  await expect(page.getByText("Your role does not allow this.", { exact: true })).toHaveCount(1);
  expect(await readProjectPreferences(api, member.token, project.id)).toEqual(initial);
  expect(
    await db.query(
      "SELECT display_properties FROM project_user_properties WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL",
      [project.id, member.id]
    )
  ).toEqual([{ display_properties: initial.display_properties }]);
  const sub = panel.getByRole("button", { name: "Show sub-work items", exact: true });
  failRead = true;
  const changed = await answerTo(page, "PATCH", path, () => sub.click());
  expect(changed.status()).toBe(200);
  await expect(sub).toHaveAttribute("aria-pressed", String(!initial.display_filters.sub_issue));
  const saved = await readProjectPreferences(api, member.token, project.id);
  expect(saved.display_filters).toEqual({ ...initial.display_filters, sub_issue: !initial.display_filters.sub_issue });
  await page.keyboard.press("Escape");
  const retry = page.getByRole("button", { name: "Retry", exact: true });
  await expect(retry).toBeVisible();
  const patchCount = watch.apiRequests.filter((request) => request === `PATCH ${path}`).length;
  expect(patchCount).toBe(2);
  expect((await answerTo(page, "GET", list, () => retry.click())).status()).toBe(200);
  await expect(retry).toHaveCount(0);
  await expect(page.locator(`[id^="issue_${row.id}_"]`)).toBeAttached();
  expect(watch.apiRequests.filter((request) => request === `PATCH ${path}`)).toHaveLength(patchCount);
  // The route that refuses the first PATCH does not reach bodiesSentTo's fallback handler.
  expect(bodies).toEqual([{ display_filters: saved.display_filters }]);
  expect(await readProjectPreferences(api, member.token, project.id)).toEqual(saved);
  await expectListPageQuiet(page, watch, browserName, 1, {
    failures: [`403 PATCH ${path}`, `503 GET ${list}`],
    errors:
      browserName === "chromium" || browserName === "webkit"
        ? [
            "Failed to load resource: the server responded with a status of 403 (Forbidden)",
            "Failed to load resource: the server responded with a status of 503 (Service Unavailable)",
          ]
        : [],
  });
});
