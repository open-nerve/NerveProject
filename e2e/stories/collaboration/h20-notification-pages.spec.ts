import type { Locator, Page } from "@playwright/test";

import { expectOutlinesWhole, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { createIssueComment } from "../../fixtures/comments";
import { recordApi, tabKey, tabStops, toast } from "../../fixtures/cycle-pages";
import { deferred } from "../../fixtures/deferred";
import {
  closedByBackdropClick,
  closedByEscape,
  enabledWithin,
  focusInDialog,
  modalDialog,
  pageBehindHidden,
} from "../../fixtures/dialogs";
import { createIssue, updateIssue } from "../../fixtures/issues";
import { chooseByKeyboard } from "../../fixtures/menus";
import {
  cardItems,
  cardSaying,
  expectActionsShown,
  listRead,
  notificationWorld,
  notificationsPath,
  unreadReach,
  zonedDay,
} from "../../fixtures/notification-pages";
import { notificationsSent } from "../../fixtures/notifications";
import { expectListBesideButton, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H20's page version for the notifications page's keys and names (M7 design 2, H20; P4 spec 2.6, 2.7, 2.11, R16,
// re-93 I3), in each browser: the tabs say their unread and which is chosen, the arrow keys moving among them; a card
// is a main button, Enter opening it, and its actions beside it, each reached by Tab, named by its row and shown as the
// card holds the focus; the type filter and the view menu are popovers of plain buttons, Tab moving among them, Enter
// and Space toggling each once, each saying whether it is pressed, Escape closing the menu and giving the focus back to
// its button; the snooze menu moves by the arrow keys, Enter choosing, Escape closing it onto its button. The custom
// snooze dialog is named, holds the focus, the page behind it inert; its missing fields say so under them, described
// by it, the first taking the focus; its time is a select of the keyboard, its list beside it, named by its field and
// the half hour picked; Escape and Cancel close it, the focus back on the row's snooze button, but not while nerve
// snoozes. A card that leaves the list by its own action gives the focus to the next card, else the previous, else the
// list; one that stays has it back on the action once nerve has answered, a refused preset snooze's on the menu's
// button. Each control's focus outline shows whole. Every request is /api/v0's; none fails but the one a proxy
// refuses. Late answers are h20-notification-late-pages.spec.ts's.

test.use({ timezoneId: "America/Los_Angeles" });

/** The main button of card: what it shows, which opens it. */
function mainOf(card: Locator): Locator {
  return card.getByRole("button").first();
}

/** Presses key on page, then checks that target has the focus. */
async function pressTo(page: Page, key: string, target: Locator): Promise<void> {
  await page.keyboard.press(key);
  await expect(target).toBeFocused();
}

test("H20 (page): bob's notifications page by the keyboard: tabs that say their unread and which is chosen; cards whose main button Enter opens and whose actions Tab reaches, each named by its row; the type filter and the view menu, Tab among their pressed buttons, Enter and Space toggling once, Escape back on their button; the snooze menu by the arrow keys; the custom snooze dialog named, holding the focus, saying what it misses, its time a select of the keyboard beside its button, closed by Escape and Cancel back onto the row's snooze button but not while nerve snoozes; a card leaving by its own action gives the focus to the next, the previous, or the list; whole focus outlines", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await notificationWorld(api, testInfo);
  // Fix, alice's, assigned to bob: its creation, its assignment, her comment and its priority, in his all tab; two
  // mentions of him, in its description and in a comment, in his mentions tab.
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix", assignee_ids: [bob.id] });
  await notificationsSent(api, bob.token, slug, 2);
  await createIssueComment(api, alice.token, fix.id, "<p>Looks good</p>");
  await notificationsSent(api, bob.token, slug, 3);
  await updateIssue(api, alice.token, fix.id, { priority: "high" });
  const mention = `<mention-component entity_name="user_mention" entity_identifier="${bob.id}"></mention-component>`;
  await updateIssue(api, alice.token, fix.id, { description_html: `<p>See ${mention}</p>` });
  await createIssueComment(api, alice.token, fix.id, `<p>Hi</p>${mention}`);
  await unreadReach(api, bob.token, slug, { all: 4, mentions: 2 });
  const ref = `WEB-${fix.sequence_id} Fix`;

  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  await page.goto(notificationsPath(slug));
  const key = tabKey(browserName);
  const cards = cardItems(page);
  await expect(cards).toHaveCount(4);
  const [priority, commented, assigned, created] = [
    cardSaying(page, "Alice set the priority to High"),
    cardSaying(page, "Alice commented"),
    cardSaying(page, "Alice assigned Bob"),
    cardSaying(page, "Alice created the work item"),
  ];

  // The inbox's link and the tabs say the unread; the chosen tab says it is; the arrow keys move the choice.
  await expect(page.getByRole("link", { name: "Inbox, 6 unread", exact: true })).toBeVisible();
  const [all, mentions] = [
    page.getByRole("tab", { name: "All, 4 unread", exact: true }),
    page.getByRole("tab", { name: "Mentions, 2 unread", exact: true }),
  ];
  await expect(all).toHaveAttribute("aria-selected", "true");
  await expect(mentions).toHaveAttribute("aria-selected", "false");
  await all.focus();
  await pressTo(page, "ArrowRight", mentions);
  await expect(mentions).toHaveAttribute("aria-selected", "true");
  await expect(cards).toHaveCount(2);
  await pressTo(page, "ArrowLeft", all);
  await expect(all).toHaveAttribute("aria-selected", "true");
  await expect(cards).toHaveCount(4);

  // A card: its main button, then its actions, each named by its row, shown as the card holds the focus; then the
  // next card's main button.
  await mainOf(priority).focus();
  await expectActionsShown(priority, `Mark ${ref} read`, true);
  expect(await tabStops(page, page.getByRole("tabpanel"), key, 4)).toEqual([
    `Mark ${ref} read`,
    `Archive ${ref}`,
    `Snooze ${ref}`,
    expect.stringContaining("Alice commented"),
  ]);
  await expectOutlinesWhole(
    page,
    key,
    all,
    mainOf(priority),
    ...[`Mark ${ref} read`, `Archive ${ref}`, `Snooze ${ref}`].map((name) =>
      priority.getByRole("button", { name, exact: true })
    ),
    ...["Mark all as read", "Refresh", "Inbox Filters", "Display"].map((name) =>
      page.getByRole("button", { name, exact: true })
    )
  );

  // The type filter: Enter opens it; Tab moves among its buttons, each saying whether it is pressed; Enter presses
  // one, once, the list read by it (none created by him); Space lets it go, once; Escape closes it, back on its
  // button.
  const filters = page.getByRole("button", { name: "Inbox Filters", exact: true });
  const button = (name: string) => page.getByRole("button", { name, exact: true });
  const [mine, made, followed] = [button("Assigned to me"), button("Created by me"), button("Subscribed by me")];
  await filters.focus();
  await page.keyboard.press("Enter");
  await pressTo(page, key, mine);
  await pressTo(page, key, made);
  await pressTo(page, key, followed);
  await pressTo(page, `Shift+${key}`, made);
  await expect(made).toHaveAttribute("aria-pressed", "false");
  const typed = recordApi(page);
  await page.keyboard.press("Enter");
  await expect(made).toHaveAttribute("aria-pressed", "true");
  await expect(
    page.getByText("Updates for your subscribed work items will appear here", { exact: true })
  ).toBeVisible();
  await expect.poll(typed).toEqual({
    pending: 0,
    requests: [listRead(slug, "tab=all&type=created&archived=false&snoozed=false&limit=100")],
  });
  await expect(made).toBeFocused();
  await page.keyboard.press(" ");
  await expect(made).toHaveAttribute("aria-pressed", "false");
  await expect(cards).toHaveCount(4);
  await page.keyboard.press("Escape");
  await expect(made).toHaveCount(0);
  await expect(filters).toBeFocused();

  // The view menu: Space opens it; the archived pressed by Space, then the snoozed by Enter, which lets the archived
  // go; Escape closes it, back on its button; the snoozed let go by the pointer.
  const display = page.getByRole("button", { name: "Display", exact: true });
  const [unreadView, archivedView, snoozedView] = [
    button("Show unread"),
    button("Show archived"),
    button("Show snoozed"),
  ];
  await display.focus();
  await page.keyboard.press(" ");
  await pressTo(page, key, unreadView);
  await pressTo(page, key, archivedView);
  await page.keyboard.press(" ");
  await expect(archivedView).toHaveAttribute("aria-pressed", "true");
  await pressTo(page, key, snoozedView);
  await page.keyboard.press("Enter");
  await expect(snoozedView).toHaveAttribute("aria-pressed", "true");
  await expect(archivedView).toHaveAttribute("aria-pressed", "false");
  await expect(unreadView).toHaveAttribute("aria-pressed", "false");
  await page.keyboard.press("Escape");
  await expect(snoozedView).toHaveCount(0);
  await expect(display).toBeFocused();
  await display.click();
  await snoozedView.click();
  await page.keyboard.press("Escape");
  await expect(cards).toHaveCount(4);

  // The snooze menu: Enter opens it, its presets and Custom its items; Escape closes it, back on its button; Custom
  // chosen by the arrow keys and Enter opens the dialog.
  const snooze = priority.getByRole("button", { name: `Snooze ${ref}`, exact: true });
  await snooze.focus();
  await page.keyboard.press("Enter");
  const menu = page.getByRole("menu");
  await expect(menu.getByRole("menuitem")).toHaveText(["1 day", "3 days", "5 days", "1 week", "2 weeks", "Custom"]);
  await page.keyboard.press("Escape");
  await expect(menu).toHaveCount(0);
  await expect(snooze).toBeFocused();
  await chooseByKeyboard(page, snooze, "Custom");

  // The dialog: named, the focus within it, the page behind it inert. Sent with nothing picked, it says so under each
  // field, which it describes, the date taking the focus; Escape closes it, back on the row's snooze button.
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Customize snooze time");
  expect(await focusInDialog(page)).toBe(true);
  expect(await pageBehindHidden(page)).toBe(true);
  const date = dialog.getByRole("button", { name: "Pick a date", exact: true });
  await dialog.getByRole("button", { name: "Snooze", exact: true }).click();
  await Promise.all(
    [date, dialog.getByRole("button", { name: "Pick a time", exact: true })].map(async (field) => {
      await expect(field).toHaveAttribute("aria-invalid", "true");
      await expect(field).toHaveAccessibleDescription("This field is required");
    })
  );
  await expect(date).toBeFocused();
  expect(await closedByEscape(page)).toBe(true);
  await expect(snooze).toBeFocused();
  // Again, closed by Cancel: back on the snooze button.
  await chooseByKeyboard(page, snooze, "Custom");
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(dialog).toHaveCount(0);
  await expect(snooze).toBeFocused();

  // Again: tomorrow, by the calendar; the time by the keyboard, its list beside its button, 12:30 AM the second half
  // hour, the select named by its field and it. Sent, nerve's answer held: no Escape, no click outside, no Cancel
  // closes it, the focus within it. Let through: closed, said, the card gone, the focus on the next card's main
  // button.
  await chooseByKeyboard(page, snooze, "Custom");
  const tomorrow = zonedDay(1, "America/Los_Angeles");
  await date.click();
  const calendar = page.getByRole("dialog", { name: "Pick a date", exact: true });
  const [year = "", month = ""] = tomorrow.split("-");
  await calendar.getByRole("combobox", { name: "Choose the Year" }).selectOption(year);
  await calendar.getByRole("combobox", { name: "Choose the Month" }).selectOption(String(Number(month) - 1));
  await calendar.locator(`[data-day="${tomorrow}"] button`).first().click();
  const time = dialog.getByRole("button", { name: "Pick a time", exact: true });
  await time.focus();
  await page.keyboard.press("Enter");
  await expectListBesideButton(page, time);
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  const picked = dialog.getByRole("button", { name: "Time, 12:30 AM", exact: true });
  await expect(picked).toBeFocused();
  await expect(page.getByRole("listbox")).toHaveCount(0);
  const note = (await notificationsSent(api, bob.token, slug, 4)).find((n) => n.data.change.field === "priority");
  const held = await sentHeld(page, "PATCH", `/api/v0/notifications/${note?.id}`, () =>
    dialog.getByRole("button", { name: "Snooze", exact: true }).click()
  );
  let released = false;
  try {
    expect(await closedByEscape(page)).toBe(false);
    expect(await closedByBackdropClick(page)).toBe(false);
    expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel", exact: true }))).toBe(false);
    expect(await focusInDialog(page)).toBe(true);
    expect((await held.release()).status()).toBe(200);
    released = true;
  } finally {
    if (!released) await held.release();
  }
  await expect(dialog).toHaveCount(0);
  await expect(toast(page, "Notification snoozed")).toBeVisible();
  await expect(priority).toHaveCount(0);
  await expect(mainOf(commented)).toBeFocused();

  // The comment's card snoozed for a day by the keyboard, nerve out of reach meanwhile (a proxy's 503, held): the menu
  // gone, its button disabled while the snooze is out; refused, the card stays, the focus back on its snooze button
  // once it is enabled again, never on the page's body (whole-branch O4).
  const commentNote = (await notificationsSent(api, bob.token, slug, 3)).find((n) => n.data.change.field === "comment");
  const commentPath = `/api/v0/notifications/${commentNote?.id}`;
  const unreachable = deferred();
  await page.route(
    (url) => url.pathname === commentPath,
    async (route) => {
      if (route.request().method() !== "PATCH") return route.fallback();
      await unreachable.promise;
      await route.fulfill({
        status: 503,
        contentType: "application/problem+json",
        body: JSON.stringify({ type: "about:blank", title: "Busy", status: 503, code: "server_busy" }),
      });
    },
    { times: 1 }
  );
  const snoozeComment = commented.getByRole("button", { name: `Snooze ${ref}`, exact: true });
  await snoozeComment.focus();
  await chooseByKeyboard(page, snoozeComment, "1 day");
  await expect(snoozeComment).toBeDisabled();
  await expect(page.getByRole("menu")).toHaveCount(0);
  // two frames on, Headless UI has tried to give the focus back to the menu's button, disabled: nothing has it
  await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  await expect(page.locator("body")).toBeFocused();
  unreachable.resolve();
  await expect(toast(page, "The server is busy. Please try again later.")).toBeVisible();
  await expect(snoozeComment).toBeEnabled();
  await expect(commented).toHaveCount(1);
  await expect(snoozeComment).toBeFocused();

  // The comment's card marked read by the keyboard: its button, disabled while nerve marks it, has the focus back,
  // named for the way back. Archived by the keyboard, the card gives the focus to the next, the assignment's; the
  // last, the creation's, to the one before it; the one left, to the list.
  await mainOf(commented).focus();
  await pressTo(page, key, commented.getByRole("button", { name: `Mark ${ref} read`, exact: true }));
  await page.keyboard.press("Enter");
  await expect(commented.getByRole("button", { name: `Mark ${ref} unread`, exact: true })).toBeFocused();
  await pressTo(page, key, commented.getByRole("button", { name: `Archive ${ref}`, exact: true }));
  await page.keyboard.press("Enter");
  await expect(commented).toHaveCount(0);
  await expect(mainOf(assigned)).toBeFocused();
  await created.getByRole("button", { name: `Archive ${ref}`, exact: true }).focus();
  await page.keyboard.press("Enter");
  await expect(created).toHaveCount(0);
  await expect(mainOf(assigned)).toBeFocused();
  await assigned.getByRole("button", { name: `Archive ${ref}`, exact: true }).focus();
  await page.keyboard.press("Enter");
  await expect(assigned).toHaveCount(0);
  const empty = page.getByText("Updates for your subscribed work items will appear here", { exact: true });
  await expect(page.locator('div[tabindex="-1"]', { has: empty })).toBeFocused();

  // The mentions tab: Enter on a card's main button opens its work item beside the list.
  await mentions.click();
  const first = cards.first();
  await mainOf(first).focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toBeVisible();

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`503 PATCH ${commentPath}`], [], []]);
  await expectQuietConsole(page, watch, {
    errors:
      browserName === "firefox"
        ? []
        : ["Failed to load resource: the server responded with a status of 503 (Service Unavailable)"],
  });
});
