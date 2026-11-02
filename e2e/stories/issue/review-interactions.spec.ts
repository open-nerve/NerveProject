import type { Locator, Page } from "@playwright/test";
import {
  addProjectMembers,
  changeProject,
  changeRole,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  readProjectPreferences,
} from "../../fixtures/api";
import { accountId, emailFor } from "../../fixtures/auth";
import { c12Evidence } from "../../fixtures/c12-evidence";
import { modalDialog } from "../../fixtures/dialogs";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import {
  expectListOrder,
  expectListPageQuiet,
  issueListProject,
  openIssueList,
  saveListPreferences,
} from "../../fixtures/issue-list-page";
import { createIssue, issueArchived, issueRead, updateIssue } from "../../fixtures/issues";
import { selectCommand } from "../../fixtures/power-k";
import { dragOnto } from "../../fixtures/project-pages";
import { answerTo, moveWithinApp, registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

async function keyboardFocus(page: Page, control: Locator, browserName: string) {
  const tab = browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab";
  await control.focus();
  await page.keyboard.press(tab);
  await page.keyboard.press(`Shift+${tab}`);
  await expect(control).toBeFocused();
  await expect(control).toHaveCSS("outline-style", /^(auto|solid)$/);
}

for (const sign of [1, -1]) {
  test(`C12 (review): board drag retains a finite midpoint between extreme neighbors, sign=${sign}`, async ({
    api,
    db,
    page,
  }, info) => {
    const seed = await issueListProject(api, info);
    const { Done: done, Todo: todo } = seed.states;
    const left = await createIssue(api, seed.member.token, seed.project.id, { name: "Extreme left", state_id: done });
    const right = await createIssue(api, seed.member.token, seed.project.id, { name: "Extreme right", state_id: done });
    const moving = await createIssue(api, seed.member.token, seed.project.id, {
      name: "Move between extremes",
      state_id: todo,
    });
    await updateIssue(api, seed.member.token, left.id, { sort_order: sign > 0 ? 1e308 : -1.6e308 });
    await updateIssue(api, seed.member.token, right.id, { sort_order: sign > 0 ? 1.6e308 : -1e308 });
    const prefs = await readProjectPreferences(api, seed.member.token, seed.project.id);
    await saveListPreferences(api, seed, {
      display_filters: {
        ...prefs.display_filters,
        layout: "kanban",
        group_by: "state_id",
        sub_group_by: null,
        order_by: "sort_order",
        show_empty_groups: true,
      },
    });
    const { watch } = await openIssueList(page, seed);
    const column = page.locator(`[id^="${done}__"]`);
    const card = (id: string) => page.locator(`[id^="issue_${id}_"]`);
    await expectListOrder(column, [left.id, right.id]);
    const held = await sentHeld(page, "PATCH", `/api/v0/issues/${moving.id}`, () =>
      dragOnto(page, card(moving.id), card(moving.id), card(right.id), { x: 100, y: 4 })
    );
    expect(held.body).toEqual({ state_id: done, sort_order: sign * 1.3e308 });
    expect((await held.release()).status()).toBe(200);
    await expectListOrder(column, [left.id, moving.id, right.id]);
    expect(await db.query("SELECT sort_order FROM issues WHERE id=$1", [moving.id])).toEqual([
      { sort_order: sign * 1.3e308 },
    ]);
    await c12Evidence(page, "B02", `review-extreme-midpoint-${sign}`);
    await expectListPageQuiet(page, watch);
  });
}

for (const layout of ["list", "kanban"] as const) {
  test(`C12 (review): ${layout} group controls fold, reopen and create through native keyboard focus`, async ({
    api,
    page,
    browserName,
  }, info) => {
    const seed = await issueListProject(api, info);
    const issue = await createIssue(api, seed.member.token, seed.project.id, {
      name: "Keyboard group row",
      state_id: seed.states.Todo,
    });
    const prefs = await readProjectPreferences(api, seed.member.token, seed.project.id);
    await saveListPreferences(api, seed, {
      display_filters: { ...prefs.display_filters, layout, group_by: "state_id", sub_group_by: null },
    });
    const { watch } = await openIssueList(page, seed);
    const fold = page.getByRole("button", { name: "Todo", exact: true });
    const card = page.locator(`[id^="issue_${issue.id}_"]`);
    await expect(card).toBeVisible();
    await keyboardFocus(page, fold, browserName);
    await c12Evidence(page, layout === "list" ? "B01" : "B02", `review-${layout}-fold-focus`);
    await page.keyboard.press("Enter");
    await expect(fold).toHaveAttribute("aria-expanded", "false");
    await expect(card).toHaveCount(0);
    await page.keyboard.press("Space");
    await expect(fold).toHaveAttribute("aria-expanded", "true");
    await expect(card).toBeVisible();
    const create = page.getByRole("button", { name: "Create work item: Todo", exact: true });
    await keyboardFocus(page, create, browserName);
    await c12Evidence(page, layout === "list" ? "B01" : "B02", `review-${layout}-create-focus`);
    await page.keyboard.press("Enter");
    const form = page.getByRole("dialog", { name: "Create new work item", exact: true });
    await expect(form.getByRole("textbox", { name: "Title", exact: true })).toBeVisible();
    await expect(form.getByRole("button", { name: "State: Todo", exact: true })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(form).toHaveCount(0);
    await expect(create).toBeFocused();
    expect(watch.apiRequests.filter((request) => /^(POST|PATCH|DELETE) \/api\/v0\/issues\//.test(request))).toEqual([]);
    await expectListPageQuiet(page, watch);
  });
}

test("C12 (review): changing the first weekday retains a navigated week outside the active month", async ({
  api,
  page,
}, info) => {
  await page.clock.setFixedTime(new Date("2026-12-31T12:00:00+08:00"));
  const seed = await issueListProject(api, info);
  const january = await createIssue(api, seed.member.token, seed.project.id, {
    name: "January week",
    target_date: "2027-01-07",
  });
  const prefs = await readProjectPreferences(api, seed.member.token, seed.project.id);
  await saveListPreferences(api, seed, {
    display_filters: {
      ...prefs.display_filters,
      layout: "calendar",
      calendar: { layout: "week", show_weekends: true },
    },
  });
  const { watch, queries } = await openIssueList(page, seed);
  await expect.poll(() => queries.at(-1)?.get("target_date")).toBe("2026-12-27..2027-01-02");
  await page.getByRole("button", { name: "Next period", exact: true }).click();
  await expect(page.locator(`#issue-${january.id}`)).toBeVisible();
  await expect.poll(() => queries.at(-1)?.get("target_date")).toBe("2027-01-03..2027-01-09");
  await page.keyboard.press("ControlOrMeta+k");
  await selectCommand(page, /^Change first day of week/);
  const held = await sentHeld(page, "PATCH", "/api/v0/me/profile", () =>
    modalDialog(page).getByRole("option", { name: "Monday", exact: true }).click()
  );
  expect(held.body).toEqual({ start_of_the_week: 1 });
  expect(queries.at(-1)?.get("target_date")).toBe("2027-01-03..2027-01-09");
  expect((await held.release()).status()).toBe(200);
  await expect.poll(() => queries.at(-1)?.get("target_date")).toBe("2027-01-04..2027-01-10");
  await expect(page.locator(`#issue-${january.id}`)).toBeVisible();
  await c12Evidence(page, "B03", "review-weekday-change-january");
  await expectListPageQuiet(page, watch);
});

test("C12 (review): a guest clears personal empty-result filters but cannot create in a swimlane header", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const email = emailFor(info, "filter-guest");
  const tokens = await registerOnboarded(api, email);
  const guest = { email, token: tokens.access_token, id: await accountId(api, tokens.access_token) };
  await inviteAndAccept(api, seed.member.token, seed.slug, guest, 15);
  await addProjectMembers(api, seed.member.token, seed.project.id, [{ member_id: guest.id, role: 15 }]);
  const own = await createIssue(api, guest.token, seed.project.id, {
    name: "Guest visible row",
    priority: "none",
    state_id: seed.states.Todo,
  });
  expect(
    (await changeProject(api, seed.member.token, seed.project.id, { guest_view_all_features: false })).status
  ).toBe(200);
  expect(
    await projectMemberWrites(api, seed.project.id).change(
      seed.member.token,
      await projectMembershipOf(api, seed.member.token, seed.project.id, guest.id),
      5
    )
  ).toEqual({ status: 200 });
  await changeRole(api, seed.member.token, seed.slug, guest.id, 5);
  const guestSeed = { ...seed, member: guest };
  const prefs = await readProjectPreferences(api, guest.token, seed.project.id);
  await saveListPreferences(api, guestSeed, {
    filters: { priority: ["urgent"] },
    display_filters: {
      ...prefs.display_filters,
      layout: "kanban",
      group_by: "state_id",
      sub_group_by: "priority",
      show_empty_groups: true,
    },
  });
  const { watch } = await openIssueList(page, guestSeed);
  const clear = page.getByRole("button", { name: "Clear all filters", exact: true });
  await expect(clear).toBeEnabled();
  await c12Evidence(page, "B07", "review-guest-empty-filter");
  expect(
    (await answerTo(page, "PATCH", `/api/v0/me/projects/${seed.project.id}/preferences`, () => clear.click())).status()
  ).toBe(200);
  await expect(page.locator(`[id^="issue_${own.id}_"]`)).toBeVisible();
  await expect(page.getByRole("button", { name: /^Create work item:/ })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add work item", exact: true })).toHaveCount(0);
  expect((await readProjectPreferences(api, guest.token, seed.project.id)).filters).toEqual({});
  await c12Evidence(page, "B02", "review-guest-swimlane-read-only");
  await expectListPageQuiet(page, watch);
});

test("C12 (review): a hot ID entry waits for its new read and a retired entry cannot navigate on the late answer", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, { name: "Fresh ID read" });
  const { watch } = await openIssueList(page, seed);
  const idPath = `${seed.path}/${issue.id}`;
  const numberPath = `/${seed.slug}/browse/WEB-${issue.sequence_id}`;
  await moveWithinApp(page, idPath);
  await expect(page).toHaveURL(numberPath);
  await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(issue.name);
  await moveWithinApp(page, seed.path);
  await expect(page.locator(`#issue-${issue.id}`)).toBeVisible();
  const fresh = await sentHeld(page, "GET", `/api/v0/issues/${issue.id}`, () => moveWithinApp(page, idPath));
  await c12Evidence(page, "B10", "review-hot-id-read-pending");
  await expect(page).toHaveURL(idPath);
  expect((await fresh.release()).status()).toBe(200);
  await expect(page).toHaveURL(numberPath);
  await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(issue.name);
  await moveWithinApp(page, seed.path);
  // The history helper changes the URL before React commits the destination. Retire
  // this entry through the real list before testing a separate entry's late answer.
  await expect(page.locator(`#issue-${issue.id}`)).toBeVisible();
  const late = await sentHeld(page, "GET", `/api/v0/issues/${issue.id}`, () => moveWithinApp(page, idPath));
  await moveWithinApp(page, `/${seed.slug}/projects`);
  await expect(page.getByRole("button", { name: "Add Project", exact: true })).toBeVisible();
  expect((await late.release()).status()).toBe(200);
  await expect(page).toHaveURL(`/${seed.slug}/projects`);
  await c12Evidence(page, "B10", "review-retired-id-stays-away");
  await expectListPageQuiet(page, watch);
});

test("C12 (review): a cold sibling menu survives parent verification and an archived detail offers no parent removal", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const parent = await createIssue(api, seed.member.token, seed.project.id, { name: "Parent facts" });
  const child = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Archived child",
    parent_id: parent.id,
    state_id: seed.states.Done,
  });
  const sibling = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Sibling target",
    parent_id: parent.id,
  });
  expect((await issueArchived(api, seed.member.token, child.id)).status).toBe(200);
  const watch = await openIssueDetail(page, seed, child, child.name);
  const trigger = page.getByRole("button", { name: "Sibling work items", exact: true });
  const held = await sentHeld(page, "GET", `/api/v0/issues/${parent.id}`, () => trigger.click());
  await expect(trigger).toBeVisible();
  await expect(page.getByRole("menu")).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "Remove parent work item", exact: true })).toHaveCount(0);
  expect((await held.release()).status()).toBe(200);
  const target = page.getByRole("menuitem", { name: `WEB-${sibling.sequence_id}`, exact: true });
  await expect(target).toBeVisible();
  await expect(page.getByRole("menuitem", { name: "Remove parent work item", exact: true })).toHaveCount(0);
  await c12Evidence(page, "B12", "review-archived-siblings");
  const opened = page.waitForEvent("popup");
  await target.click();
  const siblingPage = await opened;
  await expect(siblingPage).toHaveURL(`/${seed.slug}/browse/WEB-${sibling.sequence_id}`);
  await expect(siblingPage.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(sibling.name);
  await siblingPage.close();
  expect(await issueRead(api, seed.member.token, child.id)).toMatchObject({ issue: { parent_id: parent.id } });
  await expectListPageQuiet(page, watch);
});
