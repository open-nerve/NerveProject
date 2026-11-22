import type { Locator, Page } from "@playwright/test";

import { addProjectMembers, createProject, projectMembershipOf, projectMemberWrites } from "../../fixtures/api";
import { notificationStates, snoozedTill } from "../../fixtures/assert/notification";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { createIssueComment } from "../../fixtures/comments";
import { dismissToasts, failedLoads, tabKey, toast } from "../../fixtures/cycle-pages";
import { createIssue, updateIssue } from "../../fixtures/issues";
import { chooseByKeyboard } from "../../fixtures/menus";
import {
  cardSaying,
  expectActionsShown,
  notificationWorld,
  notificationsPath,
  unreadReach,
  zonedDay,
  zonedMoment,
} from "../../fixtures/notification-pages";
import { notificationUpdated, notificationsSent, type Notification } from "../../fixtures/notifications";
import { answerTo, sentTo } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// H6's page version (M7 design 2, 3.4; P4 spec 2.6, 2.7, 2.11, R8, R9, R15, R20's W1, R23, R25, re-93 m11): a
// receiver changes his notifications' states by their cards, by the pointer as a card shows its actions and by the
// keyboard as it holds the focus. Read and unread stay in place; archived leaves the default view and unarchived the
// archived view; a snooze, of a preset or of a time of his own, leaves the default view, and its end, which sends
// snoozed_till null, the snoozed view. Each is said, and the notification's row follows. The snoozed view's cards
// offer no archive, the archived view's no snooze (R25). His own time is a day and a half hour in his browser's zone,
// here Los Angeles: tomorrow's 12:30 AM is half past midnight (Plane's W1 made it half past noon); a time nerve takes
// as past (must_be_future, his clock behind nerve's) is said under the time, the dialog open. A snooze over is listed
// and counted again once the page reads again. Opening a card reads it and shows its work item. Mark all read in the
// all tab leaves the mentions, the archived and the snoozed unread (D5), as many marked as nerve says. Removed from a
// project, his change of one of its notifications is refused as not found, said, and its card leaves. The API version
// is h5-h7-notification-inbox.spec.ts.

test.use({ timezoneId: "America/Los_Angeles" });

const ZONE = "America/Los_Angeles";

/** The id of the notification of notes whose change is of field. */
function idOf(notes: Notification[], field: string): string {
  const note = notes.find((n) => n.data.change.field === field);
  if (!note) throw new Error(`no notification of ${field}`);
  return note.id;
}

/** Opens the view menu of page's head and toggles its item, then closes it by Escape. */
async function toggleView(page: Page, item: "Show archived" | "Show snoozed"): Promise<void> {
  await page.getByRole("button", { name: "Display", exact: true }).click();
  await page.getByRole("button", { name: item, exact: true }).click();
  await page.keyboard.press("Escape");
}

test("H6 (page): bob reads and unreads, archives and unarchives, snoozes for 3 days and unsnoozes by his cards, by the pointer and the keyboard, each said and kept by nerve, the snoozed view's cards without archive and the archived view's without snooze; his own time, tomorrow at 12:30 AM in Los Angeles, is half past midnight there; a past one is refused under the time; a snooze over is listed again; a card opened is read and shows its work item; Mark all read leaves the mentions, the archived and the snoozed unread; removed from API, his change of its card is not found and the card leaves", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await notificationWorld(api, testInfo);
  // Fix, alice's, assigned to bob; her comment on it, its priority, its state: five of his, one at a time.
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix", assignee_ids: [bob.id] });
  await notificationsSent(api, bob.token, slug, 2);
  await createIssueComment(api, alice.token, fix.id, "<p>Looks good</p>");
  await updateIssue(api, alice.token, fix.id, { priority: "high" });
  await updateIssue(api, alice.token, fix.id, { state_id: (await stateIds(api, alice.token, web.id)).Todo });
  // A mention of him in another comment: his mentions tab's.
  const mention = `<mention-component entity_name="user_mention" entity_identifier="${bob.id}"></mention-component>`;
  await createIssueComment(api, alice.token, fix.id, `<p>Hi</p>${mention}`);
  // API, of which he is a member too, and its Rate limit, his: two more.
  const apiProject = await createProject(api, alice.token, slug, { name: "API", identifier: "API" });
  await addProjectMembers(api, alice.token, apiProject.id, [{ member_id: bob.id, role: 15 }]);
  const rateLimit = await createIssue(api, alice.token, apiProject.id, { name: "Rate limit", assignee_ids: [bob.id] });
  await unreadReach(api, bob.token, slug, { all: 7, mentions: 1 });
  const all = await notificationsSent(api, bob.token, slug, 7);
  const [mentionNote] = await notificationsSent(api, bob.token, slug, 1, { tab: "mentions" });
  const fixes = all.filter((n) => n.issue_id === fix.id);
  const rates = all.filter((n) => n.issue_id === rateLimit.id);
  const ids = {
    created: idOf(fixes, "issue"),
    assigned: idOf(fixes, "assignees"),
    comment: idOf(fixes, "comment"),
    priority: idOf(fixes, "priority"),
    state: idOf(fixes, "state"),
    mention: mentionNote?.id ?? "",
  };
  const states = async (...which: (keyof typeof ids)[]) =>
    notificationStates(
      db,
      which.map((key) => ids[key])
    );
  const ref = `WEB-${fix.sequence_id} Fix`;

  // His clock is the page's own, so that it can run behind nerve's (must_be_future, below).
  const page = await signedInPage(bob.tokens);
  await page.clock.install();
  const watch = await watchPage(page);
  await page.goto(notificationsPath(slug));
  const tabs = page.getByRole("tab");
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 7 unread");
  const key = tabKey(browserName);
  const priority = cardSaying(page, "Alice set the priority to High");

  // Read, by the pointer, its actions shown as the card is hovered: in place, read; unread, by the keyboard, the
  // focus on its main button, then Tab to its first action, shown as the card holds it.
  await expectActionsShown(priority, `Mark ${ref} read`, false);
  await priority.hover();
  await expectActionsShown(priority, `Mark ${ref} read`, true);
  await priority.getByRole("button", { name: `Mark ${ref} read`, exact: true }).click();
  await expect(toast(page, "Notification marked as read")).toBeVisible();
  await expect(priority.getByText("Unread", { exact: true })).toHaveCount(0);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 6 unread");
  expect(await states("priority")).toEqual([{ read: true, archived: false, snoozed: false, by_receiver: true }]);
  await page.mouse.move(0, 0);
  await priority.getByRole("button").first().focus();
  await page.keyboard.press(key);
  const unread = priority.getByRole("button", { name: `Mark ${ref} unread`, exact: true });
  await expect(unread).toBeFocused();
  await expectActionsShown(priority, `Mark ${ref} unread`, true);
  await page.keyboard.press("Enter");
  await expect(toast(page, "Notification marked as unread")).toBeVisible();
  await expect(priority.getByText("Unread", { exact: true })).toHaveCount(1);
  expect(await states("priority")).toEqual([{ read: false, archived: false, snoozed: false, by_receiver: true }]);

  // Archived in the default view, by the pointer: it leaves; in the archived view, it offers no snooze, and unarchived
  // there by the keyboard, it leaves that view.
  const stateCard = cardSaying(page, "Alice set the state to Todo");
  await stateCard.hover();
  await stateCard.getByRole("button", { name: `Archive ${ref}`, exact: true }).click();
  await expect(toast(page, "Notification marked as archived")).toBeVisible();
  await expect(stateCard).toHaveCount(0);
  expect(await states("state")).toEqual([{ read: false, archived: true, snoozed: false, by_receiver: true }]);
  await toggleView(page, "Show archived");
  await expect(stateCard).toHaveCount(1);
  await expect(stateCard.getByRole("button", { name: `Snooze ${ref}`, exact: true })).toHaveCount(0);
  await stateCard.getByRole("button", { name: `Unarchive ${ref}`, exact: true }).focus();
  await page.keyboard.press("Enter");
  await expect(toast(page, "Notification unarchived")).toBeVisible();
  await expect(stateCard).toHaveCount(0);
  expect(await states("state")).toEqual([{ read: false, archived: false, snoozed: false, by_receiver: true }]);
  await toggleView(page, "Show archived");

  // Snoozed for 3 days by the keyboard, from the moment chosen: it leaves; in the snoozed view, till then, with no
  // archive, and its snooze ended there, snoozed_till null, it leaves that view.
  const assigned = cardSaying(page, "Alice assigned Bob").filter({ hasText: ref });
  const snoozeOf = (card: Locator) => card.getByRole("button", { name: `Snooze ${ref}`, exact: true });
  await snoozeOf(assigned).focus();
  const before = Date.now();
  const snoozed = await sentTo(page, "PATCH", `/api/v0/notifications/${ids.assigned}`, () =>
    chooseByKeyboard(page, snoozeOf(assigned), "3 days")
  );
  const till = Date.parse((snoozed.body as { snoozed_till: string }).snoozed_till);
  expect(till - before).toBeGreaterThanOrEqual(3 * 24 * 60 * 60 * 1000 - 60_000);
  expect(till - Date.now()).toBeLessThanOrEqual(3 * 24 * 60 * 60 * 1000);
  await expect(toast(page, "Notification snoozed")).toBeVisible();
  await expect(assigned).toHaveCount(0);
  expect(await states("assigned")).toEqual([{ read: false, archived: false, snoozed: true, by_receiver: true }]);
  await toggleView(page, "Show snoozed");
  await expect(assigned).toContainText("Till ");
  await expect(assigned.getByRole("button", { name: `Archive ${ref}`, exact: true })).toHaveCount(0);
  await assigned.hover();
  await snoozeOf(assigned).click();
  await expect(page.getByRole("menuitem").first()).toHaveText("Unsnooze");
  const unsnoozed = await sentTo(page, "PATCH", `/api/v0/notifications/${ids.assigned}`, () =>
    page.getByRole("menuitem", { name: "Unsnooze", exact: true }).click()
  );
  expect(unsnoozed.body).toEqual({ snoozed_till: null });
  await expect(toast(page, "Notification unsnoozed")).toBeVisible();
  await expect(assigned).toHaveCount(0);
  expect(await states("assigned")).toEqual([{ read: false, archived: false, snoozed: false, by_receiver: true }]);
  await toggleView(page, "Show snoozed");

  await dismissToasts(page);

  // His own time: tomorrow, 12:30 AM, half past midnight in Los Angeles, as the time's button says it.
  const commented = cardSaying(page, "Alice commented");
  await commented.hover();
  await snoozeOf(commented).click();
  await page.getByRole("menuitem", { name: "Custom", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Customize snooze time", exact: true });
  await expect(dialog.getByRole("heading", { name: "Customize snooze time", exact: true })).toBeVisible();
  const tomorrow = zonedDay(1, ZONE);
  await pickDay(page, dialog, tomorrow);
  await dialog.getByRole("button", { name: "Pick a time", exact: true }).click();
  await page.getByRole("option", { name: "12:30 AM", exact: true }).click();
  await expect(dialog.getByRole("button", { name: "Time, 12:30 AM", exact: true })).toBeVisible();
  const custom = await sentTo(page, "PATCH", `/api/v0/notifications/${ids.comment}`, () =>
    dialog.getByRole("button", { name: "Snooze", exact: true }).click()
  );
  const midnight = zonedMoment(tomorrow, "00:30", ZONE);
  expect(custom.body).toEqual({ snoozed_till: midnight });
  await expect(dialog).toHaveCount(0);
  await expect(toast(page, "Notification snoozed")).toBeVisible();
  await expect(commented).toHaveCount(0);
  expect(await snoozedTill(db, ids.comment)).toBe(midnight);
  await toggleView(page, "Show snoozed");
  const [year, month, day] = tomorrow.split("-");
  const shownDay = `${new Date(Date.UTC(Number(year), Number(month) - 1, 1)).toLocaleString("en-US", { month: "short", timeZone: "UTC" })} ${day}, ${year}`;
  await expect(commented).toContainText(`Till ${shownDay}, 12:30 AM`);
  await toggleView(page, "Show snoozed");

  // His clock a day behind nerve's, yesterday's noon in Los Angeles: today's 12:30 PM, the first half hour left, is
  // past for nerve, which says so under the time, the dialog open.
  const created = cardSaying(page, "Alice created the work item").filter({ hasText: ref });
  await page.clock.setSystemTime(zonedMoment(zonedDay(-1, ZONE), "12:00", ZONE));
  await created.hover();
  await snoozeOf(created).click();
  await page.getByRole("menuitem", { name: "Custom", exact: true }).click();
  await pickDay(page, dialog, zonedDay(-1, ZONE));
  await dialog.getByRole("button", { name: "Pick a time", exact: true }).click();
  await expect(page.getByRole("option").first()).toHaveText("12:30 PM");
  await page.getByRole("option", { name: "12:30 PM", exact: true }).click();
  const past = await answerTo(page, "PATCH", `/api/v0/notifications/${ids.created}`, () =>
    dialog.getByRole("button", { name: "Snooze", exact: true }).click()
  );
  expect(past.status()).toBe(422);
  const time = dialog.getByRole("button", { name: "Time, 12:30 PM", exact: true });
  await expect(time).toHaveAccessibleDescription("Must be in the future");
  await expect(time).toHaveAttribute("aria-invalid", "true");
  await expect(time).toBeFocused();
  await expect(dialog).toHaveCount(1);
  await page.clock.setSystemTime(new Date());
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(dialog).toHaveCount(0);
  // nerve wrote nothing of it: its row as it was sent
  expect(await states("created")).toEqual([{ read: false, archived: false, snoozed: false, by_receiver: false }]);

  // Snoozed till two seconds on (through the API): gone at the refresh, and listed and counted again at the refresh
  // after its end.
  const soon = new Date(Date.now() + 2000).toISOString();
  expect((await notificationUpdated(api, bob.token, ids.created, { snoozed_till: soon })).status).toBe(200);
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await expect(created).toHaveCount(0);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 5 unread");
  await unreadReach(api, bob.token, slug, { all: 6, mentions: 1 });
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await expect(created).toHaveCount(1);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 6 unread");

  // Mark all read in the all tab: the state's, archived again, the comment's, snoozed, and the mention stay unread;
  // nerve marked the five others, as the toast says.
  await stateCard.hover();
  await stateCard.getByRole("button", { name: `Archive ${ref}`, exact: true }).click();
  await expect(stateCard).toHaveCount(0);
  const marked = await answerTo(page, "POST", `/api/v0/workspaces/${slug}/notifications/mark-all-read`, () =>
    page.getByRole("button", { name: "Mark all as read", exact: true }).click()
  );
  expect(await marked.json()).toEqual({ marked: 5 });
  await expect(toast(page, "5 notifications marked as read")).toBeVisible();
  await expect(tabs.nth(0)).toHaveAccessibleName("All");
  // (the mention as it was sent; the creation's snooze over, its moment kept)
  expect(await states("state", "comment", "mention", "created", "assigned", "priority")).toEqual([
    { read: false, archived: true, snoozed: false, by_receiver: true },
    { read: false, archived: false, snoozed: true, by_receiver: true },
    { read: false, archived: false, snoozed: false, by_receiver: false },
    { read: true, archived: false, snoozed: true, by_receiver: true },
    ...[1, 2].map(() => ({ read: true, archived: false, snoozed: false, by_receiver: true })),
  ]);

  // The mention, opened: read, and its work item shown, read from nerve.
  await tabs.nth(1).click();
  const peekRead = page.waitForResponse((r) => new URL(r.url()).pathname === `/api/v0/issues/${fix.id}`);
  await cardSaying(page, "Alice mentioned you in a comment").getByRole("button").first().click();
  expect((await peekRead).status()).toBe(200);
  await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toBeVisible();
  await expect
    .poll(() => states("mention"))
    .toEqual([{ read: true, archived: false, snoozed: false, by_receiver: true }]);

  // Removed from API, the page open: his archive of Rate limit's card is refused as not found, said, the card gone.
  await tabs.nth(0).click();
  const membership = await projectMembershipOf(api, alice.token, apiProject.id, bob.id);
  expect(await projectMemberWrites(api, apiProject.id).remove(alice.token, membership)).toEqual({ status: 204 });
  const rate = cardSaying(page, "Alice assigned Bob").filter({ hasText: "Rate limit" });
  await rate.hover();
  await rate.getByRole("button", { name: "Archive API-1 Rate limit", exact: true }).click();
  await expect(toast(page, "The notification does not exist, or you cannot see it.")).toBeVisible();
  await expect(rate).toHaveCount(0);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`422 PATCH /api/v0/notifications/${ids.created}`, `404 PATCH /api/v0/notifications/${idOf(rates, "assignees")}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 422, 404) });
});

/** Picks day, YYYY-MM-DD, in the calendar of the date's button of dialog, by the calendar's year and month. */
async function pickDay(page: Page, dialog: Locator, day: string): Promise<void> {
  await dialog.getByRole("button", { name: "Pick a date", exact: true }).click();
  const calendar = page.getByRole("dialog", { name: "Pick a date", exact: true });
  const [year = "", month = ""] = day.split("-");
  await calendar.getByRole("combobox", { name: "Choose the Year" }).selectOption(year);
  await calendar.getByRole("combobox", { name: "Choose the Month" }).selectOption(String(Number(month) - 1));
  await calendar.locator(`[data-day="${day}"] button`).first().click();
}
