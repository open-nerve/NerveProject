import type { Page } from "@playwright/test";

import { createLabel } from "../../fixtures/api";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { createIssueComment } from "../../fixtures/comments";
import { recordApi, toast } from "../../fixtures/cycle-pages";
import { createIssue, issueSubscribed, updateIssue } from "../../fixtures/issues";
import { APP, WORKSPACE, valued } from "../../fixtures/mounts";
import {
  cardItems,
  cardSaying,
  countRead,
  listRead,
  notificationWorld,
  notificationsPath,
  unreadReach,
} from "../../fixtures/notification-pages";
import { notificationUpdated, notificationsSent } from "../../fixtures/notifications";
import { sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H5's page version (M7 design 2, 3.4; P4 spec 2.2, 2.5, 2.6, 2.11, R7, R9, R10, R21, R25, re-93 m6, m12): a
// receiver's notifications page lists them a hundred at a time. Its two tabs say their unread, 99+ past 99; "Load
// more" reads the next page by nerve's cursor, and Refresh the first page again in place of every page held. The all
// tab's type filter chooses its senders, each a pill, which Clear all clears; the mentions tab never sends a type, and
// keeps the all tab's choice for its return. The view menu lists the unread alone, the archived alone or the snoozed
// alone, the archived and the snoozed never both; a snoozed card offers no archive, an archived one no snooze. Each
// is one read of the list, as its query says it; Mark all read sends the tab and the filters the page holds, the types
// only when there are some. Each tab says when it lists none; a page whose cards were all archived away still offers
// "Load more" while nerve has given a cursor. Opened anew, the page reads the app's start, the workspace's and the
// list's first page alone. The API version is h5-h7-notification-inbox.spec.ts.

/** A mention of the account of id, as the editor writes it. */
const mentionOf = (id: string) =>
  `<mention-component entity_name="user_mention" entity_identifier="${id}"></mention-component>`;

/** The empty states of the two tabs. */
const NONE = {
  all: "Updates for your subscribed work items will appear here",
  mentions: "Mentions for your work items will appear here",
};

/** Opens the menu of the page's head named menu and toggles its item, then closes it by Escape. */
async function toggleIn(page: Page, menu: "Inbox Filters" | "Display", item: string): Promise<void> {
  await page.getByRole("button", { name: menu, exact: true }).click();
  await page.getByRole("button", { name: item, exact: true }).click();
  await page.keyboard.press("Escape");
}

test("H5 (page): bob's notifications page lists a hundred, then the rest by Load more, and the first hundred again by Refresh; the type filter's pills choose the all tab's senders, which the mentions tab never sends, Clear all clearing them; the unread, archived and snoozed views, the archived and the snoozed never both, a snoozed card without archive and an archived one without snooze; Mark all read sends the tab and filters the page holds; each tab's empty state; a page archived away still offers Load more", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { alice, bob, walt, slug, web } = await notificationWorld(api, testInfo);
  // Fix, alice's, assigned to bob: two notifications of his, as its assignee.
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix", assignee_ids: [bob.id] });
  const fixCreated = (await notificationsSent(api, bob.token, slug, 2)).find((n) => n.data.change.field === "issue");
  // Plan, bob's, renamed by alice: one of his, as its creator.
  const plan = await createIssue(api, bob.token, web.id, { name: "Plan" });
  await updateIssue(api, alice.token, plan.id, { name: "Plan B" });
  await notificationsSent(api, bob.token, slug, 3);
  // Many, alice's, which bob subscribes to once its creation is sent (to walt, its assignee: nerve reads who subscribes
  // as it sends): its hundred labels, added in one change (label_ids names at most 100), send him one each, as a
  // subscriber; her comment on it mentions him, his mentions tab's alone.
  const many = await createIssue(api, alice.token, web.id, { name: "Many", assignee_ids: [walt.id] });
  await notificationsSent(api, walt.token, slug, 2);
  expect(await issueSubscribed(api, bob.token, many.id)).toEqual({ status: 204 });
  const labels = await Promise.all(
    Array.from({ length: 100 }, (_, index) =>
      createLabel(api, alice.token, web.id, { name: `L${String(index + 1).padStart(3, "0")}` })
    )
  );
  await updateIssue(api, alice.token, many.id, { label_ids: labels.map((label) => label.id) });
  await createIssueComment(api, alice.token, many.id, `<p>Hi</p>${mentionOf(bob.id)}`);
  await unreadReach(api, bob.token, slug, { all: 103, mentions: 1 });
  // Fix's creation, snoozed for a day: the snoozed view's.
  const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
  expect((await notificationUpdated(api, bob.token, fixCreated?.id ?? "", { snoozed_till: tomorrow })).status).toBe(
    200
  );
  await unreadReach(api, bob.token, slug, { all: 102, mentions: 1 });

  // Opened anew: the app's start, the workspace's and the all tab's first page, a hundred of the hundred and two, and
  // "Load more"; the tabs and the inbox's link say their unread.
  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  const opened = recordApi(page);
  await page.goto(notificationsPath(slug));
  const cards = cardItems(page);
  const more = page.getByRole("button", { name: "Load more", exact: true });
  await expect(cards).toHaveCount(100);
  await expect(more).toBeVisible();
  const first = "tab=all&archived=false&snoozed=false&limit=100";
  expect(opened()).toEqual({
    pending: 0,
    requests: [
      ...APP,
      ...WORKSPACE.map((request) => valued(request, { [slug]: "{slug}" })),
      listRead(slug, first),
    ].toSorted(),
  });
  const tabs = page.getByRole("tab");
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 99+ unread");
  await expect(tabs.nth(1)).toHaveAccessibleName("Mentions, 1 unread");
  await expect(page.getByRole("link", { name: "Inbox, 103 unread", exact: true })).toBeVisible();

  /** Runs act, then checks that the page sent requests alone, once the list shows count cards. */
  const asked = async (act: () => Promise<void>, count: number, requests: string[]) => {
    const sent = recordApi(page);
    await act();
    await expect(cards).toHaveCount(count);
    await expect.poll(sent).toEqual({ pending: 0, requests: requests.toSorted() });
  };

  // Load more: the rest, by nerve's cursor, and no more "Load more". Refresh: the first hundred again, and the counts.
  const cursorAsked = page.waitForRequest((request) => new URL(request.url()).searchParams.has("cursor"));
  await more.click();
  const cursor = new URL((await cursorAsked).url()).searchParams.get("cursor") ?? "";
  await expect(cards).toHaveCount(102);
  await expect(more).toHaveCount(0);
  await expect(cardSaying(page, "Alice renamed the work item to Plan B")).toHaveCount(1);
  const refreshed = recordApi(page);
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await expect(more).toBeVisible();
  await expect(cards).toHaveCount(100);
  await expect.poll(refreshed).toEqual({ pending: 0, requests: [listRead(slug, first), countRead(slug)].toSorted() });

  // The type filter: assigned to him, then created by him too, each a pill and a read of the list.
  await asked(() => toggleIn(page, "Inbox Filters", "Assigned to me"), 1, [
    listRead(slug, "tab=all&type=assigned&archived=false&snoozed=false&limit=100"),
  ]);
  await asked(() => toggleIn(page, "Inbox Filters", "Created by me"), 2, [
    listRead(slug, "tab=all&type=assigned%2Ccreated&archived=false&snoozed=false&limit=100"),
  ]);
  await expect(page.getByRole("button", { name: /^Remove / })).toHaveText(["Assigned to me", "Created by me"]);
  // Mark all read sends the tab and the types: the two marked, said.
  const markAll = page.getByRole("button", { name: "Mark all as read", exact: true });
  const markAllPath = `/api/v0/workspaces/${slug}/notifications/mark-all-read`;
  const markedTyped = await sentTo(page, "POST", markAllPath, () => markAll.click());
  expect(markedTyped.body).toEqual({ tab: "all", type: ["assigned", "created"], archived: false, snoozed: false });
  await expect(toast(page, "2 notifications marked as read")).toBeVisible();
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 99+ unread");

  // The mentions tab: no type filter, no pill, no type sent, in the list's read nor in Mark all read.
  await asked(() => page.getByRole("tab", { name: "Mentions, 1 unread", exact: true }).click(), 1, [
    listRead(slug, "tab=mentions&archived=false&snoozed=false&limit=100"),
  ]);
  await expect(page.getByRole("button", { name: "Inbox Filters", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^Remove / })).toHaveCount(0);
  const markedMentions = await sentTo(page, "POST", markAllPath, () => markAll.click());
  expect(markedMentions.body).toEqual({ tab: "mentions", archived: false, snoozed: false });
  await expect(toast(page, "1 notification marked as read")).toBeVisible();
  await expect(tabs.nth(1)).toHaveAccessibleName("Mentions");
  // Unread alone: none, the mentions tab's empty state.
  await asked(() => toggleIn(page, "Display", "Show unread"), 0, [
    listRead(slug, "tab=mentions&read=false&archived=false&snoozed=false&limit=100"),
  ]);
  await expect(page.getByText(NONE.mentions, { exact: true })).toBeVisible();

  // Back in the all tab, its types kept: unread among them, none, its empty state; Clear all lists the hundred unread.
  await asked(() => tabs.nth(0).click(), 0, [
    listRead(slug, "tab=all&type=assigned%2Ccreated&read=false&archived=false&snoozed=false&limit=100"),
  ]);
  await expect(page.getByText(NONE.all, { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: /^Remove / })).toHaveText(["Assigned to me", "Created by me"]);
  await asked(() => page.getByRole("button", { name: "Clear all", exact: true }).click(), 100, [
    listRead(slug, "tab=all&read=false&archived=false&snoozed=false&limit=100"),
  ]);
  await expect(more).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^Remove / })).toHaveCount(0);

  // The snoozed alone: Fix's creation, which offers a snooze and no archive; the archived alone turns it off: none.
  await toggleIn(page, "Display", "Show unread");
  await asked(() => toggleIn(page, "Display", "Show snoozed"), 1, [
    listRead(slug, "tab=all&archived=false&snoozed=true&limit=100"),
  ]);
  const snoozedCard = cardSaying(page, "Alice created the work item");
  await expect(snoozedCard.getByRole("button", { name: `Snooze WEB-${fix.sequence_id} Fix`, exact: true })).toHaveCount(
    1
  );
  await expect(snoozedCard.getByRole("button", { name: /^(Archive|Unarchive) / })).toHaveCount(0);
  await asked(() => toggleIn(page, "Display", "Show archived"), 0, [
    listRead(slug, "tab=all&archived=true&snoozed=false&limit=100"),
  ]);
  await page.getByRole("button", { name: "Display", exact: true }).click();
  await expect(page.getByRole("button", { name: "Show archived", exact: true })).toHaveAttribute(
    "aria-pressed",
    "true"
  );
  await expect(page.getByRole("button", { name: "Show snoozed", exact: true })).toHaveAttribute(
    "aria-pressed",
    "false"
  );
  await page.keyboard.press("Escape");
  await expect(page.getByText(NONE.all, { exact: true })).toBeVisible();

  // The default view's first hundred, every one archived by its card (pressed in the page, all at once): none shown,
  // "Load more" still, which reads the rest. The store sends the hundred changes one at a time, each followed by a
  // read of the counts: a while in Firefox and WebKit.
  await toggleIn(page, "Display", "Show archived");
  await expect(cards).toHaveCount(100);
  const archives = page.getByRole("button", { name: `Archive WEB-${many.sequence_id} Many`, exact: true });
  await archives.evaluateAll((buttons) => buttons.forEach((button) => (button as HTMLButtonElement).click()));
  await expect(cards).toHaveCount(0, { timeout: 30_000 });
  await expect(more).toBeVisible();
  await expect(page.getByText(NONE.all, { exact: true })).toHaveCount(0);
  await asked(() => more.click(), 2, [
    listRead(slug, `tab=all&archived=false&snoozed=false&limit=100&cursor=${encodeURIComponent(cursor)}`),
  ]);
  await expect(page.getByRole("link", { name: "Inbox", exact: true })).toBeVisible();

  // The archived alone: the hundred, each offering unarchive and no snooze.
  await asked(() => toggleIn(page, "Display", "Show archived"), 100, [
    listRead(slug, "tab=all&archived=true&snoozed=false&limit=100"),
  ]);
  const archivedCard = cardSaying(page, "Alice added the label L001");
  await expect(archivedCard.getByRole("button", { name: `Unarchive WEB-${many.sequence_id} Many` })).toHaveCount(1);
  await expect(archivedCard.getByRole("button", { name: /^Snooze / })).toHaveCount(0);
  // Mark all read there sends the tab and the archived view, and no type, none being chosen: the hundred marked.
  const markedArchived = await sentTo(page, "POST", markAllPath, () => markAll.click());
  expect(markedArchived.body).toEqual({ tab: "all", archived: true, snoozed: false });
  await expect(toast(page, "100 notifications marked as read")).toBeVisible();

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
