import type { Locator, Page } from "@playwright/test";
import { test } from "../../fixtures/filter-browser";
import { expect } from "../../fixtures/test";
import { bearer } from "../../fixtures/auth";
import { expectQuietConsole, requestsElsewhere, watchPage } from "../../fixtures/browser";
import { issueListProject, openIssueList, expectListPageQuiet } from "../../fixtures/issue-list-page";
import { addCondition, displayPanel } from "../../fixtures/issue-filters-page";
import { createIssue } from "../../fixtures/issues";
import { expectPopupBesideButton, transitionsEnded } from "../../fixtures/settings-pages";
import { modalDialog, pageBehindHidden } from "../../fixtures/dialogs";

async function searchAndClear(page: Page, search: string, noResults = "No options found") {
  const input = page.getByRole("combobox");
  await expect(input).toBeFocused();
  await input.fill("no-matching-filter-option");
  await expect(page.getByText(noResults, { exact: true })).toBeVisible();
  await input.fill("");
  await input.fill(search);
  const option = page.getByRole("option", { name: search, exact: true });
  await expect(option).toBeVisible();
  await page.keyboard.press("ArrowDown");
  await expect(input).toHaveAttribute("aria-activedescendant", (await option.getAttribute("id"))!);
  // Firefox fill emits compositionstart/end, including for Latin text. Headless UI deliberately holds its Enter guard through two animation frames after compositionend.
  await input.evaluate(
    () => new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
  );
}

async function besideSearch(page: Page, button: Locator) {
  const panel = page.locator("[data-popper-placement]").filter({ has: page.getByRole("combobox") });
  await expectPopupBesideButton(panel, button);
}

test("C2: actual add, multi-value and operator selectors support keyboard search, clearing and focus; date drafts expose their error", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  await createIssue(api, seed.member.token, seed.project.id, { name: "High row", priority: "high" });
  const { watch } = await openIssueList(page, seed);
  const add = page.getByRole("button", { name: "Add filter", exact: true });
  await add.focus();
  await page.keyboard.press("Enter");
  await besideSearch(page, add);
  await searchAndClear(page, "Priority");
  await page.keyboard.press("Enter");
  const chip = page.getByRole("button", { name: "Remove filter", exact: true }).last().locator("..");
  const value = chip.getByRole("button", { name: "Select an option", exact: true });
  await expect(page.getByRole("option", { name: "High", exact: true })).toBeVisible();
  await besideSearch(page, value);
  await searchAndClear(page, "High");
  await page.keyboard.press("Enter");
  await page.keyboard.press("Escape");
  const selected = chip.getByRole("button", { name: "High", exact: true });
  await expect(selected).toBeFocused();
  await page.keyboard.press("Space");
  await expect(page.getByRole("combobox")).toBeFocused();
  // On macOS Safari, Option-Tab includes buttons in native sequential navigation.
  await page.keyboard.press(browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  const date = await addCondition(page, "Target date");
  await expect(page.getByRole("combobox", { name: "Choose the Year" })).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("combobox", { name: "Choose the Year" })).toHaveCount(0);
  await expect(date.getByRole("button", { name: "--", exact: true })).toBeFocused();
  const operator = date.getByRole("button", { name: "is", exact: true });
  await operator.focus();
  await page.keyboard.press("Space");
  await besideSearch(page, operator);
  await searchAndClear(page, "between");
  await page.keyboard.press("Enter");
  await page.getByRole("combobox", { name: "Choose the Year" }).selectOption("2026");
  await page.getByRole("combobox", { name: "Choose the Month" }).selectOption("9");
  await page.locator('[data-day="2026-10-01"] button').click();
  await expect(date.locator('button[aria-invalid="true"]')).toHaveAccessibleDescription(
    "Choose both dates to apply the range"
  );
  await page.locator('[data-day="2026-10-31"] button').click();
  await expect(date.locator('button[aria-invalid="true"]')).toHaveCount(0);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expectListPageQuiet(page, watch);
});

test("C2: display panels anchor and return focus on desktop and mobile; mobile layout has one named clickable trigger", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  await createIssue(api, seed.member.token, seed.project.id, { name: "Display row" });
  const { watch } = await openIssueList(page, seed);
  await [1280, 390].reduce(async (previous, width) => {
    await previous;
    await page.setViewportSize({ width, height: 844 });
    const display = page.getByRole("button", { name: "Display", exact: true });
    await display.focus();
    await page.keyboard.press("Space");
    const panel = await displayPanel(page);
    await transitionsEnded(panel);
    await expect(display).toHaveAttribute("aria-expanded", "true");
    if (width === 1280) await expectPopupBesideButton(panel, display);
    else {
      // Popper can shift the wide panel sideways on a narrow screen; it must remain under its trigger and inside the viewport.
      const box = await panel.boundingBox(),
        trigger = await display.boundingBox();
      expect(box).not.toBeNull();
      expect(trigger).not.toBeNull();
      if (!box || !trigger) throw new Error("Missing display geometry");
      expect(Math.abs(box.y - trigger.y - trigger.height)).toBeLessThan(8);
      expect(box.x).toBeGreaterThanOrEqual(0);
      expect(box.x + box.width).toBeLessThanOrEqual(width);
      expect(trigger.x + trigger.width / 2).toBeGreaterThanOrEqual(box.x);
      expect(trigger.x + trigger.width / 2).toBeLessThanOrEqual(box.x + box.width);
    }
    const group = panel.getByRole("button", { name: "Group by", exact: true });
    if ((await group.getAttribute("aria-expanded")) !== "true") await group.click();
    await group.locator("../..").getByRole("button", { name: "Priority", exact: true }).click();
    await page.keyboard.press("Escape");
    await expect(panel).toHaveCount(0);
    await expect(display).toBeFocused();
  }, Promise.resolve());
  const layout = page.getByRole("button", { name: "Layout", exact: true });
  await expect(layout).toHaveCount(1);
  await expect(layout.locator("button")).toHaveCount(0);
  await layout.focus();
  await page.keyboard.press("Enter");
  await page.getByRole("menuitem", { name: "Board", exact: true }).click();
  await expect(layout).toHaveAttribute("aria-expanded", "false");
  await expectListPageQuiet(page, watch);
});

test("C9: an archive confirmation traps focus, cancels without a write and returns to its row; Back closes it", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const item = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Focus archive",
    state_id: seed.states.Done,
  });
  const { watch } = await openIssueList(page, seed);
  const options = page.getByRole("button", { name: "Options: Focus archive", exact: true });
  await options.focus();
  await page.keyboard.press("Enter");
  await page.getByRole("menuitem", { name: "Archive", exact: true }).focus();
  await page.keyboard.press("Enter");
  const dialog = modalDialog(page);
  await transitionsEnded(dialog);
  expect(await pageBehindHidden(page)).toBe(true);
  await dialog.getByRole("button", { name: "Archive", exact: true }).focus();
  await page.keyboard.press("Tab");
  await expect.poll(() => dialog.evaluate((element) => element.contains(document.activeElement))).toBe(true);
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  expect(await pageBehindHidden(page)).toBe(false);
  await expect(options).toBeFocused();
  await page
    .getByRole("complementary", { name: "Main sidebar", exact: true })
    .getByRole("link", { name: "Projects", exact: true })
    .click();
  await page.getByRole("heading", { name: "Web", exact: true }).click();
  await options.click();
  await page.getByRole("menuitem", { name: "Archive", exact: true }).click();
  await expect(dialog).toHaveCount(1);
  await page.goBack();
  await expect(page).toHaveURL(`/${seed.slug}/projects`);
  await expect(dialog).toHaveCount(0);
  expect(watch.apiRequests.filter((request) => request === `POST /api/v0/issues/${item.id}/archive`)).toEqual([]);
  await expectListPageQuiet(page, watch);
});

test("C2: a parent's real custom display and filter buttons anchor their panels and return focus inside the peek", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  const parent = await createIssue(api, seed.member.token, seed.project.id, { name: "Filter parent" });
  await createIssue(api, seed.member.token, seed.project.id, { name: "Filter child", parent_id: parent.id });
  const { watch } = await openIssueList(page, seed);
  await page.locator(`#issue-${parent.id}`).click();
  const peek = page.locator("#full-screen-portal");
  await expect(peek.getByText("Filter child", { exact: true })).toBeVisible();
  const toolbar = peek.getByRole("button", { name: "Display", exact: true }).locator("../..");
  await ["Display", "Filters"].reduce(async (previous, name) => {
    await previous;
    const trigger = toolbar.getByRole("button", { name, exact: true });
    await trigger.focus();
    await page.keyboard.press("Enter");
    await expect(trigger).toHaveAttribute("aria-expanded", "true");
    const id = await trigger.getAttribute("aria-controls");
    expect(id).toBeTruthy();
    const panel = page.locator(`[id="${id}"]`);
    await transitionsEnded(panel);
    await expectPopupBesideButton(panel, trigger);
    await expect(trigger.locator("button")).toHaveCount(0);
    if (name === "Filters") await panel.getByPlaceholder("Search").fill("priority");
    else await panel.getByRole("button", { name: "ID", exact: true }).click();
    await page.keyboard.press("Escape");
    await expect(panel).toHaveCount(0);
    await expect(trigger).toBeFocused();
    await expect(peek.getByText("Filter child", { exact: true })).toBeVisible();
  }, Promise.resolve());
  await expectListPageQuiet(page, watch);
});

for (const kind of ["single", "multi"] as const) {
  for (const language of ["en", "zh-CN"] as const) {
    test(`C2: browser compatibility fixture ${kind}/${language} opens during loading, searches, clears, selects and returns keyboard focus`, async ({
      page,
      filterBrowserURL,
    }) => {
      let release!: () => void;
      const held = new Promise<void>((resolve) => {
        release = resolve;
      });
      await page.route(`${filterBrowserURL}/options`, async (route) => {
        await held;
        await route.fulfill({ status: 200, body: "ready" });
      });
      const watch = await watchPage(page);
      await page.goto(`${filterBrowserURL}/?kind=${kind}&language=${language}`);
      try {
        await expect(
          page.getByText(language === "en" ? "Loading options…" : "正在加载选项…", { exact: true })
        ).toBeVisible();
        await expect(page.getByRole("combobox")).toBeFocused();
      } finally {
        release();
      }
      const ready = language === "en" ? "Ready" : "已就绪";
      await expect(page.getByRole("option", { name: ready, exact: true })).toBeVisible();
      const value = page.getByRole("region", { name: "Value control" }).getByRole("button");
      await besideSearch(page, value);
      await expect(page.getByRole("combobox")).toHaveAttribute("placeholder", language === "en" ? "Search" : "搜索");
      await searchAndClear(page, ready, language === "en" ? "No options found" : "未找到选项");
      await page.keyboard.press("Enter");
      await expect(page.getByRole("status", { name: "Selected value" })).toHaveText(
        kind === "single" ? '"ready"' : '["ready"]'
      );
      if (kind === "multi") await page.keyboard.press("Escape");
      await expect(page.getByRole("listbox")).toHaveCount(0);
      await expect(value).toBeFocused();
      await page.keyboard.press("Space");
      await expect(page.getByRole("combobox")).toHaveValue("");
      await page.keyboard.press("Escape");
      await expect(value).toBeFocused();
      expect(watch.pageErrors).toEqual([]);
      expect(watch.apiFailures).toEqual([]);
      expect(watch.oldApiRequests).toEqual([]);
      expect(watch.cspViolations).toEqual([]);
      expect(requestsElsewhere(page, watch)).toEqual([]);
      await expectQuietConsole(page, watch);
    });
  }
}

test("C2: Chinese project filters translate condition names, built-in choices and operators", async ({
  api,
  page,
}, info) => {
  const seed = await issueListProject(api, info);
  await createIssue(api, seed.member.token, seed.project.id, { name: "Chinese row", priority: "high" });
  expect(
    (await api.PATCH("/api/v0/me/profile", { headers: bearer(seed.member.token), body: { language: "zh-CN" } }))
      .response.status
  ).toBe(200);
  const { watch } = await openIssueList(page, seed);
  await page.getByRole("button", { name: "添加筛选", exact: true }).click();
  await expect(page.getByRole("option", { name: "优先级", exact: true })).toBeVisible();
  await page.getByRole("option", { name: "优先级", exact: true }).click();
  await page.getByRole("option", { name: "高", exact: true }).click();
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "添加筛选", exact: true }).click();
  await page.getByRole("option", { name: "目标日期", exact: true }).click();
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "是", exact: true }).last().click();
  await expect(page.getByRole("option", { name: "介于", exact: true })).toBeVisible();
  await page.keyboard.press("Escape");
  await expectListPageQuiet(page, watch);
});
