/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createLabel } from "../../fixtures/api";
import { intakeRowOf } from "../../fixtures/assert/intake";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { createIssueComment } from "../../fixtures/comments";
import { failedLoads, recordApi, seenBy, tabKey } from "../../fixtures/cycle-pages";
import { shownWithin } from "../../fixtures/dialogs";
import { submitIntakeIssue, triageIntakeIssue } from "../../fixtures/intake";
import { cardButton, embedClose, listedIds, listRegion, opened, previewOf } from "../../fixtures/intake-embed";
import { expectLanded, focusedName, NARROW, optionsOf, tabTo } from "../../fixtures/intake-focus";
import {
  expectHeaderOf,
  expectTabs,
  identifierOf,
  intakeDetail,
  intakeEditPath,
  intakeRow,
  intakeWorld,
  selecting,
  shownStatus,
  submitNamed,
  triagePath,
} from "../../fixtures/intake-pages";
import { detailPath, expectNothingSelected, notificationsPath } from "../../fixtures/notification-pages";
import { notificationsSent } from "../../fixtures/notifications";
import { answerTo, expectPopupBesideButton, moveWithinApp } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H10's page version, a work item of a project's intake embedded in the notifications page (M7 design 2, 3.5; P5b
// spec 2.5, R16–R18, R26, R27, R30; re-95 I3, O5, O7): a notification of one not accepted (pending, snoozed, declined)
// embeds the intake's detail in the preview, its header the intake's (its status, copy, open, the triages its caller is
// offered) and a Close, never previous or next, ↑ and ↓ stepping nowhere; its labels and assignees show, and the accept
// modal's state has its default, the embed outside any project's pages. Triaged there, the selection stays, the list
// does not change, and the embed shows the new status, the focus on its primary action, else on its region. One
// accepted is peeked at. Every leaving of the notification (the embed's Close, its deletion, the peek's close) clears
// the selection, the focus on the card's main button, or, the card no longer listed, on the list. The embed reads the
// work item twice over (the preview's read that judges it, then its own: P5b R16, re-95 O1), the peek once; each fetch
// reads it twice, its sub-work items' read checking the parent again (sub_issues.store.ts:99, since M4). H20 (page):
// the browse page of a work item in the intake gives way to the intake page selecting it, in its tab (R14), Back
// never returning to it.

/** The text of a notification of a work item's assignment to Alice by Bob. */
const BOB_ASSIGNED_ALICE = "Bob assigned Alice";

test("H10 (page): alice, Web's admin, opens the notifications of three pending work items of its intake: each embeds its detail, the work item read four times; its header offers the triages and a Close, no previous or next, ↑ and ↓ stepping nowhere, its menu and its assignees beside their buttons, its label and assignee shown; declined by the keyboard, snoozed by the pointer and accepted on a narrow screen, through the modal and its default state, each stays selected, the list unchanged, the embed saying its new status, the focus on its primary action or its region", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const bug = await createLabel(api, alice.token, web.id, { name: "Bug" });
  const forAlice = { assignee_ids: [alice.id] };
  const dark = await submitIntakeIssue(api, bob.token, web.id, { name: "Dark mode", ...forAlice });
  const slow = await submitIntakeIssue(api, bob.token, web.id, { name: "Slow search", ...forAlice });
  const crash = await submitIntakeIssue(api, bob.token, web.id, {
    name: "Crash on save",
    ...forAlice,
    label_ids: [bug.id],
  });
  await notificationsSent(api, alice.token, slug, 6);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const sent = recordApi(page);
  await page.goto(notificationsPath(slug));
  const detail = intakeDetail(page);

  // Crash on save: embedded, read four times; pending, its triages, copy, open (the intake page, selecting it), its
  // menu, and Close; no plain peek, no previous or next; its label and its assignee shown.
  expect(await opened(page, cardButton(page, BOB_ASSIGNED_ALICE, crash), crash, { gets: 4, versions: true })).toBe(4);
  await expectHeaderOf(page, crash);
  await expect(shownStatus(page, "Pending")).toBeVisible();
  await expect(embedClose(page)).toBeVisible();
  await Promise.all(
    ["Accept", "Decline", "Copy work item link", "Options"].map((name) =>
      expect(detail.getByRole("button", { name, exact: true })).toBeVisible()
    )
  );
  await expect(detail.getByRole("link", { name: "Open work item", exact: true })).toHaveAttribute(
    "href",
    `/${slug}/projects/${web.id}/intake?currentTab=open&inboxIssueId=${crash.id}`
  );
  await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toHaveCount(0);
  await expect(previewOf(page).getByRole("button", { name: /^(Previous|Next) work item$/ })).toHaveCount(0);
  await expect(detail.getByRole("button", { name: "Bug", exact: true })).toBeVisible();
  const assignees = detail.getByRole("button", { name: "Add assignees Alice", exact: true });
  await expect(assignees).toBeVisible();
  const ids = await listedIds(page);

  // Its menu and its assignees' list open beside their buttons (shared popovers are fixed: b1aeccb1); Escape closes
  // each, the menu's back on its button. ↑ and ↓ on the embed step to no other work item.
  const options = detail.getByRole("button", { name: "Options", exact: true });
  await options.click();
  const menu = page.getByRole("menu");
  await expectPopupBesideButton(menu, options);
  await expect(menu.getByRole("menuitem")).toHaveText(["Snooze", "Mark as duplicate", "Delete"]);
  await page.keyboard.press("Escape");
  await expect(menu).toHaveCount(0);
  await expectLanded(options);
  await assignees.click();
  const list = page.getByRole("listbox");
  await expectPopupBesideButton(list, assignees);
  await page.keyboard.press("Escape");
  await expect(list).toHaveCount(0);
  await detail.focus();
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("ArrowUp");
  expect(await shownWithin(detail.getByRole("heading", { name: /^WEB-[12]$/ }))).toBe(false);
  await expectHeaderOf(page, crash);

  // Declined by the keyboard: still selected, Declined, the list as it was, the focus on the embed's region.
  await detail.getByRole("button", { name: "Decline", exact: true }).focus();
  await page.keyboard.press("Enter");
  const declining = page.getByRole("dialog", { name: "Decline work item", exact: true });
  await tabTo(page, tabKey(browserName), declining.getByRole("button", { name: "Decline", exact: true }));
  const declined = await answerTo(page, "POST", triagePath(crash), () => page.keyboard.press("Enter"));
  expect(declined.status()).toBe(200);
  await expect.poll(() => focusedName(page)).toBe("region Intake work item");
  await expectLanded(detail);
  await expectHeaderOf(page, crash);
  await expect(shownStatus(page, "Declined")).toBeVisible();
  expect(await listedIds(page)).toEqual(ids);
  await expect.poll(() => sent().pending).toBe(0);

  // Slow search snoozed by the pointer, from the menu: still selected, its snooze's days to go, the focus on Accept.
  expect(await opened(page, cardButton(page, BOB_ASSIGNED_ALICE, slow), slow, { gets: 4, versions: true })).toBe(4);
  await expectHeaderOf(page, slow);
  await options.click();
  await page.getByRole("menuitem", { name: "Snooze", exact: true }).click();
  const snoozing = page.getByRole("dialog", { name: "Snooze work item", exact: true });
  const snoozed = await answerTo(page, "POST", triagePath(slow), () =>
    snoozing.getByRole("button", { name: "Snooze", exact: true }).click()
  );
  expect(snoozed.status()).toBe(200);
  await expect.poll(() => focusedName(page)).toBe("button Accept");
  await expectLanded(detail.getByRole("button", { name: "Accept", exact: true }));
  await expectHeaderOf(page, slow);
  await expect(detail.getByText(/^\d+ days? to go$/).filter({ visible: true })).toBeVisible();
  expect(await listedIds(page)).toEqual(ids);
  await expect.poll(() => sent().pending).toBe(0);

  // On a narrow screen, Dark mode accepted from the narrow header's menu, no list toggle beside it, through the modal,
  // its state Web's default: still selected, Accepted, the list as it was, the focus on the embed's region.
  await page.setViewportSize(NARROW);
  expect(await opened(page, cardButton(page, BOB_ASSIGNED_ALICE, dark), dark, { gets: 4, versions: true })).toBe(4);
  await expect(shownStatus(page, "Pending")).toBeVisible();
  await expect(page.getByRole("button", { name: "Intake list", exact: true })).toHaveCount(0);
  await optionsOf(page).click();
  await page.getByRole("menuitem", { name: "Accept", exact: true }).click();
  const moving = page.getByRole("dialog", { name: `Move ${identifierOf(dark)} to project work items`, exact: true });
  await expect(moving.getByRole("button", { name: "State: Backlog", exact: true })).toBeVisible();
  const accepted = await answerTo(page, "POST", triagePath(dark), () =>
    moving.getByRole("button", { name: "Add to project", exact: true }).click()
  );
  expect(accepted.status()).toBe(200);
  await expect.poll(() => focusedName(page)).toBe("region Intake work item");
  await expectLanded(detail);
  await expect(moving).toHaveCount(0);
  await expect(shownStatus(page, "Accepted")).toBeVisible();
  await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toHaveCount(0);
  expect(await listedIds(page)).toEqual(ids);
  await expect.poll(() => sent().pending).toBe(0);
  expect(await Promise.all([crash, slow, dark].map(async (issue) => (await intakeRowOf(db, issue.id)).status))).toEqual(
    ["declined", "snoozed", "accepted"]
  );

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H10 (page): bob, Web's member, opens the notifications of a pending, a declined and an accepted work item: the two embedded, read four times, their headers offering him copy and open alone; the accepted one peeked at, read twice, its close leaving the focus on its card's main button; gus, its guest, opens the notification of his own, embedded, offered copy and open alone", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const { alice, bob, gus, slug, web } = await intakeWorld(api, testInfo);
  const forBob = (name: string) => submitIntakeIssue(api, alice.token, web.id, { name, assignee_ids: [bob.id] });
  const spam = await forBob("Buy now");
  const dark = await forBob("Dark mode");
  const crash = await forBob("Crash on save");
  await triageIntakeIssue(api, alice.token, spam.id, { status: "declined" });
  await triageIntakeIssue(api, alice.token, dark.id, { status: "accepted" });
  const idea = await submitIntakeIssue(api, gus.token, web.id, { name: "Gus's idea" });
  await createIssueComment(api, alice.token, idea.id, "<p>Thanks</p>");
  await Promise.all([notificationsSent(api, bob.token, slug, 6), notificationsSent(api, gus.token, slug, 1)]);

  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  await page.goto(notificationsPath(slug));
  const detail = intakeDetail(page);
  /** Checks that the embedded work item issue, its status status, offers bob copy, open and Close alone. */
  const readOnly = async (issue: typeof crash, status: string, tab: "open" | "closed") => {
    await expectHeaderOf(page, issue);
    await expect(shownStatus(page, status)).toBeVisible();
    await expect(embedClose(page)).toBeVisible();
    await expect(detail.getByRole("button", { name: "Copy work item link", exact: true })).toBeVisible();
    await expect(detail.getByRole("link", { name: "Open work item", exact: true })).toHaveAttribute(
      "href",
      `/${slug}/projects/${web.id}/intake?currentTab=${tab}&inboxIssueId=${issue.id}`
    );
    await expect(detail.getByRole("button", { name: /^(Accept|Decline|Options)$/ })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Close the peek view", exact: true })).toHaveCount(0);
  };

  // Crash on save, pending, and Buy now, declined: each embedded, read four times, nothing to triage.
  const assigned = "Alice assigned Bob";
  expect(await opened(page, cardButton(page, assigned, crash), crash, { gets: 4, versions: true })).toBe(4);
  await readOnly(crash, "Pending", "open");
  expect(await opened(page, cardButton(page, assigned, spam), spam, { gets: 4, versions: true })).toBe(4);
  await readOnly(spam, "Declined", "closed");

  // Dark mode, accepted: peeked at, no intake detail, read twice (the peek's read joins the preview's: SWR dedupes);
  // its close clears the selection, the focus on its card's main button (O7).
  const darkCard = cardButton(page, assigned, dark);
  expect(await opened(page, darkCard, dark, { gets: 2, versions: true })).toBe(2);
  const close = page.getByRole("button", { name: "Close the peek view", exact: true });
  await expect(close).toBeVisible();
  await expect(detail).toHaveCount(0);
  await close.click();
  await expectNothingSelected(page);
  await expect.poll(() => darkCard.evaluate((button) => button === document.activeElement)).toBe(true);
  await expectLanded(darkCard);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);

  // gus, Web's guest: his own, embedded, read four times (no description's versions, which a guest does not read).
  await seenBy(browser, baseURL ?? "", api, gus.email, notificationsPath(slug), async (other) => {
    const card = cardButton(other, "Alice commented", idea);
    expect(await opened(other, card, idea, { gets: 4, versions: false })).toBe(4);
    await expectHeaderOf(other, idea);
    await expect(shownStatus(other, "Pending")).toBeVisible();
    await expect(embedClose(other)).toBeVisible();
    await expect(intakeDetail(other).getByRole("link", { name: "Open work item", exact: true })).toBeVisible();
    await expect(intakeDetail(other).getByRole("button", { name: /^(Accept|Decline|Options)$/ })).toHaveCount(0);
  });
});

test("H10 (page): alice leaves embedded work items: Close, by the keyboard, clears the selection, the focus on the card's main button; a deletion, the same; Close once the tab no longer lists the card, the focus on the list; Open, on a pending one, goes to the intake page selecting it", async ({
  api,
  browserName,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const forAlice = (name: string) => submitIntakeIssue(api, bob.token, web.id, { name, assignee_ids: [alice.id] });
  const one = await forAlice("One");
  const two = await forAlice("Two");
  const three = await forAlice("Three");
  const four = await forAlice("Four");
  await notificationsSent(api, alice.token, slug, 8);

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  await page.goto(notificationsPath(slug));
  const card = (issue: typeof one) => cardButton(page, BOB_ASSIGNED_ALICE, issue);
  const reads = { gets: 4, versions: true };

  // One, closed by Enter on the embed's Close: nothing selected, the focus on One's card.
  expect(await opened(page, card(one), one, reads)).toBe(4);
  await expectHeaderOf(page, one);
  await embedClose(page).focus();
  await page.keyboard.press("Enter");
  await expectNothingSelected(page);
  await expect.poll(() => focusedName(page)).toContain("WEB-1 One");
  await expectLanded(card(one));

  // Two, deleted from the menu: nothing selected, its card still listed, the focus on it. The deletion's read again,
  // in its turn, finds the work item gone (nerve's 404).
  expect(await opened(page, card(two), two, reads)).toBe(4);
  await optionsOf(page).click();
  await page.getByRole("menuitem", { name: "Delete", exact: true }).click();
  const gone = page.waitForResponse(
    (r) => r.request().method() === "GET" && new URL(r.url()).pathname === `/api/v0/issues/${two.id}`
  );
  const deleted = await answerTo(page, "DELETE", intakeEditPath(two), () =>
    page
      .getByRole("dialog", { name: "Delete work item", exact: true })
      .getByRole("button", { name: "Delete", exact: true })
      .click()
  );
  expect([deleted.status(), (await gone).status()]).toEqual([204, 404]);
  await expectNothingSelected(page);
  await expect.poll(() => focusedName(page)).toContain("WEB-2 Two");
  await expectLanded(card(two));

  // Three, then the mentions tab, which lists none of them: still embedded; closed by the pointer, the focus on the
  // tab's list.
  expect(await opened(page, card(three), three, reads)).toBe(4);
  await page.getByRole("tab", { name: /^Mentions/ }).click();
  await expect(listRegion(page)).toHaveText("Mentions for your work items will appear here");
  await expectHeaderOf(page, three);
  await embedClose(page).click();
  await expectNothingSelected(page);
  await expectLanded(listRegion(page));

  // Four, from the all tab: Open goes to the intake page, its open tab selecting Four.
  await page.getByRole("tab", { name: /^All/ }).click();
  expect(await opened(page, card(four), four, reads)).toBe(4);
  await intakeDetail(page).getByRole("link", { name: "Open work item", exact: true }).click();
  await expect(page).toHaveURL(selecting(four));
  await expect(page).toHaveURL(/[?&]currentTab=open(&|$)/);
  await expect(intakeRow(page, four)).toBeVisible();
  await expectHeaderOf(page, four);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 GET /api/v0/issues/${two.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 404) });
});

test("H20 (page): alice opens the browse page of a pending work item of Web's intake: once its read answers, the intake page takes its place, its open tab selecting it; moved to a declined one's, the closed tab selects it; Back returns to the pending one's intake page, never to a browse page", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const [spam, crash] = await submitNamed(api, bob.token, web.id, ["Buy now", "Crash on save"]);
  await triageIntakeIssue(api, alice.token, spam.id, { status: "declined" });

  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const sent = recordApi(page);
  /** The browse page's read of the work item issue by its identifier, once nerve answers it. */
  const lookedUp = (issue: typeof crash) =>
    page.waitForResponse(
      (r) => new URL(r.url()).pathname === `/api/v0/workspaces/${slug}/issues/${identifierOf(issue)}`
    );

  // Crash on save, pending, by its browse address: the intake page in its place, the open tab selecting it.
  const crashRead = lookedUp(crash);
  await page.goto(detailPath(slug, "WEB", crash.sequence_id));
  expect((await crashRead).status()).toBe(200);
  await expect(page).toHaveURL(selecting(crash));
  await expect(page).toHaveURL(/[?&]currentTab=open(&|$)/);
  await expectTabs(page, "Open, 1 work item", "open");
  await expectHeaderOf(page, crash);
  await expect.poll(() => sent().pending).toBe(0);

  // Buy now, declined, its browse address reached within the app: the closed tab selecting it.
  const spamRead = lookedUp(spam);
  await moveWithinApp(page, detailPath(slug, "WEB", spam.sequence_id));
  expect((await spamRead).status()).toBe(200);
  await expect(page).toHaveURL(selecting(spam));
  await expect(page).toHaveURL(/[?&]currentTab=closed(&|$)/);
  await expectTabs(page, "Open", "closed");
  await expectHeaderOf(page, spam);
  await expect(shownStatus(page, "Declined")).toBeVisible();
  await expect.poll(() => sent().pending).toBe(0);

  // Back: the pending one's intake page, which replaced its browse page; no browse page is left to go back to.
  await page.goBack();
  await expect(page).toHaveURL(selecting(crash));
  await expect(page).not.toHaveURL(/\/browse\//);
  await expectHeaderOf(page, crash);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
