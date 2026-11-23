import type { Page } from "@playwright/test";

import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { failedLoads, tabKey } from "../../fixtures/cycle-pages";
import {
  closedByBackdropClick,
  closedByEscape,
  closedWithin,
  enabledWithin,
  focusInDialog,
  modalDialog,
  pageBehindHidden,
} from "../../fixtures/dialogs";
import { triageIntakeIssue } from "../../fixtures/intake";
import {
  expectLanded,
  focusedName,
  fromMenu,
  listToggle,
  NARROW,
  narrowOn,
  optionsOf,
  tabTo,
  type MenuPath,
} from "../../fixtures/intake-focus";
import {
  browserDay,
  expectHeaderOf,
  filterOption,
  identifierOf,
  intakeDetail,
  intakePath,
  intakeRow,
  intakeWorld,
  openFilters,
  rowName,
  selecting,
  shownRows,
  shownStatus,
  submitNamed,
  triagePath,
} from "../../fixtures/intake-pages";
import { createIssue, type Issue } from "../../fixtures/issues";
import { zonedDay, zonedMoment } from "../../fixtures/notification-pages";
import { sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H20's page version for the intake page's focus as a triage ends (M7 design 2, H20; P5a spec 2.8's R35, 2.12; re-94
// I4): a duplicate marked, a snooze and a snooze anew, each opened once by the keyboard, once by the pointer (whose
// press focuses the menu's item, never its button: P4 O4) and once from the narrow header's menu (the list hidden by a
// row's press, shown again by its toggle). Each modal is named, holds the focus, the page behind it inert; cancelled,
// the focus is back on the menu's button. Sent, nerve's answer held, the modal stays (neither Escape nor a click
// outside closes it, its Cancel out of use, the focus within it: checked on each modal's first path); once nerve
// answers, the header drawn again, the focus lands as R35 says, where the screen shows it, never on the page's body:
// on the detail for a duplicate, which stays selected; on the next row for a snooze, the selection moving with it, or
// on the detail once a narrow screen hides the list. On a narrow screen a decline, an acceptance and a deletion land
// on the detail the page then selects, the last deletion on the list's toggle. The keys and names of the page's other
// controls are h20-intake-keys-pages.spec.ts's; its late answers, h20-intake-late-pages.spec.ts's.

/** The browser's time zone, whose calendar's days a snooze ends at the first moment of. */
const ZONE = "America/Los_Angeles";

test.use({ timezoneId: ZONE });

/** The dialog of the action named name: its modal, open. */
const dialogNamed = (page: Page, name: string) => page.getByRole("dialog", { name, exact: true });

/** Presses the Cancel of the modal named name, by the pointer, and resolves with whether it closed (closedWithin). */
async function closedByCancel(page: Page, name: string): Promise<boolean> {
  const closed = closedWithin(page);
  await dialogNamed(page, name).getByRole("button", { name: "Cancel", exact: true }).click();
  return closed;
}

/**
 * Opens the modal named name by item of the header's menu along path, checks that it is named, holds the focus and
 * makes the page behind it inert, and cancels it: by Escape on the keyboard's path, else by cancel, a press of the
 * pointer; the focus is then back on the menu's button.
 */
async function openAndCancel(page: Page, path: MenuPath, item: string, name: string, cancel: () => Promise<boolean>) {
  await fromMenu(page, path, item);
  await expect(dialogNamed(page, name)).toHaveCount(1);
  expect([await focusInDialog(page), await pageBehindHidden(page)]).toEqual([true, true]);
  expect(path === "keyboard" ? await closedByEscape(page) : await cancel()).toBe(true);
  await expectLanded(optionsOf(page));
}

/**
 * Sends the triage of issue that send makes the modal named name send, nerve's answer held; checked (the first path
 * of each modal), meanwhile neither Escape nor a click outside closes it, its cancel (if it has one) out of use, the
 * focus within it. Then lets the answer through and gives the body sent.
 */
async function sentLocked(
  page: Page,
  issue: Issue,
  name: string,
  send: () => Promise<void>,
  checked: { cancel: boolean } | false
) {
  const held = await sentHeld(page, "POST", triagePath(issue), send);
  if (checked) await expectLocked(page, name, checked.cancel);
  expect((await held.release()).status()).toBe(200);
  return held.body;
}

/** Checks that the modal named name, its triage out, stays: no Escape, no click outside, its cancel out of use. */
async function expectLocked(page: Page, name: string, cancel: boolean) {
  expect(await closedByEscape(page)).toBe(false);
  expect(await closedByBackdropClick(page)).toBe(false);
  if (cancel)
    expect(await enabledWithin(dialogNamed(page, name).getByRole("button", { name: "Cancel", exact: true }))).toBe(
      false
    );
  expect(await focusInDialog(page)).toBe(true);
}

/**
 * Waits for the focus to land on target, named name, once the header is drawn again (the page's body is never the name
 * waited for), then checks that no modal is left open. In that order: the focus lands once the triage is settled,
 * while the modal closes a moment after nerve answers, so a modal still open would be seen only in passing first.
 */
async function landsOn(page: Page, name: string, target: ReturnType<typeof intakeDetail>) {
  await expect.poll(() => focusedName(page)).toBe(name);
  await expectLanded(target);
  await expect(modalDialog(page)).toHaveCount(0);
}

test("H20 (page): alice marks three work items duplicates of Fix the login, by the picker opened from the header's menu by the keyboard, by the pointer, and from the narrow header's menu: each time it is named, holds the focus and makes the page behind it inert, cancelled the focus back on the menu's button; chosen, nerve's answer held, nothing closes it; once nerve answers, the header says Duplicate and the focus is on the detail, the selection kept", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix the login" });
  const [broken, cannot, fails] = await submitNamed(api, bob.token, web.id, ["Login broken", "Cannot log in", "Fails"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([fails, cannot, broken].map(rowName));
  await expect(page).toHaveURL(selecting(fails));
  const key = tabKey(browserName);
  const picker = dialogNamed(page, "Mark as duplicate");
  const candidate = `${identifierOf(fix)}Fix the login`;

  /** Marks issue a duplicate of Fix the login along path, the picker first opened and cancelled. */
  const markDuplicate = async (path: MenuPath, issue: Issue) => {
    await openAndCancel(page, path, "Mark as duplicate", "Mark as duplicate", () => closedByBackdropClick(page));
    await fromMenu(page, path, "Mark as duplicate");
    await expect(picker.getByRole("option")).toHaveText([candidate]);
    const body = await sentLocked(
      page,
      issue,
      "Mark as duplicate",
      async () => {
        if (path !== "keyboard") return picker.getByRole("option", { name: /Fix the login/ }).click();
        // the search, then the candidate by the arrow, chosen by Enter
        await tabTo(page, key, picker.getByRole("combobox"));
        await page.keyboard.press("ArrowDown");
        await page.keyboard.press("Enter");
      },
      path === "keyboard" && { cancel: false }
    );
    expect(body).toEqual({ status: "duplicate", duplicate_to_id: fix.id });
    await landsOn(page, "region Intake work item", intakeDetail(page));
    await expect(shownStatus(page, "Duplicate")).toBeVisible();
    await expect(page).toHaveURL(selecting(issue));
  };

  // By the keyboard, Fails, selected first; by the pointer, Cannot log in, pressed; on a narrow screen, Login broken.
  await expectHeaderOf(page, fails);
  await markDuplicate("keyboard", fails);
  await intakeRow(page, cannot).click();
  await expectHeaderOf(page, cannot);
  await markDuplicate("pointer", cannot);
  await narrowOn(page, broken);
  await markDuplicate("narrow", broken);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice snoozes three work items from the header's menu, by the keyboard, by the pointer, and from the narrow header's menu: the modal named, holding the focus, the page behind it inert, cancelled back on the menu's button; sent, nerve's answer held, nothing closes it and its Cancel is out of use; once nerve answers, the selection moves to the next row and the focus with it, or, the list hidden on a narrow screen, to the detail", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [rest, dark, slow, crash] = await submitNamed(api, bob.token, web.id, ["Rest", "Dark", "Slow", "Crash"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([crash, slow, dark, rest].map(rowName));
  await expect(page).toHaveURL(selecting(crash));
  const key = tabKey(browserName);
  const snoozing = dialogNamed(page, "Snooze work item");
  const cancel = () => closedByCancel(page, "Snooze work item");

  // By the keyboard: Crash snoozed, the Snooze button reached by Tab; Slow, the next row, selected and focused.
  await openAndCancel(page, "keyboard", "Snooze", "Snooze work item", cancel);
  const tomorrow = zonedMoment(await browserDay(page, 1), "00:00", ZONE);
  await fromMenu(page, "keyboard", "Snooze");
  const byKeyboard = await sentLocked(
    page,
    crash,
    "Snooze work item",
    async () => {
      await tabTo(page, key, snoozing.getByRole("button", { name: "Snooze", exact: true }));
      await page.keyboard.press("Enter");
    },
    { cancel: true }
  );
  expect(byKeyboard).toEqual({ status: "snoozed", snoozed_till: tomorrow });
  await landsOn(page, `link ${rowName(slow)}`, intakeRow(page, slow));
  await expect(page).toHaveURL(selecting(slow));

  // By the pointer: Slow snoozed; Dark selected and focused.
  await expectHeaderOf(page, slow);
  await openAndCancel(page, "pointer", "Snooze", "Snooze work item", cancel);
  await fromMenu(page, "pointer", "Snooze");
  await sentLocked(
    page,
    slow,
    "Snooze work item",
    () => snoozing.getByRole("button", { name: "Snooze", exact: true }).click(),
    false
  );
  await landsOn(page, `link ${rowName(dark)}`, intakeRow(page, dark));
  await expect(page).toHaveURL(selecting(dark));

  // On a narrow screen, Dark pressed, the list hidden: snoozed from the narrow header's menu, Rest selected, the
  // focus on its detail, which the screen shows, not on its row, which it does not.
  await narrowOn(page, dark);
  await openAndCancel(page, "narrow", "Snooze", "Snooze work item", cancel);
  await fromMenu(page, "narrow", "Snooze");
  await sentLocked(
    page,
    dark,
    "Snooze work item",
    () => snoozing.getByRole("button", { name: "Snooze", exact: true }).click(),
    false
  );
  await landsOn(page, "region Intake work item", intakeDetail(page));
  await expect(page).toHaveURL(selecting(rest));
  await expect(listToggle(page)).toHaveAttribute("aria-expanded", "false");

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): alice snoozes anew three snoozed work items, shown by the snoozed filter, from the header's menu by the keyboard, by the pointer, and from the narrow header's menu: cancelled, back on the menu's button; sent till the day each holds, nerve's answer held, nothing closes the modal; once nerve answers, the next row has the selection and the focus, or, the list hidden on a narrow screen, the detail", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [rest, one, two, three] = await submitNamed(api, bob.token, web.id, ["Rest", "One", "Two", "Three"]);
  const held = zonedMoment(zonedDay(2, ZONE), "00:00", ZONE);
  await Promise.all(
    [one, two, three].map((issue) =>
      triageIntakeIssue(api, alice.token, issue.id, { status: "snoozed", snoozed_till: held })
    )
  );

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([rowName(rest)]);
  const panel = await openFilters(page);
  await filterOption(panel, "Status", "Snoozed").click();
  await page.keyboard.press("Escape");
  await expect.poll(() => shownRows(page)).toEqual([three, two, one, rest].map(rowName));
  const snoozing = dialogNamed(page, "Snooze work item");
  const cancel = () => closedByCancel(page, "Snooze work item");
  const snoozeAgain = () => snoozing.getByRole("button", { name: "Snooze", exact: true }).click();

  // Three, chosen, by the keyboard; its modal at the day it holds; sent: Two selected and focused.
  await intakeRow(page, three).click();
  await expectHeaderOf(page, three);
  await openAndCancel(page, "keyboard", "Change snooze date", "Snooze work item", cancel);
  await fromMenu(page, "keyboard", "Change snooze date");
  expect(await sentLocked(page, three, "Snooze work item", snoozeAgain, { cancel: true })).toEqual({
    status: "snoozed",
    snoozed_till: held,
  });
  await landsOn(page, `link ${rowName(two)}`, intakeRow(page, two));
  await expect(page).toHaveURL(selecting(two));

  // Two, by the pointer: One selected and focused.
  await expectHeaderOf(page, two);
  await openAndCancel(page, "pointer", "Change snooze date", "Snooze work item", cancel);
  await fromMenu(page, "pointer", "Change snooze date");
  await sentLocked(page, two, "Snooze work item", snoozeAgain, false);
  await landsOn(page, `link ${rowName(one)}`, intakeRow(page, one));
  await expect(page).toHaveURL(selecting(one));

  // One, on a narrow screen, the list hidden by its press: Rest selected, the focus on its detail.
  await narrowOn(page, one);
  await openAndCancel(page, "narrow", "Change snooze date", "Snooze work item", cancel);
  await fromMenu(page, "narrow", "Change snooze date");
  await sentLocked(page, one, "Snooze work item", snoozeAgain, false);
  await landsOn(page, "region Intake work item", intakeDetail(page));
  await expect(page).toHaveURL(selecting(rest));

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H20 (page): on a narrow screen, the list over the detail until a row's press hides it and its toggle shows it again, alice declines, accepts and deletes from the narrow header's menu: the focus lands on the detail the page then selects, which the screen shows, never on a row or a list it hides; the last one deleted, on the list's toggle", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [d1, d2, d3, d4] = await submitNamed(api, bob.token, web.id, ["D1", "D2", "D3", "D4"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.setViewportSize(NARROW);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([d4, d3, d2, d1].map(rowName));
  await expect(page).toHaveURL(selecting(d4));
  await expect(listToggle(page)).toHaveAttribute("aria-expanded", "true");

  // D4 declined: D3 selected, the focus on its detail.
  await narrowOn(page, d4);
  await fromMenu(page, "narrow", "Decline");
  await dialogNamed(page, "Decline work item").getByRole("button", { name: "Decline", exact: true }).click();
  await landsOn(page, "region Intake work item", intakeDetail(page));
  await expect(page).toHaveURL(selecting(d3));

  // D3 accepted through the modal: D2 selected, the focus on its detail.
  await narrowOn(page, d3);
  await fromMenu(page, "narrow", "Accept");
  await dialogNamed(page, `Move ${identifierOf(d3)} to project work items`)
    .getByRole("button", { name: "Add to project", exact: true })
    .click();
  await landsOn(page, "region Intake work item", intakeDetail(page));
  await expect(page).toHaveURL(selecting(d2));

  // D2 deleted: the selection cleared, then D1, the page's first row, selected, the focus on its detail.
  await narrowOn(page, d2);
  await fromMenu(page, "narrow", "Delete");
  await dialogNamed(page, "Delete work item").getByRole("button", { name: "Delete", exact: true }).click();
  await landsOn(page, "region Intake work item", intakeDetail(page));
  await expect(page).toHaveURL(selecting(d1));

  // D1, the last, deleted: nothing selected nor to be, the focus on the list's toggle, which says the list is hidden.
  await narrowOn(page, d1);
  await fromMenu(page, "narrow", "Delete");
  await dialogNamed(page, "Delete work item").getByRole("button", { name: "Delete", exact: true }).click();
  await landsOn(page, "button Intake list", listToggle(page));
  await expect(page).not.toHaveURL(/inboxIssueId/);
  await expect(listToggle(page)).toHaveAttribute("aria-expanded", "false");

  // Each deletion's read again, in its turn, finds its work item gone (nerve's 404), as the deletion means.
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 GET /api/v0/issues/${d2.id}`, `404 GET /api/v0/issues/${d1.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 404, 404) });
});
