import type { Locator, Page } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  createProject,
  inviteAndAccept,
  projectMemberWrites,
  projectMembershipOf,
} from "../../fixtures/api";
import { subscriptionRows } from "../../fixtures/assert/notification";
import { expectQuietConsole, watchPage, type PageWatch } from "../../fixtures/browser";
import { dismissToasts, failedLoads, seenBy, toast } from "../../fixtures/cycle-pages";
import { createIssue, issueArchived } from "../../fixtures/issues";
import { detailPath, notificationWorld, person } from "../../fixtures/notification-pages";
import { openCommands, selectCommand } from "../../fixtures/power-k";
import { holdAnswer, moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// H1's page version (M7 design 2, 3.1, 3.10; P4 spec 2.3, 2.6, 2.11, R13, R14): subscribing to a work item and
// unsubscribing from it in a person's browser, by the button of the detail's header and of the peek's, and by the
// command palette (Shift+S, or its command picked). The change is never shown before nerve has made it: while it is
// out the button is out of use and says Loading; once nerve answers, the page reads the work item again (the detail
// withdrawn meanwhile, M4's re-read after a write), and only that read turns the button over; a toast says it. Each is
// a row of the subscriber's, as the API version (h1-subscription.spec.ts) asserts them. Whoever has a role in the
// project and holds the work item's detail may: a guest of a project that shows its guests every work item on
// another's, and a guest who sees only his own on his own, made while he was a member (a guest creates none, access
// rules.go:133). An archived work item offers neither. Removed from the project with the page open, the person's
// change is refused as not found, said, and no row is made. Every request is /api/v0's.

/** The subscription button of within, whichever it says. */
function subscriptionButton(within: Page | Locator): Locator {
  return within.getByRole("button", { name: /^(Subscribe|Unsubscribe|Loading)$/ });
}

/**
 * Presses the subscription button of within, which says from, with nerve's answers to its write (POST path) and to the
 * work item's read that follows (GET read) held each in turn: while the write is out the button says Loading, out of
 * use, and nothing else is sent; once its answer is let through, the read follows, and while that is out the button
 * says to nowhere (the detail withdrawn); once the read is let through, the button says to and a toast says said.
 */
async function toggleHeld(
  page: Page,
  within: Page | Locator,
  watch: PageWatch,
  calls: { write: string; read: string },
  turn: { from: string; to: string; said: string }
): Promise<void> {
  const button = subscriptionButton(within);
  await expect(button).toHaveText(turn.from);
  const mark = watch.apiRequests.length;
  const releaseRead = await holdAnswer(page, "GET", calls.read);
  const { release } = await sentHeld(page, "POST", calls.write, () => button.click());
  await expect(button).toHaveText("Loading");
  await expect(button).toBeDisabled();
  expect(watch.apiRequests.slice(mark)).toEqual([`POST ${calls.write}`]);
  expect((await release()).status()).toBe(204);
  await expect.poll(() => watch.apiRequests.slice(mark)).toEqual([`POST ${calls.write}`, `GET ${calls.read}`]);
  await expect(within.getByRole("button", { name: turn.to, exact: true })).toHaveCount(0);
  await releaseRead();
  await expect(button).toHaveText(turn.to);
  await expect(button).toBeEnabled();
  await expect(toast(page, turn.said)).toBeVisible();
  await dismissToasts(page);
}

const subscribed = "Work item subscribed successfully";
const unsubscribed = "Work item unsubscribed successfully";

test("H1 (page): bob subscribes to alice's Fix the login by its detail's header and unsubscribes, each out of use and Loading while nerve has not answered, then the work item read again before the button turns; Shift+S subscribes him and the palette's command unsubscribes him, each a row of his; removed from Web with the page open, his subscription is refused as not found, said, the detail not found, and no row made", async ({
  api,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await notificationWorld(api, testInfo);
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix the login" });
  const calls = { read: `/api/v0/issues/${fix.id}`, write: `/api/v0/issues/${fix.id}/subscribe` };
  const live = { live: true, hers: true, stamped: true };
  const deleted = { live: false, hers: true, stamped: true };

  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  await page.goto(detailPath(slug, "WEB", fix.sequence_id));
  await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toBeVisible();
  const button = subscriptionButton(page);
  await expect(button).toHaveText("Subscribe");
  expect(await subscriptionRows(db, fix.id, bob.id)).toEqual([]);
  // The header's button, each way.
  await toggleHeld(page, page, watch, calls, { from: "Subscribe", to: "Unsubscribe", said: subscribed });
  expect(await subscriptionRows(db, fix.id, bob.id)).toEqual([live]);
  const unsubscribeCalls = { ...calls, write: `/api/v0/issues/${fix.id}/unsubscribe` };
  await toggleHeld(page, page, watch, unsubscribeCalls, { from: "Unsubscribe", to: "Subscribe", said: unsubscribed });
  expect(await subscriptionRows(db, fix.id, bob.id)).toEqual([deleted]);

  // Shift+S, the focus outside any field: he subscribes again.
  await page.keyboard.press("Shift+S");
  await expect(button).toHaveText("Unsubscribe");
  await expect(toast(page, subscribed)).toBeVisible();
  await dismissToasts(page);
  expect(await subscriptionRows(db, fix.id, bob.id)).toEqual([deleted, live]);
  // The palette names its command by what it does, with its shortcut; picked, he unsubscribes.
  await openCommands(page);
  await expect(page.getByRole("option", { name: /^Unsubscribe from notifications/ })).toBeVisible();
  await selectCommand(page, /^Unsubscribe from notifications/);
  await expect(button).toHaveText("Subscribe");
  await expect(toast(page, unsubscribed)).toBeVisible();
  await dismissToasts(page);
  expect(await subscriptionRows(db, fix.id, bob.id)).toEqual([deleted, deleted]);

  // alice removes bob from Web, his page open: his subscription is refused as not found, which the toast says, and
  // the work item read again is not found either; no row is made.
  const membership = await projectMembershipOf(api, alice.token, web.id, bob.id);
  expect(await projectMemberWrites(api, web.id).remove(alice.token, membership)).toEqual({ status: 204 });
  await button.click();
  await expect(toast(page, "The work item does not exist, or you cannot see it.")).toBeVisible();
  await expect(page.getByText("Work item does not exist", { exact: true })).toBeVisible();
  expect(await subscriptionRows(db, fix.id, bob.id)).toEqual([deleted, deleted]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 POST ${calls.write}`, `404 GET ${calls.read}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 404, 404) });
});

test("H1 (page): bob subscribes to Fix the login by the peek's header on Web's list and unsubscribes; Web's guest gus, who sees every work item, subscribes to alice's; olga, a guest of Ops who sees only her own, unsubscribes from hers, made while she was its member; Old, archived, offers no subscription", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await notificationWorld(api, testInfo);
  const [gus, olga] = await Promise.all([person(api, testInfo, "gus", "Gus"), person(api, testInfo, "olga", "Olga")]);
  await inviteAndAccept(api, alice.token, slug, { email: gus.email, token: gus.token }, 5);
  await inviteAndAccept(api, alice.token, slug, { email: olga.email, token: olga.token }, 15);
  await addProjectMembers(api, alice.token, web.id, [{ member_id: gus.id, role: 5 }]);
  expect((await changeProject(api, alice.token, web.id, { guest_view_all_features: true })).status).toBe(200);
  // Ops, whose guests see only their own: olga makes Her note as its member, then is its guest.
  const ops = await createProject(api, alice.token, slug, { name: "Ops", identifier: "OPS" });
  expect(ops.guest_view_all_features).toBe(false);
  const [olgas] = await addProjectMembers(api, alice.token, ops.id, [{ member_id: olga.id, role: 15 }]);
  const note = await createIssue(api, olga.token, ops.id, { name: "Her note" });
  expect(await projectMemberWrites(api, ops.id).change(alice.token, olgas?.id ?? "", 5)).toEqual({ status: 200 });
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix the login" });
  const old = await createIssue(api, alice.token, web.id, {
    name: "Old",
    state_id: (await stateIds(api, alice.token, web.id)).Done,
  });
  expect((await issueArchived(api, alice.token, old.id)).status).toBe(200);

  // bob's peek of Fix the login, on Web's list: its header's button, each way.
  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects/${web.id}/issues`);
  await page.locator(`#issue-${fix.id}`).click();
  const peek = page.locator("#full-screen-portal");
  await expect(peek.getByRole("button", { name: "Close the peek view", exact: true })).toBeVisible();
  const read = `/api/v0/issues/${fix.id}`;
  await toggleHeld(
    page,
    peek,
    watch,
    { read, write: `${read}/subscribe` },
    {
      from: "Subscribe",
      to: "Unsubscribe",
      said: subscribed,
    }
  );
  await toggleHeld(
    page,
    peek,
    watch,
    { read, write: `${read}/unsubscribe` },
    {
      from: "Unsubscribe",
      to: "Subscribe",
      said: unsubscribed,
    }
  );
  expect(await subscriptionRows(db, fix.id, bob.id)).toEqual([{ live: false, hers: true, stamped: true }]);
  // Old, archived: its detail offers no subscription.
  await moveWithinApp(page, detailPath(slug, "WEB", old.sequence_id));
  await expect(page.getByText("Old", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Copy link", exact: true })).toBeVisible();
  await expect(subscriptionButton(page)).toHaveCount(0);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);

  // gus, Web's guest, who sees every work item, subscribes to alice's Fix the login.
  await seenBy(browser, baseURL ?? "", api, gus.email, detailPath(slug, "WEB", fix.sequence_id), async (seen) => {
    await subscriptionButton(seen).click();
    await expect(subscriptionButton(seen)).toHaveText("Unsubscribe");
    await expect(toast(seen, subscribed)).toBeVisible();
  });
  expect(await subscriptionRows(db, fix.id, gus.id)).toEqual([{ live: true, hers: true, stamped: true }]);
  // olga, Ops's guest, who sees only her own, unsubscribes from Her note, which she made as its member.
  await seenBy(browser, baseURL ?? "", api, olga.email, detailPath(slug, "OPS", note.sequence_id), async (seen) => {
    await expect(subscriptionButton(seen)).toHaveText("Unsubscribe");
    await subscriptionButton(seen).click();
    await expect(subscriptionButton(seen)).toHaveText("Subscribe");
    await expect(toast(seen, unsubscribed)).toBeVisible();
  });
  expect(await subscriptionRows(db, note.id, olga.id)).toEqual([{ live: false, hers: true, stamped: true }]);
  await expectQuietConsole(page, watch);
});
