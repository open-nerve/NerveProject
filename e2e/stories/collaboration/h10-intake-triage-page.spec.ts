import type { Page } from "@playwright/test";

import { intakeRowOf, triageActivities } from "../../fixtures/assert/intake";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { seenBy, shownDay } from "../../fixtures/cycle-pages";
import { submitIntakeIssue } from "../../fixtures/intake";
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
import { createIssue } from "../../fixtures/issues";
import { zonedMoment } from "../../fixtures/notification-pages";
import { sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H10's page version, its other triages (M7 design 2, 3.5; P5a spec 2.6, 2.8, 2.9, 2.12, R11, R13, R14, R15, R17,
// R35): a project's intake triaged by its admin on its page, its members and its guests offered no triage. A snooze is
// till the first moment of a day of the browser's calendar, tomorrow at the earliest and at first, a snooze anew from
// the day it holds; snoozing, snoozing anew, ending a snooze and declining move the selection to the next row and the
// focus with it; marking a duplicate, through the picker of the project's work items, keeps both on the work item,
// which then names the one it duplicates. A member who submitted a work item is offered its deletion alone. The
// acceptance is h10-intake-accept-page.spec.ts's; the API version is h10-intake-triage.spec.ts.

/** The browser's time zone, whose calendar's days a snooze ends at the first moment of. */
const ZONE = "America/Los_Angeles";

test.use({ timezoneId: ZONE });

/** How a live intake row reads, neither snoozed nor a duplicate. */
const live = { snoozed_till: null, duplicate_to_id: null };

/** Chooses item from the menu of the header of the intake page's detail. */
async function fromMenu(page: Page, item: string): Promise<void> {
  await intakeDetail(page).getByRole("button", { name: "Options", exact: true }).click();
  await page.getByRole("menuitem", { name: item, exact: true }).click();
}

/**
 * Checks that the header of the intake page's detail, drawn, offers no triage: copying the work item, not opening it
 * (the page offers that on one accepted alone, P5b R15, and its readers here look at pending ones), and in its menu
 * menu alone, no menu when menu is empty.
 */
async function expectNoTriage(page: Page, menu: string[]): Promise<void> {
  const detail = intakeDetail(page);
  await expect(detail.getByRole("button", { name: "Copy work item link", exact: true })).toBeVisible();
  await expect(detail.getByRole("link", { name: "Open work item", exact: true })).toHaveCount(0);
  await expect(detail.getByRole("button", { name: /^(Accept|Decline)$/ })).toHaveCount(0);
  const options = detail.getByRole("button", { name: "Options", exact: true });
  await expect(options).toHaveCount(menu.length === 0 ? 0 : 1);
  if (menu.length === 0) return;
  await options.click();
  await expect(page.getByRole("menuitem")).toHaveText(menu);
  await page.keyboard.press("Escape");
}

test("H10 (page): in Los Angeles, alice snoozes the work item selected from the header's menu till tomorrow there, its first moment, the selection moving to the next row; shown again by the snoozed filter, she snoozes it anew till a later day, then ends its snooze, the selection moving each time", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [dark, slow, crash] = await submitNamed(api, bob.token, web.id, ["Dark mode", "Slow search", "Crash on save"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([crash, slow, dark].map(rowName));
  await expect(page).toHaveURL(selecting(crash));

  // Snoozed till tomorrow, the modal's day at first: its first moment in Los Angeles; Slow search focused, as the
  // triage ends, and selected.
  const tomorrow = await browserDay(page, 1);
  await fromMenu(page, "Snooze");
  const snoozing = page.getByRole("dialog", { name: "Snooze work item", exact: true });
  const day = snoozing.getByRole("button", { name: "Snooze until", exact: true });
  await expect(day).toHaveText(shownDay(tomorrow));
  const snoozed = await sentTo(page, "POST", triagePath(crash), () =>
    snoozing.getByRole("button", { name: "Snooze", exact: true }).click()
  );
  expect([snoozed.body, snoozed.answer.status()]).toEqual([
    { status: "snoozed", snoozed_till: zonedMoment(tomorrow, "00:00", ZONE) },
    200,
  ]);
  await expect(intakeRow(page, slow)).toBeFocused();
  await expect(page).toHaveURL(selecting(slow));
  await expect(snoozing).toHaveCount(0);
  await expect.poll(() => shownRows(page)).toEqual([slow, dark].map(rowName));

  // The snoozed shown too; Crash on save, chosen, snoozed anew from its day to a later one, by the calendar.
  const panel = await openFilters(page);
  await filterOption(panel, "Status", "Snoozed").click();
  await page.keyboard.press("Escape");
  await expect.poll(() => shownRows(page)).toEqual([crash, slow, dark].map(rowName));
  await intakeRow(page, crash).click();
  await expectHeaderOf(page, crash);
  await fromMenu(page, "Change snooze date");
  await expect(day).toHaveText(shownDay(tomorrow));
  const later = await browserDay(page, 3);
  await day.click();
  const calendar = page.getByRole("dialog", { name: "Snooze until", exact: true });
  const [year = "", month = ""] = later.split("-");
  await calendar.getByRole("combobox", { name: "Choose the Year" }).selectOption(year);
  await calendar.getByRole("combobox", { name: "Choose the Month" }).selectOption(String(Number(month) - 1));
  await calendar.locator(`[data-day="${later}"] button`).click();
  await expect(day).toHaveText(shownDay(later));
  const resnoozed = await sentTo(page, "POST", triagePath(crash), () =>
    snoozing.getByRole("button", { name: "Snooze", exact: true }).click()
  );
  expect([resnoozed.body, resnoozed.answer.status()]).toEqual([
    { status: "snoozed", snoozed_till: zonedMoment(later, "00:00", ZONE) },
    200,
  ]);
  await expect(intakeRow(page, slow)).toBeFocused();
  await expect(page).toHaveURL(selecting(slow));
  await expect(snoozing).toHaveCount(0);
  expect(await intakeRowOf(db, crash.id)).toEqual({
    ...live,
    status: "snoozed",
    snoozed_till: zonedMoment(later, "00:00", ZONE),
    written_by: alice.id,
  });

  // Its snooze ended, from the menu, without a modal: pending again; Slow search selected and focused.
  await intakeRow(page, crash).click();
  await expectHeaderOf(page, crash);
  const ended = await sentTo(page, "POST", triagePath(crash), () => fromMenu(page, "Un snooze"));
  expect([ended.body, ended.answer.status()]).toEqual([{ status: "pending" }, 200]);
  await expect(intakeRow(page, slow)).toBeFocused();
  await expect(page).toHaveURL(selecting(slow));
  expect(await intakeRowOf(db, crash.id)).toEqual({ ...live, status: "pending", written_by: alice.id });
  expect(await triageActivities(db, crash.id)).toEqual([
    { verb: "snoozed", old_value: "pending", new_value: "snoozed", actor: alice.id },
    { verb: "snoozed", old_value: "snoozed", new_value: "snoozed", actor: alice.id },
    { verb: "pending", old_value: "snoozed", new_value: "pending", actor: alice.id },
  ]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H10 (page): alice marks the work item selected a duplicate of Fix the login through the picker, which lists neither it nor the intake's others, the selection and the focus staying on it, now a duplicate of WEB-1; she declines another, the selection moving on; bob, a member, is offered on his own work item its deletion alone, on another nothing, and gus, a guest, nothing on his own", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, gus, slug, web } = await intakeWorld(api, testInfo);
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix the login" });
  const idea = await submitIntakeIssue(api, gus.token, web.id, { name: "Gus's idea" });
  const [dark, spam, again] = await submitNamed(api, bob.token, web.id, ["Dark mode", "Buy now", "The login fails"]);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([again, spam, dark, idea].map(rowName));
  await expect(page).toHaveURL(selecting(again));

  // The picker lists Fix the login alone; chosen: a duplicate of it, still selected, its detail focused, the header
  // saying so and naming it.
  await fromMenu(page, "Mark as duplicate");
  const picker = page.getByRole("dialog", { name: "Mark as duplicate", exact: true });
  await expect(picker.getByRole("option")).toHaveText([`${identifierOf(fix)}Fix the login`]);
  const marked = await sentTo(page, "POST", triagePath(again), () =>
    picker.getByRole("option", { name: /Fix the login/ }).click()
  );
  expect([marked.body, marked.answer.status()]).toEqual([{ status: "duplicate", duplicate_to_id: fix.id }, 200]);
  await expect(shownStatus(page, "Duplicate")).toBeVisible();
  await expect(
    intakeDetail(page).getByRole("link", { name: `${identifierOf(fix)} · Fix the login`, exact: true })
  ).toBeVisible();
  await expect(page).toHaveURL(selecting(again));
  await expect(intakeDetail(page)).toBeFocused();
  await expect(picker).toHaveCount(0);
  await expect.poll(() => shownRows(page)).toEqual([spam, dark, idea].map(rowName));

  // Buy now, chosen, declined: Dark mode, the next row, selected and focused.
  await intakeRow(page, spam).click();
  await expectHeaderOf(page, spam);
  await intakeDetail(page).getByRole("button", { name: "Decline", exact: true }).click();
  const declining = page.getByRole("dialog", { name: "Decline work item", exact: true });
  const declined = await sentTo(page, "POST", triagePath(spam), () =>
    declining.getByRole("button", { name: "Decline", exact: true }).click()
  );
  expect([declined.body, declined.answer.status()]).toEqual([{ status: "declined" }, 200]);
  await expect(intakeRow(page, dark)).toBeFocused();
  await expect(page).toHaveURL(selecting(dark));
  await expect.poll(() => shownRows(page)).toEqual([dark, idea].map(rowName));
  expect(await Promise.all([again, spam].map((issue) => intakeRowOf(db, issue.id)))).toEqual([
    { ...live, status: "duplicate", duplicate_to_id: fix.id, written_by: alice.id },
    { ...live, status: "declined", written_by: alice.id },
  ]);
  expect(await Promise.all([again, spam].map((issue) => triageActivities(db, issue.id)))).toEqual([
    [{ verb: "duplicate", old_value: "pending", new_value: "duplicate", actor: alice.id }],
    [{ verb: "declined", old_value: "pending", new_value: "declined", actor: alice.id }],
  ]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);

  // bob, a member: on Dark mode, his, its deletion alone beside its copy and opening; on Gus's idea, nothing.
  await seenBy(browser, baseURL ?? "", api, bob.email, intakePath(slug, web.id), async (other) => {
    await expect.poll(() => shownRows(other)).toEqual([dark, idea].map(rowName));
    await expect(other).toHaveURL(selecting(dark));
    await expectHeaderOf(other, dark);
    await expectNoTriage(other, ["Delete"]);
    await intakeRow(other, idea).click();
    await expectHeaderOf(other, idea);
    await expectNoTriage(other, []);
  });
  // gus, a guest: his own alone, and nothing on it.
  await seenBy(browser, baseURL ?? "", api, gus.email, intakePath(slug, web.id), async (other) => {
    await expect.poll(() => shownRows(other)).toEqual([rowName(idea)]);
    await expectHeaderOf(other, idea);
    await expectNoTriage(other, []);
  });
});
