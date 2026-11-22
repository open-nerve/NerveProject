import {
  addProjectMembers,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { notificationStates } from "../../fixtures/assert/notification";
import { anotherTabSignsIn } from "../../fixtures/auth";
import { expectQuietConsole, refuseClipboardWrites, watchPage } from "../../fixtures/browser";
import { dismissToasts, toast } from "../../fixtures/cycle-pages";
import { shownWithin } from "../../fixtures/dialogs";
import { createIssue, issueSubscribed, updateIssue } from "../../fixtures/issues";
import {
  cardItems,
  cardSaying,
  countRead,
  expectNothingSelected,
  notificationWorld,
  notificationsPath,
  shownCards,
  unreadReach,
} from "../../fixtures/notification-pages";
import { notificationsSent } from "../../fixtures/notifications";
import { holdAnswer, moveWithinApp, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H20's page version for the notifications page's late answers (M7 design 2, H20; P4 spec 2.3, 2.5, 2.6, 2.11, R3,
// R20's W3, W5, W6; v0 design 7.7): what the page sent belongs to its session and its workspace. A move Back while a
// card's read is out leaves nothing of it behind, Forward finding nothing selected and no peek (W6). A workspace's list,
// its next page and its counts answered late, after a move to another workspace, enter nothing of the other's list,
// "Load more", tabs or dot (W3, W5); back, the first workspace's are read again. Answers late for a session another
// tab ended say nothing and change nothing on the other account's page. The work item beside the list unsubscribes and
// subscribes by its header's button, and its link copied with the clipboard refused says the copy failed. Every
// request is /api/v0's. The keys and names are h20-notification-pages.spec.ts's.

test("H20 (page): bob's card read while he goes Back leaves no peek and nothing selected Forward; the peek beside the list unsubscribes and subscribes him, and its link copied with the clipboard refused says it failed; Acme's late first page, next page and counts, after a move to Beta, change nothing of Beta's, and back in Acme are read again; his read and refresh answered after alice signs in in another tab say and change nothing on her page", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, walt, slug, web } = await notificationWorld(api, testInfo);
  // Fix, alice's, assigned to bob: his two; Many, alice's, walt's, which bob subscribes to once its creation is sent:
  // its hundred labels, his hundred more, the newest.
  const fix = await createIssue(api, alice.token, web.id, { name: "Fix", assignee_ids: [bob.id] });
  await notificationsSent(api, bob.token, slug, 2);
  const many = await createIssue(api, alice.token, web.id, { name: "Many", assignee_ids: [walt.id] });
  await notificationsSent(api, walt.token, slug, 2);
  expect(await issueSubscribed(api, bob.token, many.id)).toEqual({ status: 204 });
  const labels = await Promise.all(
    Array.from({ length: 100 }, (_, index) => createLabel(api, alice.token, web.id, { name: `L${index + 1}` }))
  );
  await updateIssue(api, alice.token, many.id, { label_ids: labels.map((label) => label.id) });
  // bob's Review, assigned to alice: her two in Acme.
  await createIssue(api, bob.token, web.id, { name: "Review", assignee_ids: [alice.id] });
  // Beta, alice's other workspace, of which bob is a member: Core's Plan, his two there.
  const beta = await createWorkspace(api, alice.token, { name: "Beta", slug: slugFor(testInfo, "beta") });
  await inviteAndAccept(api, alice.token, beta.slug, { email: bob.email, token: bob.token }, 15);
  const core = await createProject(api, alice.token, beta.slug, { name: "Core", identifier: "CORE" });
  await addProjectMembers(api, alice.token, core.id, [{ member_id: bob.id, role: 15 }]);
  await createIssue(api, alice.token, core.id, { name: "Plan", assignee_ids: [bob.id] });
  await Promise.all([
    unreadReach(api, bob.token, slug, { all: 102, mentions: 0 }),
    unreadReach(api, alice.token, slug, { all: 2, mentions: 0 }),
    unreadReach(api, bob.token, beta.slug, { all: 2, mentions: 0 }),
  ]);
  const notes = (await notificationsSent(api, bob.token, slug, 100, { limit: 100 })).filter(
    (n) => n.issue_id === many.id
  );
  const noteOf = (label: string) => notes.find((n) => n.data.change.new_value === label)?.id ?? "";
  // the labels whose cards no other card's text holds (L1's is in L10's to L19's, and L100's)
  const [l1, l2] = [noteOf("L100"), noteOf("L99")];

  // From Web's list to the notifications page; L100's card opened, its peek shown, nerve's answer to its read held;
  // Back to Web's list, then the answer let through: nothing said, no peek; Forward: nothing selected, no peek.
  const page = await signedInPage(bob.tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects/${web.id}/issues`);
  await expect(page.locator(`#issue-${fix.id}`)).toBeVisible();
  await moveWithinApp(page, notificationsPath(slug));
  const cards = cardItems(page);
  const more = page.getByRole("button", { name: "Load more", exact: true });
  await expect(cards).toHaveCount(100);
  const peekClose = page.getByRole("button", { name: "Close the peek view", exact: true });
  // L98's peek opened and closed first, its modules loaded before any route: WebKit fetches again what it preloaded once
  // a route is installed, and reports it unused (g11, i13)
  await cardSaying(page, "Alice added the label L98").getByRole("button").first().click();
  await peekClose.click();
  await expect(peekClose).toHaveCount(0);
  const l1Card = cardSaying(page, "Alice added the label L100");
  const opened = await sentHeld(page, "PATCH", `/api/v0/notifications/${l1}`, () =>
    l1Card.getByRole("button").first().click()
  );
  await expect(peekClose).toBeVisible();
  await page.goBack();
  await expect(page.locator(`#issue-${fix.id}`)).toBeVisible();
  expect((await opened.release()).status()).toBe(200);
  expect(await shownWithin(page.getByLabel("Notifications", { exact: true }).getByRole("dialog"))).toBe(false);
  await expect(peekClose).toHaveCount(0);
  await page.goForward();
  await expect(cards).toHaveCount(100);
  await expect(peekClose).toHaveCount(0);
  await expectNothingSelected(page);
  expect(await notificationStates(db, [l1])).toEqual([
    { read: true, archived: false, snoozed: false, by_receiver: true },
  ]);

  // L100's card again: the peek beside the list unsubscribes him from Many and subscribes him again; its link, the
  // clipboard refused, is said not copied.
  await l1Card.getByRole("button").first().click();
  const subscription = page.getByRole("button", { name: /^(Subscribe|Unsubscribe|Loading)$/ });
  await subscription.click();
  await expect(subscription).toHaveText("Subscribe");
  await expect(toast(page, "Work item unsubscribed successfully")).toBeVisible();
  await subscription.click();
  await expect(subscription).toHaveText("Unsubscribe");
  await expect(toast(page, "Work item subscribed successfully")).toBeVisible();
  await refuseClipboardWrites(page);
  await page.getByRole("button", { name: "Copy link", exact: true }).click();
  await expect(toast(page, "Something went wrong. Please try again.")).toBeVisible();
  expect(await shownWithin(toast(page, "Work item link copied to clipboard"))).toBe(false);
  await peekClose.click();
  await dismissToasts(page);

  // Acme's next page, then its first page and counts by Refresh, each answer held; a move to Beta's notifications
  // page, which shows Beta's two and its counts; Acme's answers let through: Beta's list, "Load more", tabs and dot as
  // they were. Back in Acme, its list and counts read again.
  const acmeList = `/api/v0/workspaces/${slug}/notifications`;
  const releaseList = await holdAnswer(page, "GET", acmeList, 2);
  const releaseCount = await holdAnswer(page, "GET", `${acmeList}/unread-count`);
  const sent = watch.apiRequests.length;
  await more.click();
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await expect
    .poll(() => watch.apiRequests.slice(sent).filter((request) => request.startsWith(`GET ${acmeList}`)))
    .toEqual([`GET ${acmeList}`, `GET ${acmeList}`, countRead(slug)]);
  await moveWithinApp(page, notificationsPath(beta.slug));
  const betaCards = ["Alice assigned Bob", "Alice created the work item"];
  const tabs = page.getByRole("tab");
  await expect.poll(async () => (await shownCards(page)).map((card) => card.split(" | ")[2])).toEqual(betaCards);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 2 unread");
  await expect(page.getByRole("link", { name: "Inbox, 2 unread", exact: true })).toBeVisible();
  const answered = page.waitForResponse((r) => new URL(r.url()).pathname === `${acmeList}/unread-count`);
  await releaseList();
  await releaseCount();
  await answered;
  expect(await shownWithin(more)).toBe(false);
  expect((await shownCards(page)).map((card) => card.split(" | ")[2])).toEqual(betaCards);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 2 unread");
  await expect(page.getByRole("link", { name: "Inbox, 2 unread", exact: true })).toBeVisible();
  const reread = watch.apiRequests.length;
  await page.goBack();
  await expect(cards).toHaveCount(100);
  await expect(more).toBeVisible();
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 99+ unread");
  await expect.poll(() => watch.apiRequests.slice(reread)).toEqual(expect.arrayContaining([`GET ${acmeList}`]));

  // L99's card marked read and the list refreshed, nerve's answers held; another tab signs alice in, whose page this
  // becomes: her two. bob's answers let through: nothing said, her cards and counts as they were; L99 is read.
  const l2Card = cardSaying(page, "Alice added the label L99");
  await l2Card.hover();
  const marked = await sentHeld(page, "PATCH", `/api/v0/notifications/${l2}`, () =>
    l2Card.getByRole("button", { name: `Mark WEB-${many.sequence_id} Many read`, exact: true }).click()
  );
  const refreshed = await sentHeld(page, "GET", acmeList, () =>
    page.getByRole("button", { name: "Refresh", exact: true }).click()
  );
  await (await anotherTabSignsIn(context, api, alice.email)).close();
  const hers = ["Bob assigned Alice", "Bob created the work item"];
  await expect.poll(async () => (await shownCards(page)).map((card) => card.split(" | ")[2])).toEqual(hers);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 2 unread");
  expect((await marked.release()).status()).toBe(200);
  expect((await refreshed.release()).status()).toBe(200);
  expect(await shownWithin(toast(page, "Notification marked as read"))).toBe(false);
  expect((await shownCards(page)).map((card) => card.split(" | ")[2])).toEqual(hers);
  await expect(tabs.nth(0)).toHaveAccessibleName("All, 2 unread");
  expect(await notificationStates(db, [l2])).toEqual([
    { read: true, archived: false, snoozed: false, by_receiver: true },
  ]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
