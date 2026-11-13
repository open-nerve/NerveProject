import { c12Evidence } from "../../fixtures/c12-evidence";
import { bearer } from "../../fixtures/auth";
import { focusOutline } from "../../fixtures/browser";
import type { Locator } from "@playwright/test";
import { createDraft, createIssue } from "../../fixtures/issues";
import { openIssueDetail } from "../../fixtures/issue-detail-page";
import { expectListPageQuiet, issueListProject, openIssueList } from "../../fixtures/issue-list-page";
import { composingKey, modalDialog, pageBehindHidden } from "../../fixtures/dialogs";
import { answerTo, moveWithinApp } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

/** Check the actual focus indicator against the viewport and each clipping ancestor (focusOutline). */
async function visibleFocus(control: Locator) {
  await expect(control).toBeFocused();
  await expect(control).toHaveCSS("opacity", "1");
  const focus = await focusOutline(control);
  expect(focus.width).toBeGreaterThanOrEqual(2);
  expect(focus.style).not.toBe("none");
  expect(focus.clippedBy).toEqual([]);
}

test("C12 (cleanup): continue writing is in the native Tab order, keeps a visible focus outline and appends after the final table", async ({
  api,
  db,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "Continue after table",
    description_html: "<table><tbody><tr><td><p>Last cell</p></td></tr></tbody></table>",
  });
  const path = `/api/v0/issues/${issue.id}`;
  const migrated = page.waitForResponse(
    (response) => response.request().method() === "PATCH" && new URL(response.url()).pathname === path
  );
  const watch = await openIssueDetail(page, seed, issue);
  const container = page.locator(`[id="editor-container-${issue.id}"]`);
  const editor = container.locator('[contenteditable="true"]');
  await expect(editor).toContainText("Last cell");
  expect((await migrated).status()).toBe(200);
  const button = container.getByRole("button", { name: "Continue writing", exact: true });
  const tab = browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab";
  await button.focus();
  // Leave toward the preceding control: Shift-Tab inside a table intentionally navigates its cells.
  await page.keyboard.press(`Shift+${tab}`);
  await page.keyboard.press(tab);
  await visibleFocus(button);
  await c12Evidence(page, "B14", "continue-writing-tab");
  await page.keyboard.press(tab);
  await expect(editor).toBeFocused();
  await button.focus();
  expect(
    (
      await answerTo(page, "PATCH", path, async () => {
        await page.keyboard.press("Enter");
        await expect(editor).toBeFocused();
        await page.keyboard.insertText("After table");
      })
    ).status()
  ).toBe(200);
  await expect(editor.locator(":scope > p").last()).toHaveText("After table");
  await c12Evidence(page, "B14", "written-after-final-table");
  expect(await db.query("SELECT description_stripped FROM issues WHERE id=$1", [issue.id])).toEqual([
    { description_stripped: "Last cellAfter table" },
  ]);
  await expectListPageQuiet(page, watch);
});

for (const kind of ["row", "column"] as const) {
  test(`C12 (cleanup): a table ${kind} menu closes before its draft dialog or peek and leaves no stale keyboard guard`, async ({
    api,
    page,
    browserName,
  }, info) => {
    const seed = await issueListProject(api, info);
    const description = "<table><tbody><tr><td><p>Nested cell</p></td></tr></tbody></table>";
    const draft = await createDraft(api, seed.member.token, seed.slug, {
      payload: { name: "Nested table", description_html: description },
    });
    const issue = await createIssue(api, seed.member.token, seed.project.id, {
      name: "Peek table",
      description_html: description,
    });
    const { watch } = await openIssueList(page, seed);
    await moveWithinApp(page, `/${seed.slug}/drafts`);
    await page.locator(`[id="issue-${draft.id}"] button[aria-haspopup="menu"]:visible`).last().click();
    await page.getByRole("menuitem", { name: "Make a copy", exact: true }).click();
    const form = page.getByRole("dialog", { name: "Create a draft", exact: true });
    const name = kind === "row" ? "Table row options" : "Table column options";
    const panel = page.getByRole("dialog", { name, exact: true });
    const first = panel.getByRole("button", { name: kind === "row" ? "Header row" : "Header column", exact: true });
    await form.locator('[contenteditable="true"]').getByText("Nested cell", { exact: true }).click();
    const formTrigger = form.getByRole("button", { name, exact: true }).first();
    const tab = browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab";
    // Return via the neighbouring control; crossing the editable table uses its own cell-navigation keys.
    const away = kind === "row" ? `Shift+${tab}` : tab;
    const back = kind === "row" ? tab : `Shift+${tab}`;
    await formTrigger.focus();
    await page.keyboard.press(away);
    await page.keyboard.press(back);
    await visibleFocus(formTrigger);
    await page.keyboard.press("Enter");
    await expect(first).toBeFocused();
    await expect(first).toHaveCSS("outline-style", /^(auto|solid)$/);
    expect(await pageBehindHidden(page)).toBe(true);
    await c12Evidence(page, "B15", `draft-${kind}-menu`);
    await page.keyboard.press("Escape");
    await expect(panel).toHaveCount(0);
    await expect(form.getByRole("heading", { name: "Create a draft", exact: true })).toBeVisible();
    await c12Evidence(page, "B15", `draft-${kind}-returned-focus`);
    await expect(formTrigger).toBeFocused();
    expect(await pageBehindHidden(page)).toBe(true);
    await page.keyboard.press("Escape");
    await expect(modalDialog(page)).toHaveCount(0);
    expect(await pageBehindHidden(page)).toBe(false);
    expect(watch.apiRequests.filter((request) => /^(POST|PATCH|DELETE) .*drafts/.test(request))).toEqual([]);
    await moveWithinApp(page, seed.path);
    const row = page.locator(`#issue-${issue.id}`);
    await row.click();
    const peek = page.locator("#full-screen-portal");
    await peek.locator('[contenteditable="true"]').getByText("Nested cell", { exact: true }).click();
    const peekTrigger = peek.getByRole("button", { name, exact: true }).first();
    await peekTrigger.focus();
    await page.keyboard.press(away);
    await page.keyboard.press(back);
    await visibleFocus(peekTrigger);
    await page.keyboard.press("Enter");
    await expect(first).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(panel).toHaveCount(0);
    await expect(peekTrigger).toBeFocused();
    await c12Evidence(page, "B15", `peek-${kind}-returned-focus`);
    await expect(peek.getByRole("button", { name: "Close the peek view", exact: true })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(peek.getByRole("button", { name: "Close the peek view", exact: true })).toHaveCount(0);
    // The list may have remounted during sub-issue reconciliation; its retired row is not a focus owner.
    await expect(panel).toHaveCount(0);
    await expectListPageQuiet(page, watch);
  });

  test(`C12 (cleanup): table ${kind} menus keep keyboard focus, edit the selected ${kind} and retire on Back`, async ({
    api,
    page,
    db,
    browserName,
  }, info) => {
    const seed = await issueListProject(api, info);
    const issue = await createIssue(api, seed.member.token, seed.project.id, {
      name: "Keyboard table",
      description_html:
        "<table><tbody><tr><td><p>First</p></td><td><p>Second</p></td></tr><tr><td><p>Third</p></td><td><p>Fourth</p></td></tr></tbody></table>",
    });
    const path = `/api/v0/issues/${issue.id}`;
    const migrated = page.waitForResponse(
      (response) => response.request().method() === "PATCH" && new URL(response.url()).pathname === path
    );
    const watch = await openIssueDetail(page, seed, issue);
    expect((await migrated).status()).toBe(200);
    const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
    await expect(editor.getByText("First", { exact: true })).toBeVisible();
    // Enter keyboard modality from the title; the initial selection is already in the first table.
    await page.getByRole("textbox", { name: "Work item title", exact: true }).focus();
    await page.keyboard.press("Shift+Tab");
    const trigger = editor
      .getByRole("button", { name: kind === "row" ? "Table row options" : "Table column options", exact: true })
      .first();
    await page.mouse.move(0, 0);
    await trigger.focus();
    const tab = browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab";
    await page.keyboard.press(tab);
    await page.keyboard.press(`Shift+${tab}`);
    await visibleFocus(trigger);
    await c12Evidence(page, "B15", `${kind}-trigger-tab`);
    await page.keyboard.press("Enter");
    const panel = page.getByRole("dialog", {
      name: kind === "row" ? "Table row options" : "Table column options",
      exact: true,
    });
    const first = panel.getByRole("button", { name: kind === "row" ? "Header row" : "Header column", exact: true });
    await expect(first).toBeFocused();
    await composingKey(first, "Escape");
    await expect(panel).toBeVisible();
    await page.keyboard.press(tab);
    await expect(first).not.toBeFocused();
    await expect.poll(() => panel.evaluate((element) => element.contains(document.activeElement))).toBe(true);
    await c12Evidence(page, "B15", `${kind}-menu-tab`);
    await page.keyboard.press("Escape");
    await expect(panel).toHaveCount(0);
    await expect(trigger).toBeFocused();
    const second = editor
      .getByRole("button", { name: kind === "row" ? "Table row options" : "Table column options", exact: true })
      .nth(1);
    expect(
      (
        await answerTo(page, "PATCH", path, async () => {
          await second.focus();
          await page.keyboard.press("Enter");
          await expect(first).toBeFocused();
          await panel
            .getByRole("button", { name: kind === "row" ? "Insert above" : "Insert left", exact: true })
            .focus();
          await page.keyboard.press("Enter");
        })
      ).status()
    ).toBe(200);
    await expect(editor).toBeFocused();
    await expect(editor.locator("tr")).toHaveCount(kind === "row" ? 3 : 2);
    await expect(editor.locator("td")).toHaveCount(6);
    await c12Evidence(page, "B15", `${kind}-second-target-inserted`);
    await expect(editor.locator("tr").first().locator("td").first()).toHaveText("First");
    expect(
      await db.query(
        "SELECT regexp_count(description_html, '<tr') AS rows, regexp_count(description_html, '<td') AS cells FROM issues WHERE id=$1",
        [issue.id]
      )
    ).toEqual([{ rows: kind === "row" ? 3 : 2, cells: 6 }]);
    const detailPath = new URL(page.url()).pathname;
    await moveWithinApp(page, seed.path);
    await expect(page.locator(`#issue-${issue.id}`)).toBeVisible();
    await moveWithinApp(page, detailPath);
    // Handles belong to the currently selected table; select a cell in this newly mounted editor.
    await editor.getByText("Fourth", { exact: true }).click();
    await trigger.focus();
    await page.keyboard.press("Enter");
    await expect(panel).toBeVisible();
    const beforeBack = watch.apiRequests.length;
    await page.goBack();
    await expect(page).toHaveURL(seed.path);
    await expect(panel).toHaveCount(0);
    expect(watch.apiRequests.slice(beforeBack).filter((request) => /^(POST|PATCH|DELETE) /.test(request))).toEqual([]);
    await expectListPageQuiet(page, watch);
  });
}

test("C12 (cleanup): Chinese editor names identify the native table menus and continue-writing focus target", async ({
  api,
  page,
  browserName,
}, info) => {
  const seed = await issueListProject(api, info);
  const issue = await createIssue(api, seed.member.token, seed.project.id, {
    name: "中文表格",
    description_html: "<table><tbody><tr><td><p>末尾单元格</p></td></tr></tbody></table>",
  });
  expect(
    (
      await api.PATCH("/api/v0/me/profile", {
        headers: bearer(seed.member.token),
        body: { language: "zh-CN" },
      })
    ).response.status
  ).toBe(200);
  const { watch } = await openIssueList(page, seed);
  await moveWithinApp(page, `/${seed.slug}/browse/WEB-${issue.sequence_id}`);
  const editor = page.locator(`[id="editor-container-${issue.id}"] [contenteditable="true"]`);
  await expect(editor).toContainText("末尾单元格");
  const tab = browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab";
  await editor.getByText("末尾单元格", { exact: true }).click();
  const checkMenu = async (kind: "row" | "column") => {
    const name = kind === "row" ? "表格行选项" : "表格列选项";
    const trigger = editor.getByRole("button", { name, exact: true }).first();
    await trigger.focus();
    await page.keyboard.press(kind === "row" ? `Shift+${tab}` : tab);
    await page.keyboard.press(kind === "row" ? tab : `Shift+${tab}`);
    await visibleFocus(trigger);
    await page.keyboard.press("Enter");
    const panel = page.getByRole("dialog", { name, exact: true });
    await expect(panel).toBeVisible();
    await page.keyboard.press(tab);
    await expect.poll(() => panel.evaluate((element) => element.contains(document.activeElement))).toBe(true);
    await c12Evidence(page, "B15", `chinese-${kind}-menu`);
    await page.keyboard.press("Escape");
    await expect(panel).toHaveCount(0);
    await visibleFocus(trigger);
  };
  await checkMenu("row");
  await checkMenu("column");
  const continueWriting = page
    .locator(`[id="editor-container-${issue.id}"]`)
    .getByRole("button", { name: "继续输入", exact: true });
  await continueWriting.focus();
  await page.keyboard.press(`Shift+${tab}`);
  await page.keyboard.press(tab);
  await visibleFocus(continueWriting);
  await c12Evidence(page, "B14", "chinese-continue-writing");
  await expectListPageQuiet(page, watch);
});
