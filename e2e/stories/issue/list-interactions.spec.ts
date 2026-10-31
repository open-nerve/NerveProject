import type { Locator, Page, TestInfo } from "@playwright/test";
import { type Api, changeProject, createLabel, readProjectPreferences } from "../../fixtures/api";
import { createIssue, issueRead } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";
import {
  issueListProject,
  saveListPreferences,
  openIssueList,
  expectListPageQuiet,
} from "../../fixtures/issue-list-page";
import {
  answerTo,
  sentHeld,
  expectPopupBesideButton,
  expectListBesideButton,
  transitionsEnded,
} from "../../fixtures/settings-pages";
import { modalDialog, pageBehindHidden, composingKey } from "../../fixtures/dialogs";

async function controlsProject(api: Api, info: TestInfo) {
  const seed = await issueListProject(api, info);
  const label = await createLabel(api, seed.member.token, seed.project.id, { name: "Visible label" });
  const row = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Control row",
    state_id: seed.states.Todo,
  });
  const ranged = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Range row",
    state_id: seed.states.Todo,
    start_date: "2026-11-10",
    target_date: "2026-11-20",
  });
  const prefs = await readProjectPreferences(api, seed.member.token, seed.project.id);
  await saveListPreferences(api, seed, {
    filters: {},
    display_filters: { ...prefs.display_filters, layout: "list", group_by: "state_id", order_by: "sort_order" },
    display_properties: {
      ...prefs.display_properties,
      state: true,
      priority: true,
      assignee: true,
      labels: true,
      start_date: true,
      due_date: true,
    },
  });
  return { ...seed, row, ranged, label };
}

/** Keyboard opens the real search panel; its complete panel (including the input) owns the placement. */
async function searchPanel(page: Page, button: Locator, option: string, insideModal = false) {
  await button.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("option", { name: option, exact: true })).toBeVisible();
  const panel = page.locator("[data-popper-placement]").filter({ has: page.getByRole("combobox") });
  await expectPopupBesideButton(panel, button);
  await expect(page.getByRole("combobox")).toBeFocused();
  if (insideModal) expect(await pageBehindHidden(page)).toBe(true);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(button).toBeFocused();
}

test("C3: list dates use one keyboard state, adjacent controls close each other, and seven property panels anchor to their triggers", async ({
  api,
  page,
}, info) => {
  const seed = await controlsProject(api, info);
  const { watch } = await openIssueList(page, seed);
  const row = page.locator(`[id^="issue_${seed.row.id}_"]`);
  const start = row.getByRole("button", { name: "Start date", exact: true });
  const due = row.getByRole("button", { name: "Due date", exact: true });
  await start.focus();
  await page.keyboard.press("Enter");
  const startCalendar = page.getByRole("dialog", { name: "Start date", exact: true });
  await expectPopupBesideButton(startCalendar, start);
  await due.click();
  await expect(startCalendar).toHaveCount(0);
  const dueCalendar = page.getByRole("dialog", { name: "Due date", exact: true });
  await expectPopupBesideButton(dueCalendar, due);
  await page.keyboard.press("Escape");
  await expect(dueCalendar).toHaveCount(0);
  await expect(due).toBeFocused();
  await searchPanel(page, row.getByRole("button", { name: "State: Todo", exact: true }), "Done");
  await searchPanel(page, row.getByRole("button", { name: "Priority: none", exact: true }), "High");
  await searchPanel(page, row.getByRole("button", { name: "Labels: Control row", exact: true }), "Visible label");
  await searchPanel(
    page,
    row.getByRole("button", { name: "Assignees", exact: true }),
    `${seed.member.email.split("@")[0]} You`
  );
  const range = page
    .locator(`[id^="issue_${seed.ranged.id}_"]`)
    .getByRole("button", { name: "Date Range", exact: true });
  await range.focus();
  await page.keyboard.press("Enter");
  const rangeCalendar = page.getByRole("dialog", { name: "Date Range", exact: true });
  await expectPopupBesideButton(rangeCalendar, range);
  await page.keyboard.press("Escape");
  await expect(rangeCalendar).toHaveCount(0);
  await expect(range).toBeFocused();
  const options = row.getByRole("button", { name: "Options: Control row", exact: true });
  await options.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menu")).toBeVisible();
  await page.keyboard.press("ArrowDown");
  await expect(page.getByRole("menu")).toHaveAttribute("aria-activedescendant", /.+/);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("menu")).toHaveCount(0);
  await expect(options).toBeFocused();
  const rowLink = page.locator(`#issue-${seed.row.id}`);
  await rowLink.focus();
  await row.getByText("Control row", { exact: true }).click({ button: "right" });
  const menu = page.getByRole("menu");
  await expect(menu.getByRole("menuitem").first()).toBeFocused();
  await page.keyboard.press("ArrowDown");
  await expect(menu.getByRole("menuitem").nth(1)).toBeFocused();
  await composingKey(menu, "Escape");
  await expect(menu).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(menu).toHaveCount(0);
  await expect(rowLink).toBeFocused();
  expect(
    await page
      .locator('[data-context-menu="true"]')
      .evaluateAll((panels) => panels.every((panel) => !!panel.closest('[inert][aria-hidden="true"]')))
  ).toBe(true);
  await expectListPageQuiet(page, watch);
});

test("C3: a current row property refusal is announced once without an unhandled promise or a false confirmed value", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await controlsProject(api, info);
  const { watch } = await openIssueList(page, seed);
  const priority = page
    .locator(`[id^="issue_${seed.row.id}_"]`)
    .getByRole("button", { name: "Priority: none", exact: true });
  const path = `/api/v0/issues/${seed.row.id}`;
  await page.route(`**${path}`, async (route) => {
    if (route.request().method() !== "PATCH") return route.fallback();
    await route.fulfill({
      status: 403,
      contentType: "application/problem+json",
      body: JSON.stringify({ type: "about:blank", title: "Forbidden", status: 403, code: "forbidden" }),
    });
  });
  await priority.focus();
  await page.keyboard.press("Enter");
  expect(
    (
      await answerTo(page, "PATCH", path, () => page.getByRole("option", { name: "High", exact: true }).click())
    ).status()
  ).toBe(403);
  await expect(page.getByText("Your role does not allow this.", { exact: true })).toHaveCount(1);
  await expect(priority).toBeVisible();
  expect(await issueRead(api, seed.member.token, seed.row.id)).toMatchObject({
    status: 200,
    issue: { priority: "none" },
  });
  await expectListPageQuiet(page, watch, {
    failures: [`403 PATCH ${path}`],
    errors:
      browserName === "chromium" || browserName === "webkit"
        ? ["Failed to load resource: the server responded with a status of 403 (Forbidden)"]
        : [],
  });
});

test("C3: create form project and label panels keep the outer modal isolated and return focus without submitting", async ({
  api,
  page,
}, info) => {
  const seed = await controlsProject(api, info);
  const { watch } = await openIssueList(page, seed);
  const create = page.getByRole("button", { name: "Add work item", exact: true });
  await create.click();
  await expect(modalDialog(page)).toHaveCount(1);
  await transitionsEnded(modalDialog(page));
  await searchPanel(page, page.getByRole("button", { name: "Project: Web", exact: true }), "Web", true);
  await searchPanel(page, page.getByRole("button", { name: "Labels", exact: true }), "Visible label", true);
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);
  expect(await pageBehindHidden(page)).toBe(false);
  await expect(create).toBeFocused();
  expect(watch.apiRequests.filter((request) => request === `POST /api/v0/projects/${seed.project.id}/issues`)).toEqual(
    []
  );
  await expectListPageQuiet(page, watch);
});

test("C3: closing a loading peek returns keyboard focus to its still-connected trigger and a late read stays closed", async ({
  api,
  page,
}, info) => {
  const seed = await controlsProject(api, info);
  const { watch } = await openIssueList(page, seed);
  const trigger = page.locator(`#issue-${seed.row.id}`);
  const peek = page.locator("#full-screen-portal");
  const held = await sentHeld(page, "GET", `/api/v0/issues/${seed.row.id}`, () => trigger.click());
  try {
    const close = peek.getByRole("button", { name: "Close the peek view", exact: true });
    await expect(close).toBeVisible();
    await close.focus();
    await page.keyboard.press("Enter");
    await expect(close).toHaveCount(0);
    await expect(trigger).toBeFocused();
  } finally {
    const refreshed = page.waitForResponse(
      (response) => new URL(response.url()).pathname === `/api/v0/projects/${seed.project.id}/issues`
    );
    await held.release();
    expect(await (await refreshed).finished()).toBeNull();
  }
  await expect(peek.getByRole("button", { name: "Toggle peek view layout", exact: true })).toHaveCount(0);
  await expectListPageQuiet(page, watch);
});

test("C3: peek layout selects its current mode, anchors the list and leaves a retired row alone; detail labels keep one popup", async ({
  api,
  page,
}, info) => {
  const seed = await controlsProject(api, info);
  const { watch } = await openIssueList(page, seed);
  const trigger = page.locator(`#issue-${seed.row.id}`);
  const originalTrigger = await trigger.elementHandle();
  await trigger.click();
  const peek = page.locator("#full-screen-portal");
  const mode = peek.getByRole("button", { name: "Toggle peek view layout", exact: true });
  await expect(mode).toBeVisible();
  await mode.focus();
  await composingKey(mode, "Enter");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await page.keyboard.press("Enter");
  await expectListBesideButton(page, mode);
  await expect(page.getByRole("option", { name: "Side Peek", exact: true })).toHaveAttribute("aria-selected", "true");
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(mode).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("option", { name: "Modal", exact: true })).toHaveAttribute("aria-selected", "true");
  await page.keyboard.press("Escape");
  await transitionsEnded(peek);
  await searchPanel(page, peek.getByRole("button", { name: "Labels: Control row", exact: true }), "Visible label");
  const close = peek.getByRole("button", { name: "Close the peek view", exact: true });
  await close.focus();
  await page.keyboard.press("Enter");
  await expect(mode).toHaveCount(0);
  // Sub-issue reconciliation revalidates the parent and remounts the list; that old trigger no longer owns focus.
  expect(await originalTrigger!.evaluate((element) => element.isConnected)).toBe(false);
  await expect(trigger).not.toBeFocused();
  await expect(page.locator("body")).toBeFocused();
  await expectListPageQuiet(page, watch);
});

test("C3: settings selects and API-token calendar preserve keyboard focus, placement and outer modal isolation", async ({
  api,
  page,
}, info) => {
  const seed = await controlsProject(api, info);
  expect(await changeProject(api, seed.member.token, seed.project.id, { archive_in: 3 })).toMatchObject({
    status: 200,
  });
  const { watch } = await openIssueList(page, seed);
  await expect(page.locator(`#issue-${seed.row.id}`)).toBeVisible();
  await page.goto(`/${seed.slug}/settings/projects/${seed.project.id}/automations`);
  const months = page.getByRole("button", { name: "3 months", exact: true });
  await months.focus();
  await page.keyboard.press("Enter");
  await expectListBesideButton(page, months);
  await expect(page.getByRole("option", { name: "3 months", exact: true })).toHaveAttribute("aria-selected", "true");
  await page.keyboard.press("Escape");
  await expect(months).toBeFocused();

  await page.goto(`/${seed.slug}/projects`);
  await page.getByRole("button", { name: "Filters", exact: true }).click();
  await page.getByText("Custom", { exact: true }).click();
  await expect(modalDialog(page)).toHaveCount(1);
  await transitionsEnded(modalDialog(page));
  const filter = page.getByRole("button", { name: "Created date range", exact: true });
  await filter.focus();
  await page.keyboard.press("Enter");
  await expectListBesideButton(page, filter);
  expect(await pageBehindHidden(page)).toBe(true);
  await page.getByRole("option", { name: "Created date before", exact: true }).click();
  await expect(page.getByRole("button", { name: "Created date before", exact: true })).toBeFocused();
  expect(await pageBehindHidden(page)).toBe(true);
  await page.keyboard.press("Escape");
  await expect(modalDialog(page)).toHaveCount(0);

  await page.goto("/settings/profile/api-tokens");
  await page.getByRole("button", { name: "Add access token", exact: true }).first().click();
  await expect(modalDialog(page)).toHaveAccessibleName("Create token");
  await transitionsEnded(modalDialog(page));
  await page.getByRole("button", { name: "Set expiration date", exact: true }).click();
  await page.getByRole("option", { name: "Custom date", exact: true }).click();
  const day = page.getByRole("button", { name: "Set date", exact: true }).first();
  await day.focus();
  await page.keyboard.press("Enter");
  const calendar = page.getByRole("dialog", { name: "Set date", exact: true });
  await expectPopupBesideButton(calendar, day);
  expect(await pageBehindHidden(page)).toBe(true);
  await page.keyboard.press("Escape");
  await expect(calendar).toHaveCount(0);
  await expect(day).toBeFocused();
  await page.keyboard.press("Enter");
  await page.getByRole("grid").getByRole("button", { disabled: false }).last().click();
  await expect(calendar).toHaveCount(0);
  await expect(page.getByText(/^Expires .+ at /)).toBeVisible();
  expect(await pageBehindHidden(page)).toBe(true);
  expect(watch.apiRequests.filter((request) => request === "POST /api/v0/me/api-tokens")).toEqual([]);
  await expectListPageQuiet(page, watch);
});
