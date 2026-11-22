import type { Locator, Page } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { createIssueComment } from "../../fixtures/comments";
import { recordApi, seenBy, toast } from "../../fixtures/cycle-pages";
import { sentWithin } from "../../fixtures/dialogs";
import { createIssue, issueSubscribed, updateIssue } from "../../fixtures/issues";
import {
  cardSaying,
  countRead,
  notificationWorld,
  notificationsPath,
  person,
  unreadReach,
} from "../../fixtures/notification-pages";
import { notificationUpdated, notificationsSent } from "../../fixtures/notifications";
import { moveWithinApp } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";
import { switchWorkspace } from "../../fixtures/workspace-pages";

// H7's page version (M7 design 2, 3.4; P4 spec 2.4, 2.6, 2.11, R5, R6, re-93 m12): the caller's unread notifications
// of the route's workspace, as the top navigation's inbox shows them on every page of it, a dot and its link's name,
// and as the notifications page's two tabs say them: the all tab's and the mentions tab's, neither counting the
// archived nor the snoozed. Every page of a workspace reads them once as it opens; the notifications page reads them
// again as it mounts, the same read; each of the caller's own changes, Refresh and Mark all read read them again. Nothing
// else does: the browser's tab regaining focus or showing again reads nothing (R6), so another's change is shown only
// then. Each workspace has its own; a guest sees his. After a move within the app the story waits for what the page
// shows, not counting the reads, which SWR's deduping decides (m12). The API version is h5-h7-notification-inbox.spec.ts.

/** The inbox's link of the top navigation, which says the unread of the route's workspace in its name. */
function inbox(page: Page, unread: number): Locator {
  return page.getByRole("link", { name: unread === 0 ? "Inbox" : `Inbox, ${unread} unread`, exact: true });
}

/** The dot of the inbox's link, shown while there is an unread notification. */
function dotOf(link: Locator): Locator {
  return link.locator('[class*="bg-danger-primary"]');
}

test("H7 (page): bob's inbox dot and the notifications page's tabs say his unread in Acme, the archived and the snoozed left out, read once as a page opens and again as the notifications page mounts, after each of his changes, Refresh and Mark all read, never as the browser's tab regains focus; Beta has its own; gus, a guest, sees his", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await notificationWorld(api, testInfo);
  // Fix, alice's, assigned to bob: his two; her comment, archived by him, and its priority, snoozed by him, uncounted;
  // her mention of him, his mentions tab's.
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix", assignee_ids: [bob.id] });
  await notificationsSent(api, bob.token, slug, 2);
  await createIssueComment(api, alice.token, fix.id, "<p>Looks good</p>");
  const [looks] = await notificationsSent(api, bob.token, slug, 3);
  await updateIssue(api, alice.token, fix.id, { priority: "high" });
  const [priority] = await notificationsSent(api, bob.token, slug, 4);
  const mention = `<mention-component entity_name="user_mention" entity_identifier="${bob.id}"></mention-component>`;
  await createIssueComment(api, alice.token, fix.id, `<p>Hi</p>${mention}`);
  await notificationsSent(api, bob.token, slug, 1, { tab: "mentions" });
  const day = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
  expect((await notificationUpdated(api, bob.token, looks?.id ?? "", { archived: true })).status).toBe(200);
  expect((await notificationUpdated(api, bob.token, priority?.id ?? "", { snoozed_till: day })).status).toBe(200);
  await unreadReach(api, bob.token, slug, { all: 2, mentions: 1 });
  // Beta, alice's other workspace, of which bob is a member: Core's Plan, his, two there.
  const beta = await createWorkspace(api, alice.token, { name: "Beta", slug: slugFor(testInfo, "beta") });
  await inviteAndAccept(api, alice.token, beta.slug, { email: bob.email, token: bob.token }, 15);
  const core = await createProject(api, alice.token, beta.slug, { name: "Core", identifier: "CORE" });
  await addProjectMembers(api, alice.token, core.id, [{ member_id: bob.id, role: 15 }]);
  await createIssue(api, alice.token, core.id, { name: "Plan", assignee_ids: [bob.id] });
  await unreadReach(api, bob.token, beta.slug, { all: 2, mentions: 0 });

  // Web's list, opened anew: one read of the counts among the page's; the dot, and the link's name, say three. The
  // page's clock is its own, so that the story can let SWR's deduping interval pass without waiting it out.
  const page = await signedInPage(bob.tokens);
  await page.clock.install();
  const watch = await watchPage(page);
  const opened = recordApi(page);
  await page.goto(`/${slug}/projects/${web.id}/issues`);
  await expect(page.locator(`#issue-${fix.id}`)).toBeVisible();
  await expect(dotOf(inbox(page, 3))).toBeVisible();
  expect(opened().requests.filter((request) => request === countRead(slug))).toEqual([countRead(slug)]);

  // alice changes Fix's state: four for nerve, still three on the page, its tab regaining focus and showing again
  // reading nothing, SWR's deduping interval long past.
  await updateIssue(api, alice.token, fix.id, { state_id: (await stateIds(api, alice.token, web.id)).Todo });
  await unreadReach(api, bob.token, slug, { all: 3, mentions: 1 });
  await page.clock.fastForward("00:05");
  const focused = await sentWithin(page, "GET", `/api/v0/workspaces/${slug}/notifications/unread-count`, () =>
    page.evaluate(() => {
      window.dispatchEvent(new Event("focus"));
      document.dispatchEvent(new Event("visibilitychange"));
    })
  );
  expect(focused).toBe(false);
  await expect(inbox(page, 3)).toBeVisible();

  // The inbox's link: the notifications page, its tabs read the counts as they mount, four now.
  await inbox(page, 3).click();
  const tabs = page.getByRole("tab");
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 3 unread");
  await expect(tabs.nth(1)).toHaveAccessibleName("Mentions, 1 unread");
  await expect(dotOf(inbox(page, 4))).toBeVisible();

  // His own changes: Fix's assignment read, then unread; its state's archived.
  const assigned = cardSaying(page, "Alice assigned Bob");
  await assigned.hover();
  await assigned.getByRole("button", { name: `Mark WEB-${fix.sequence_id} Fix read`, exact: true }).click();
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 2 unread");
  await expect(inbox(page, 3)).toBeVisible();
  await assigned.getByRole("button", { name: `Mark WEB-${fix.sequence_id} Fix unread`, exact: true }).click();
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 3 unread");
  const stateCard = cardSaying(page, "Alice set the state to Todo");
  await stateCard.hover();
  await stateCard.getByRole("button", { name: `Archive WEB-${fix.sequence_id} Fix`, exact: true }).click();
  await expect(stateCard).toHaveCount(0);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 2 unread");
  await expect(inbox(page, 3)).toBeVisible();

  // alice's comment: nothing on the page until Refresh, which reads the counts too.
  await createIssueComment(api, alice.token, fix.id, "<p>Ship it</p>");
  await unreadReach(api, bob.token, slug, { all: 3, mentions: 1 });
  await expect(cardSaying(page, "Alice commented")).toHaveCount(0);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 2 unread");
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await expect(cardSaying(page, "Alice commented")).toHaveCount(1);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 3 unread");
  await expect(inbox(page, 4)).toBeVisible();

  // Mark all read, the all tab's: its count gone, the mention's left; then the mentions tab's: the dot gone.
  await page.getByRole("button", { name: "Mark all as read", exact: true }).click();
  await expect(toast(page, "3 notifications marked as read")).toBeVisible();
  await expect(tabs.nth(0)).toHaveAccessibleName("All");
  await expect(tabs.nth(1)).toHaveAccessibleName("Mentions, 1 unread");
  await expect(dotOf(inbox(page, 1))).toBeVisible();
  await tabs.nth(1).click();
  await page.getByRole("button", { name: "Mark all as read", exact: true }).click();
  await expect(toast(page, "1 notification marked as read")).toBeVisible();
  await expect(tabs.nth(1)).toHaveAccessibleName("Mentions");
  await expect(inbox(page, 0)).toBeVisible();
  await expect(dotOf(inbox(page, 0))).toHaveCount(0);

  // Beta, through the workspace menu: its own two; back in Acme, none.
  await switchWorkspace(page, beta.id);
  await expect(dotOf(inbox(page, 2))).toBeVisible();
  await moveWithinApp(page, notificationsPath(slug));
  await expect(page.getByRole("tab").nth(1)).toHaveAccessibleName("Mentions");
  await expect(dotOf(inbox(page, 0))).toHaveCount(0);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);

  // gus, a guest of Acme and of Web, who sees every work item there, subscribed to Fix: alice's change of it is his
  // one unread, which his notifications page, opened anew, reads once with the page's.
  const gus = await person(api, testInfo, "gus", "Gus");
  await inviteAndAccept(api, alice.token, slug, { email: gus.email, token: gus.token }, 5);
  await addProjectMembers(api, alice.token, web.id, [{ member_id: gus.id, role: 5 }]);
  expect((await changeProject(api, alice.token, web.id, { guest_view_all_features: true })).status).toBe(200);
  expect(await issueSubscribed(api, gus.token, fix.id)).toEqual({ status: 204 });
  await updateIssue(api, alice.token, fix.id, { priority: "low" });
  await unreadReach(api, gus.token, slug, { all: 1, mentions: 0 });
  await seenBy(browser, baseURL ?? "", api, gus.email, notificationsPath(slug), async (seen, seenWatch) => {
    await expect(dotOf(inbox(seen, 1))).toBeVisible();
    expect(seenWatch.apiRequests.filter((request) => request === countRead(slug))).toEqual([countRead(slug)]);
    await expect(seen.getByRole("tab").nth(0)).toHaveAccessibleName("All, 1 unread");
    await expect(cardSaying(seen, "Alice set the priority to Low")).toHaveCount(1);
  });
});
