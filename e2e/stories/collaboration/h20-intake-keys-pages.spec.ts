import type { Locator, Page } from "@playwright/test";

import { expectQuietConsole, refuseClipboardWrites, watchPage } from "../../fixtures/browser";
import { failedLoads, recordApi, tabKey, toast } from "../../fixtures/cycle-pages";
import {
  closedByBackdropClick,
  closedByEscape,
  enabledWithin,
  focusInDialog,
  modalDialog,
  pageBehindHidden,
  shownWithin,
} from "../../fixtures/dialogs";
import { expectLanded, focusedName, optionsOf, tabTo } from "../../fixtures/intake-focus";
import {
  expectHeaderOf,
  expectTabs,
  filterOption,
  identifierOf,
  intakeDescription,
  intakeDetail,
  intakeEditPath,
  intakeListRead,
  intakeListReads,
  intakePath,
  intakeRow,
  intakeWorld,
  rowName,
  selecting,
  shownRows,
  submitNamed,
  triagePath,
} from "../../fixtures/intake-pages";
import { chooseByKeyboard } from "../../fixtures/menus";
import { expectPopupBesideButton, holdAnswer, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H20's page version for the intake page's keys and names (M7 design 2, H20; P5a spec 2.6–2.8, 2.12, R5, R14, R35;
// M3 closeout 48): the tabs say their total and which is chosen, the arrow keys choosing; the filters are a named
// button that says it is expanded, its panel beside it, Tab going through its search and its options, each saying
// whether it is pressed, Enter and Space pressing one once, Escape closing it back onto its button; the order and the
// header's actions are menus of the keyboard, Escape closing each back onto its button; ↑ and ↓ step through the list
// only while the focus is outside the description's editor. The accept, decline and delete modals are named, hold the
// focus, the page behind them inert, and Escape closes each back onto the control that opened it; sent by Enter,
// Space or a press, nerve's answer held, nothing closes them, their cancel out of use, the focus within; once nerve
// answers, the focus goes with the selection, to the next row, or to the list once the selection is cleared. A link
// copied with the clipboard refused says the copy failed. The snooze modal and the duplicate picker, the focus as a
// triage ends by each of the three paths, are h20-intake-pages.spec.ts's; late answers,
// h20-intake-late-pages.spec.ts's.

/**
 * Presses Enter on control, which has the focus, and checks the modal it opens: named name, the focus within it, the
 * page behind it inert. Escape then closes it, back on control.
 */
async function openedAndEscaped(page: Page, control: Locator, name: string): Promise<void> {
  await control.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("dialog", { name, exact: true })).toHaveCount(1);
  expect([await focusInDialog(page), await pageBehindHidden(page)]).toEqual([true, true]);
  expect(await closedByEscape(page)).toBe(true);
  await expectLanded(control);
}

/** Checks that the one modal open stays while its request is out: no Escape, no click outside, cancel out of use. */
async function expectLocked(page: Page, cancel: Locator): Promise<void> {
  expect(await closedByEscape(page)).toBe(false);
  expect(await closedByBackdropClick(page)).toBe(false);
  expect(await enabledWithin(cancel)).toBe(false);
  expect(await focusInDialog(page)).toBe(true);
}

test("H20 (page): alice works the tabs and the filters of Web's intake page by the keyboard: the tabs say their total and which is chosen, the arrow keys choosing; the filters are a named button saying whether it is expanded, its panel beside it, Tab going through its search and its options, each saying whether it is pressed, Enter and Space pressing one once, Escape closing it back onto its button", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [slow, crash] = await submitNamed(api, bob.token, web.id, ["Slow", "Crash"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([crash, slow].map(rowName));
  await expect(page).toHaveURL(selecting(crash));
  const key = tabKey(browserName);

  // The tabs: the open one says its total and is chosen; ArrowRight chooses the closed one, ArrowLeft the open again.
  const [open, closed] = [
    page.getByRole("tab", { name: "Open, 2 work items", exact: true }),
    page.getByRole("tab", { name: "Closed", exact: true }),
  ];
  await expectTabs(page, "Open, 2 work items", "open");
  await open.focus();
  await page.keyboard.press("ArrowRight");
  await expect(closed).toBeFocused();
  await expectTabs(page, "Open", "closed");
  await expect(page).toHaveURL(/[?&]currentTab=closed(&|$)/);
  await page.keyboard.press("ArrowLeft");
  await expect(open).toBeFocused();
  await expectTabs(page, "Open, 2 work items", "open");
  await expect(page).toHaveURL(selecting(crash));

  // The filters: Enter opens the panel beside its button, which says so; Tab reaches its search, then Status, whose
  // options say whether each is pressed; Enter presses Snoozed once (one read of the list), Space lets it go once;
  // Escape closes the panel, back on its button.
  const filters = page.getByLabel("Filters", { exact: true }).and(page.getByRole("button"));
  await expect(filters).toHaveAttribute("aria-expanded", "false");
  await filters.focus();
  await page.keyboard.press("Enter");
  await expect(filters).toHaveAttribute("aria-expanded", "true");
  const panel = page.locator(`[id="${await filters.getAttribute("aria-controls")}"]`);
  await expectPopupBesideButton(panel, filters);
  await tabTo(page, key, panel.getByRole("textbox", { name: "Search", exact: true }), 1);
  // Firefox stops at the panel's scrolling list before its first control
  await tabTo(page, key, panel.getByRole("button", { name: "Status (1)", exact: true }), 2);
  const [pending, snoozed] = [filterOption(panel, "Status", "Pending"), filterOption(panel, "Status", "Snoozed")];
  await tabTo(page, key, pending, 1);
  await expect(pending).toHaveAttribute("aria-pressed", "true");
  await tabTo(page, key, snoozed, 1);
  await expect(snoozed).toHaveAttribute("aria-pressed", "false");
  const reads = recordApi(page);
  await page.keyboard.press("Enter");
  await expect(snoozed).toHaveAttribute("aria-pressed", "true");
  await expect
    .poll(() => intakeListReads(reads().requests))
    .toEqual([intakeListRead(web.id, "status=pending,snoozed&order_by=-created_at&limit=50")]);
  await page.keyboard.press(" ");
  await expect(snoozed).toHaveAttribute("aria-pressed", "false");
  await page.keyboard.press("Escape");
  await expect(filters).toHaveAttribute("aria-expanded", "false");
  await expectLanded(filters);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice works the order, the header's menu, ↑ and ↓, and the accept, decline and delete modals of Web's intake page by the keyboard: each menu beside its button, closed by Escape back onto it; ↑ and ↓ step through the list outside the description's editor only; each modal named, holding the focus, the page behind it inert, closed by Escape back onto the control that opened it", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [slow, crash, spam] = await submitNamed(api, bob.token, web.id, ["Slow", "Crash", "Spam"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([spam, crash, slow].map(rowName));
  await expect(page).toHaveURL(selecting(spam));

  // The order: Enter opens its menu beside its button; Escape closes it, back on its button; Ascending chosen by the
  // keyboard, the button says so and the list is the oldest first; Descending again, the newest first.
  const order = page.getByRole("button", { name: "Order by Created at, Descending", exact: true });
  await order.focus();
  await page.keyboard.press("Enter");
  const menu = page.getByRole("menu");
  await expect(menu.getByRole("menuitem")).toHaveText(["Created at", "Updated at", "ID", "Ascending", "Descending"]);
  await expectPopupBesideButton(menu, order);
  await page.keyboard.press("Escape");
  await expect(menu).toHaveCount(0);
  await expectLanded(order);
  await chooseByKeyboard(page, order, "Ascending");
  const ascending = page.getByRole("button", { name: "Order by Created at, Ascending", exact: true });
  await expectLanded(ascending);
  await expect.poll(() => shownRows(page)).toEqual([slow, crash, spam].map(rowName));
  await chooseByKeyboard(page, ascending, "Descending");
  await expect.poll(() => shownRows(page)).toEqual([spam, crash, slow].map(rowName));

  // The header's actions: Enter opens the menu of those not shown as buttons; Escape closes it, back on its button.
  const options = optionsOf(page);
  await options.focus();
  await page.keyboard.press("Enter");
  await expect(menu.getByRole("menuitem")).toHaveText(["Snooze", "Mark as duplicate", "Delete"]);
  await page.keyboard.press("Escape");
  await expect(menu).toHaveCount(0);
  await expectLanded(options);

  // ↓ in the description's editor steps nowhere: had it stepped, the ↓ on the detail would step on past Crash. On
  // the detail, ↓ steps to Crash, ↑ back to Spam.
  await intakeDescription(page, spam).click();
  await page.keyboard.press("ArrowDown");
  await intakeDetail(page).focus();
  await page.keyboard.press("ArrowDown");
  await expect(page).toHaveURL(selecting(crash));
  await expectHeaderOf(page, crash);
  await page.keyboard.press("ArrowUp");
  await expect(page).toHaveURL(selecting(spam));
  await expectHeaderOf(page, spam);

  // Each modal by the keyboard, closed by Escape, back on its opener: accept, decline, and delete from the menu.
  const detail = intakeDetail(page);
  const accept = detail.getByRole("button", { name: "Accept", exact: true });
  await openedAndEscaped(page, accept, `Move ${identifierOf(spam)} to project work items`);
  await openedAndEscaped(page, detail.getByRole("button", { name: "Decline", exact: true }), "Decline work item");
  await options.focus();
  await chooseByKeyboard(page, options, "Delete");
  await expect(page.getByRole("dialog", { name: "Delete work item", exact: true })).toHaveCount(1);
  expect([await focusInDialog(page), await pageBehindHidden(page)]).toEqual([true, true]);
  expect(await closedByEscape(page)).toBe(true);
  await expectLanded(options);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice accepts, declines and deletes from Web's intake page by the keyboard, nerve's answers held: each modal stays, its cancel out of use, the focus within it; once nerve answers, the focus is on the next row, or, the selection cleared, on the list; a link copied with the clipboard refused says it failed", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [dark, slow, crash, spam] = await submitNamed(api, bob.token, web.id, ["Dark", "Slow", "Crash", "Spam"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([spam, crash, slow, dark].map(rowName));
  await expect(page).toHaveURL(selecting(spam));
  const key = tabKey(browserName);
  const detail = intakeDetail(page);
  const options = optionsOf(page);

  // Spam accepted, nerve's triage held: the modal stays, Discard out of use; once nerve answers and the update is
  // confirmed, Crash selected and focused.
  await detail.getByRole("button", { name: "Accept", exact: true }).focus();
  await page.keyboard.press("Enter");
  const moving = page.getByRole("dialog", { name: `Move ${identifierOf(spam)} to project work items`, exact: true });
  const accepted = await sentHeld(page, "POST", triagePath(spam), () =>
    moving.getByRole("button", { name: "Add to project", exact: true }).click()
  );
  await expectLocked(page, moving.getByRole("button", { name: "Discard", exact: true }));
  expect((await accepted.release()).status()).toBe(200);
  await expect.poll(() => focusedName(page)).toBe(`link ${rowName(crash)}`);
  await expectLanded(intakeRow(page, crash));
  await expect(page).toHaveURL(selecting(crash));

  // Crash declined by Tab and Space, nerve's answer held: locked; then Slow selected and focused.
  await expectHeaderOf(page, crash);
  await detail.getByRole("button", { name: "Decline", exact: true }).focus();
  await page.keyboard.press("Enter");
  const declining = page.getByRole("dialog", { name: "Decline work item", exact: true });
  const declined = await sentHeld(page, "POST", triagePath(crash), async () => {
    await tabTo(page, key, declining.getByRole("button", { name: "Decline", exact: true }));
    await page.keyboard.press(" ");
  });
  await expectLocked(page, declining.getByRole("button", { name: "Cancel", exact: true }));
  expect((await declined.release()).status()).toBe(200);
  await expect.poll(() => focusedName(page)).toBe(`link ${rowName(slow)}`);
  await expectLanded(intakeRow(page, slow));
  await expect(page).toHaveURL(selecting(slow));

  // Slow deleted by Tab and Enter, nerve's answer held: locked; then the selection cleared, the focus on the list,
  // which selects Dark, and the modal gone (checked after the focus, which lands once the deletion is settled).
  await expectHeaderOf(page, slow);
  await options.focus();
  await chooseByKeyboard(page, options, "Delete");
  const deleting = page.getByRole("dialog", { name: "Delete work item", exact: true });
  const release = await holdAnswer(page, "DELETE", intakeEditPath(slow));
  await tabTo(page, key, deleting.getByRole("button", { name: "Delete", exact: true }));
  await page.keyboard.press("Enter");
  await expectLocked(page, deleting.getByRole("button", { name: "Cancel", exact: true }));
  await release();
  await expect.poll(() => focusedName(page)).toBe("region Intake work items");
  await expectLanded(page.getByRole("region", { name: "Intake work items", exact: true }));
  await expect(page).toHaveURL(selecting(dark));
  await expect(modalDialog(page)).toHaveCount(0);

  // Dark's link copied, the clipboard refused: said not copied.
  await expectHeaderOf(page, dark);
  await refuseClipboardWrites(page);
  await detail.getByRole("button", { name: "Copy work item link", exact: true }).click();
  await expect(toast(page, "Something went wrong. Please try again.")).toBeVisible();
  expect(await shownWithin(toast(page, "Link copied to clipboard"))).toBe(false);

  // The deletion's read again, in its turn, finds the work item gone (nerve's 404), as the deletion means.
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 GET /api/v0/issues/${slow.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 404) });
});
